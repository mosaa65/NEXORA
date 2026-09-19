package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// writeFile creates a media file of the given size, creating parent directories.
func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestScanContinuesPastUnreadableDirectory is the central fault-tolerance
// property: one locked folder must not abort a scan of the rest of the disk.
//
// The assertion is deliberately about the *other* files being indexed. A
// permission error is recorded and classified, but traversal continues.
func TestScanContinuesPastUnreadableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows; covered by the classified-error test")
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Disk1", "Movies", "one.mp4"), 16)
	writeFile(t, filepath.Join(root, "Disk1", "Series", "two.mkv"), 16)
	writeFile(t, filepath.Join(root, "Disk2", "three.avi"), 16)

	locked := filepath.Join(root, "Disk1", "Private")
	if err := os.MkdirAll(locked, 0o000); err != nil {
		t.Skipf("cannot create an unreadable directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	var found int
	report, err := New(Options{Workers: 2}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(FileInfo) error {
			found++
			return nil
		})

	if err != nil {
		t.Fatalf("scan failed despite isolating the unreadable directory: %v", err)
	}
	if found != 3 {
		t.Fatalf("found %d files, want 3 (the unreadable folder must not abort the scan)", found)
	}
	if report.Progress.Status != StatusCompleted && report.Progress.Status != StatusPartial {
		t.Errorf("unexpected status %q", report.Progress.Status)
	}
}

// TestUnavailableRootDoesNotStopOtherRoots verifies per-root isolation: a
// missing disk must leave the other disks scanning normally.
func TestUnavailableRootDoesNotStopOtherRoots(t *testing.T) {
	goodRoot := t.TempDir()
	writeFile(t, filepath.Join(goodRoot, "Movies", "film.mp4"), 16)
	writeFile(t, filepath.Join(goodRoot, "Series", "S01E01.mkv"), 16)

	missingRoot := filepath.Join(t.TempDir(), "not-plugged-in")

	var found int
	report, err := New(Options{Workers: 2}).ScanTree(context.Background(),
		[]string{missingRoot, goodRoot},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true},
		func(FileInfo) error { found++; return nil })

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if found != 2 {
		t.Fatalf("found %d files, want 2 from the available root", found)
	}
	if state := report.RootStates[missingRoot]; state != RootUnavailable {
		t.Errorf("missing root state = %q, want %q", state, RootUnavailable)
	}
	if state := report.RootStates[goodRoot]; state != RootCompleted {
		t.Errorf("good root state = %q, want %q", state, RootCompleted)
	}
	// An unavailable root must make the session PARTIAL, not failed: the
	// catalogue is still valid, it is simply incomplete for that disk.
	if report.Progress.Status != StatusPartial {
		t.Errorf("status = %q, want %q", report.Progress.Status, StatusPartial)
	}
}

// TestIncrementalScanSkipsUnchangedFiles is the core incremental property. A
// second scan with the same catalogue must not re-emit or re-parse anything.
func TestIncrementalScanSkipsUnchangedFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Movies", "film.mp4"), 32)
	writeFile(t, filepath.Join(root, "Series", "S01E01.mkv"), 32)

	s := New(Options{Workers: 2})

	// First pass: full scan discovers both files.
	known := make([]KnownFile, 0, 2)
	first, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			known = append(known, KnownFile{
				ID:      int64(len(known) + 1),
				Path:    file.Path,
				Size:    file.Size,
				ModTime: file.ModTime,
				FileID:  file.FileID,
			})
			return nil
		})
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if first.Progress.NewFiles != 2 {
		t.Fatalf("first scan new = %d, want 2", first.Progress.NewFiles)
	}

	// Second pass: an incremental scan must skip both files.
	var emitted int
	second, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeIncremental, Known: known}, func(FileInfo) error {
			emitted++
			return nil
		})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if emitted != 0 {
		t.Errorf("incremental scan emitted %d unchanged files, want 0", emitted)
	}
	if second.Progress.UnchangedFiles != 2 {
		t.Errorf("unchanged = %d, want 2", second.Progress.UnchangedFiles)
	}
	if second.Progress.NewFiles != 0 || second.Progress.ModifiedFiles != 0 {
		t.Errorf("incremental scan reported new=%d changed=%d, want 0/0",
			second.Progress.NewFiles, second.Progress.ModifiedFiles)
	}
	if second.Progress.SkippedBytes == 0 {
		t.Error("expected skipped bytes to be reported for unchanged files")
	}
}

// TestIncrementalScanDetectsModification verifies that a changed size is
// detected even when the path is identical.
func TestIncrementalScanDetectsModification(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Movies", "film.mp4")
	writeFile(t, path, 32)

	s := New(Options{Workers: 2})
	var original FileInfo
	if _, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			original = file
			return nil
		}); err != nil {
		t.Fatalf("first scan: %v", err)
	}

	known := []KnownFile{{
		ID: 1, Path: original.Path, Size: original.Size,
		ModTime: original.ModTime, FileID: original.FileID,
	}}

	// Grow the file and move its modification time forward.
	writeFile(t, path, 4096)
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	var changed []FileInfo
	report, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeIncremental, Known: known}, func(file FileInfo) error {
			changed = append(changed, file)
			return nil
		})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected the modified file to be emitted, got %d files", len(changed))
	}
	if changed[0].Change != ChangeChanged {
		t.Errorf("change kind = %q, want %q", changed[0].Change, ChangeChanged)
	}
	if report.Progress.ModifiedFiles != 1 {
		t.Errorf("modified = %d, want 1", report.Progress.ModifiedFiles)
	}
}

// TestRenameIsNotDeletePlusCreate is the duplicate-prevention test. A moved file
// must be reported as a rename against the same record, not as a new file.
func TestRenameIsNotDeletePlusCreate(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "Movies", "old-name.mp4")
	writeFile(t, oldPath, 64)

	s := New(Options{Workers: 2})
	var original FileInfo
	if _, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			original = file
			return nil
		}); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if original.FileID == "" {
		t.Skip("this filesystem exposes no file identity; rename inference needs it")
	}

	known := []KnownFile{{
		ID: 42, Path: original.Path, Size: original.Size,
		ModTime: original.ModTime, FileID: original.FileID,
	}}

	newPath := filepath.Join(root, "Movies", "Renamed Title 2024.mp4")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("rename: %v", err)
	}

	var results []FileInfo
	report, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeIncremental, Known: known}, func(file FileInfo) error {
			results = append(results, file)
			return nil
		})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected one result for the renamed file, got %d", len(results))
	}
	if results[0].Change != ChangeRenamed {
		t.Fatalf("change kind = %q, want %q (a rename must not be a delete plus create)",
			results[0].Change, ChangeRenamed)
	}
	if results[0].KnownID != 42 {
		t.Errorf("known id = %d, want 42 (the record must keep its identity)", results[0].KnownID)
	}
	if results[0].RenamedFrom != original.Path {
		t.Errorf("renamed from = %q, want %q", results[0].RenamedFrom, original.Path)
	}
	if report.Progress.NewFiles != 0 {
		t.Errorf("rename reported %d new files; it must be 0", report.Progress.NewFiles)
	}
	if report.Progress.RenamedFiles != 1 {
		t.Errorf("renamed = %d, want 1", report.Progress.RenamedFiles)
	}
}

// TestMissingFilesAreReportedNotDeleted verifies reconciliation input: a deleted
// file becomes a MISSING candidate, and the scanner itself deletes nothing.
func TestMissingFilesAreReportedNotDeleted(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "Movies", "keep.mp4")
	gone := filepath.Join(root, "Movies", "gone.mp4")
	writeFile(t, keep, 16)
	writeFile(t, gone, 16)

	s := New(Options{Workers: 2})
	known := []KnownFile{
		{ID: 1, Path: NormalizePathKey(keep), Size: 16},
		{ID: 2, Path: NormalizePathKey(gone), Size: 16},
	}
	// Seed real fingerprints so nothing is treated as changed.
	if _, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			for i := range known {
				if NormalizePathKey(known[i].Path) == NormalizePathKey(file.Path) {
					known[i].Size = file.Size
					known[i].ModTime = file.ModTime
					known[i].FileID = file.FileID
				}
			}
			return nil
		}); err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	if err := os.Remove(gone); err != nil {
		t.Fatalf("remove: %v", err)
	}

	report, err := s.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeReconcile, Known: known}, func(FileInfo) error { return nil })
	if err != nil {
		t.Fatalf("reconcile scan: %v", err)
	}

	if len(report.Missing) != 1 {
		t.Fatalf("missing = %d, want 1", len(report.Missing))
	}
	if report.Missing[0].ID != 2 {
		t.Errorf("missing id = %d, want 2", report.Missing[0].ID)
	}
	if report.Progress.RemovedFiles != 0 {
		t.Error("the scanner must not report deletions; reconciliation owns that decision")
	}
}

// TestRunnerRejectsUnsupportedExtensionsAndTempFiles checks candidate filtering,
// including the in-progress download artefacts that would otherwise be indexed.
func TestRunnerRejectsUnsupportedExtensionsAndTempFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "good.mp4"), 16)
	writeFile(t, filepath.Join(root, "notes.txt"), 16)
	writeFile(t, filepath.Join(root, "cover.jpg"), 16)
	writeFile(t, filepath.Join(root, "movie.mkv.part"), 16)
	writeFile(t, filepath.Join(root, "other.crdownload"), 16)

	var seen []string
	report, err := New(Options{Workers: 2}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			seen = append(seen, filepath.Base(file.Path))
			return nil
		})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(seen) != 1 || seen[0] != "good.mp4" {
		t.Fatalf("accepted %v, want only good.mp4", seen)
	}
	if report.Progress.Rejected == 0 {
		t.Error("expected rejected files to be counted")
	}
	if report.Progress.FilesSeen < 5 {
		t.Errorf("files seen = %d, want at least 5", report.Progress.FilesSeen)
	}
}

// TestIgnoredDirectoriesAreSkipped verifies the ignore policy, including the
// system directories that make a naive scan crawl on Windows.
func TestIgnoredDirectoriesAreSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Media", "film.mp4"), 16)
	writeFile(t, filepath.Join(root, "$RECYCLE.BIN", "junk.mp4"), 16)
	writeFile(t, filepath.Join(root, "Media", "node_modules", "pkg", "x.mp4"), 16)
	writeFile(t, filepath.Join(root, "Media", ".hidden", "y.mp4"), 16)
	writeFile(t, filepath.Join(root, "Media", "CustomSkip", "z.mp4"), 16)

	var seen []string
	if _, err := New(Options{
		Workers: 2,
		Discovery: DiscoveryConfig{
			IgnoreHidden: true,
			IgnoreDirs:   []string{"CustomSkip"},
		},
	}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			seen = append(seen, filepath.Base(filepath.Dir(file.Path))+"/"+filepath.Base(file.Path))
			return nil
		}); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(seen) != 1 {
		t.Fatalf("scanned %v, want only the Media/film.mp4 entry", seen)
	}
}

// TestScanHonoursContextCancellation verifies that cancelling a scan stops it
// promptly and reports CANCELLED rather than FAILED.
func TestScanHonoursContextCancellation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 200; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 8)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := New(Options{Workers: 1})
	_, err := s.ScanTree(ctx, []string{root}, ScanOptions{Mode: ModeFull, EmitUnchanged: true},
		func(FileInfo) error {
			// Cancel as soon as scanning is genuinely under way.
			cancel()
			return nil
		})
	if err == nil {
		t.Fatal("expected the scan to report cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// TestScanIsDeterministicAboutRootDeduplication ensures the same root listed
// twice does not double-index the library.
func TestScanIsDeterministicAboutRootDeduplication(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "film.mp4"), 16)

	var count int
	if _, err := New(Options{Workers: 2}).ScanTree(context.Background(),
		[]string{root, root, filepath.ToSlash(root)},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true},
		func(FileInfo) error { count++; return nil }); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 1 {
		t.Fatalf("file emitted %d times for duplicated roots, want 1", count)
	}
}

// TestArtworkResolverCachesPerDirectory proves the artwork optimisation: many
// videos in one folder must not produce one ReadDir per video.
func TestArtworkResolverCachesPerDirectory(t *testing.T) {
	root := t.TempDir()
	seasonDir := filepath.Join(root, "Series", "Show", "Season 01")
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(seasonDir, "S01E"+itoa(i)+".mkv"), 8)
	}
	writeFile(t, filepath.Join(root, "Series", "Show", "poster.jpg"), 8)

	resolver := newArtworkResolver()
	var resolved string
	if _, err := New(Options{Workers: 4}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			if file.ArtworkPath != "" {
				resolved = file.ArtworkPath
			}
			return nil
		}); err != nil {
		t.Fatalf("scan: %v", err)
	}
	_ = resolver
	if resolved == "" {
		t.Fatal("expected artwork to be discovered from the parent folder")
	}

	// Direct proof of the cache: resolving many paths in one directory performs
	// a bounded number of directory lookups, not one per file.
	cached := newArtworkResolver()
	for i := 0; i < 50; i++ {
		cached.Resolve(filepath.Join(seasonDir, "S01E"+itoa(i)+".mkv"))
	}
	if got := cached.Size(); got > 4 {
		t.Errorf("directory lookups memoised = %d, want a small constant (not one per file)", got)
	}
}

// TestArtworkRankingPrefersPoster verifies deterministic artwork selection when
// several candidate images exist in one folder.
func TestArtworkRankingPrefersPoster(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "banner.jpg"), 8)
	writeFile(t, filepath.Join(dir, "fanart.jpg"), 8)
	writeFile(t, filepath.Join(dir, "poster.jpg"), 8)
	writeFile(t, filepath.Join(dir, "random-image.jpg"), 8)

	got := searchDirForArtwork(dir)
	if filepath.Base(got) != "poster.jpg" {
		t.Errorf("artwork = %q, want poster.jpg to win the ranking", filepath.Base(got))
	}
}

// TestProgressCountersAreConsistent checks the report is internally coherent,
// which is what makes it trustworthy as an operational signal.
func TestProgressCountersAreConsistent(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 100)
	}
	writeFile(t, filepath.Join(root, "readme.txt"), 100)

	report, err := New(Options{Workers: 4}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(FileInfo) error { return nil })
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	progress := report.Progress
	if progress.NewFiles+progress.ModifiedFiles+progress.UnchangedFiles+progress.RenamedFiles != 20 {
		t.Errorf("processed counters do not sum to the 20 media files: %+v", progress)
	}
	if progress.TotalBytes != 2000 {
		t.Errorf("total bytes = %d, want 2000", progress.TotalBytes)
	}
	if progress.Directories == 0 {
		t.Error("directories visited was not counted")
	}
	if progress.Duration.Seconds <= 0 {
		t.Error("duration was not recorded")
	}
	if progress.Throughput.FilesPerSecond <= 0 {
		t.Error("throughput was not measured")
	}
}

// BenchmarkScanSyntheticTree measures the pipeline against a synthetic library.
// It builds real directory entries (no file contents) so it exercises traversal,
// parsing, artwork lookup and change detection together.
func BenchmarkScanSyntheticTree(b *testing.B) {
	for _, files := range []int{1000, 10000} {
		b.Run("files="+itoa(files), func(b *testing.B) {
			root := b.TempDir()
			buildSyntheticTree(b, root, files)
			scanner := New(Options{Workers: 8})

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := scanner.ScanTree(context.Background(), []string{root},
					ScanOptions{Mode: ModeFull, EmitUnchanged: true},
					func(FileInfo) error { return nil })
				if err != nil {
					b.Fatalf("scan: %v", err)
				}
			}
		})
	}
}

// BenchmarkIncrementalScanSkipsUnchanged measures the incremental fast path,
// which is the mode a real server runs most often.
func BenchmarkIncrementalScanSkipsUnchanged(b *testing.B) {
	root := b.TempDir()
	const files = 5000
	buildSyntheticTree(b, root, files)

	scanner := New(Options{Workers: 8})
	known := make([]KnownFile, 0, files)
	if _, err := scanner.ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true}, func(file FileInfo) error {
			known = append(known, KnownFile{
				ID: int64(len(known) + 1), Path: file.Path, Size: file.Size,
				ModTime: file.ModTime, FileID: file.FileID,
			})
			return nil
		}); err != nil {
		b.Fatalf("seed scan: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := scanner.ScanTree(context.Background(), []string{root},
			ScanOptions{Mode: ModeIncremental, Known: known},
			func(FileInfo) error { return nil })
		if err != nil {
			b.Fatalf("incremental scan: %v", err)
		}
	}
}

// BenchmarkParseFilePath measures raw parse throughput, which dominates CPU cost
// on a library with many files.
func BenchmarkParseFilePath(b *testing.B) {
	paths := []string{
		"D:/Media/Series/Breaking Bad/Season 01/Breaking.Bad.S01E01.1080p.WEB-DL.x264.mkv",
		"D:/Media/مسلسلات تركية/طائر الرفراف/الجزء الثاني/akoam_ep05.mp4",
		"D:/Media/Anime - أنمي/Attack on Titan - هجوم العمالقة/Season 01/[SubGroup] Attack on Titan - 001 [1080p].mkv",
		"D:/Media/Movies/Inception (2010)/Inception.2010.2160p.BluRay.HEVC.mkv",
		"D:/Media/كرتون/Tom and Jerry/Season 01/Tom.and.Jerry.S01E05.mp4",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFilePath(paths[i%len(paths)])
	}
}

// buildSyntheticTree creates a realistic directory shape: category folders,
// per-show folders, season folders and artwork, all without file contents.
func buildSyntheticTree(tb testing.TB, root string, count int) {
	tb.Helper()
	categories := []string{"Movies", "Series", "Anime", "Kids", "Documentaries", "وثائقي"}
	showsPerCategory := 20
	for i := 0; i < count; i++ {
		category := categories[i%len(categories)]
		show := "Show " + itoa(i/showsPerCategory)
		season := "Season " + itoa((i%5)+1)
		dir := filepath.Join(root, category, show, season)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatalf("mkdir: %v", err)
		}
		// One artwork file per show folder, which is the realistic shape and the
		// case the artwork cache exists for.
		if i%10 == 0 {
			_ = os.WriteFile(filepath.Join(root, category, show, "poster.jpg"), []byte("jpeg"), 0o644)
		}
		name := "S" + itoa((i%5)+1) + "E" + itoa(i%50+1) + ".1080p.mkv"
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			tb.Fatalf("write: %v", err)
		}
	}
}

// TestNoGoroutineLeakAfterScan guards the pipeline against leaking workers. A
// scanner that leaks goroutines per run cannot survive days of operation.
func TestNoGoroutineLeakAfterScan(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 8)
	}

	before := runtime.NumGoroutine()
	for i := 0; i < 5; i++ {
		if _, err := New(Options{Workers: 4}).ScanTree(context.Background(), []string{root},
			ScanOptions{Mode: ModeFull, EmitUnchanged: true},
			func(FileInfo) error { return nil }); err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
	}
	// Allow the runtime to retire finished goroutines.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+3 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("goroutines before=%d after=%d; the scan pipeline appears to leak", before, runtime.NumGoroutine())
}

// TestConcurrentScansDoNotInterfere confirms two scans of different roots can run
// simultaneously without corrupting each other's counters.
func TestConcurrentScansDoNotInterfere(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	for i := 0; i < 20; i++ {
		writeFile(t, filepath.Join(rootA, "film"+itoa(i)+".mp4"), 8)
		writeFile(t, filepath.Join(rootB, "film"+itoa(i)+".mp4"), 8)
	}

	var wg sync.WaitGroup
	results := make([]Progress, 2)
	roots := []string{rootA, rootB}
	for index := range roots {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			report, err := New(Options{Workers: 4}).ScanTree(context.Background(), []string{roots[i]},
				ScanOptions{Mode: ModeFull, EmitUnchanged: true},
				func(FileInfo) error { return nil })
			if err != nil {
				t.Errorf("scan %d: %v", i, err)
				return
			}
			results[i] = report.Progress
		}(index)
	}
	wg.Wait()

	for i, progress := range results {
		if progress.NewFiles != 20 {
			t.Errorf("scan %d counted %d new files, want 20", i, progress.NewFiles)
		}
	}
}
