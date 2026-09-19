package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReconcileMarksMissingOnlyWhenRootIsReadable is the single most important
// safety test in this package.
//
// A record may only become MISSING when the root that owns it was actually read.
// If the disk is offline or a share timed out, the record must become
// UNAVAILABLE so an unplugged drive can never lose catalogue entries.
func TestReconcileMarksMissingOnlyWhenRootIsReadable(t *testing.T) {
	rootOnline := "D:/Media"
	rootOffline := "E:/Archive"
	roots := []string{rootOnline, rootOffline}

	missing := []KnownFile{
		{ID: 1, Path: "D:/Media/Movies/gone.mp4", State: StateActive},
		{ID: 2, Path: "E:/Archive/Movies/unplugged.mp4", State: StateActive},
	}

	cases := []struct {
		name        string
		states      map[string]RootStatus
		wantMissing int
		wantUnavail int
	}{
		{
			name: "both roots readable",
			states: map[string]RootStatus{
				rootOnline:  RootCompleted,
				rootOffline: RootCompleted,
			},
			wantMissing: 2,
		},
		{
			name: "one disk unplugged",
			states: map[string]RootStatus{
				rootOnline:  RootCompleted,
				rootOffline: RootUnavailable,
			},
			wantMissing: 1,
			wantUnavail: 1,
		},
		{
			name: "disk offline reported explicitly",
			states: map[string]RootStatus{
				rootOnline:  RootCompleted,
				rootOffline: RootOffline,
			},
			wantMissing: 1,
			wantUnavail: 1,
		},
		{
			name: "root failed with an error",
			states: map[string]RootStatus{
				rootOnline:  RootCompleted,
				rootOffline: RootError,
			},
			wantMissing: 1,
			wantUnavail: 1,
		},
		{
			name:        "no root information at all refuses to declare anything missing",
			states:      map[string]RootStatus{},
			wantUnavail: 2,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			report := Report{RootStates: testCase.states, Missing: missing}
			result := Reconcile(report, roots)
			if result.MarkedMissing != testCase.wantMissing {
				t.Errorf("marked missing = %d, want %d", result.MarkedMissing, testCase.wantMissing)
			}
			if result.MarkedUnavailable != testCase.wantUnavail {
				t.Errorf("marked unavailable = %d, want %d", result.MarkedUnavailable, testCase.wantUnavail)
			}
		})
	}
}

// TestCleanupRefusesWhileRootIsOffline verifies the cleanup policy cannot be
// tricked into wiping records for an offline disk.
func TestCleanupRefusesWhileRootIsOffline(t *testing.T) {
	candidates := []CleanupCandidate{
		{ID: 1, Path: "E:/Archive/a.mp4", State: StateMissing, LastSeen: time.Now().Add(-30 * 24 * time.Hour)},
	}
	policy := DefaultCleanupPolicy()

	// With the owning root offline, cleanup must refuse and explain why.
	selected, refusal := SelectCleanupCandidates(candidates,
		map[string]RootStatus{"E:/Archive": RootUnavailable}, policy, time.Now())
	if len(selected) != 0 {
		t.Errorf("cleanup selected %d records while the root was offline", len(selected))
	}
	if refusal == "" {
		t.Error("expected an explicit refusal reason")
	}

	// With the root reachable and the record old enough, it may proceed.
	selected, refusal = SelectCleanupCandidates(candidates,
		map[string]RootStatus{"E:/Archive": RootCompleted}, policy, time.Now())
	if refusal != "" {
		t.Errorf("unexpected refusal: %s", refusal)
	}
	if len(selected) != 1 {
		t.Errorf("selected %d records, want 1", len(selected))
	}
}

// TestCleanupSkipsRecentlyMissingRecords ensures a temporarily unavailable file
// is not deleted simply because it was not seen in one pass.
func TestCleanupSkipsRecentlyMissingRecords(t *testing.T) {
	now := time.Now()
	candidates := []CleanupCandidate{
		{ID: 1, Path: "D:/Media/recent.mp4", State: StateMissing, LastSeen: now.Add(-1 * time.Hour)},
		{ID: 2, Path: "D:/Media/old.mp4", State: StateMissing, LastSeen: now.Add(-30 * 24 * time.Hour)},
	}
	selected, _ := SelectCleanupCandidates(candidates,
		map[string]RootStatus{"D:/Media": RootCompleted}, DefaultCleanupPolicy(), now)

	if len(selected) != 1 {
		t.Fatalf("selected %d records, want only the aged one", len(selected))
	}
	if selected[0].ID != 2 {
		t.Errorf("selected id = %d, want 2", selected[0].ID)
	}
}

// TestCleanupRespectsDeletionCap verifies the safety cap that prevents a single
// misconfiguration from emptying the catalogue.
func TestCleanupRespectsDeletionCap(t *testing.T) {
	now := time.Now()
	candidates := make([]CleanupCandidate, 0, 100)
	for i := 0; i < 100; i++ {
		candidates = append(candidates, CleanupCandidate{
			ID: int64(i), Path: "D:/Media/f" + itoa(i) + ".mp4", State: StateMissing,
			LastSeen: now.Add(-30 * 24 * time.Hour),
		})
	}
	policy := DefaultCleanupPolicy()
	policy.MaxDeletionsPerRun = 10

	selected, _ := SelectCleanupCandidates(candidates,
		map[string]RootStatus{"D:/Media": RootCompleted}, policy, now)
	if len(selected) != 10 {
		t.Fatalf("selected %d records, want the cap of 10", len(selected))
	}
}

// TestDecideActionsOrdersMovesBeforeInserts guards against a rename creating a
// duplicate row: the move must be applied before any insert for that path.
func TestDecideActionsOrdersMovesBeforeInserts(t *testing.T) {
	files := []FileInfo{
		{Path: "D:/Media/new-name.mp4", Change: ChangeRenamed, KnownID: 7, RenamedFrom: "D:/Media/old-name.mp4"},
		{Path: "D:/Media/brand-new.mp4", Change: ChangeNew},
	}

	decisions := DecideActions(files, nil)
	if len(decisions) != 2 {
		t.Fatalf("decisions = %d, want 2", len(decisions))
	}
	if decisions[0].Action != ActionMovePath {
		t.Errorf("first action = %q, want %q", decisions[0].Action, ActionMovePath)
	}
	if decisions[1].Action != ActionInsert {
		t.Errorf("second action = %q, want %q", decisions[1].Action, ActionInsert)
	}
}

// TestQualityAnalyzerFindsRealProblems checks the data-quality report detects the
// conditions the owner asked to be surfaced rather than hidden.
func TestQualityAnalyzerFindsRealProblems(t *testing.T) {
	files := []FileInfo{
		// Missing category, low confidence, no artwork.
		{Path: "D:/Unsorted/01.mp4", Parsed: ParsedName{Title: "01", Confidence: ConfidenceLow}},
		// Episode without a number.
		{Path: "D:/Series/Show/Season 01/mystery.mkv", Parsed: ParsedName{
			Title: "Show", IsEpisode: true, EpisodeNumber: 0, CategorySlug: "series",
			Confidence: ConfidenceMedium,
		}, ArtworkPath: "poster.jpg"},
		// Impossible episode number.
		{Path: "D:/Series/Show/Season 01/S01E99999.mkv", Parsed: ParsedName{
			Title: "Show", IsEpisode: true, SeasonNumber: 1, EpisodeNumber: 99999,
			CategorySlug: "series", Confidence: ConfidenceMedium,
		}, ArtworkPath: "poster.jpg"},
		// Invalid season.
		{Path: "D:/Series/Show/Season 500/S01E01.mkv", Parsed: ParsedName{
			Title: "Show", IsEpisode: true, SeasonNumber: 500, EpisodeNumber: 1,
			CategorySlug: "series", Confidence: ConfidenceMedium,
		}, ArtworkPath: "poster.jpg"},
	}

	issues := NewQualityAnalyzer(3).Analyze(files)
	codes := map[string]int{}
	for _, issue := range issues {
		codes[issue.Code] = issue.Count
	}

	for _, expected := range []string{
		"missing_category", "low_confidence", "missing_artwork",
		"episode_without_number", "impossible_episode", "invalid_season",
	} {
		if codes[expected] == 0 {
			t.Errorf("quality report did not detect %q; got %v", expected, codes)
		}
	}

	// Every issue must be explainable, not just counted.
	for _, issue := range issues {
		if issue.Title == "" || issue.Detail == "" || issue.AffectsData == "" {
			t.Errorf("issue %q is not self-explanatory: %+v", issue.Code, issue)
		}
	}
}

// TestStabilityTrackerDefersGrowingFile verifies the debounce that stops a large
// download from being ingested dozens of times while it is still being written.
func TestStabilityTrackerDefersGrowingFile(t *testing.T) {
	tracker := newStabilityTracker(30 * time.Millisecond)
	path := "D:/Media/downloading.mkv"
	modTime := time.Now()

	// Simulate a download growing in chunks.
	for size := int64(100); size <= 500; size += 100 {
		if tracker.Observe(path, size, modTime) {
			t.Fatalf("a growing file was reported stable at size %d", size)
		}
	}

	// Now the size stops changing: after the window it becomes stable.
	size := int64(500)
	if tracker.Observe(path, size, modTime) {
		t.Fatal("the file was reported stable before the window elapsed")
	}
	time.Sleep(40 * time.Millisecond)
	if !tracker.Observe(path, size, modTime) {
		t.Fatal("a settled file was never reported stable")
	}

	// A successful observation must clear the pending entry: state is bounded.
	if tracker.Pending() != 0 {
		t.Errorf("pending entries = %d, want 0 after settling", tracker.Pending())
	}
}

// TestDebouncerBoundsMemory verifies the debounce state cannot grow without
// limit, which would be a slow memory leak in a long-running watcher.
func TestDebouncerBoundsMemory(t *testing.T) {
	debouncer := newDebouncer(64)
	for i := 0; i < 500; i++ {
		debouncer.schedule("D:/Media/file"+itoa(i)+".mkv", time.Second)
	}
	if size := debouncer.Size(); size > 64 {
		t.Fatalf("debouncer retained %d paths, want at most the capacity of 64", size)
	}
}

// TestWatchedSetPreventsDuplicateRegistration verifies a directory created event
// cannot register the same path twice, which would double every subsequent event.
func TestWatchedSetPreventsDuplicateRegistration(t *testing.T) {
	set := newWatchedSet()
	if !set.add("D:/Media/Series") {
		t.Fatal("first registration should succeed")
	}
	if set.add("d:/media/series/") {
		t.Error("a case/separator variant of the same path was registered twice")
	}
	if set.add(`D:\Media\Series`) {
		t.Error("a Windows-separator variant of the same path was registered twice")
	}
	if set.Count() != 1 {
		t.Errorf("registered %d paths, want 1", set.Count())
	}
}

// TestWatcherNeverDeletesOnRemoveEvent asserts the noise-level contract: a
// filesystem remove event must not translate into a catalogue deletion. The
// watcher reports it so reconciliation can decide, because a remove can be the
// first half of a rename or an unplugged disk.
func TestWatcherNeverDeletesOnRemoveEvent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "film.mp4"), 16)

	watcher := NewEventWatcherWithOptions(New(Options{}), WatcherOptions{
		Recursive:         true,
		Debounce:          5 * time.Millisecond,
		Stability:         5 * time.Millisecond,
		RootRetryInterval: 5 * time.Millisecond,
	})
	watcher.retryInterval = watcher.options.RootRetryInterval

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan Event, 16)
	go func() {
		_ = watcher.Watch(ctx, []string{root}, func(event Event) error {
			select {
			case events <- event:
			default:
			}
			return nil
		})
	}()

	// Give the watcher time to register the root.
	time.Sleep(80 * time.Millisecond)

	path := filepath.Join(root, "film.mp4")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Kind == EventRemoved {
				// The key assertion: no File payload means no ingest and,
				// crucially, no deletion instruction.
				if event.File != nil {
					t.Fatal("remove event carried a file payload; it must not trigger an ingest")
				}
				return
			}
		case <-deadline:
			t.Skip("fsnotify did not deliver a remove event in this environment")
		}
	}
}

// TestWatcherSurvivesMissingRootAtStartup verifies a removable disk that is not
// present at boot does not prevent the watcher from starting.
func TestWatcherSurvivesMissingRootAtStartup(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "removable-media")
	watcher := NewEventWatcher(New(Options{}), false)
	watcher.retryInterval = 5 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	err := watcher.Watch(ctx, []string{missingRoot}, func(Event) error { return nil })
	if err == nil {
		t.Fatal("expected the watcher to keep waiting rather than exit silently")
	}
	// The watcher must exit only because the context ended, never because the
	// root was absent.
	if ctx.Err() == nil {
		t.Fatalf("watcher returned before the context ended: %v", err)
	}
}

// TestNormalizePathKeyFoldsCaseAndSeparators is the foundation of duplicate
// prevention: the same file seen as D:\Media\X and d:/media/x must compare equal.
func TestNormalizePathKeyFoldsCaseAndSeparators(t *testing.T) {
	variants := []string{
		`D:\Media\Movies\Film.mkv`,
		"D:/Media/Movies/Film.mkv",
		"d:/media/movies/film.mkv",
		"D:/Media/Movies/./Film.mkv",
	}
	base := NormalizePathKey(variants[0])
	for _, variant := range variants[1:] {
		if got := NormalizePathKey(variant); got != base {
			t.Errorf("NormalizePathKey(%q) = %q, want %q", variant, got, base)
		}
	}
}

// TestRootIDIsStablePerVolume verifies that root identity survives a changed
// drive letter for an external disk, which is what makes recovery scans possible.
func TestRootIDIsStablePerVolume(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"D:/Media/Movies", "d:"},
		{"D:/Other", "d:"},
		{"E:/", "e:"},
		{"//nas/share/films", "//nas/share"},
	}
	for _, testCase := range tests {
		if got := RootID(testCase.path); got != testCase.want {
			t.Errorf("RootID(%q) = %q, want %q", testCase.path, got, testCase.want)
		}
	}
}

// TestFingerprintEquality documents the change-detection contract, including the
// deliberate preference for a stable file ID over a touched modification time.
func TestFingerprintEquality(t *testing.T) {
	base := Fingerprint{Size: 100, ModTime: 1000, FileID: "id-1"}

	if !base.Equal(Fingerprint{Size: 100, ModTime: 1000, FileID: "id-1"}) {
		t.Error("identical fingerprints must be equal")
	}
	if base.Equal(Fingerprint{Size: 200, ModTime: 1000, FileID: "id-1"}) {
		t.Error("a changed size must be detected")
	}
	if base.Equal(Fingerprint{Size: 100, ModTime: 9999, FileID: "id-2"}) {
		t.Error("a changed file identity must be detected")
	}
	// Some filesystems rewrite mtime without changing content; a stable file ID
	// plus identical size is the stronger signal and must win.
	if !base.Equal(Fingerprint{Size: 100, ModTime: 5555, FileID: "id-1"}) {
		t.Error("a touched mtime with identical identity and size must count as unchanged")
	}
}

// TestErrorClassification verifies the error model that drives the report and,
// more importantly, the transient/permanent distinction that protects data.
func TestErrorClassification(t *testing.T) {
	cases := []struct {
		message   string
		wantCode  ErrorCode
		transient bool
	}{
		{"open D:/media: access is denied", ErrPermissionDenied, true},
		{"The device is not ready.", ErrDiskUnavailable, true},
		{"The network path was not found.", ErrNetworkError, true},
		{"no such file or directory", ErrNotFound, false},
		{"too many links", ErrSymlinkLoop, false},
		{"something unexpected", ErrCodeUnknown, false},
	}

	for _, testCase := range cases {
		code, _ := ClassifyError(errString(testCase.message))
		if code != testCase.wantCode {
			t.Errorf("ClassifyError(%q) = %q, want %q", testCase.message, code, testCase.wantCode)
		}
		if code.Transient() != testCase.transient {
			t.Errorf("%q transient = %v, want %v", testCase.wantCode, code.Transient(), testCase.transient)
		}
	}
}

// TestErrorSinkIsBounded verifies the report cannot grow with the number of
// failures, which is what makes it safe on a library with a million bad files.
func TestErrorSinkIsBounded(t *testing.T) {
	sink := NewErrorSink(5)
	for i := 0; i < 1000; i++ {
		sink.AddError(errString("access is denied"), "D:/path/"+itoa(i), "D:/", "stat")
	}
	if sink.Total() != 1000 {
		t.Errorf("total = %d, want 1000", sink.Total())
	}
	if len(sink.Samples()) != 5 {
		t.Errorf("retained samples = %d, want the limit of 5", len(sink.Samples()))
	}
	if sink.Count(ErrPermissionDenied) != 1000 {
		t.Errorf("permission count = %d, want 1000", sink.Count(ErrPermissionDenied))
	}
}

// errString is a minimal error implementation for classification tests.
type errString string

func (e errString) Error() string { return string(e) }
