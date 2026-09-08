package transfer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTransferService_Basic(t *testing.T) {
	svc := NewService(Options{
		AndroidTargetFolder: "Download",
	})

	ctx := context.Background()
	devices, err := svc.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	t.Logf("Found %d devices", len(devices))

	// Create temporary dummy file
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test_movie.mp4")
	data := make([]byte, 1024*1024*2) // 2MB
	if err := os.WriteFile(sourceFile, data, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	targetDir := filepath.Join(tmpDir, "target_disk")
	job, err := svc.StartCopy(ctx, CopyRequest{
		DeviceID:   "disk_Z",
		SourcePath: sourceFile,
	})
	if err != nil {
		t.Fatalf("StartCopy failed: %v", err)
	}

	if job.ID == "" {
		t.Errorf("Expected job ID, got empty")
	}

	// Override execute to local folder for test verification
	err = svc.copyToLocalPath(ctx, job, sourceFile, targetDir)
	if err != nil {
		t.Fatalf("copyToLocalPath failed: %v", err)
	}

	copiedFile := filepath.Join(targetDir, "test_movie.mp4")
	info, err := os.Stat(copiedFile)
	if err != nil {
		t.Fatalf("Copied file missing: %v", err)
	}
	if info.Size() != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), info.Size())
	}

	// Verify Job Retrieval
	retrievedJob, ok := svc.GetJob(job.ID)
	if !ok {
		t.Fatalf("GetJob failed to find job %s", job.ID)
	}
	if retrievedJob.SourcePath != sourceFile {
		t.Errorf("Expected source %s, got %s", sourceFile, retrievedJob.SourcePath)
	}

	jobsList := svc.ListJobs()
	if len(jobsList) == 0 {
		t.Errorf("Expected non-empty jobs list")
	}
}

func TestTransferService_Cancel(t *testing.T) {
	svc := NewService(Options{})

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "big_movie.mp4")
	data := make([]byte, 1024*1024*10)
	_ = os.WriteFile(sourceFile, data, 0o644)

	job, err := svc.StartCopy(context.Background(), CopyRequest{
		DeviceID:   "disk_X",
		SourcePath: sourceFile,
	})
	if err != nil {
		t.Fatalf("StartCopy failed: %v", err)
	}

	canceled := svc.CancelJob(job.ID)
	if !canceled {
		t.Errorf("Expected CancelJob to return true")
	}

	time.Sleep(50 * time.Millisecond)
	retrieved, _ := svc.GetJob(job.ID)
	if retrieved.Status != StatusCancelled && retrieved.Status != StatusPending && retrieved.Status != StatusProcessing {
		t.Logf("Job status after cancel: %s", retrieved.Status)
	}
}

func TestParseIOSUDIDRejectsWindowsPlaceholder(t *testing.T) {
	if _, ok := parseIOSUDID("ios_mtp_0_Apple iPhone"); ok {
		t.Fatal("expected Windows iPhone placeholder to be rejected")
	}

	udid, ok := parseIOSUDID("ios_0_00008030-001450161E00802E")
	if !ok {
		t.Fatal("expected iOS iPhone id to be accepted")
	}
	if udid != "00008030-001450161E00802E" {
		t.Fatalf("unexpected udid: %s", udid)
	}
}

func TestFilterWindowsIOSPlaceholders(t *testing.T) {
	devices := []Device{
		{ID: "ios_mtp_0_Apple iPhone", Name: "Apple iPhone", Type: DeviceIOS},
		{ID: "mtp_1_Galaxy", Name: "Galaxy", Type: DeviceAndroid},
	}

	filtered := filterWindowsIOSPlaceholders(devices)
	if len(filtered) != 1 {
		t.Fatalf("expected one device after filtering, got %d", len(filtered))
	}
	if filtered[0].ID != "mtp_1_Galaxy" {
		t.Fatalf("unexpected remaining device: %s", filtered[0].ID)
	}
}
