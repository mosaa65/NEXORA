package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Options configures a Scanner. Every field has a working default, so
// `New(Options{})` is always safe.
type Options struct {
	// Workers is the size of the metadata/parse worker pool. It bounds
	// concurrent stat calls, which is the actual I/O pressure on a disk.
	Workers int
	// Extensions overrides the accepted video extensions.
	Extensions []string
	// QueueSize bounds the candidate channel. Backpressure is intentional: an
	// unbounded queue is how a scanner exhausts memory on a huge library.
	QueueSize int
	// Discovery controls traversal policy (symlinks, ignore rules, depth).
	Discovery DiscoveryConfig
	// ProgressInterval is how often aggregated progress is logged. Per-file
	// logging is never used: a million-file scan must not emit a million lines.
	ProgressInterval time.Duration
	// SampleErrorLimit bounds how many example errors are retained.
	SampleErrorLimit int
	// RootWorkers limits how many roots are scanned concurrently. Each root
	// already uses Workers goroutines, so this multiplies I/O parallelism.
	RootWorkers int
	// GroupTrackingCap bounds how many directory summaries are retained for
	// neighbourhood evidence. Zero uses DefaultTrackedDirectories.
	GroupTrackingCap int
	// Control, when set, exposes cooperative pause/resume/cancel and live
	// per-worker state for the duration of the run.
	Control *ScanControlHandle
}

// Scanner turns a media filesystem into a streaming inventory.
type Scanner struct {
	extensions map[string]struct{}
	workers    int
	queueSize  int
	options    Options
	roots      []string
}

// FileInfo is the scanner's output for one accepted media file. It carries
// filesystem metadata, the parse result, artwork, and the change decision, so
// the persistence layer needs no second pass over the disk.
type FileInfo struct {
	Path        string     `json:"path"`
	Size        int64      `json:"size"`
	ModTime     time.Time  `json:"modTime"`
	Parsed      ParsedName `json:"parsed"`
	Extension   string     `json:"extension"`
	ArtworkPath string     `json:"artworkPath,omitempty"`

	// Identity and change detection.
	RootID      string      `json:"rootId,omitempty"`
	Root        string      `json:"root,omitempty"`
	FileID      string      `json:"fileId,omitempty"`
	Fingerprint Fingerprint `json:"fingerprint"`
	Change      ChangeKind  `json:"change"`
	KnownID     int64       `json:"knownId,omitempty"`
	RenamedFrom string      `json:"renamedFrom,omitempty"`
	State       FileState   `json:"state"`

	// Search-ready derived data.
	SearchTokens []string `json:"searchTokens,omitempty"`
	RelativePath string   `json:"relativePath,omitempty"`

	// PathSegments is the hierarchy below the media root, outermost first. Entity
	// resolution classifies from segments rather than from a raw path string, so a
	// folder like "/NotMovies/" cannot be misread.
	PathSegments []string `json:"pathSegments,omitempty"`

	// Context is the meaning extracted from the path hierarchy: which folder names
	// the work, which folder declares the season, and the category and origin.
	Context PathContext `json:"context,omitempty"`

	// Group summarises the sibling files in the same directory. It is the evidence
	// that lets a folder of bare numbers resolve as one season.
	Group GroupContext `json:"group,omitempty"`
}

// PathContext is the labelled folder hierarchy for one file.
type PathContext struct {
	// WorkTitle is the folder most likely naming the work.
	WorkTitle string `json:"workTitle,omitempty"`
	// SeasonNumber when a folder declares one.
	SeasonNumber int `json:"seasonNumber,omitempty"`
	// SeasonFolderTitle is the season folder name, which sometimes still carries
	// the work name ("Silo الموسم الثالث").
	SeasonFolderTitle string `json:"seasonFolderTitle,omitempty"`
	// Category is the content category decided from the hierarchy.
	Category string `json:"category,omitempty"`
	// Origin is the production-origin hint decided from the hierarchy.
	Origin string `json:"origin,omitempty"`
}

// GroupContext mirrors the identity package's neighbourhood summary without
// importing it, because the scanner must not depend on resolution logic.
type GroupContext struct {
	Size             int    `json:"size"`
	EpisodicSiblings int    `json:"episodicSiblings"`
	EpisodeNumbers   []int  `json:"episodeNumbers,omitempty"`
	YearSiblings     int    `json:"yearSiblings"`
	SeasonFolder     bool   `json:"seasonFolder"`
	ConsensusTitle   string `json:"consensusTitle,omitempty"`
	ConsensusVotes   int    `json:"consensusVotes"`
}

var imageExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
	".jfif": {},
	".bmp":  {},
	".tif":  {},
	".tiff": {},
}

var defaultVideoExtensions = []string{
	".mp4", ".mkv", ".avi", ".mov", ".wmv", ".m4v", ".webm",
	".ts", ".m2ts", ".mpg", ".mpeg", ".flv", ".vob", ".iso", ".rmvb", ".3gp",
}

// New creates a Scanner from options, applying defaults.
func New(options Options) *Scanner {
	extensions := options.Extensions
	if len(extensions) == 0 {
		extensions = defaultVideoExtensions
	}

	extensionSet := make(map[string]struct{}, len(extensions))
	for _, extension := range extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		extensionSet[extension] = struct{}{}
	}

	workers := options.Workers
	if workers <= 0 {
		workers = 4
	}
	queueSize := options.QueueSize
	if queueSize <= 0 {
		queueSize = workers * 32
	}
	if queueSize < 64 {
		queueSize = 64
	}

	return &Scanner{
		extensions: extensionSet,
		workers:    workers,
		queueSize:  queueSize,
		options:    options,
	}
}

// SetConfiguredRoots records the operator-configured media roots so background
// reconciliation can run without every caller passing the list again.
func (s *Scanner) SetConfiguredRoots(roots []string) {
	s.roots = append([]string{}, roots...)
}

// ConfiguredRoots returns the media roots the scanner was configured with.
func (s *Scanner) ConfiguredRoots() []string {
	return append([]string{}, s.roots...)
}

// IsVideoFile reports whether a path has an accepted video extension.
func (s *Scanner) IsVideoFile(path string) bool {
	_, ok := s.extensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

// Extension returns the lowercased extension of an accepted file.
func (s *Scanner) Extension(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// ParsePath parses a single file into a FileInfo without scanning a tree.
func (s *Scanner) ParsePath(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	if info.IsDir() {
		return FileInfo{}, fs.ErrInvalid
	}
	return s.fileInfo(path, info, "", nil), nil
}

// Scan collects every media file under the roots into one slice.
//
// This exists for small libraries, tests and tooling. It deliberately loads the
// whole result into memory, which is why the pipeline itself uses Walk and the
// API uses WalkStream: a million-file scan must never call this.
func (s *Scanner) Scan(ctx context.Context, roots []string) ([]FileInfo, error) {
	files := make([]FileInfo, 0, 1024)
	err := s.Walk(ctx, roots, func(file FileInfo) error {
		files = append(files, file)
		return nil
	})
	return files, err
}

// Walk streams every discovered media file to `emit` and returns an error only
// when the scan as a whole could not be set up (no roots) or the context was
// cancelled. Per-file and per-directory failures are collected in the summary
// and never abort the run — that is what makes a single locked folder harmless.
//
// `emit` is invoked from one goroutine, so callers may batch into a database
// without adding their own locking.
func (s *Scanner) Walk(ctx context.Context, roots []string, emit func(FileInfo) error) error {
	// Walk always behaves as a full scan: callers of this API expect every file.
	_, err := s.ScanTree(ctx, roots, ScanOptions{Mode: ModeFull, EmitUnchanged: true}, emit)
	return err
}

// WalkWithKnown performs an incremental scan against previously indexed files.
// WalkWithKnown performs an incremental scan against previously indexed files
// and returns the full report, including the records that were not observed.
//
// It is the convenience wrapper over ScanTree for callers that only need the
// report and want unchanged files filtered out.
func (s *Scanner) WalkWithKnown(
	ctx context.Context,
	roots []string,
	known []KnownFile,
	emit func(FileInfo) error,
) (Report, error) {
	return s.ScanTree(ctx, roots, ScanOptions{
		Mode:  ModeIncremental,
		Known: known,
	}, emit)
}

// normalizeRoots trims, deduplicates and validates the root list.
func normalizeRoots(roots []string) ([]string, error) {
	seen := make(map[string]struct{}, len(roots))
	clean := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		key := NormalizePathKey(root)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, root)
	}
	if len(clean) == 0 {
		return nil, errors.New("at least one media root is required")
	}
	return clean, nil
}

// fileInfo builds the scanner's output record for one file.
func (s *Scanner) fileInfo(path string, info fs.FileInfo, root string, artwork *artworkResolver) FileInfo {
	parsed := ParseFilePath(path)

	artworkPath := ""
	if artwork != nil {
		artworkPath = artwork.Resolve(path)
	}

	extension := strings.ToLower(filepath.Ext(path))
	fileID := fileIDFromStat(info)

	segments := PathSegmentsBelowRoot(root, path)
	context := BuildPathContext(segments)

	return FileInfo{
		Path:         path,
		Size:         info.Size(),
		ModTime:      info.ModTime().UTC(),
		Parsed:       parsed,
		Extension:    extension,
		ArtworkPath:  artworkPath,
		Root:         root,
		RootID:       RootID(root),
		FileID:       fileID,
		Fingerprint:  Fingerprint{Size: info.Size(), ModTime: info.ModTime().UTC().UnixNano(), FileID: fileID},
		SearchTokens: SearchTokens(parsed.Title),
		RelativePath: relativeToRoot(root, path),
		PathSegments: segments,
		Context:      context,
	}
}

// PathSegmentsBelowRoot returns the folder names between a media root and a file,
// outermost first. Classification and origin detection work on these segments
// rather than on the raw path, which removes substring false positives.
func PathSegmentsBelowRoot(root, path string) []string {
	if root == "" {
		return nil
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	dir := filepath.Dir(relative)
	if dir == "." {
		return nil
	}
	raw := strings.Split(filepath.ToSlash(dir), "/")
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

// BuildPathContext labels the folder hierarchy for resolution.
//
// Owners organise libraries as Root/Category/Origin/Work/Season, so the work
// name is normally the folder directly above the season folder. Reading that
// structure is far more reliable than reading a release filename.
func BuildPathContext(segments []string) PathContext {
	context := PathContext{}
	for index, segment := range segments {
		// Category and origin are cumulative: a deeper folder overrides a shallower
		// one, matching how owners nest "Movies/Series/...".
		if slug := DetectCategoryFromSegments([]string{segment}); slug != "" {
			context.Category = slug
		}
		if origin := DetectOriginTagsFromFolders([]string{segment}); origin != "" {
			context.Origin = origin
		}

		if season := parseSeasonFromFolder(segment); season > 0 && context.SeasonNumber == 0 {
			context.SeasonNumber = season
			context.SeasonFolderTitle = segment
			// A folder can name both the work and the season: "Silo الموسم الثالث".
			if prefix := titlePrefixBeforeSeason(segment); prefix != "" {
				context.WorkTitle = prefix
			}
		}
		_ = index
	}

	// The work folder is the deepest segment that is neither a category, an
	// origin, nor a season, and that carries a usable title. Walking from the
	// deepest segment outward finds the closest possible name.
	if context.WorkTitle == "" {
		for index := len(segments) - 1; index >= 0; index-- {
			segment := segments[index]
			if DetectCategoryFromSegments([]string{segment}) != "" {
				continue
			}
			if DetectOriginTagsFromFolders([]string{segment}) != "" {
				continue
			}
			if parseSeasonFromFolder(segment) > 0 {
				continue
			}
			cleaned := cleanTitle(normalizeWorkingName(segment))
			if cleaned == "" || isJunkOrWatermarkTitle(cleaned) || isNumericOnly(cleaned) {
				continue
			}
			context.WorkTitle = cleaned
			break
		}
	}

	return context
}

// relativeToRoot returns the path below the media root, which is the field
// search and the UI should use instead of the raw absolute path.
func relativeToRoot(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// newScanID produces a sortable, collision-resistant scan identifier.
func newScanID() string {
	return fmt.Sprintf("scan-%d", time.Now().UTC().UnixNano())
}

// -----------------------------------------------------------------------------
// Artwork discovery
// -----------------------------------------------------------------------------

// FindLocalArtwork returns the artwork next to (or just above) a video file.
//
// It performs two ReadDir calls per invocation, so it must not be called per
// file on a large library: the scan pipeline uses artworkResolver, which
// memoises each directory. This function remains for one-off lookups.
func FindLocalArtwork(videoPath string) string {
	dir := filepath.Dir(videoPath)
	if img := searchDirForArtwork(dir); img != "" {
		return img
	}
	parent := filepath.Dir(dir)
	if parent != dir && parent != "." && parent != string(filepath.Separator) {
		if img := searchDirForArtwork(parent); img != "" {
			return img
		}
	}
	return ""
}

// artworkPriority ranks artwork file names. A folder-level poster is much more
// likely to be the intended cover than an arbitrary first image, so the order
// is explicit and extensible rather than incidental.
var artworkPriority = []struct {
	name  string
	score int
}{
	{"poster", 100},
	{"folder", 95},
	{"cover", 90},
	{"tvshow", 88},
	{"season", 85},
	{"front", 80},
	{"default", 70},
	{"show", 68},
	{"thumb", 50},
	{"fanart", 40},
	{"banner", 30},
	{"backdrop", 20},
	{"logo", 10},
}

// searchDirForArtwork scans a single directory for the best artwork candidate.
func searchDirForArtwork(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	best := ""
	bestScore := -1
	fallback := ""
	fallbackScore := -1

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := imageExtensions[ext]; !ok {
			continue
		}
		base := strings.ToLower(strings.TrimSuffix(entry.Name(), ext))

		// Rank every known keyword rather than returning on the first hit, so
		// "season01.jpg" and "poster.jpg" in one folder resolve deterministically.
		score := -1
		for _, candidate := range artworkPriority {
			if strings.Contains(base, candidate.name) {
				if candidate.score > score {
					score = candidate.score
				}
			}
		}

		full := filepath.Join(dir, entry.Name())
		if score >= 0 {
			if score > bestScore {
				best = full
				bestScore = score
			}
			continue
		}
		// Unknown image name: usable, but always ranked below a named candidate.
		if fallbackScore < 0 {
			fallback = full
			fallbackScore = 0
		}
	}

	if best != "" {
		return best
	}
	return fallback
}

// sortFilesByState is a small helper used by the reconciliation report to keep
// output deterministic.
func sortFilesByState(files []FileInfo) {
	sortSliceStable(files, func(a, b FileInfo) bool { return a.Path < b.Path })
}

func sortSliceStable[T any](items []T, less func(a, b T) bool) {
	stableSort(items, less)
}

func stableSort[T any](items []T, less func(a, b T) bool) {
	// Insertion sort is adequate for the report-sized slices this is used on.
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && less(items[j], items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// mutexPool is a tiny helper to keep a shared accumulator safe when a caller
// chooses to emit from multiple goroutines itself.
type mutexPool struct {
	mu sync.Mutex
}
