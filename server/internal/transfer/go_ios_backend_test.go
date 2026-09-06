package transfer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGoIOSBackendRequiresConnection(t *testing.T) {
	backend := NewGoIOSBackend()
	ctx := context.Background()

	if _, err := backend.List(ctx, "/Documents"); err == nil {
		t.Fatal("List should reject an unopened backend")
	}
	if _, err := backend.Stat(ctx, "/Documents/file.mp4"); err == nil {
		t.Fatal("Stat should reject an unopened backend")
	}
	if err := backend.Mkdir(ctx, "/Documents/Series"); err == nil {
		t.Fatal("Mkdir should reject an unopened backend")
	}
	if err := backend.Delete(ctx, "/Documents/file.mp4"); err == nil {
		t.Fatal("Delete should reject an unopened backend")
	}
}

func TestGoIOSBackendHonorsCancellationBeforeConnection(t *testing.T) {
	backend := NewGoIOSBackend()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := backend.Connect(ctx, Device{Type: DeviceIOS}, TransferDestination{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Connect error = %v, want context.Canceled", err)
	}
}

func TestGoIOSBackendRejectsInvalidDeviceID(t *testing.T) {
	backend := NewGoIOSBackend()
	err := backend.Connect(context.Background(), Device{Type: DeviceIOS}, TransferDestination{
		DeviceID: "ios_mtp_0_Apple iPhone",
		AppID:    "org.videolan.vlc-ios",
	})
	if err == nil {
		t.Fatal("Connect should reject a Windows MTP placeholder")
	}
}

func TestGoIOSBackendRenameExplainsUnsupportedOperation(t *testing.T) {
	backend := NewGoIOSBackend()
	err := backend.Rename(context.Background(), "/Documents/a", "/Documents/b")
	if err == nil {
		t.Fatal("Rename should report the unsupported AFC operation")
	}
}

func TestGoIOSBackendRealDeviceSmoke(t *testing.T) {
	if os.Getenv("NEXORA_IOS_INTEGRATION") != "1" {
		t.Skip("set NEXORA_IOS_INTEGRATION=1 to use a connected iPhone")
	}

	deviceID := os.Getenv("NEXORA_IOS_UDID")
	if deviceID == "" {
		deviceID = "ios_0_00008030-001450161E00802E"
	}
	bundleID := os.Getenv("NEXORA_IOS_BUNDLE_ID")
	if bundleID == "" {
		bundleID = "org.videolan.vlc-ios"
	}

	backend := NewGoIOSBackend()
	ctx := context.Background()
	if err := backend.Connect(ctx, Device{ID: deviceID, Type: DeviceIOS}, TransferDestination{DeviceID: deviceID, DeviceType: DeviceIOS, AppID: bundleID}); err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	entries, err := backend.List(ctx, "/Documents")
	if err != nil {
		t.Fatalf("list Documents: %v", err)
	}
	t.Logf("Documents entries: %d", len(entries))

	source := filepath.Join(t.TempDir(), "go-ios-smoke.txt")
	if err := os.WriteFile(source, []byte("NEXORA go-ios smoke test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	remotePath := "/Documents/NEXORA-go-ios-smoke.txt"
	t.Cleanup(func() { _ = backend.Delete(ctx, remotePath) })
	if err := backend.Put(ctx, source, remotePath, PutOptions{BufferSize: 1024}); err != nil {
		t.Fatalf("put smoke file: %v", err)
	}
	entry, err := backend.Stat(ctx, remotePath)
	if err != nil {
		t.Fatalf("stat smoke file: %v", err)
	}
	if entry.Size != int64(len("NEXORA go-ios smoke test\n")) {
		t.Fatalf("smoke file size = %d", entry.Size)
	}
}
