package transfer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// runEngineToCompletion submits a job to a fresh engine backed by a local
// StorageBackend and waits until the job reaches a terminal state.
func runEngineToCompletion(t *testing.T, cfg TransferConfig, job *TransferJobV2) {
	t.Helper()

	eng := NewTransferEngine(cfg, func(device Device) (TransferBackend, error) {
		return NewStorageBackend(), nil
	})

	if err := eng.Submit(job); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := eng.GetJob(job.ID)
		if ok {
			switch cur.Status {
			case StatusCompleted, StatusFailed, StatusCancelled:
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, ok := eng.GetJob(job.ID); !ok {
		return
	}
	t.Fatalf("job did not finish within deadline (status=%q)", job.Status)
}

func writeTempFile(t *testing.T, dir, name string, size int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEngineCopiesMultipleFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	names := []string{"a.txt", "b.mp4", "c.txt"}
	wantSizes := map[string]int64{
		"a.txt": 1_000,
		"b.mp4": 5_000,
		"c.txt": 100,
	}
	for _, n := range names {
		writeTempFile(t, srcDir, n, int(wantSizes[n]))
	}

	files := make([]TransferFile, 0, len(names))
	for _, n := range names {
		info, err := os.Stat(filepath.Join(srcDir, n))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, TransferFile{SourcePath: filepath.Join(srcDir, n), Size: info.Size()})
	}

	job := &TransferJobV2{
		ID:          "job-1",
		DeviceID:    "USB-DRIVE",
		Destination: TransferDestination{DeviceID: "USB-DRIVE", DeviceType: DeviceStorage, RemotePath: dstDir},
		Files:       files,
		Status:      StatusQueued,
	}

	runEngineToCompletion(t, DefaultTransferConfig(), job)

	if job.Status != StatusCompleted {
		t.Fatalf("expected completed, got status=%q err=%v", job.Status, job.Error)
	}
	if job.Progress != 100 {
		t.Fatalf("expected progress 100, got %v", job.Progress)
	}
	if job.TransferredBytes != job.TotalBytes {
		t.Fatalf("transferred %d != total %d", job.TransferredBytes, job.TotalBytes)
	}
	for _, n := range names {
		src := filepath.Join(srcDir, n)
		dst := filepath.Join(dstDir, n)
		sinfo, serr := os.Stat(src)
		dinfo, derr := os.Stat(dst)
		if derr != nil {
			t.Fatalf("destination %s missing: %v", dst, derr)
		}
		if serr != nil || sinfo.Size() != dinfo.Size() {
			t.Fatalf("size mismatch for %s: src=%v dst=%v", n, sinfo.Size(), dinfo.Size())
		}
	}
}

func TestEngineMissingSourceFailsJob(t *testing.T) {
	dstDir := t.TempDir()

	job := &TransferJobV2{
		ID:          "job-bad",
		DeviceID:    "USB-DRIVE",
		Destination: TransferDestination{DeviceID: "USB-DRIVE", DeviceType: DeviceStorage, RemotePath: dstDir},
		Files:       []TransferFile{{SourcePath: filepath.Join(t.TempDir(), "does-not-exist.txt")}},
		Status:      StatusQueued,
	}

	runEngineToCompletion(t, DefaultTransferConfig(), job)

	if job.Status != StatusFailed {
		t.Fatalf("expected failed, got status=%q", job.Status)
	}
	if job.Error == nil || job.Error.Code != CodeSourceNotFound {
		t.Fatalf("expected source-not-found error, got %+v", job.Error)
	}
}

func TestSchedulerRoutesJobsConcurrentlyByDevice(t *testing.T) {
	eng := NewTransferEngine(DefaultTransferConfig(), func(device Device) (TransferBackend, error) {
		return NewStorageBackend(), nil
	})

	j1 := &TransferJobV2{ID: "j1", DeviceID: "d1", Destination: TransferDestination{DeviceID: "d1", DeviceType: DeviceStorage, RemotePath: t.TempDir()}, Files: nil, Status: StatusQueued}
	j2 := &TransferJobV2{ID: "j2", DeviceID: "d1", Destination: TransferDestination{DeviceID: "d1", DeviceType: DeviceStorage, RemotePath: t.TempDir()}, Files: nil, Status: StatusQueued}

	if err := eng.Submit(j1); err != nil {
		t.Fatal(err)
	}
	sched := eng.sched
	if len(sched.workers) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(sched.workers))
	}
	// A job with a different device id creates a second worker.
	if err := eng.Submit(j2); err != nil {
		t.Fatal(err)
	}
	if err := eng.Submit(&TransferJobV2{ID: "j3", DeviceID: "d2", Destination: TransferDestination{DeviceID: "d2", DeviceType: DeviceStorage, RemotePath: t.TempDir()}, Files: nil, Status: StatusQueued}); err != nil {
		t.Fatal(err)
	}
	if len(sched.workers) != 2 {
		t.Fatalf("expected 2 workers, got %d", len(sched.workers))
	}
}

func TestRetryBackoff(t *testing.T) {
	cfg := DefaultTransferConfig()
	if got := retryBackoff(cfg, 1); got != cfg.RetryInitialDelay {
		t.Fatalf("attempt 1 backoff = %v, want %v", got, cfg.RetryInitialDelay)
	}
	if got := retryBackoff(cfg, 10); got > cfg.RetryMaxDelay {
		t.Fatalf("backoff exceeded max: %v", got)
	}
}
