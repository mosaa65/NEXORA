package identity

import (
	"fmt"
	"testing"
)

// TestProbeBrowseFoldersAndParts is a diagnostic that documents the correct
// behaviour for two folder shapes that are easy to get wrong:
//
//  1. A browsing folder. "مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/" means
//     "library by actors / ... / works". The work name is in the FILENAME, so
//     the folder must not be used as the title.
//
//  2. A franchise folder. "Franchises/DC/Part 3 - The Dark Knight Rises.mkv"
//     names the film in the filename while the folder names the franchise, so
//     the filename must win.
//
// It prints rather than asserts so it is useful while tuning, but the key
// expectations are asserted so a regression fails the build.
func TestProbeBrowseFoldersAndParts(t *testing.T) {
	resolver := NewResolver()

	cases := []struct {
		name          string
		path          string
		parsedTitle   string
		folderTitle   string
		category      string
		wantWorkTitle string
	}{
		{
			name:          "browsing folder must not become the work",
			path:          "مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv",
			parsedTitle:   "Inception",
			folderTitle:   "Inception",
			category:      "movies",
			wantWorkTitle: "Inception",
		},
		{
			name:          "franchise filename names the film",
			path:          "سلاسل وأجزاء/Franchises/DC/Part 3 - The Dark Knight Rises.mkv",
			parsedTitle:   "The Dark Knight Rises",
			folderTitle:   "DC",
			category:      "movies",
			wantWorkTitle: "The Dark Knight Rises",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			evidence := Evidence{
				Path:             testCase.path,
				OriginalName:     testCase.path,
				ParsedTitle:      testCase.parsedTitle,
				WorkFolderTitle:  testCase.folderTitle,
				CategoryHint:     testCase.category,
				ParserConfidence: 0.8,
				Group:            GroupContext{Size: 1},
			}
			resolution := resolver.Resolve(ResolverInput{Evidence: evidence})

			fmt.Printf("%s\n", testCase.name)
			fmt.Printf("   parsed=%q folder=%q\n", testCase.parsedTitle, testCase.folderTitle)
			fmt.Printf("   => work=%q type=%s decision=%s reason=%s\n",
				resolution.WorkTitleForDisplay(), resolution.MediaType,
				resolution.Decision, resolution.ReasonCode)

			if got := resolution.WorkTitleForDisplay(); got != testCase.wantWorkTitle {
				t.Errorf("work = %q, want %q", got, testCase.wantWorkTitle)
			}
		})
	}
}
