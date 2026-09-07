package transfer

import (
	"context"
	"testing"
)

func TestBuildV2JobStorage(t *testing.T) {
	s := NewService(Options{})
	j := s.buildV2Job(CopyRequest{DeviceID: "disk_E", SourcePath: `C:\movies\a.mp4`}, []TransferFile{{SourcePath: `C:\movies\a.mp4`, Size: 12345}})
	if j == nil {
		t.Fatal("expected a v2 job for disk device")
	}
	if j.DeviceType != DeviceStorage {
		t.Fatalf("device type = %q, want storage", j.DeviceType)
	}
	if j.Destination.RemotePath != `E:\Movies` {
		t.Fatalf("remote path = %q, want E:\\Movies", j.Destination.RemotePath)
	}
	if len(j.Files) != 1 || j.Files[0].Size != 12345 {
		t.Fatalf("unexpected files: %+v", j.Files)
	}
	if j.Status != StatusQueued {
		t.Fatalf("status = %q, want queued", j.Status)
	}
}

func TestBuildV2JobStorageMultiFile(t *testing.T) {
	s := NewService(Options{})
	j := s.buildV2Job(CopyRequest{DeviceID: "disk_E", SourcePath: `C:\movies\a.mp4`}, []TransferFile{
		{SourcePath: `C:\movies\a.mp4`, Size: 100},
		{SourcePath: `C:\movies\b.mp4`, Size: 200},
		{SourcePath: `C:\movies\c.mp4`, Size: 300},
	})
	if j == nil {
		t.Fatal("expected a v2 job for disk device")
	}
	if len(j.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(j.Files))
	}
	if j.TotalBytes != 600 {
		t.Fatalf("total bytes = %d, want 600", j.TotalBytes)
	}
}

func TestBuildV2JobIOS(t *testing.T) {
	s := NewService(Options{IOSBundleID: "org.test.app"})
	j := s.buildV2Job(CopyRequest{DeviceID: "ios_0_00008100-000123456789A1CE10203040506070", SourcePath: `C:\movies\a.mp4`, SubFolder: "Series"}, []TransferFile{{SourcePath: `C:\movies\a.mp4`, Size: 100}})
	if j == nil {
		t.Fatal("expected v2 job for iOS device")
	}
	if j.DeviceType != DeviceIOS {
		t.Fatalf("device type = %q, want ios", j.DeviceType)
	}
	if j.Destination.AppID != "org.test.app" {
		t.Fatalf("app id = %q", j.Destination.AppID)
	}
	if j.Destination.RemotePath != "/Documents/Series" {
		t.Fatalf("remote path = %q, want /Documents/Series", j.Destination.RemotePath)
	}
}

func TestBuildV2JobAndroid(t *testing.T) {
	s := NewService(Options{AndroidTargetFolder: "Movies"})
	j := s.buildV2Job(CopyRequest{DeviceID: "mtp_0_SM-G991B", SourcePath: `C:\movies\a.mp4`, SubFolder: "Kids"}, []TransferFile{{SourcePath: `C:\movies\a.mp4`, Size: 100}})
	if j == nil {
		t.Fatal("expected v2 job for Android device")
	}
	if j.DeviceType != DeviceAndroid {
		t.Fatalf("device type = %q, want android", j.DeviceType)
	}
	if j.Destination.RemotePath != "Movies/Kids" {
		t.Fatalf("remote path = %q, want Movies/Kids", j.Destination.RemotePath)
	}
}

func TestBuildV2JobNilCases(t *testing.T) {
	s := NewService(Options{})
	defer s.Close()
	if j := s.buildV2Job(CopyRequest{SourcePath: "", DeviceID: "disk_E"}, nil); j != nil {
		t.Fatalf("empty source should return nil, got %+v", j)
	}
	if j := s.buildV2Job(CopyRequest{SourcePath: `C:\a.mp4`, DeviceID: ""}, []TransferFile{{SourcePath: `C:\a.mp4`}}); j != nil {
		t.Fatalf("empty device should return nil, got %+v", j)
	}
}

func TestV2Outcome(t *testing.T) {
	completed := &TransferJobV2{Status: StatusCompleted}
	if err := v2Outcome(completed); err != nil {
		t.Fatalf("completed should yield nil, got %v", err)
	}

	cancelled := &TransferJobV2{Status: StatusCancelled}
	if err := v2Outcome(cancelled); err != context.Canceled {
		t.Fatalf("cancelled should yield context.Canceled, got %v", err)
	}

	failed := &TransferJobV2{Status: StatusFailed, Error: &TransferError{Code: CodeSourceNotFound, Message: "missing"}}
	if err := v2Outcome(failed); err == nil {
		t.Fatal("failed should yield a non-nil error")
	}
}
