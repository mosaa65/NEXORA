package transfer

import (
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
