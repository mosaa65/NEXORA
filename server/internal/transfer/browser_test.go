package transfer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// browserService builds a Service wired to a Storage backend so File Browser
// methods (Phase 3) can be exercised without a physical device.
func browserService(t *testing.T) *Service {
	t.Helper()
	svc := NewService(Options{})
	svc.engine = NewTransferEngine(DefaultTransferConfig(), func(device Device) (TransferBackend, error) {
		return NewStorageBackend(), nil
	})
	return svc
}

func TestBrowserStorageListAndStat(t *testing.T) {
	svc := browserService(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "SubDir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry, err := svc.ListDevicePath(context.Background(), "disk_X", root, "", "")
	if err != nil {
		t.Fatalf("ListDevicePath: %v", err)
	}
	foundFile, foundDir := false, false
	for _, e := range entry {
		if e.Name == "a.txt" && !e.IsDir {
			foundFile = true
		}
		if e.Name == "SubDir" && e.IsDir {
			foundDir = true
		}
	}
	if !foundFile || !foundDir {
		t.Fatalf("list = %+v, want a.txt and SubDir", entry)
	}

	st, err := svc.StatDevicePath(context.Background(), "disk_X", filepath.Join(root, "a.txt"), "", "")
	if err != nil {
		t.Fatalf("StatDevicePath: %v", err)
	}
	if st.Name != "a.txt" || st.Size != 5 {
		t.Fatalf("stat = %+v", st)
	}
}

func TestBrowserStorageMkdir(t *testing.T) {
	svc := browserService(t)
	root := t.TempDir()
	newDir := filepath.Join(root, "P", "Q")
	if err := svc.CreateDeviceFolder(context.Background(), "disk_X", newDir, "", ""); err != nil {
		t.Fatalf("CreateDeviceFolder: %v", err)
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("expected folder to exist: %v", err)
	}
}

func TestDeviceTypeOf(t *testing.T) {
	cases := []struct {
		id   string
		want DeviceType
	}{
		{"ios_0_00008100-000123456789A1CE10203040506070", DeviceIOS},
		{"disk_E", DeviceStorage},
		{"mtp_0_SM-G991B", DeviceAndroid},
	}
	for _, c := range cases {
		if got := deviceTypeOf(c.id); got != c.want {
			t.Errorf("deviceTypeOf(%q) = %q, want %q", c.id, got, c.want)
		}
	}
}
