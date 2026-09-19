package scanner

import (
	"fmt"
	"testing"
)

// TestProFranchiseNumbering is a diagnostic for ordered franchise folders.
//
// The shape "Franchises/DC/01 - Batman Begins.2005.1080p.mkv" is extremely
// common: the leading number orders the film inside the franchise and is not an
// episode. It prints the parser's reasoning so the fix can be verified, and it
// asserts the outcome so a regression fails the build.
func TestProbeFranchiseNumbering(t *testing.T) {
	cases := []struct {
		path        string
		wantEpisode int
		wantTitle   string
	}{
		{"D:/Media/Franchises/DC/01 - Batman Begins.2005.1080p.mkv", 0, "Batman Begins"},
		{"D:/Media/Franchises/John Wick/01 - John Wick.2014.1080p.mkv", 0, "John Wick"},
		{"D:/Media/Franchises/John Wick/04 - John Wick Chapter 4.2023.1080p.mkv", 0, "John Wick Chapter 4"},
		{"D:/Media/Franchises/Marvel/01 - Iron Man.2008.1080p.mkv", 0, "Iron Man"},
		// A genuine episode file must still be an episode.
		{"D:/Media/Series/Silo/Season 01/Silo.S01E04.1080p.mkv", 4, "Silo"},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			parsed := ParseFilePath(testCase.path)
			fmt.Printf("%-58s => title=%q ep=%d isEp=%v year=%d src=%s reason=%v\n",
				filepathBase(testCase.path), parsed.Title, parsed.EpisodeNumber,
				parsed.IsEpisode, parsed.ReleaseYear, parsed.EpisodeSource, parsed.Reasons)

			if parsed.IsEpisode && testCase.wantEpisode == 0 {
				t.Errorf("leading franchise number became episode %d", parsed.EpisodeNumber)
			}
			if testCase.wantEpisode > 0 && parsed.EpisodeNumber != testCase.wantEpisode {
				t.Errorf("episode = %d, want %d", parsed.EpisodeNumber, testCase.wantEpisode)
			}
		})
	}
}

func filepathBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
