package copybridge

import (
	"path/filepath"
	"testing"

	"nexora/server/internal/transfer"
)

func TestRemoteTargetPath(t *testing.T) {
	cases := []struct {
		name       string
		deviceType transfer.DeviceType
		req        CopyRequest
		wantDir    string
		wantPath   string
	}{
		{
			name:       "storage with sub folder",
			deviceType: transfer.DeviceStorage,
			req:        CopyRequest{DeviceID: "disk_E", TargetFolder: "Movies", SubFolder: "Action"},
			wantDir:    filepath.Join(`E:\`, "Movies", "Action"),
			wantPath:   filepath.Join(`E:\`, "Movies", "Action", "Film.mkv"),
		},
		{
			name:       "storage default folder",
			deviceType: transfer.DeviceStorage,
			req:        CopyRequest{DeviceID: "disk_D"},
			wantDir:    filepath.Join(`D:\`, "Movies"),
			wantPath:   filepath.Join(`D:\`, "Movies", "Film.mkv"),
		},
		{
			name:       "ios with sub folder",
			deviceType: transfer.DeviceIOS,
			req:        CopyRequest{DeviceID: "ios_1", SubFolder: "Series"},
			wantDir:    "/Documents/Series",
			wantPath:   "/Documents/Series/Film.mkv",
		},
		{
			name:       "ios root",
			deviceType: transfer.DeviceIOS,
			req:        CopyRequest{DeviceID: "ios_1"},
			wantDir:    "/Documents",
			wantPath:   "/Documents/Film.mkv",
		},
		{
			name:       "android with sub folder",
			deviceType: transfer.DeviceAndroid,
			req:        CopyRequest{DeviceID: "mtp_1_x", SubFolder: "Kids"},
			wantDir:    "Download/Kids",
			wantPath:   "Download/Kids/Film.mkv",
		},
		{
			name:       "android custom target folder",
			deviceType: transfer.DeviceAndroid,
			req:        CopyRequest{DeviceID: "mtp_1_x", TargetFolder: "Movies", SubFolder: "Action"},
			wantDir:    "Movies/Action",
			wantPath:   "Movies/Action/Film.mkv",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, path := remoteTargetPath(tc.deviceType, tc.req, "Film.mkv")
			if dir != tc.wantDir {
				t.Fatalf("remoteDir = %q, want %q", dir, tc.wantDir)
			}
			if path != tc.wantPath {
				t.Fatalf("remotePath = %q, want %q", path, tc.wantPath)
			}
		})
	}
}

func TestSafeRelativeSub(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"  ":               "",
		".":                "",
		"..":               "",
		"a/b":              "a/b",
		"a\\b":             "a/b",
		"../escape":        "escape",
		"a/../b":           "a/b",
		"in:valid?":        "in_valid_",
		"Movies\\Season 1": "Movies/Season 1",
	}
	for input, want := range cases {
		if got := safeRelativeSub(input); got != want {
			t.Errorf("safeRelativeSub(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoteTargetPathSanitizesDisplayName(t *testing.T) {
	cases := []struct {
		name        string
		displayName string
		wantPath    string
	}{
		{name: "dot dot", displayName: "..", wantPath: filepath.Join(`E:\`, "Movies", "file")},
		{name: "dot", displayName: ".", wantPath: filepath.Join(`E:\`, "Movies", "file")},
		{name: "separators", displayName: `a\..\b.mp4`, wantPath: filepath.Join(`E:\`, "Movies", "a_.._b.mp4")},
		{name: "empty", displayName: "  ", wantPath: filepath.Join(`E:\`, "Movies", "file")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := remoteTargetPath(transfer.DeviceStorage, CopyRequest{DeviceID: "disk_E"}, tc.displayName)
			if got != tc.wantPath {
				t.Fatalf("remoteTargetPath displayName=%q = %q, want %q", tc.displayName, got, tc.wantPath)
			}
		})
	}
}
