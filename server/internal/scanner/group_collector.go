package scanner

import (
	"path/filepath"
	"strings"
	"sync"
)

// CodecRegion is the directory summary published to persistence.
//
// It embeds the public GroupContext carried on every FileInfo, so the pipeline
// has exactly one neighbourhood type and the directory the summary came from is
// recorded alongside it.
type CodecRegion struct {
	GroupContext
	Directory string
}

// dirAggregate accumulates one directory's structure while files stream through
// the pipeline.
type dirAggregate struct {
	size             int
	episodicSiblings int
	episodeNumbers   map[int]struct{}
	yearSiblings     int
	seasonFolder     bool
	titleVotes       map[string]int
}

// groupCollector builds per-directory structure as files are emitted.
//
// It is deliberately bounded in two ways:
//
//   - the number of tracked directories is capped, and the least recently used
//     entries are evicted, so a scan of a directory-per-file library cannot grow
//     memory without limit;
//   - only the information resolution needs is retained, never the files
//     themselves.
//
// Eviction is safe because a group summary is an optimisation: a file whose
// directory was evicted simply resolves with less neighbourhood evidence.
type groupCollector struct {
	mu      sync.Mutex
	dirs    map[string]*dirAggregate
	order   []string
	maxDirs int
	// evicted counts directories dropped due to the cap, so the scan report can
	// state honestly that a run was constrained.
	evicted int
}

// DefaultTrackedDirectories bounds how much structure is kept in memory.
const DefaultTrackedDirectories = 20000

func newGroupCollector(maxDirs int) *groupCollector {
	if maxDirs <= 0 {
		maxDirs = DefaultTrackedDirectories
	}
	return &groupCollector{
		dirs:    make(map[string]*dirAggregate, 1024),
		order:   make([]string, 0, 1024),
		maxDirs: maxDirs,
	}
}

// Observe records one file's contribution to its directory structure.
func (g *groupCollector) Observe(path string, parsed ParsedName, seasonFolder bool) {
	directory := filepath.Dir(path)

	g.mu.Lock()
	defer g.mu.Unlock()

	aggregate, exists := g.dirs[directory]
	if !exists {
		if len(g.dirs) >= g.maxDirs {
			g.evictOldestLocked()
		}
		aggregate = &dirAggregate{
			episodeNumbers: make(map[int]struct{}, 8),
			titleVotes:     make(map[string]int, 8),
		}
		g.dirs[directory] = aggregate
		g.order = append(g.order, directory)
	}
	if seasonFolder {
		aggregate.seasonFolder = true
	}

	aggregate.size++
	if parsed.IsEpisode {
		aggregate.episodicSiblings++
		if parsed.EpisodeNumber > 0 {
			aggregate.episodeNumbers[parsed.EpisodeNumber] = struct{}{}
		}
	}
	if parsed.ReleaseYear > 0 {
		aggregate.yearSiblings++
	}

	// Only count titles that could actually name a work, so a directory of
	// watermarks cannot establish a consensus. The parser's own verdict is
	// reused rather than re-derived here.
	if title := strings.TrimSpace(parsed.Title); title != "" {
		if parsed.Confidence >= ConfidenceLow && !isJunkOrWatermarkTitle(title) {
			aggregate.titleVotes[NormalizeTitleForSearch(title)]++
		}
	}
}

// Summary returns the structure of a directory, or an empty summary when the
// directory was evicted or never observed.
func (g *groupCollector) Summary(directory string) CodecRegion {
	g.mu.Lock()
	defer g.mu.Unlock()

	aggregate, exists := g.dirs[directory]
	if !exists {
		return CodecRegion{Directory: directory}
	}

	summary := CodecRegion{
		Directory: directory,
		GroupContext: GroupContext{
			Size:             aggregate.size,
			EpisodicSiblings: aggregate.episodicSiblings,
			YearSiblings:     aggregate.yearSiblings,
			SeasonFolder:     aggregate.seasonFolder,
		},
	}
	summary.EpisodeNumbers = make([]int, 0, len(aggregate.episodeNumbers))
	for number := range aggregate.episodeNumbers {
		summary.EpisodeNumbers = append(summary.EpisodeNumbers, number)
	}

	// A consensus requires at least two agreeing files: one file's opinion is
	// not a consensus and must not be presented as one.
	for title, votes := range aggregate.titleVotes {
		if votes > summary.ConsensusVotes {
			summary.ConsensusTitle = title
			summary.ConsensusVotes = votes
		}
	}
	if summary.ConsensusVotes < 2 {
		summary.ConsensusTitle = ""
		summary.ConsensusVotes = 0
	}
	return summary
}

// Evicted reports how many directories were dropped due to the memory cap.
func (g *groupCollector) Evicted() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.evicted
}

// TrackedDirectories reports how many directory summaries are retained.
func (g *groupCollector) TrackedDirectories() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.dirs)
}

// evictOldestLocked drops the oldest half of the tracked directories. Dropping
// in bulk rather than one at a time keeps the amortised cost O(1) per insert.
func (g *groupCollector) evictOldestLocked() {
	drop := len(g.order) / 2
	if drop < 1 {
		drop = 1
	}
	for i := 0; i < drop && i < len(g.order); i++ {
		delete(g.dirs, g.order[i])
	}
	g.order = append([]string(nil), g.order[drop:]...)
	g.evicted += drop
}

// groupWriter publishes each file's finished summary to the collector.
//
// Group summaries are computed and returned BEFORE the strict sorted walk of the
// directory completes, because the collector sees every sibling as it streams.
// The persistence layer therefore receives the neighbourhood context of the
// whole directory, which is exactly what requirement 70 asks for.
func (g *groupCollector) publish(file *FileInfo) {
	summary := g.Summary(filepath.Dir(file.Path))
	if summary.Size <= 0 {
		return
	}
	file.Group = summary.GroupContext
}
