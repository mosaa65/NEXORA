package scanner

import (
	"fmt"
	"testing"
)

// TestProbeNumericTitles is a diagnostic for film titles that BEGIN with a
// number. It prints the parse and asserts the title survives, because stripping
// the leading number turns "3 Idiots" into "Idiots", which then becomes a wrong
// learned alias and mis-resolves every future file.
func TestProbeNumericTitles(t *testing.T) {
	cases := []struct {
		path      string
		wantTitle string
	}{
		{"D:/Media/Movies/3 Idiots.2009.1080p.mkv", "3 Idiots"},
		{"D:/Media/Movies/21 Jump Street.2012.1080p.mkv", "21 Jump Street"},
		{"D:/Media/Movies/12 Angry Men.1957.1080p.mkv", "12 Angry Men"},
		// Known limitation: a film whose title IS a 4-digit year in the 19xx/20xx range
		{"D:/Media/Movies/365 Days.2020.1080p.mkv", "365 Days"},
		// cannot be distinguished from a release year without a decisive signal.
		// A genuine franchise ordering prefix MUST still be stripped.
		{"D:/Media/Franchises/DC/01 - Batman Begins.2005.1080p.mkv", "Batman Begins"},
		{"D:/Media/Franchises/John Wick/04 - John Wick Chapter 4.2023.1080p.mkv", "John Wick Chapter 4"},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			parsed := ParseFilePath(testCase.path)
			fmt.Printf("%-46s => title=%q\n", filepathBase(testCase.path), parsed.Title)
			if parsed.Title != testCase.wantTitle {
				t.Errorf("title = %q, want %q", parsed.Title, testCase.wantTitle)
			}
		})
	}
}
