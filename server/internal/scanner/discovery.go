package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Visit is the unit the discovery engine emits. It carries only what was cheap
// to obtain during traversal (the directory entry and its parent context), so
// no metadata syscall happens inside the walk callback.
type Visit struct {
	Path     string
	Root     string
	Entry    fs.DirEntry
	Segments []string // normalized path segments below the root
	Depth    int
}

// DiscoveryConfig controls traversal behaviour. Every field has a safe default
// so a zero value still produces a correct, bounded scan.
type DiscoveryConfig struct {
	// FollowSymlinks decides whether directory symlinks/junctions are entered.
	// Default false: media libraries contain link farms and recursive
	// junctions that would otherwise cause unbounded traversal.
	FollowSymlinks bool
	// IgnoreHidden skips dot-directories and directories marked hidden.
	IgnoreHidden bool
	// IgnoreDirs is matched case-insensitively against individual path
	// segments (not the whole path), which avoids the "/NotMovies/" false
	// positive a substring search produces.
	IgnoreDirs []string
	// IgnoreFileSuffixes drops in-progress download artefacts outright.
	IgnoreFileSuffixes []string
	// MaxDepth limits how deep traversal goes below a root. 0 means unlimited.
	MaxDepth int
	// DiscoveryWorkers is how many roots walk concurrently. Individual
	// directories are walked sequentially inside a root to preserve locality
	// for spinning disks and network shares.
	DiscoveryWorkers int
}

// defaultIgnoredDirs are directories that never contain media and are extremely
// expensive to traverse on real libraries.
var defaultIgnoredDirs = []string{
	"$recycle.bin", "system volume information", "recycler",
	"#recycle", ".trash", ".trashes", ".trash-1000",
	"node_modules", ".git", ".svn", "__macosx",
	"@eadir", "lost+found", "windows", "program files", "program files (x86)",
	"programdata", "$windows.~bt", "$windows.~ws", "recovery",
}

// defaultIgnoredSuffixes are partial/in-progress download files. Indexing them
// would create bogus entries that must later be removed.
var defaultIgnoredSuffixes = []string{
	".part", ".!qb", ".crdownload", ".download", ".tmp", ".temp",
	".partial", ".incomplete", ".aria2", ".bup", ".filepart", ".opdownload",
	".!ut", ".bc!", ".td", ".xltd",
}

// discovery implements the DISCOVERY stage: traversal plus candidate filtering.
type discovery struct {
	config   DiscoveryConfig
	jobCh    chan<- Visit
	ignore   map[string]struct{}
	suffixes []string
}

func newDiscovery(config DiscoveryConfig, jobCh chan<- Visit) *discovery {
	ignore := make(map[string]struct{}, len(defaultIgnoredDirs)+len(config.IgnoreDirs))
	for _, name := range defaultIgnoredDirs {
		ignore[strings.ToLower(name)] = struct{}{}
	}
	for _, name := range config.IgnoreDirs {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		ignore[name] = struct{}{}
	}
	suffixes := append([]string{}, defaultIgnoredSuffixes...)
	for _, suffix := range config.IgnoreFileSuffixes {
		suffix = strings.ToLower(strings.TrimSpace(suffix))
		if suffix == "" {
			continue
		}
		suffixes = append(suffixes, suffix)
	}
	return &discovery{config: config, jobCh: jobCh, ignore: ignore, suffixes: suffixes}
}

// isIgnoredDirName reports whether a single path segment is on the deny list.
func (d *discovery) isIgnoredDirName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return true
	}
	if _, denied := d.ignore[lower]; denied {
		return true
	}
	if d.config.IgnoreHidden && strings.HasPrefix(lower, ".") {
		return true
	}
	return false
}

// isIgnoredFileName drops in-progress or temporary artefacts.
func (d *discovery) isIgnoredFileName(name string) bool {
	lower := strings.ToLower(name)
	for _, suffix := range d.suffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// walkRoot traverses one root and forwards candidates. Crucially, it never
// returns a filesystem error to its caller: a failure to read one directory is
// recorded and traversal continues with its siblings. This is the single
// change that stops one locked folder from aborting an entire disk scan.
func (d *discovery) walkRoot(ctx context.Context, root string, tracker *ProgressTracker) {
	info, err := os.Lstat(root)
	if err != nil {
		tracker.SetRootStatus(root, RootUnavailable)
		tracker.Errors().AddError(err, root, root, "stat_root")
		if IsRootUnavailable(err) || os.IsNotExist(err) {
			tracker.Counts().rootsUnavailable.Add(1)
		} else {
			tracker.Counts().rootsFailed.Add(1)
		}
		return
	}
	if !info.IsDir() {
		tracker.SetRootStatus(root, RootError)
		tracker.Errors().Add(ScanError{
			Code: ErrInvalidPath, Path: root, Root: root, Op: "stat_root",
			Err: "media root is not a directory",
		})
		tracker.Counts().rootsFailed.Add(1)
		return
	}

	tracker.SetRootStatus(root, RootScanning)
	completed := d.walkDirectory(ctx, root, root, nil, 0, tracker)
	if tracker.RootStatuses()[root] == RootScanning {
		if completed {
			tracker.SetRootStatus(root, RootCompleted)
			tracker.Counts().rootsCompleted.Add(1)
		} else {
			tracker.SetRootStatus(root, RootUnavailable)
			tracker.Counts().rootsUnavailable.Add(1)
		}
	}
}

// walkDirectory reads one directory and recurses into its subdirectories.
// It returns false when the directory itself could not be read, which lets the
// caller mark a whole root unavailable without per-file error noise.
func (d *discovery) walkDirectory(ctx context.Context, root, dir string, ancestors []string, depth int, tracker *ProgressTracker) bool {
	if err := ctx.Err(); err != nil {
		return true
	}
	if d.config.MaxDepth > 0 && depth > d.config.MaxDepth {
		return true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		tracker.Errors().AddError(err, dir, root, "read_dir")
		code, _ := ClassifyError(err)
		switch code {
		case ErrPermissionDenied:
			tracker.Counts().permissionErrors.Add(1)
		case ErrDiskUnavailable, ErrNetworkError:
			tracker.Counts().filesystemErrors.Add(1)
			return false
		default:
			tracker.Counts().filesystemErrors.Add(1)
		}
		// Permission and transient errors on a subdirectory never stop the
		// parent traversal; only an unreadable root is treated as fatal.
		return dir != root
	}
	tracker.Counts().directories.Add(1)

	var subdirs []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return true
		}
		name := entry.Name()

		if entry.IsDir() {
			if d.isIgnoredDirName(name) {
				continue
			}
			subdirs = append(subdirs, name)
			continue
		}

		// A symlink or junction can point at a directory. Entering it blindly
		// is how scanners create infinite loops, so the policy is explicit.
		if entry.Type()&fs.ModeSymlink != 0 {
			if !d.config.FollowSymlinks {
				continue
			}
			target, statErr := os.Stat(filepath.Join(dir, name))
			if statErr != nil {
				tracker.Errors().AddError(statErr, filepath.Join(dir, name), root, "stat_symlink")
				continue
			}
			if target.IsDir() {
				subdirs = append(subdirs, name)
				continue
			}
		}

		tracker.Counts().filesSeen.Add(1)
		if d.isIgnoredFileName(name) {
			tracker.Counts().rejected.Add(1)
			continue
		}

		path := filepath.Join(dir, name)
		visit := Visit{
			Path:     path,
			Root:     root,
			Entry:    entry,
			Segments: appendSegment(ancestors, name),
			Depth:    depth,
		}
		select {
		case d.jobCh <- visit:
		case <-ctx.Done():
			return true
		}
	}

	sort.Strings(subdirs)
	for _, name := range subdirs {
		if err := ctx.Err(); err != nil {
			return true
		}
		if !d.walkDirectory(ctx, root, filepath.Join(dir, name), appendSegment(ancestors, name), depth+1, tracker) {
			// A subdirectory vanished; keep walking its siblings.
			continue
		}
	}
	return true
}

func appendSegment(segments []string, name string) []string {
	out := make([]string, len(segments)+1)
	copy(out, segments)
	out[len(segments)] = name
	return out
}

// artworkResolver caches per-directory artwork discovery. Without this cache a
// season folder holding 100,000 episodes triggers 100,000 ReadDir calls just to
// find the same poster.jpg. The cache is bounded by the number of directories
// visited, not by the number of files, and it is discarded with the scan.
type artworkResolver struct {
	mu      sync.Mutex
	byDir   map[string]string
	checked map[string]struct{}
}

func newArtworkResolver() *artworkResolver {
	return &artworkResolver{byDir: make(map[string]string), checked: make(map[string]struct{})}
}

// Resolve returns the artwork for a video path, consulting each directory once.
func (a *artworkResolver) Resolve(videoPath string) string {
	dir := filepath.Dir(videoPath)
	parent := filepath.Dir(dir)

	a.mu.Lock()
	if cached, checked := a.byDir[dir]; checked {
		a.mu.Unlock()
		if cached != "" {
			return cached
		}
		// Directory has no artwork of its own; the parent may still have one.
		return a.resolveParent(parent, dir)
	}
	a.mu.Unlock()

	found := searchDirForArtwork(dir)

	a.mu.Lock()
	a.checked[dir] = struct{}{}
	a.byDir[dir] = found
	a.mu.Unlock()

	if found != "" {
		return found
	}
	return a.resolveParent(parent, dir)
}

func (a *artworkResolver) resolveParent(parent, dir string) string {
	if parent == dir || parent == "." || parent == string(filepath.Separator) || parent == "" {
		return ""
	}
	a.mu.Lock()
	if cached, checked := a.byDir[parent]; checked {
		a.mu.Unlock()
		return cached
	}
	a.mu.Unlock()

	found := searchDirForArtwork(parent)
	a.mu.Lock()
	a.checked[parent] = struct{}{}
	a.byDir[parent] = found
	a.mu.Unlock()
	return found
}

// Size reports how many directory lookups were memoised. It is used by tests
// and benchmarks to prove the cache actually collapses ReadDir calls.
func (a *artworkResolver) Size() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.byDir)
}

// stabilityTracker suppresses ingestion until a file has stopped growing. This
// is what prevents a download that is written in 40 chunks from producing 40
// catalogue inserts.
type stabilityTracker struct {
	mu      sync.Mutex
	pending map[string]*stabilityEntry
	minAge  time.Duration
}

type stabilityEntry struct {
	size    int64
	modTime time.Time
	first   time.Time
}

func newStabilityTracker(minAge time.Duration) *stabilityTracker {
	if minAge <= 0 {
		minAge = 5 * time.Second
	}
	return &stabilityTracker{pending: make(map[string]*stabilityEntry), minAge: minAge}
}

// Observe records the current size of a path and reports whether it has been
// stable for at least minAge.
func (s *stabilityTracker) Observe(path string, size int64, modTime time.Time) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.pending[path]
	switch {
	case !exists:
		s.pending[path] = &stabilityEntry{size: size, modTime: modTime, first: now}
		return false
	case entry.size != size || !entry.modTime.Equal(modTime):
		// Still growing: restart the stability window.
		entry.size = size
		entry.modTime = modTime
		entry.first = now
		return false
	default:
		if now.Sub(entry.first) >= s.minAge {
			delete(s.pending, path)
			return true
		}
		return false
	}
}

// Pending reports how many paths are awaiting stability. Bounded by event
// volume between ticks, and pruned on every successful observation.
func (s *stabilityTracker) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}
