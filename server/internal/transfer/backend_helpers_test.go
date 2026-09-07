package transfer

import "testing"

func TestIOSDocumentsPath(t *testing.T) {
	cases := map[string]string{
		"":                           "/Documents",
		"/Documents":                 "/Documents",
		"Documents":                  "/Documents",
		"/Documents/Series/Season 1": "/Documents/Series/Season 1",
		"Series/Season 1":            "/Documents/Series/Season 1",
	}
	for input, want := range cases {
		if got := iosDocumentsPath(input); got != want {
			t.Errorf("iosDocumentsPath(%q) = %q, want %q", input, got, want)
		}
	}
}
