package identity

import (
	"testing"
)

// -----------------------------------------------------------------------------
// Normalization and similarity: the foundation of "one work, many spellings"
// -----------------------------------------------------------------------------

// TestNormalizeUnifiesSpellings is requirement 64: the same work written in
// different ways must normalize to one key, so the library does not split into
// five works for one show.
func TestNormalizeUnifiesSpellings(t *testing.T) {
	groups := [][]string{
		{"Breaking Bad", "Breaking.Bad", "Breaking_Bad", "breaking-bad", "Breaking  Bad", "BREAKING BAD"},
		{"One Piece", "One.Piece", "One_Piece", "one piece"},
		{"Fate Apocrypha", "Fate_Apocrypha", "Fate-Apocrypha"},
		// Arabic letter variants that are written differently but read identically.
		{"أسامة", "اسامة"},
		{"موسم", "موسم"},
		// Arabic-Indic digits fold to ASCII.
		{"الحلقة ١٢٣", "الحلقة 123"},
	}

	for _, group := range groups {
		t.Run(group[0], func(t *testing.T) {
			want := Normalize(group[0])
			if want == "" {
				t.Fatalf("Normalize(%q) returned empty", group[0])
			}
			for _, variant := range group[1:] {
				if got := Normalize(variant); got != want {
					t.Errorf("Normalize(%q) = %q, want %q (same as %q)", variant, got, want, group[0])
				}
			}
		})
	}
}

// TestNormalizePreservesIdentity distinguishes genuinely different works, which
// is the other half of the requirement: unification must not merge titles that
// only look similar.
func TestNormalizePreservesIdentity(t *testing.T) {
	different := [][2]string{
		{"Silo", "Severance"},
		{"The Office", "The Office US"},
		{"Breaking Bad", "Better Call Saul"},
		{"One Piece", "One Punch Man"},
	}

	for _, pair := range different {
		if Normalize(pair[0]) == Normalize(pair[1]) {
			t.Errorf("Normalize collapsed %q and %q into one key", pair[0], pair[1])
		}
	}
}

// TestSimilarityBehaviour documents the scoring scale the resolver relies on.
func TestSimilarityBehaviour(t *testing.T) {
	cases := []struct {
		name     string
		a, b     string
		minScore float64
		maxScore float64
	}{
		{"identical", "Silo", "Silo", 1.0, 1.0},
		{"separator variant", "Breaking Bad", "Breaking.Bad", 0.9, 1.0},
		{"compact equal", "One Piece", "onepiece", 0.9, 1.0},
		{"with qualifier", "Silo", "Silo Arabic", 0.7, 1.0},
		{"one typo", "Interstellar", "Intersteller", 0.5, 0.95},
		{"unrelated", "Silo", "Planet Earth", 0, 0.35},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := Similarity(testCase.a, testCase.b)
			if got < testCase.minScore || got > testCase.maxScore {
				t.Errorf("Similarity(%q, %q) = %.3f, want between %.2f and %.2f",
					testCase.a, testCase.b, got, testCase.minScore, testCase.maxScore)
			}
		})
	}
}

// TestSimilarityIsWordAligned is the guard against substring false positives:
// "silo" must not match "silosomethingelse" as if it were the same work.
func TestSimilarityIsWordAligned(t *testing.T) {
	// "silo" is a prefix of "silosomething", but not a word inside it.
	containmentOnly := Similarity("silo", "silosomething")
	realContainment := Similarity("silo", "silo arabic")
	if containmentOnly >= realContainment {
		t.Errorf("prefix-only match (%.3f) must score below real word containment (%.3f)",
			containmentOnly, realContainment)
	}
}

// -----------------------------------------------------------------------------
// Title quality: the guard against inventing works from noise
// -----------------------------------------------------------------------------

// TestAssessTitleRejectsRealLibraryNoise uses names taken from the actual
// database, where the old per-file ingest created these as works.
func TestAssessTitleRejectsRealLibraryNoise(t *testing.T) {
	cases := []struct {
		filename string
		title    string
		usable   bool
	}{
		// A bare episode marker is never a work name.
		{"01.mkv", "01", false},
		{"03.mkv", "03", false},
		// A trailing separator is a parse artifact seen as work "Fate Stay Night -".
		{"Fate Stay Night -.mkv", "Fate Stay Night -", false},
		// A structural keyword consumed as the title.
		{"الكنز ج1 الحلقه.mkv", "الحلقة", false},
		// A site watermark.
		{"الكنزنت 01.avi", "الكنزنت", false},
		{"akoam_ep05.mp4", "akoam", false},
		// Real titles must survive.
		{"Inception.2010.1080p.mkv", "Inception", true},
		{"Breaking.Bad.S01E01.mkv", "Breaking Bad", true},
		{"الكنز.mkv", "الكنز", true},
	}

	for _, testCase := range cases {
		t.Run(testCase.filename, func(t *testing.T) {
			quality := AssessTitle(testCase.title, testCase.filename)
			if quality.Usable != testCase.usable {
				t.Errorf("AssessTitle(%q, %q).Usable = %v, want %v (reason: %s)",
					testCase.title, testCase.filename, quality.Usable, testCase.usable, quality.Reason)
			}
			if !quality.Usable && quality.Reason == "" {
				t.Error("a rejection must always explain itself")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Media type: movie vs series from structure, never from naming
// -----------------------------------------------------------------------------

func TestDetectMediaTypePrefersStructureOverNaming(t *testing.T) {
	cases := []struct {
		name     string
		evidence Evidence
		want     MediaType
	}{
		{
			name: "explicit episode marker is a series",
			evidence: Evidence{
				ParserEpisodeStrong: true,
				ParsedSeason:        1,
				ParsedEpisode:       1,
			},
			want: MediaTypeSeries,
		},
		{
			name: "season folder plus neighbourhood is a series",
			evidence: Evidence{
				SeasonFolderNumber: 2,
				Group: GroupContext{
					SeasonFolder:     true,
					EpisodicSiblings: 5,
					EpisodeNumbers:   []int{1, 2, 3, 4, 5},
				},
			},
			want: MediaTypeSeries,
		},
		{
			name: "a sequel number is NOT an episode and the file is a movie",
			evidence: Evidence{
				ParsedTitle:   "Toy Story 2",
				ParsedEpisode: 0,
				ParsedYear:    1999,
				CategoryHint:  "movies",
				Group:         GroupContext{Size: 1, YearSiblings: 0},
			},
			want: MediaTypeMovie,
		},
		{
			name: "multi-part film is a movie",
			evidence: Evidence{
				ParsedTitle:  "The Godfather",
				ParsedPart:   2,
				CategoryHint: "movies",
				Group:        GroupContext{Size: 1},
			},
			want: MediaTypeMovie,
		},
		{
			name: "anime series is a series, not a separate type",
			evidence: Evidence{
				CategoryHint:        "anime",
				ParserEpisodeStrong: true,
				ParsedEpisode:       12,
			},
			want: MediaTypeSeries,
		},
		{
			name: "genuinely ambiguous stays unknown instead of guessing",
			evidence: Evidence{
				ParsedTitle:  "Silo",
				CategoryHint: "",
				Group:        GroupContext{Size: 1},
			},
			want: MediaTypeUnknown,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := DetectMediaType(testCase.evidence)
			if got.MediaType != testCase.want {
				t.Errorf("DetectMediaType = %q (score %.0f), want %q; reasons=%v",
					got.MediaType, got.Score, testCase.want, got.Reasons)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Season and episode resolution
// -----------------------------------------------------------------------------

// TestResolveSeasonPrefersFolder is requirement 67: a season folder is
// authoritative, and a season never becomes part of the work name.
func TestResolveSeasonPrefersFolder(t *testing.T) {
	cases := []struct {
		name     string
		evidence Evidence
		want     int
	}{
		{"season folder wins", Evidence{SeasonFolderNumber: 3, ParsedSeason: 1}, 3},
		{"filename supplies the season", Evidence{ParserEpisodeStrong: true, ParsedSeason: 4, ParsedEpisode: 2}, 4},
		{"episode folder defaults to 1", Evidence{ParserEpisodeStrong: true, Group: GroupContext{SeasonFolder: true}}, 1},
		{"no season structure", Evidence{}, 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ResolveSeason(testCase.evidence)
			if got.Number != testCase.want {
				t.Errorf("ResolveSeason = %d, want %d (reason: %s)", got.Number, testCase.want, got.Reason)
			}
		})
	}
}

// TestResolveEpisodeRequiresContext is requirement 68: a bare number must not
// become an episode without structural support.
func TestResolveEpisodeRequiresContext(t *testing.T) {
	// A bare number alone in an unstructured folder: refused.
	lone := ResolveEpisode(Evidence{ParsedEpisode: 4, ParserEpisodeStrong: false})
	if lone.Valid {
		t.Errorf("a bare number with no context became episode %d", lone.Number)
	}

	// The same number inside an episode folder: accepted.
	inFolder := ResolveEpisode(Evidence{
		ParsedEpisode: 4,
		CategoryHint:  "series",
	})
	if !inFolder.Valid {
		t.Errorf("a bare number with series context was rejected: %s", inFolder.Reason)
	}

	// The same number inside a contiguous sibling run: accepted even without a
	// category, because the neighbourhood is the evidence.
	inRun := ResolveEpisode(Evidence{
		ParsedEpisode: 4,
		Group: GroupContext{
			Size:             4,
			EpisodicSiblings: 4,
			EpisodeNumbers:   []int{1, 2, 3, 4},
			SeasonFolder:     true,
		},
	})
	if !inRun.Valid {
		t.Errorf("a bare number inside an episode run was rejected: %s", inRun.Reason)
	}

	// An explicit marker is always trusted.
	explicit := ResolveEpisode(Evidence{ParsedEpisode: 7, ParserEpisodeStrong: true})
	if !explicit.Valid || explicit.Number != 7 {
		t.Errorf("an explicit episode pattern was not trusted: %+v", explicit)
	}
}

// -----------------------------------------------------------------------------
// Group context: batch resolution (requirement 70)
// -----------------------------------------------------------------------------

// TestGroupContextRecognisesEpisodeRun proves that four bare-numbered files are
// recognised as one season even though no single filename says anything.
func TestGroupContextRecognisesEpisodeRun(t *testing.T) {
	siblings := []GroupInput{
		{Path: "D:/Media/Series/Show/01.mkv", ParsedTitle: "Show", Episode: 1, IsEpisode: true, EpisodeStrong: false},
		{Path: "D:/Media/Series/Show/02.mkv", ParsedTitle: "Show", Episode: 2, IsEpisode: true, EpisodeStrong: false},
		{Path: "D:/Media/Series/Show/03.mkv", ParsedTitle: "Show", Episode: 3, IsEpisode: true, EpisodeStrong: false},
		{Path: "D:/Media/Series/Show/04.mkv", ParsedTitle: "Show", Episode: 4, IsEpisode: true, EpisodeStrong: false},
	}

	context := BuildGroupContext("D:/Media/Series/Show", siblings, true)
	episodeRange := context.ContiguousEpisodeRange()
	if !episodeRange.Contiguous {
		t.Fatalf("a contiguous 1-4 run was not recognised: %+v", context)
	}
	if episodeRange.Low != 1 || episodeRange.High != 4 {
		t.Errorf("range = %d-%d, want 1-4", episodeRange.Low, episodeRange.High)
	}
	if context.ConsensusVotes < 4 {
		t.Errorf("consensus votes = %d, want 4", context.ConsensusVotes)
	}
}

// TestGroupContextRejectsUnreliableConsensus ensures one file cannot create a
// "consensus", which would otherwise let a single bad filename dominate.
func TestGroupContextRejectsUnreliableConsensus(t *testing.T) {
	siblings := []GroupInput{
		{Path: "D:/Media/Movies/a.mkv", ParsedTitle: "Inception"},
		{Path: "D:/Media/Movies/b.mkv", ParsedTitle: "Interstellar"},
		{Path: "D:/Media/Movies/c.mkv", ParsedTitle: "Oppenheimer"},
	}
	context := BuildGroupContext("D:/Media/Movies", siblings, false)
	if context.ConsensusTitle != "" {
		t.Errorf("consensus was declared from disagreeing siblings: %q", context.ConsensusTitle)
	}

	// A single file is never a consensus either.
	alone := BuildGroupContext("D:/Media/Movies", []GroupInput{
		{Path: "D:/Media/Movies/a.mkv", ParsedTitle: "Inception"},
	}, false)
	if alone.ConsensusTitle != "" {
		t.Errorf("consensus was declared from a single file: %q", alone.ConsensusTitle)
	}
}

// TestGroupContextIgnoresWatermarkTitles ensures a folder full of watermarked
// names cannot establish a consensus.
func TestGroupContextIgnoresWatermarkTitles(t *testing.T) {
	siblings := []GroupInput{
		{Path: "D:/Media/x/01.avi", ParsedTitle: "الكنزنت"},
		{Path: "D:/Media/x/02.avi", ParsedTitle: "الكنزنت"},
		{Path: "D:/Media/x/03.avi", ParsedTitle: "الكنزنت"},
	}
	context := BuildGroupContext("D:/Media/x", siblings, false)
	if context.ConsensusTitle != "" {
		t.Errorf("a watermark became the consensus title: %q", context.ConsensusTitle)
	}
}
