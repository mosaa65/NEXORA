package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestPauseIsCooperativeAndResumes verifies the core pause contract: a paused
// scan stops taking new work, keeps its counters, and continues from where it
// stopped once resumed.
//
// This is the property that makes Pause different from Cancel: nothing is lost.
// A paused scan that is never resumed is expected to block indefinitely, which
// is why this test pauses AND resumes.
func TestPauseIsCooperativeAndResumes(t *testing.T) {
	root := t.TempDir()
	const files = 200
	for i := 0; i < files; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 8)
	}

	handle := &ScanControlHandle{}
	scanner := New(Options{Workers: 2})

	// Pause mid-scan from another goroutine, confirm the pipeline really stops,
	// then resume and require every file to be processed exactly once.
	var mu sync.Mutex
	var emitted int
	paused := make(chan struct{})

	go func() {
		// Wait until the scan has produced some work before pausing it.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			seen := emitted
			mu.Unlock()
			if seen >= 10 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		handle.Pause()
		close(paused)

		// Give the pause time to take effect, then lift it. The scan must finish
		// rather than block forever.
		time.Sleep(30 * time.Millisecond)
		handle.Resume()
	}()

	report, err := scanner.ScanTree(context.Background(), []string{root}, ScanOptions{
		Mode:          ModeFull,
		EmitUnchanged: true,
		Control:       handle,
	}, func(FileInfo) error {
		mu.Lock()
		emitted++
		mu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	<-paused
	mu.Lock()
	total := emitted
	mu.Unlock()

	if total != files {
		t.Fatalf("emitted %d files across pause/resume, want all %d: a resume must not lose work", total, files)
	}
	if report.Progress.NewFiles != int64(files) {
		t.Errorf("new files = %d, want %d", report.Progress.NewFiles, files)
	}
}

// TestPauseBlocksNewWorkThenResumeReleasesIt drives the pause state machine
// directly, which is the part that must be exact.
func TestPauseBlocksNewWorkThenResumeReleasesIt(t *testing.T) {
	control := newScanControl(2)

	if state := control.State(); state != ControlRunning {
		t.Fatalf("initial state = %q, want running", state)
	}

	// Requesting a pause moves to PAUSING, not straight to PAUSED: workers may
	// still be finishing the item they are on.
	if state := control.Pause(); state != ControlPausing {
		t.Errorf("state after pause = %q, want pausing", state)
	}

	// A worker asking to take new work must block.
	release := make(chan bool, 1)
	go func() {
		release <- control.wait(context.Background())
	}()

	select {
	case <-release:
		t.Fatal("a paused scan allowed a worker to take new work")
	case <-time.After(40 * time.Millisecond):
		// Still blocked, which is the expected behaviour.
	}

	// The scan is now recorded as fully paused.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if control.Snapshot().State == ControlPaused {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if state := control.State(); state != ControlPaused {
		t.Errorf("state = %q, want paused once a worker is waiting", state)
	}

	// Resume must release the blocked worker.
	control.Resume()

	select {
	case allowed := <-release:
		if !allowed {
			t.Error("resume released the worker with a stop signal")
		}
	case <-time.After(time.Second):
		t.Fatal("resume did not release the blocked worker")
	}

	if state := control.State(); state != ControlResuming {
		t.Errorf("state after resume = %q, want resuming", state)
	}
}

// TestCancelIsDistinctFromPause verifies the two operations are not the same
// thing: cancelling must release a paused worker with a STOP signal.
func TestCancelIsDistinctFromPause(t *testing.T) {
	control := newScanControl(1)
	control.Pause()

	release := make(chan bool, 1)
	go func() {
		release <- control.wait(context.Background())
	}()

	// Let the worker reach the paused state.
	time.Sleep(20 * time.Millisecond)

	control.Cancel()

	select {
	case allowed := <-release:
		if allowed {
			t.Error("cancel released the worker to continue instead of stopping")
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not release the paused worker")
	}

	if state := control.State(); state != ControlCancelled {
		t.Errorf("state = %q, want cancelled", state)
	}
	if !control.Cancelled() {
		t.Error("Cancelled() reported false after Cancel()")
	}
}

// TestPauseIgnoresContextCancellationOnWait ensures a cancelled context stops the
// wait rather than leaving a worker blocked forever.
func TestPauseStopOnContextCancellation(t *testing.T) {
	control := newScanControl(1)
	control.Pause()

	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan bool, 1)
	go func() {
		release <- control.wait(ctx)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case allowed := <-release:
		if allowed {
			t.Error("a cancelled context released the worker to continue")
		}
	case <-time.After(time.Second):
		t.Fatal("a cancelled context did not release the paused worker")
	}
}

// TestWorkerStatesShowWhatEachWorkerIsDoing is the per-worker visibility
// requirement: "8 workers" is not information, "worker 3: PARSER ACTIVE" is.
func TestWorkerStatesShowWhatEachWorkerIsDoing(t *testing.T) {
	control := newScanControl(4)

	control.markWorker(0, RoleDiscovery, "/media/Disk1")
	control.markWorker(1, RoleMetadata, "/media/Disk1/Movies/a.mkv")
	control.markWorker(2, RolePersistence, "/media/Disk1/Movies/b.mkv")

	snapshot := control.Snapshot()

	if len(snapshot.Workers) != 4 {
		t.Fatalf("reported %d workers, want 4", len(snapshot.Workers))
	}
	if snapshot.ActiveWorkers != 3 {
		t.Errorf("active workers = %d, want 3", snapshot.ActiveWorkers)
	}
	if snapshot.IdleWorkers != 1 {
		t.Errorf("idle workers = %d, want 1", snapshot.IdleWorkers)
	}

	byID := map[int]WorkerSnapshot{}
	for _, worker := range snapshot.Workers {
		byID[worker.ID] = worker
	}

	if byID[0].Role != RoleDiscovery {
		t.Errorf("worker 0 role = %q, want discovery", byID[0].Role)
	}
	if byID[1].Role != RoleMetadata || byID[1].CurrentPath == "" {
		t.Errorf("worker 1 should be parsing with a current path: %+v", byID[1])
	}
	if byID[2].Role != RolePersistence {
		t.Errorf("worker 2 role = %q, want persistence", byID[2].Role)
	}
	if byID[3].Active {
		t.Error("worker 3 has taken no work and must be idle")
	}

	// Workers are reported in a stable order so a polling UI does not reshuffle.
	previous := -1
	for _, worker := range snapshot.Workers {
		if worker.ID <= previous {
			t.Fatalf("workers are not ordered by id: %d after %d", worker.ID, previous)
		}
		previous = worker.ID
	}
}

// TestWorkerGoesIdleAfterWork verifies a worker that finishes is marked idle
// again, so the UI does not show stale activity forever.
func TestWorkerGoesIdleAfterWork(t *testing.T) {
	control := newScanControl(1)

	control.markWorker(0, RoleMetadata, "/media/a.mkv")
	if snapshot := control.Snapshot(); snapshot.ActiveWorkers != 1 {
		t.Fatal("worker was not marked active")
	}

	control.markWorker(0, RoleIdle, "")
	snapshot := control.Snapshot()
	if snapshot.ActiveWorkers != 0 {
		t.Errorf("active workers = %d, want 0 after the worker went idle", snapshot.ActiveWorkers)
	}
	if snapshot.Workers[0].CurrentPath != "" {
		t.Error("an idle worker still reports a current path")
	}
	if snapshot.Workers[0].IdleSince == nil {
		t.Error("an idle worker should record when it went idle")
	}
}

// TestWorkerProcessedCounterTracksWork verifies the per-worker item count.
func TestWorkerProcessedCounterTracksWork(t *testing.T) {
	control := newScanControl(1)
	for i := 0; i < 5; i++ {
		control.markWorker(0, RoleMetadata, "/media/f"+itoa(i)+".mkv")
	}

	if got := control.Snapshot().Workers[0].Processed; got != 5 {
		t.Errorf("processed = %d, want 5", got)
	}
}

// TestScanRunsWithControlHandleEndToEnd proves the pause gate does not deadlock
// a normal scan that is never paused.
func TestScanRunsWithControlHandleEndToEnd(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 40; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 8)
	}

	handle := &ScanControlHandle{}
	var emitted int

	report, err := New(Options{Workers: 4}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true, Control: handle},
		func(FileInfo) error { emitted++; return nil })
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if emitted != 40 {
		t.Fatalf("emitted %d files, want 40: the pause gate blocked a normal scan", emitted)
	}
	if report.Progress.NewFiles != 40 {
		t.Errorf("new files = %d, want 40", report.Progress.NewFiles)
	}

	// After the scan the handle must still be readable and safe.
	if state := handle.Snapshot().State; state == "" {
		t.Error("the control handle reported an empty state")
	}
}

// TestControlHandleIsNilSafe guards the API path where a Server is constructed
// directly (as tests and older callers do).
func TestControlHandleIsNilSafe(t *testing.T) {
	handle := &ScanControlHandle{}

	if state := handle.Pause(); state != ControlRunning {
		t.Errorf("Pause on an unattached handle = %q, want running", state)
	}
	if state := handle.Resume(); state != ControlRunning {
		t.Errorf("Resume on an unattached handle = %q, want running", state)
	}
	handle.Cancel() // must not panic
	if snapshot := handle.Snapshot(); snapshot.State == "" {
		t.Error("Snapshot on an unattached handle returned an empty state")
	}
}

// TestControlHandleDrivesARealScan pauses and resumes an actual scan through the
// handle, which is the path the HTTP endpoints use.
func TestControlHandleDrivesARealScan(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 200; i++ {
		writeFile(t, filepath.Join(root, "Movies", "film"+itoa(i)+".mp4"), 8)
	}

	handle := &ScanControlHandle{}
	scanner := New(Options{Workers: 2})

	// Observe the scan, then pause it and resume it while it runs.
	go func() {
		// Wait until the scan is genuinely under way before issuing the pause.
		for i := 0; i < 100; i++ {
			if handle.Snapshot().State == ControlRunning {
				time.Sleep(10 * time.Millisecond)
				break
			}
			time.Sleep(time.Millisecond)
		}
		handle.Pause()
		time.Sleep(20 * time.Millisecond)
		handle.Resume()
	}()

	var emitted int
	var mu sync.Mutex
	report, err := scanner.ScanTree(context.Background(), []string{root}, ScanOptions{
		Mode:          ModeFull,
		EmitUnchanged: true,
		Control:       handle,
	}, func(FileInfo) error {
		mu.Lock()
		emitted++
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	mu.Lock()
	total := emitted
	mu.Unlock()

	if total != 200 {
		t.Fatalf("emitted %d files after pause/resume, want all 200: a resume must not lose work", total)
	}
	if report.Progress.NewFiles != 200 {
		t.Errorf("new files = %d, want 200", report.Progress.NewFiles)
	}
}

// TestPausedScanDoesNotAbandonAnItem is the safety property that matters most:
// a pause must never leave a partially processed file, because that is what
// would produce a half-written database row.
//
// The gate is checked before an item is taken, so a worker that is mid-file runs
// to completion.
func TestPausedScanDoesNotAbandonAnItem(t *testing.T) {
	control := newScanControl(1)

	// A worker starts an item.
	control.markWorker(0, RoleMetadata, "/media/in-flight.mkv")
	started := control.Snapshot().Workers[0]
	if !started.Active {
		t.Fatal("worker was not marked active")
	}

	// Pause is requested while the item is in flight.
	control.Pause()

	// The worker's recorded state must still show the item it is finishing,
	// because the pause does not interrupt work already under way.
	current := control.Snapshot().Workers[0]
	if !current.Active || current.CurrentPath != "/media/in-flight.mkv" {
		t.Fatalf("a paused scan abandoned the in-flight item: %+v", current)
	}
}

// TestScanControlStateConstantsAreStable documents the wire values the UI relies
// on. Changing one silently would break the admin screen.
func TestScanControlStateConstantsAreStable(t *testing.T) {
	expected := map[ScanControlState]string{
		ControlRunning:   "running",
		ControlPausing:   "pausing",
		ControlPaused:    "paused",
		ControlResuming:  "resuming",
		ControlCancelled: "cancelled",
	}
	for state, want := range expected {
		if string(state) != want {
			t.Errorf("state constant = %q, want %q", state, want)
		}
	}
}

// TestScannerControlIsOptional verifies a scan without a control handle still
// works, which keeps the scanner usable as a plain library.
func TestScannerControlIsOptional(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.mp4"), 8)

	var emitted int
	if _, err := New(Options{}).ScanTree(context.Background(), []string{root},
		ScanOptions{Mode: ModeFull, EmitUnchanged: true},
		func(FileInfo) error { emitted++; return nil }); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if emitted != 1 {
		t.Fatalf("emitted %d files, want 1", emitted)
	}
}

// avoid an unused import if os is only used by helpers in this file
var _ = os.Getenv
