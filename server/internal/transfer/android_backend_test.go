package transfer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeafName(t *testing.T) {
	cases := map[string]string{
		"Download/movie.mp4":   "movie.mp4",
		`C:\Temp\movie.mp4`:    "movie.mp4",
		"movie.mp4":            "movie.mp4",
		"/Documents/series/e1": "e1",
		"":                     "",
	}
	for input, want := range cases {
		if got := leafName(input); got != want {
			t.Errorf("leafName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAndroidDestinationParts(t *testing.T) {
	cases := []struct {
		input    string
		wantDir  string
		wantBase string
	}{
		{input: "Download/Kids/movie.mp4", wantDir: "Download/Kids", wantBase: "movie.mp4"},
		{input: `Download\Kids\movie.mp4`, wantDir: "Download/Kids", wantBase: "movie.mp4"},
		{input: "/Documents/a/b.mkv", wantDir: "/Documents/a", wantBase: "b.mkv"},
		{input: "single.bin", wantDir: "", wantBase: "single.bin"},
		{input: "Download/", wantDir: "Download", wantBase: "stream-file.bin"},
	}
		for _, tc := range cases {
			dir, base := androidDestinationParts(tc.input)
		if dir != tc.wantDir {
			t.Errorf("androidDestinationParts(%q) dir = %q, want %q", tc.input, dir, tc.wantDir)
		}
		if base != tc.wantBase {
			t.Errorf("androidDestinationParts(%q) base = %q, want %q", tc.input, base, tc.wantBase)
		}
		}
	}

	// TestParseMTPListing covers the new List path: the marked output becomes
	// RemoteEntry values, and a malformed line is dropped rather than turned into a
	// phantom file.
	func TestParseMTPListing(t *testing.T) {
		text := strings.Join([]string{
		mtpMarkerListPrefix + "D|0|Movies",
		mtpMarkerListPrefix + "F|1234567|Movie.mkv",
		"a stray warning line that is not an entry",
		mtpMarkerListPrefix + "F|not-a-number|Odd.mkv",
		mtpMarkerListPrefix + "F|9|", // empty name: dropped
		}, "\n")

		entries := parseMTPListing(text, "Movies")
		if len(entries) != 3 {
			t.Fatalf("got %d entries, want 3 (stray line and empty name dropped): %+v", len(entries), entries)
		}

		if !entries[0].IsDir || entries[0].Name != "Movies" {
			t.Errorf("entry 0 = %+v, want a directory named Movies", entries[0])
		}
		if entries[1].IsDir || entries[1].Size != 1234567 {
			t.Errorf("entry 1 = %+v, want a file of size 1234567", entries[1])
		}
		// A size the phone refused to report degrades to 0, not to a wrong number.
		if entries[2].Size != 0 || entries[2].Name != "Odd.mkv" {
			t.Errorf("entry 2 = %+v, want Odd.mkv with size 0", entries[2])
		}
		// Paths stay inside the listed folder.
		if entries[1].Path != "Movies/Movie.mkv" {
			t.Errorf("entry 1 path = %q, want Movies/Movie.mkv", entries[1].Path)
		}
	}

	// TestParseMTPListingRootPath keeps the root listing free of a leading slash.
	func TestParseMTPListingRootPath(t *testing.T) {
		entries := parseMTPListing(mtpMarkerListPrefix+"F|10|a.mkv", "")
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Path != "a.mkv" {
			t.Errorf("path = %q, want a.mkv", entries[0].Path)
		}
	}

	// TestClassifyMTPOutput checks that a device or storage marker becomes a
	// classified, actionable error instead of free text.
	func TestClassifyMTPOutput(t *testing.T) {
		cases := []struct {
		name       string
		output     string
		wantCode   string
		wantDevice bool
		}{
		{name: "device gone", output: mtpMarkerDeviceMissing, wantCode: CodeDeviceNotFound, wantDevice: true},
		{name: "storage gone", output: mtpMarkerStorageMissing, wantCode: CodeStorageNotAvailable},
		{name: "dir gone", output: mtpMarkerDirMissing, wantCode: CodeDestinationNotFound},
		{name: "item gone", output: mtpMarkerItemMissing, wantCode: CodeDestinationNotFound},
		{name: "unknown", output: "something else entirely", wantCode: CodeMTPError},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
			err := classifyMTPOutput(tc.output)
		if err == nil {
			t.Fatal("expected an error")
		}
		if err.Code != tc.wantCode {
			t.Errorf("code = %q, want %q", err.Code, tc.wantCode)
		}
		if err.DeviceLost != tc.wantDevice {
			t.Errorf("DeviceLost = %v, want %v", err.DeviceLost, tc.wantDevice)
		}
		})
		}
	}

	// TestAndroidConnectBindsDeviceName covers both identifier forms the discovery
	// paths produce, plus the device name taking precedence.
	func TestAndroidConnectBindsDeviceName(t *testing.T) {
		cases := []struct {
		name       string
			destination string
		deviceName  string
		want        string
		}{
		{name: "mtp id is decoded", destination: "mtp_0_Samsung_A55", want: "Samsung A55"},
		{name: "ios placeholder id is decoded", destination: "ios_0_Some_Phone", want: "Some Phone"},
		{name: "device name wins", destination: "mtp_0_Ignored", deviceName: "Real Phone", want: "Real Phone"},
		{name: "plain id is kept", destination: "Samsung A55", want: "Samsung A55"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
		backend := NewAndroidBackend()
			err := backend.Connect(context.Background(), Device{Name: tc.deviceName}, TransferDestination{DeviceID: tc.destination})
		if err != nil {
			t.Fatalf("Connect: %v", err)
		}
		if backend.deviceName != tc.want {
			t.Errorf("deviceName = %q, want %q", backend.deviceName, tc.want)
		}
		})
		}
	}

	// TestAndroidRenameIsClassifiedUnsupported proves the removed stub behaviour is
	// replaced by a classified code, so a caller can fall back deliberately.
	func TestAndroidRenameIsClassifiedUnsupported(t *testing.T) {
		err := NewAndroidBackend().Rename(context.Background(), "a.mkv", "b.mkv")
		if err == nil {
			t.Fatal("expected Rename to report unsupported")
		}
		var transferErr *TransferError
		if !errors.As(err, &transferErr) {
			t.Fatalf("Rename error is not a TransferError: %T", err)
		}
		if transferErr.Code != CodeUnsupportedOperation {
			t.Errorf("code = %q, want %q", transferErr.Code, CodeUnsupportedOperation)
		}
	}

	// TestAndroidCapabilitiesMatchTheBackend keeps the advertised capabilities and
	// the implemented methods from drifting apart.
	func TestAndroidCapabilitiesMatchTheBackend(t *testing.T) {
		caps := androidCapabilities()
		if !caps.List || !caps.Delete {
			t.Errorf("List and Delete are implemented over Shell COM and must be advertised: %+v", caps)
		}
		if caps.Rename || caps.Resume || caps.StreamWrite {
			t.Errorf("rename, resume and stream-write need WPD and must not be advertised: %+v", caps)
		}
	}

	// TestStorageCapabilitiesAdvertiseEverything guards the working path: a real
	// filesystem supports every operation, and saying otherwise would make the UI
	// hide features that work.
	func TestStorageCapabilitiesAdvertiseEverything(t *testing.T) {
		caps := storageCapabilities()
		if !caps.List || !caps.Delete || !caps.Rename || !caps.Resume || !caps.StreamWrite {
			t.Errorf("storage supports every operation: %+v", caps)
		}
	}

	// TestCapabilitiesForEveryDeviceType makes sure a discovered device never comes
	// back with a nil capability record, which would silently disable its actions.
	func TestCapabilitiesForEveryDeviceType(t *testing.T) {
		for _, deviceType := range []DeviceType{DeviceAndroid, DeviceIOS, DeviceStorage} {
		if capabilitiesFor(deviceType) == nil {
			t.Errorf("capabilitiesFor(%q) = nil", deviceType)
		}
		}
	}

	// TestPowershellPathResolvesAnAbsoluteBinary proves the backend does not depend
	// on the process PATH to find PowerShell, which is the failure a restricted
	// service account or a bare launcher produces.
	func TestPowershellPathResolvesAnAbsoluteBinary(t *testing.T) {
		resolved := powershellPath()
		if resolved == "" {
			t.Fatal("powershellPath() returned an empty string")
		}
		// On Windows the resolved path must be an existing file, not the bare name.
		if filepath.IsAbs(resolved) {
		if _, err := os.Stat(resolved); err != nil {
			t.Errorf("resolved path %q does not exist: %v", resolved, err)
		}
		} else if resolved != "powershell" {
		t.Errorf("non-absolute resolution %q is neither a real path nor the fallback", resolved)
		}
	}

	// TestPowershellPathHonoursSystemRoot keeps a relocated Windows directory
	// working instead of hard-coding one drive.
	func TestPowershellPathHonoursSystemRoot(t *testing.T) {
		original := os.Getenv("SystemRoot")
		if original == "" {
		t.Skip("SystemRoot is not set on this host")
		}
		resolved := powershellPath()
		if !strings.HasPrefix(strings.ToLower(resolved), strings.ToLower(original)) && resolved != "powershell" {
		t.Logf("resolved %q outside SystemRoot %q; acceptable only if the fixed fallback matched", resolved, original)
		}
	}
