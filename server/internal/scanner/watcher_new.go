package scanner

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventKind classifies a filesystem event. Rename is its own kind because
// treating it as remove+create is what causes duplicate catalogue rows.
type EventKind string

const (
	EventCreated    EventKind = "created"
	EventModified   EventKind = "modified"
	EventRemoved    EventKind = "removed"
	EventRenamed    EventKind = "renamed"
	EventDirCreated EventKind = "dir_created"
	EventDirRemoved EventKind = "dir_removed"
	// EventRootOnline / EventRootOffline describe a removable disk appearing or
	// disappearing, which triggers a recovery scan rather than a deletion.
	EventRootOnline  EventKind = "root_online"
	EventRootOffline EventKind = "root_offline"
)

// Event is one coalesced, stability-checked filesystem change.
type Event struct {
	Kind EventKind `json:"kind"`
	Path string    `json:"path"`
	File *FileInfo `json:"file,omitempty"`
	// RenamedFrom carries the previous path when the filesystem reported it.
	RenamedFrom string `json:"renamedFrom,omitempty"`
	// Root is the media root the path belongs to.
	Root string `json:"root,omitempty"`
}

// WatcherOptions configures the event watcher's resilience behaviour.
type WatcherOptions struct {
	// Recursive registers every subdirectory of each root.
	Recursive bool
	// Debounce coalesces a burst of events on the same path into one action.
	Debounce time.Duration
	// Stability is how long a file's size and mtime must be unchanged before it
	// is considered finished. This is what prevents a large download from being
	// ingested dozens of times.
	Stability time.Duration
	// RootRetryInterval is how often an absent root is re-checked.
	RootRetryInterval time.Duration
	// ErrorRetryInterval is how long to wait before recreating a broken watcher.
	ErrorRetryInterval time.Duration
	// QueueSize bounds the coalescing queue.
	QueueSize int
	// Logger receives one line per watcher recovery, not per event.
	Logger *slog.Logger
}

func (o WatcherOptions) withDefaults() WatcherOptions {
	if o.Debounce <= 0 {
		o.Debounce = 2 * time.Second
	}
	if o.Stability <= 0 {
		o.Stability = 5 * time.Second
	}
	if o.RootRetryInterval <= 0 {
		o.RootRetryInterval = 15 * time.Second
	}
	if o.ErrorRetryInterval <= 0 {
		o.ErrorRetryInterval = 5 * time.Second
	}
	if o.QueueSize <= 0 {
		o.QueueSize = 1024
	}
	return o
}

// EventWatcher turns fsnotify events into debounced, stability-checked actions.
//
// The watcher is the FAST PATH only. It can lose events, overflow its buffer and
// fail entirely on some network filesystems, so reconciliation remains the
// source of truth. Nothing in this file deletes catalogue rows: a remove event
// marks a record for reconciliation instead.
type EventWatcher struct {
	scanner *Scanner
	options WatcherOptions
	// retryInterval is kept as a field so tests can shrink it.
	retryInterval time.Duration
}

// NewEventWatcher creates a watcher with sensible resilience defaults.
func NewEventWatcher(scanner *Scanner, recursive bool) *EventWatcher {
	return NewEventWatcherWithOptions(scanner, WatcherOptions{Recursive: recursive})
}

// NewEventWatcherWithOptions creates a fully configured watcher.
func NewEventWatcherWithOptions(scanner *Scanner, options WatcherOptions) *EventWatcher {
	options = options.withDefaults()
	return &EventWatcher{
		scanner:       scanner,
		options:       options,
		retryInterval: options.RootRetryInterval,
	}
}

// Watch runs the watcher until the context is cancelled.
//
// It never returns on a transient fsnotify error: a watcher that dies on one
// error stops protecting the library forever, so the loop rebuilds and
// continues. It returns only when the context ends.
func (w *EventWatcher) Watch(ctx context.Context, roots []string, handle func(Event) error) error {
	if handle == nil {
		return errors.New("event handler is required")
	}

	watched := newWatchedSet()
	pendingRoots := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		pendingRoots[root] = struct{}{}
	}
	if len(pendingRoots) == 0 {
		return fs.ErrNotExist
	}

	stability := newStabilityTracker(w.options.Stability)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := w.watchLoop(ctx, pendingRoots, watched, stability, handle); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Rebuild the watcher rather than giving up. The library must keep
			// being monitored even after an inotify queue overflow.
			if w.options.Logger != nil {
				w.options.Logger.Warn("watcher restarting after error", slog.Any("error", err))
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.options.ErrorRetryInterval):
		}
	}
}

// watchLoop runs one watcher generation.
func (w *EventWatcher) watchLoop(
	ctx context.Context,
	pendingRoots map[string]struct{},
	watched *watchedSet,
	stability *stabilityTracker,
	handle func(Event) error,
) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Re-register everything that was already being watched in the previous
	// generation, so a restart does not reopen the same directory twice.
	watched.reset()
	for root := range pendingRoots {
		if err := w.tryAddRoot(watcher, root, watched); err == nil {
			delete(pendingRoots, root)
			_ = handle(Event{Kind: EventRootOnline, Path: root, Root: root})
		}
	}

	retryTicker := time.NewTicker(w.retryInterval)
	defer retryTicker.Stop()

	debouncer := newDebouncer(w.options.QueueSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-retryTicker.C:
			for root := range pendingRoots {
				if err := w.tryAddRoot(watcher, root, watched); err == nil {
					delete(pendingRoots, root)
					_ = handle(Event{Kind: EventRootOnline, Path: root, Root: root})
				}
			}
		case err := <-watcher.Errors:
			if err != nil {
				return err
			}
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("watcher event channel closed")
			}
			if err := w.processRawEvent(ctx, watcher, event, watched, debouncer, stability, handle); err != nil {
				return err
			}
		}
	}
}

// processRawEvent translates one fsnotify event into zero or more pending
// actions, registering new directories as they appear.
func (w *EventWatcher) processRawEvent(
	ctx context.Context,
	watcher *fsnotify.Watcher,
	event fsnotify.Event,
	watched *watchedSet,
	debouncer *debouncer,
	stability *stabilityTracker,
	handle func(Event) error,
) error {
	switch {
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// Never delete on a remove event. Mark it for reconciliation instead: the
		// file may be mid-rename, on a disconnecting share, or inside a folder that
		// is being replaced.
		return handle(Event{Kind: EventRemoved, Path: event.Name})
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// fsnotify cannot always tell us the destination. Route it through the
		// stability pipeline so the new path is discovered by reconciliation.
		return handle(Event{Kind: EventRenamed, Path: event.Name})
	}

	if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return nil
	}

	info, err := os.Lstat(event.Name)
	if err != nil {
		// The path may already be gone; reconciliation will settle it.
		return nil
	}

	if info.IsDir() {
		if event.Op&fsnotify.Create == fsnotify.Create {
			if w.options.Recursive {
				// Register the new subtree once; watchedSet prevents duplicates.
				if err := w.addRootDeduped(watcher, event.Name, watched); err != nil {
					return nil
				}
			}
			return handle(Event{Kind: EventDirCreated, Path: event.Name})
		}
		return nil
	}

	if !w.scanner.IsVideoFile(event.Name) {
		return nil
	}

	// A stability check is the difference between one ingest and forty: an
	// in-progress download grows, so its size keeps changing and the action is
	// deferred until it settles.
	if !stability.Observe(event.Name, info.Size(), info.ModTime()) {
		if !debouncer.schedule(event.Name, w.options.Debounce) {
			return nil
		}
		return nil
	}

	file := w.scanner.fileInfo(event.Name, info, "", nil)
	kind := EventModified
	if event.Op&fsnotify.Create == fsnotify.Create {
		kind = EventCreated
	}
	return handle(Event{Kind: kind, Path: event.Name, File: &file})
}

// addRootDeduped registers a directory tree without duplicate watcher entries.
func (w *EventWatcher) addRootDeduped(watcher *fsnotify.Watcher, root string, watched *watchedSet) error {
	if watched.has(root) {
		return nil
	}
	return w.addRoot(watcher, root, watched)
}

// addRoot registers a root (recursively when configured).
func (w *EventWatcher) addRoot(watcher *fsnotify.Watcher, root string, watched *watchedSet) error {
	if !w.options.Recursive {
		if watched.add(root) {
			return watcher.Add(root)
		}
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// A single unreadable subdirectory must not abort registration for
			// the rest of the tree.
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if watched.add(path) {
			if err := watcher.Add(path); err != nil {
				return nil
			}
		}
		return nil
	})
}

// tryAddRoot adds a root if it is currently reachable, leaving removable disks
// pending instead of failing.
func (w *EventWatcher) tryAddRoot(watcher *fsnotify.Watcher, root string, watched *watchedSet) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) || IsRootUnavailable(err) {
			return err
		}
		return err
	}
	if !info.IsDir() {
		return fs.ErrInvalid
	}
	return w.addRoot(watcher, root, watched)
}

// -----------------------------------------------------------------------------
// Supporting structures
// -----------------------------------------------------------------------------

// watchedSet tracks registered directories so a directory created event or a
// watcher restart cannot register the same path twice.
type watchedSet struct {
	mu    sync.Mutex
	paths map[string]struct{}
}

func newWatchedSet() *watchedSet {
	return &watchedSet{paths: make(map[string]struct{})}
}

// add returns true when the path was not previously registered.
func (s *watchedSet) add(path string) bool {
	key := NormalizePathKey(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.paths[key]; exists {
		return false
	}
	s.paths[key] = struct{}{}
	return true
}

func (s *watchedSet) has(path string) bool {
	key := NormalizePathKey(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.paths[key]
	return exists
}

// reset clears the registry, used when a watcher generation is rebuilt.
func (s *watchedSet) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paths = make(map[string]struct{})
}

// Count reports how many directories are registered.
func (s *watchedSet) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.paths)
}

// debouncer coalesces bursts of events on the same path. It is a bounded map
// with time-based expiry, so a long-running watcher cannot accumulate state.
type debouncer struct {
	mu       sync.Mutex
	seen     map[string]time.Time
	capacity int
}

func newDebouncer(capacity int) *debouncer {
	if capacity <= 0 {
		capacity = 1024
	}
	return &debouncer{seen: make(map[string]time.Time), capacity: capacity}
}

// schedule records an event time and reports whether this burst should be
// actioned yet. It returns false while the burst is still active.
func (d *debouncer) schedule(path string, window time.Duration) bool {
	now := time.Now()
	key := NormalizePathKey(path)
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.seen) >= d.capacity {
		// Bound memory: drop the oldest half of the entries rather than growing.
		d.evictOldestLocked(d.capacity / 2)
	}

	last, exists := d.seen[key]
	d.seen[key] = now
	if !exists {
		return false
	}
	return now.Sub(last) >= window
}

func (d *debouncer) evictOldestLocked(count int) {
	if count <= 0 || len(d.seen) == 0 {
		return
	}
	type entry struct {
		key  string
		when time.Time
	}
	entries := make([]entry, 0, len(d.seen))
	for key, when := range d.seen {
		entries = append(entries, entry{key: key, when: when})
	}
	// Partial selection is unnecessary at these sizes; a full sort of a bounded
	// map is microseconds and runs rarely.
	sortSliceStable(entries, func(a, b entry) bool { return a.when.Before(b.when) })
	for i := 0; i < count && i < len(entries); i++ {
		if entries[i].key == "" {
			continue
		}
		delete(d.seen, entries[i].key)
	}
}

// Size reports the number of tracked paths (test/observability helper).
func (d *debouncer) Size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.seen)
}
