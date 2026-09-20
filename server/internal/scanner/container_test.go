package scanner

import (
	"testing"
)

// TestContainerFolderIsNeverAWorkTitle is the regression test for the defect
// that created works literally named "أعمال" and "Media".
//
// The real layout that produced them:
//
//	…/مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Titanic.1997.mkv
//
// "أعمال" means "works" and is a browse grouping. The filename carries the film.
// The parser must therefore ignore the container AND ignore the actor folder
// above it, because an actor is not a work either.
func TestContainerFolderIsNeverAWorkTitle(t *testing.T) {
	root := "D:/Media"
	cases := []struct {
		name      string
		path      string
		wantTitle string
	}{
		{
			name:      "actor browse folder named works",
			path:      root + "/مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Titanic.1997.mkv",
			wantTitle: "Titanic",
		},
		{
			name:      "another actor, same grouping",
			path:      root + "/مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv",
			wantTitle: "Inception",
		},
		{
			// A category folder followed by the film name. The category is a
			// container, so the title comes from the file.
			name:      "english category then film",
			path:      root + "/Library/Movies/Iron Man.2008.mkv",
			wantTitle: "Iron Man",
		},
		{
			name:      "arabic films grouping",
			path:      root + "/أفلام/Interstellar.2014.mkv",
			wantTitle: "Interstellar",
		},
		{
			// A category folder combined with an origin folder is still a
			// container, so the film name must come from the file.
			name:      "category and origin combined",
			path:      root + "/مسلسلات تركية/طائر الرفراف/الجزء الثاني/akoam_ep05.mp4",
			wantTitle: "طائر الرفراف",
		},
		{
			// An actor folder with a container beneath it: the container is skipped
			// and the actor folder above it is not a work either, so the filename
			// supplies the title. This is the layout that produced rows named
			// "أعمال".
			name:      "english actor folder with a container beneath",
			path:      root + "/Library/Actors/Robert Downey Jr/أعمال/Iron Man.2008.mkv",
			wantTitle: "Iron Man",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed := ParseFilePath(testCase.path)
			if parsed.Title != testCase.wantTitle {
				t.Errorf("title = %q, want %q", parsed.Title, testCase.wantTitle)
			}
			if IsContainerFolderForTitle(parsed.Title) {
				t.Errorf("title %q is itself a container folder name", parsed.Title)
			}
		})
	}
}

// TestContainerFolderClassification documents exactly which folder names are
// treated as containers rather than works.
//
// The scanner and the identity package must agree on this, because the catalogue
// repair and the ingest path both consult it. A disagreement would mean a
// repaired row and a freshly ingested row differ for the same folder.
func TestContainerFolderClassification(t *testing.T) {
	containers := []string{
		"أعمال", "اعمال", "افلام", "أفلام", "مسلسلات", "مسلسل",
		"مكتبة", "مكتبه", "متنوع", "أخرى",
		"مكتبة حسب الممثلين", "الممثلين",
		"Movies", "movies", "Films", "Series", "TV", "Library", "Media",
		"Franchises", "Collection", "Boxset", "Extras", "Samples", "Unsorted",
		"مسلسلات تركية",
	}

	for _, folder := range containers {
		if !IsContainerFolderForTitle(folder) {
			t.Errorf("IsContainerFolderForTitle(%q) = false, want true", folder)
		}
	}

	// A real work name must never be classified as a container, or the parser
	// would refuse every title.
	works := []string{
		"Titanic", "Inception", "Iron Man", "Breaking Bad", "One Piece",
		"صراع العروش", "طائر الرفراف", "Silo", "The Office",
	}

	for _, folder := range works {
		if IsContainerFolderForTitle(folder) {
			t.Errorf("IsContainerFolderForTitle(%q) = true, want false", folder)
		}
	}
}

// TestActorFolderIsNotUsedAsTitle guards the second half of the same rule.
//
// An actor's folder is a real name, so a naive "use the nearest non-container
// ancestor" would title every film in it after the actor. The filename must win.
func TestActorFolderIsNotUsedAsTitle(t *testing.T) {
	parsed := ParseFilePath("D:/Media/مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Titanic.1997.mkv")

	if parsed.Title == "Leonardo DiCaprio" {
		t.Fatal("the actor folder was used as the work title")
	}
	if parsed.Title != "Titanic" {
		t.Errorf("title = %q, want the film name from the filename", parsed.Title)
	}
	if parsed.ReleaseYear != 1997 {
		t.Errorf("year = %d, want 1997", parsed.ReleaseYear)
	}
}

// TestWatermarkOnlyFilenameProducesNoTitle verifies the parser refuses to invent
// a title when the filename is only site branding.
func TestWatermarkOnlyFilenameProducesNoTitle(t *testing.T) {
	parsed := ParseFilePath("D:/Media/الانطلاقه نت 5.mp4")

	if parsed.Title != "" {
		t.Errorf("title = %q, want empty: a watermark is not a title", parsed.Title)
	}
	if parsed.Confidence > 0.2 {
		t.Errorf("confidence = %.2f, want near the floor for no evidence", parsed.Confidence)
	}
	if parsed.ConfidenceBand == "HIGH" || parsed.ConfidenceBand == "MEDIUM" {
		t.Errorf("band = %q, want LOW for a watermark-only name", parsed.ConfidenceBand)
	}
}

// TestContainerDetectionDoesNotBreakNormalFolderLayouts is the counter-test: a
// library organised as Category/Work/Season must still resolve from the folder,
// because that layout puts the work name in a folder and the filename may be
// nothing but a number.
func TestContainerDetectionDoesNotBreakNormalFolderLayouts(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		wantTitle string
	}{
		{
			name:      "series work folder",
			path:      "D:/Media/Series/Breaking Bad/Season 01/Breaking.Bad.S01E01.mkv",
			wantTitle: "Breaking Bad",
		},
		{
			name:      "numeric filename needs the folder",
			path:      "D:/Media/Anime/One Piece/الحلقات 1-50/001.mkv",
			wantTitle: "One Piece",
		},
		{
			name:      "bilingual anime folder",
			path:      "D:/Media/Anime - أنمي/Attack on Titan - هجوم العمالقة/Season 01/E01.mkv",
			wantTitle: "Attack on Titan - هجوم العمالقة",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed := ParseFilePath(testCase.path)
			if parsed.Title != testCase.wantTitle {
				t.Errorf("title = %q, want %q", parsed.Title, testCase.wantTitle)
			}
		})
	}
}
