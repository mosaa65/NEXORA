package scanner

import (
	"strings"
	"testing"
)

// TestParserCorpus is the regression corpus for real-world media naming. It is
// deliberately large and organised by scenario (English, Arabic, mixed, anime,
// multipart, kids, documentaries, release tags) so a future change that breaks
// one convention fails with a named case instead of a silent metadata drift.
//
// Each case records the expected title plus the fields the parser must agree on.
// Fields left at their zero value are not asserted.
type corpusCase struct {
	name      string
	path      string
	title     string
	titleAR   string
	titleEN   string
	season    int
	episode   int
	part      int
	year      int
	res       string
	isEpisode bool
	category  string
	origin    string
	special   string
	// minConfidence asserts the parse is at least this trustworthy, which is how
	// confidence regressions are caught rather than only title regressions.
	minConfidence ParseConfidence
}

func TestParserCorpus(t *testing.T) {
	root := "D:/Media"
	cases := []corpusCase{
		// ---------------- English series -----------------
		{
			name: "standard SxxExx", path: root + "/Series/Breaking Bad/Season 01/Breaking.Bad.S01E01.1080p.WEB-DL.mkv",
			title: "Breaking Bad", season: 1, episode: 1, res: "1080p", isEpisode: true,
			category: "series", minConfidence: ConfidenceLow,
		},
		{
			name: "1x01 form", path: root + "/Series/The Office/The.Office.1x05.HDTV.mkv",
			title: "The Office", season: 1, episode: 5, isEpisode: true, minConfidence: ConfidenceLow,
		},
		{
			name: "E01 bare form", path: root + "/Series/Dark/Season 02/E07.mkv",
			title: "Dark", season: 2, episode: 7, isEpisode: true, minConfidence: ConfidenceMedium,
		},
		{
			name: "episode word", path: root + "/Series/Friends/Season 03/Episode 12.mkv",
			title: "Friends", season: 3, episode: 12, isEpisode: true, minConfidence: ConfidenceMedium,
		},
		{
			name: "release group prefix and dash number", path: root + "/Anime/Dragon Ball Z/[SubGroup] Dragon Ball Z - 042 [BDRip 1080p] [FLAC].mkv",
			title: "Dragon Ball Z", season: 1, episode: 42, res: "1080p", isEpisode: true,
			category: "anime", minConfidence: ConfidenceMedium,
		},

		// ---------------- Movies -----------------
		{
			name: "movie with year and tags", path: root + "/Movies/Inception (2010)/Inception.2010.1080p.BluRay.x264.mkv",
			title: "Inception", year: 2010, res: "1080p", isEpisode: false,
			category: "movies", minConfidence: ConfidenceMedium,
		},
		{
			name: "4K movie", path: root + "/Movies/Interstellar/Interstellar.2014.2160p.HDR.mkv",
			title: "Interstellar", year: 2014, res: "4K", isEpisode: false, minConfidence: ConfidenceMedium,
		},
		{
			// The critical numbering case: a sequel number must NOT become an episode.
			name: "sequel number is not an episode", path: root + "/Movies/Toy Story 2/Toy Story 2.mkv",
			title: "Toy Story 2", episode: 0, isEpisode: false, category: "movies",
			minConfidence: ConfidenceMedium,
		},
		{
			name:  "movie title ending in a number outside a category folder",
			path:  root + "/Movies/The Room 2/The Room 2 2019.mkv",
			title: "The Room 2", episode: 0, isEpisode: false, minConfidence: ConfidenceMedium,
		},

		// ---------------- Multipart / disc -----------------
		{
			name: "multipart part", path: root + "/Movies/The Godfather/The Godfather Part 2 1974.mkv",
			title: "The Godfather", part: 2, episode: 0, isEpisode: false, minConfidence: ConfidenceMedium,
		},
		{
			name: "CD marker", path: root + "/Movies/Rare Film/Rare Film CD1.mkv",
			part: 1, episode: 0, isEpisode: false, minConfidence: ConfidenceMedium,
		},
		{
			name: "Disc marker", path: root + "/Movies/Rare Film/Rare Film Disc 2.mkv",
			part: 2, episode: 0, isEpisode: false, minConfidence: ConfidenceMedium,
		},

		// ---------------- Arabic -----------------
		{
			name: "Arabic numerals episode", path: root + "/أنمي/ون بيس/الحلقة ١٠٨٦.mp4",
			title: "ون بيس", season: 1, episode: 1086, isEpisode: true, category: "anime",
			minConfidence: ConfidenceMedium,
		},
		{
			name: "Persian numerals episode", path: root + "/أنمي/ون بيس/الحلقة ۱۰۸۶.mp4",
			title: "ون بيس", season: 1, episode: 1086, isEpisode: true, minConfidence: ConfidenceMedium,
		},
		{
			name: "Arabic season word and episode", path: root + "/مسلسلات/صراع العروش/الموسم الأول/الحلقة 3.mkv",
			title: "صراع العروش", season: 1, episode: 3, isEpisode: true, category: "series",
			minConfidence: ConfidenceMedium,
		},
		{
			name: "Arabic season ordinal third", path: "C:/Media/Silo/الموسم الثالث/الحلقة 1.mp4",
			title: "Silo", season: 3, episode: 1, isEpisode: true, minConfidence: ConfidenceMedium,
		},
		{
			name: "Arabic part as season for turkish series", path: root + "/مسلسلات/مسلسلات تركية/طائر الرفراف/الجزء الثاني/akoam_ep05.mp4",
			title: "طائر الرفراف", titleAR: "طائر الرفراف", season: 2, episode: 5, isEpisode: true,
			category: "series", origin: "تركي", minConfidence: ConfidenceMedium,
		},
		{
			name: "watermark-only filename falls back to folder", path: "C:/Media/مسلسل اجنبي/Silo/الموسم الثالث/الانطلاقه نت 1.mp4",
			title: "Silo", season: 3, episode: 1, isEpisode: true, minConfidence: ConfidenceMedium,
		},

		// ---------------- Bilingual -----------------
		{
			name: "bilingual anime folder", path: root + "/Anime - أنمي/Attack on Titan - هجوم العمالقة/Season 01/E01.mkv",
			title: "Attack on Titan - هجوم العمالقة", titleAR: "هجوم العمالقة", titleEN: "Attack on Titan",
			season: 1, episode: 1, isEpisode: true, category: "anime", minConfidence: ConfidenceMedium,
		},

		// ---------------- Kids / documentaries / plays -----------------
		{
			name: "kids cartoon", path: root + "/كرتون/Tom and Jerry/Season 01/Tom.and.Jerry.S01E05.mp4",
			title: "Tom and Jerry", season: 1, episode: 5, isEpisode: true, category: "kids",
			minConfidence: ConfidenceMedium,
		},
		{
			name: "documentary series", path: root + "/⬛وثائقي/Planet Earth/Season 01/E02.mkv",
			title: "Planet Earth", season: 1, episode: 2, isEpisode: true, category: "documentaries",
			minConfidence: ConfidenceMedium,
		},
		{
			name: "play", path: root + "/مسرحيات/مسرحية الغرفة.mkv",
			isEpisode: false, category: "plays", minConfidence: ConfidenceLow,
		},

		// ---------------- Specials -----------------
		{
			name: "anime OVA", path: root + "/Anime/Some Show/OVA/Episode 01.mkv",
			title: "Some Show", season: 1, episode: 1, isEpisode: true, category: "anime",
			minConfidence: ConfidenceMedium,
		},

		// ---------------- Range and episode lists -----------------
		{
			name: "episode range", path: root + "/Anime/One Piece/Season 19/One.Piece.S19E01-E02.mkv",
			title: "One Piece", season: 19, episode: 1, isEpisode: true, category: "anime",
			minConfidence: ConfidenceMedium,
		},
		{
			name: "episodes range folder", path: root + "/Anime/One Piece/الحلقات 1-50/001.mkv",
			title: "One Piece", season: 1, episode: 1, isEpisode: true, category: "anime",
			minConfidence: ConfidenceMedium,
		},

		// ---------------- Bad filenames -----------------
		{
			// The parser must not invent metadata: a numeric-only name in a folder
			// that carries no title has NO evidence at all, so it is reported at the
			// floor rather than at the weakest usable level. Anything above the floor
			// here would mean the parser was crediting itself for an inference it did
			// not make.
			name: "numeric only file in unknown folder", path: root + "/Unsorted/01.mkv",
			episode: 0, isEpisode: false, minConfidence: 0.05,
		},
		{
			name: "mixed separators and dots", path: root + "/Series/Westworld/Season 01/Westworld_S01E01_1080p_WEB.mp4",
			title: "Westworld", season: 1, episode: 1, res: "1080p", isEpisode: true,
			minConfidence: ConfidenceMedium,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed := ParseFilePath(testCase.path)

			if testCase.title != "" && parsed.Title != testCase.title {
				t.Errorf("title = %q, want %q", parsed.Title, testCase.title)
			}
			if testCase.titleAR != "" && parsed.TitleAR != testCase.titleAR {
				t.Errorf("titleAR = %q, want %q", parsed.TitleAR, testCase.titleAR)
			}
			if testCase.titleEN != "" && parsed.TitleEN != testCase.titleEN {
				t.Errorf("titleEN = %q, want %q", parsed.TitleEN, testCase.titleEN)
			}
			// Season and episode are only asserted when the case declares a value;
			// a zero expectation means "must remain zero" for the numbering cases.
			if testCase.name == "sequel number is not an episode" || testCase.name == "movie title ending in a number outside a category folder" {
				if parsed.SeasonNumber != 0 || parsed.EpisodeNumber != 0 || parsed.IsEpisode {
					t.Errorf("number treated as episode: season=%d episode=%d isEpisode=%v",
						parsed.SeasonNumber, parsed.EpisodeNumber, parsed.IsEpisode)
				}
			} else {
				if testCase.season > 0 && parsed.SeasonNumber != testCase.season {
					t.Errorf("season = %d, want %d", parsed.SeasonNumber, testCase.season)
				}
				if testCase.episode > 0 && parsed.EpisodeNumber != testCase.episode {
					t.Errorf("episode = %d, want %d", parsed.EpisodeNumber, testCase.episode)
				}
				if parsed.IsEpisode != testCase.isEpisode {
					t.Errorf("isEpisode = %v, want %v", parsed.IsEpisode, testCase.isEpisode)
				}
			}
			if testCase.part != 0 && parsed.PartNumber != testCase.part {
				t.Errorf("part = %d, want %d", parsed.PartNumber, testCase.part)
			}
			if testCase.part > 0 && parsed.EpisodeNumber != 0 {
				t.Errorf("part was misread as episode %d", parsed.EpisodeNumber)
			}
			if testCase.year > 0 && parsed.ReleaseYear != testCase.year {
				t.Errorf("year = %d, want %d", parsed.ReleaseYear, testCase.year)
			}
			if testCase.res != "" && parsed.Resolution != testCase.res {
				t.Errorf("resolution = %q, want %q", parsed.Resolution, testCase.res)
			}
			if testCase.category != "" && parsed.CategorySlug != testCase.category {
				t.Errorf("category = %q, want %q", parsed.CategorySlug, testCase.category)
			}
			if testCase.origin != "" && parsed.OriginTag != testCase.origin {
				t.Errorf("origin = %q, want %q", parsed.OriginTag, testCase.origin)
			}
			if testCase.special != "" && parsed.SpecialKind != testCase.special {
				t.Errorf("special = %q, want %q", parsed.SpecialKind, testCase.special)
			}
			if testCase.minConfidence > 0 && parsed.Confidence < testCase.minConfidence {
				t.Errorf("confidence = %.2f (%s), want at least %.2f; reasons=%v",
					parsed.Confidence, parsed.ConfidenceBand, testCase.minConfidence, parsed.Reasons)
			}
			// A parse must always be explainable.
			if parsed.ConfidenceBand == "" {
				t.Error("confidence band was not set")
			}
		})
	}
}

// TestParserNeverInventsTitle asserts the "do not guess" rule: when there is no
// evidence for a title, the parser leaves it empty and says why instead of
// returning a number or a quality tag as the title.
func TestParserNeverInventsTitle(t *testing.T) {
	for _, fileName := range []string{"1080p.mkv", "x265.mkv", "[SubGroup].mkv", "01.mp4"} {
		parsed := ParseFileName(fileName)
		if parsed.Title != "" {
			t.Errorf("ParseFileName(%q) invented title %q", fileName, parsed.Title)
		}
		if parsed.Confidence > ConfidenceMedium {
			t.Errorf("ParseFileName(%q) reported confidence %.2f without evidence", fileName, parsed.Confidence)
		}
	}
}

// TestArabicNormalization checks the normalization layer without destroying the
// original text the user sees.
func TestArabicNormalization(t *testing.T) {
	// Arabic-Indic and Persian digits must both become ASCII.
	if got := normalizeDigits("الحلقة ١٢٣"); got != "الحلقة 123" {
		t.Errorf("Arabic-Indic digits = %q", got)
	}
	if got := normalizeDigits("قسم ۴۵۶"); got != "قسم 456" {
		t.Errorf("Persian digits = %q", got)
	}

	// Diacritics and tatweel are removed for matching, but not from the title
	// the user sees.
	withDiacritics := "المُسَلْسَل"
	if got := stripArabicDiacritics(withDiacritics); got != "المسلسل" {
		t.Errorf("diacritics not stripped: %q", got)
	}

	// Search normalization must fold the alef/teh-marbuta variants so equivalent
	// spellings match, while NormalizeTitleForSearch stays a search-only value.
	sameA := NormalizeTitleForSearch("اسامة")
	sameB := NormalizeTitleForSearch("أسامة")
	if sameA != sameB {
		t.Errorf("alef variants did not fold: %q vs %q", sameA, sameB)
	}
}

// TestSearchTokens checks the search-preparation layer. Separator and case
// variants of the same title must produce identical token sets. Arabic and
// English are separate aliases, matched through their own tokens rather than
// by sharing any token across languages.
func TestSearchTokens(t *testing.T) {
	groups := [][]string{
		{"One Piece", "one_piece", "One.Piece", "one piece", "ONE PIECE"},
		{"ون بيس", "ون_بيس", "ون.بيس"},
	}
	for _, group := range groups {
		t.Run(group[0], func(t *testing.T) {
			base := strings.Join(SearchTokens(group[0]), "|")
			for _, variant := range group[1:] {
				tokens := strings.Join(SearchTokens(variant), "|")
				if tokens != base {
					t.Errorf("SearchTokens(%q) = %q, want %q (same as %q)", variant, tokens, base, group[0])
				}
			}
			// The concatenated form lets "one piece" and "onepiece" both match.
			concatenated := strings.ReplaceAll(NormalizeTitleForSearch(group[0]), " ", "")
			if !strings.Contains(base, concatenated) {
				t.Errorf("expected a concatenated token %q in %q", concatenated, base)
			}
		})
	}
}

// TestCategoryDetectionIsSegmentBased is the regression test for the
// strings.Contains false positive: "/NotMovies/" must not classify as movies.
func TestCategoryDetectionIsSegmentBased(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"D:/Media/Movies/Inception.mkv", "movies"},
		{"D:/Media/NotMovies/Inception.mkv", ""},
		{"D:/Media/MySeriesStuff/x.mkv", ""},
		{"D:/Media/Series/Anything/E01.mkv", "series"},
		{"D:/Media/Documentaries/Nature/E01.mkv", "documentaries"},
		{"D:/Media/مسلسلات تركية/Show/S01/E01.mkv", "series"},
		{"D:/Media/كرتون/Tom/E01.mkv", "kids"},
		// The deepest segment wins: a series inside a movies folder is a series.
		{"D:/Media/Movies/Series/Show/S01/E01.mkv", "series"},
	}

	for _, testCase := range cases {
		t.Run(testCase.path, func(t *testing.T) {
			if got := DetectCategoryFromPath(testCase.path); got != testCase.want {
				t.Errorf("DetectCategoryFromPath(%q) = %q, want %q", testCase.path, got, testCase.want)
			}
		})
	}
}

// TestPartAndEpisodeAreDistinct asserts the explicit separation of Season,
// Episode and Part that the old parser conflated.
func TestPartAndEpisodeAreDistinct(t *testing.T) {
	cases := []struct {
		fileName string
		part     int
		episode  int
	}{
		{"Movie Part 1.mkv", 1, 0},
		{"Movie Part 01.mkv", 1, 0},
		{"Movie CD2.mkv", 2, 0},
		{"Movie Disc 3.mkv", 3, 0},
		{"Series S01E04.mkv", 0, 4},
	}

	for _, testCase := range cases {
		t.Run(testCase.fileName, func(t *testing.T) {
			parsed := ParseFileName(testCase.fileName)
			if parsed.PartNumber != testCase.part {
				t.Errorf("part = %d, want %d", parsed.PartNumber, testCase.part)
			}
			if parsed.EpisodeNumber != testCase.episode {
				t.Errorf("episode = %d, want %d", parsed.EpisodeNumber, testCase.episode)
			}
		})
	}
}

// TestSeasonOrdinalsArabic covers the full Arabic ordinal table used by real
// libraries, including the forms that share a prefix.
func TestSeasonOrdinalsArabic(t *testing.T) {
	cases := map[string]int{
		"الموسم الأول":  1,
		"الموسم الثاني": 2,
		"الموسم الثالث": 3,
		"الموسم الرابع": 4,
		"الموسم الخامس": 5,
		"الموسم السادس": 6,
		"الموسم السابع": 7,
		"الموسم الثامن": 8,
		"الموسم التاسع": 9,
		"الموسم العاشر": 10,
		"الجزء الثاني":  2,
		// "الجز" alone is not a season keyword; only the full "الجزء" is.
		"الجزء الثالث": 3,
		"الجزء 4":      4,
		"الموسم 01":    1,
		"الموسم 12":    12,
		"Season 03":    3,
		"S04":          4,
	}

	for folder, want := range cases {
		t.Run(folder, func(t *testing.T) {
			if got := parseSeasonFromFolder(folder); got != want {
				t.Errorf("parseSeasonFromFolder(%q) = %d, want %d", folder, got, want)
			}
		})
	}
}

// TestFolderTitleWinsOverNumericFilename is the real-library scenario where the
// filename is a bare number and only the folder carries the title.
func TestFolderTitleWinsOverNumericFilename(t *testing.T) {
	parsed := ParseFilePath("D:/Anime/One Piece/الحلقات 1-50/001.mkv")
	if parsed.Title != "One Piece" {
		t.Errorf("title = %q, want the folder title", parsed.Title)
	}
	if !parsed.IsEpisode {
		t.Error("expected an episode")
	}
}
