package identity

import (
	"testing"
)

// -----------------------------------------------------------------------------
// Requirement 63: never create a new work too quickly
// -----------------------------------------------------------------------------

// TestResolverRefusesToInventWorkFromBareNumber is the central guarantee. A file
// called "03.mkv" must never become a work named "03".
func TestResolverRefusesToInventWorkFromBareNumber(t *testing.T) {
	resolver := NewResolver()

	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:         "D:/Media/Unsorted/03.mkv",
			OriginalName: "03.mkv",
			ParsedTitle:  "03",
			Segments:     []string{"Media", "Unsorted"},
			Group:        GroupContext{Size: 1},
			// No folder title, no episode structure, no category.
		},
	})

	if resolution.Decision == DecisionCreate {
		t.Fatalf("a bare number created a work named %q", resolution.NewWorkTitle)
	}
	if resolution.State != StateUnresolved && resolution.State != StateNeedsReview {
		t.Errorf("state = %q, want unresolved or needs_review", resolution.State)
	}
	if resolution.ReasonCode == "" || resolution.Reason == "" {
		t.Error("an unresolved file must explain why")
	}
}

// TestResolverCreatesWorkOnlyWithStrongEvidence verifies creation is still
// possible, but only when folder and structure agree.
func TestResolverCreatesWorkOnlyWithStrongEvidence(t *testing.T) {
	resolver := NewResolver()

	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:                "D:/Media/Series/Breaking Bad/Season 01/Breaking.Bad.S01E01.1080p.mkv",
			OriginalName:        "Breaking.Bad.S01E01.1080p.mkv",
			ParsedTitle:         "Breaking Bad",
			Segments:            []string{"Media", "Series", "Breaking Bad", "Season 01"},
			WorkFolderTitle:     "Breaking Bad",
			SeasonFolderNumber:  1,
			ParsedSeason:        1,
			ParsedEpisode:       1,
			ParserIsEpisode:     true,
			ParserEpisodeStrong: true,
			ParserConfidence:    0.9,
			CategoryHint:        "series",
			Group: GroupContext{
				Size:             10,
				EpisodicSiblings: 10,
				EpisodeNumbers:   []int{1, 2, 3},
				SeasonFolder:     true,
			},
		},
	})

	if resolution.Decision != DecisionCreate {
		t.Fatalf("a well-evidenced new series was not created: %+v", resolution)
	}
	if resolution.MediaType != MediaTypeSeries {
		t.Errorf("media type = %q, want series", resolution.MediaType)
	}
	if resolution.Season != 1 || resolution.Episode != 1 {
		t.Errorf("season/episode = %d/%d, want 1/1", resolution.Season, resolution.Episode)
	}
	if resolution.State != StateEnrichmentPending {
		t.Errorf("a new work should await enrichment, got %q", resolution.State)
	}
}

// TestResolverRefusesUnknownMediaType is requirement 69: when movie and series
// evidence conflict, the file is queued rather than guessed at.
func TestResolverRefusesUnknownMediaType(t *testing.T) {
	resolver := NewResolver()

	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:            "D:/Media/Something/thing.mkv",
			OriginalName:    "thing.mkv",
			ParsedTitle:     "Thing",
			WorkFolderTitle: "Something",
			Group:           GroupContext{Size: 1},
		},
	})

	if resolution.Decision == DecisionCreate {
		t.Fatalf("an ambiguous file created a %q work", resolution.MediaType)
	}
	if resolution.ReasonCode != "unknown_media_type" {
		t.Errorf("reason = %q, want unknown_media_type", resolution.ReasonCode)
	}
}

// -----------------------------------------------------------------------------
// Requirement 66: existing entity first
// -----------------------------------------------------------------------------

// TestResolverAttachesToExistingWorkInsteadOfCreating proves that a differently
// spelled filename joins the existing work rather than making a duplicate.
func TestResolverAttachesToExistingWorkInsteadOfCreating(t *testing.T) {
	resolver := NewResolver()

	existing := Work{
		ID:              42,
		TitleEN:         "Breaking Bad",
		TitleAR:         "بريكنغ باد",
		TitleNormalized: "breaking bad",
		MediaType:       "series",
		CategorySlug:    "series",
		ReleaseYear:     2008,
		Aliases:         []string{"Breaking.Bad", "بريكنغ باد"},
		SeasonNumbers:   []int{1, 2, 3, 4, 5},
		EpisodeCounts:   map[int]int{1: 7, 2: 13, 3: 13},
	}

	// A file whose folder says "Breaking_Bad" and whose filename is generic.
	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:                "D:/Media/Series/Breaking_Bad/Season 02/Episode 05.mkv",
			OriginalName:        "Episode 05.mkv",
			ParsedTitle:         "Breaking Bad",
			WorkFolderTitle:     "Breaking_Bad",
			SeasonFolderNumber:  2,
			ParsedSeason:        2,
			ParsedEpisode:       5,
			ParserEpisodeStrong: true,
			CategoryHint:        "series",
			ParserConfidence:    0.8,
			Group:               GroupContext{Size: 13, SeasonFolder: true},
		},
		Known: []Work{existing},
	})

	if resolution.Work == nil || resolution.Work.ID != 42 {
		t.Fatalf("did not attach to the existing work: %+v", resolution)
	}
	if resolution.Decision != DecisionAuto {
		t.Errorf("decision = %q, want auto_attach; reason=%s", resolution.Decision, resolution.Reason)
	}
	if resolution.NewWorkTitle == "Breaking_Bad" && resolution.Work == nil {
		t.Error("created a duplicate instead of attaching")
	}
}

// TestResolverUsesLearnedAlias verifies the rule-based library memory: once an
// alias is learned, resolution becomes exact rather than a similarity guess.
func TestResolverUsesLearnedAlias(t *testing.T) {
	resolver := NewResolver()
	_ = resolver
	existing := Work{
		ID:        7,
		TitleEN:   "One Piece",
		MediaType: "series",
	}

	// "ون بيس" was previously confirmed by an operator as One Piece.
	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:                "D:/Media/Anime/ون بيس/الحلقة 1000.mkv",
			OriginalName:        "الحلقة 1000.mkv",
			ParsedTitle:         "ون بيس",
			WorkFolderTitle:     "ون بيس",
			ParserEpisodeStrong: true,
			ParsedEpisode:       1000,
			ParsedSeason:        1,
		},
		Known:       []Work{existing},
		AliasLookup: map[string]int64{Normalize("ون بيس"): 7},
	})

	if resolution.Work == nil || resolution.Work.ID != 7 {
		t.Fatalf("learned alias was not used: %+v", resolution)
	}
	if resolution.ReasonCode != "alias_hit" {
		t.Errorf("reason = %q, want alias_hit", resolution.ReasonCode)
	}
	if resolution.ResolverConfidence < 0.9 {
		t.Errorf("an alias hit should be near-certain, got %.2f", resolution.ResolverConfidence)
	}
}

// -----------------------------------------------------------------------------
// Requirement 67: a new season attaches to the work, never becomes a new work
// -----------------------------------------------------------------------------

// TestSeasonNeverBecomesItsOwnWork is the explicit rule from requirement 67.
func TestSeasonNeverBecomesItsOwnWork(t *testing.T) {
	resolver := NewResolver()

	silo := Work{
		ID:              100,
		TitleEN:         "Silo",
		TitleNormalized: "silo",
		MediaType:       "series",
		CategorySlug:    "series",
		SeasonNumbers:   []int{1, 2},
		EpisodeCounts:   map[int]int{1: 10, 2: 10},
	}

	// Season 3 does not exist yet.
	resolution := resolver.Resolve(ResolverInput{
		Evidence: Evidence{
			Path:                "D:/Media/Series/Silo/الموسم الثالث/الحلقة 04.mkv",
			OriginalName:        "الحلقة 04.mkv",
			ParsedTitle:         "Silo",
			WorkFolderTitle:     "Silo",
			SeasonFolderNumber:  3,
			ParsedSeason:        3,
			ParsedEpisode:       4,
			ParserEpisodeStrong: true,
			CategoryHint:        "series",
			ParserConfidence:    0.85,
			Group:               GroupContext{Size: 8, SeasonFolder: true},
		},
		Known: []Work{silo},
	})

	if resolution.Work == nil || resolution.Work.ID != 100 {
		t.Fatalf("season 3 did not attach to work Silo: %+v", resolution)
	}
	if resolution.Season != 3 {
		t.Errorf("season = %d, want 3", resolution.Season)
	}
	if resolution.Episode != 4 {
		t.Errorf("episode = %d, want 4", resolution.Episode)
	}
	if resolution.NewWorkTitle == "Silo الموسم الثالث" {
		t.Error("the season folder became a new work name")
	}
}

// -----------------------------------------------------------------------------
// Requirement 62: ambiguity must not be resolved arbitrarily
// -----------------------------------------------------------------------------

// TestAmbiguousCandidatesGoToReview verifies that two equally good candidates
// produce a review request instead of picking one.
func TestAmbiguousCandidatesGoToReview(t *testing.T) {
	candidates := []Candidate{
		{WorkID: 1, Title: "Silo", Score: 72},
		{WorkID: 2, Title: "Silo (2023)", Score: 70},
	}

	_, ambiguous := ResolveAmbiguity(candidates, AmbiguityMargin)
	if !ambiguous {
		t.Fatal("two near-identical scores were not treated as ambiguous")
	}

	// A clear winner is not ambiguous.
	candidates[1].Score = 20
	if _, ambiguous := ResolveAmbiguity(candidates, AmbiguityMargin); ambiguous {
		t.Error("a clear winner was incorrectly treated as ambiguous")
	}
}

// TestContradictoryEvidencePenalisesCandidate verifies the scoring can say "no".
func TestContradictoryEvidencePenalisesCandidate(t *testing.T) {
	// A movie work offered a file from a 40-episode season must lose points.
	movieWork := Work{
		ID:           1,
		TitleEN:      "Silo",
		MediaType:    "movie",
		CategorySlug: "movies",
	}

	evidence := Evidence{
		ParsedTitle:         "Silo",
		WorkFolderTitle:     "Silo",
		CategoryHint:        "series",
		ParserEpisodeStrong: true,
		ParsedSeason:        4,
		ParsedEpisode:       12,
		Group: GroupContext{
			Size:             20,
			EpisodicSiblings: 20,
			EpisodeNumbers:   []int{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}

	candidate := ScoreCandidate(movieWork, evidence)

	foundConflict := false
	for _, item := range candidate.Breakdown.Items {
		if item.Points < 0 {
			foundConflict = true
		}
	}
	if !foundConflict {
		t.Errorf("a movie work matched against an episode run recorded no contradiction: %+v",
			candidate.Breakdown.Items)
	}
}

// -----------------------------------------------------------------------------
// Score explainability
// -----------------------------------------------------------------------------

// TestScoreBreakdownIsExplainable ensures every decision can be justified in the
// admin review screen rather than being an opaque number.
func TestScoreBreakdownIsExplainable(t *testing.T) {
	work := Work{
		ID:            5,
		TitleEN:       "Breaking Bad",
		TitleAR:       "بريكنغ باد",
		Aliases:       []string{"Breaking.Bad"},
		MediaType:     "series",
		CategorySlug:  "series",
		ReleaseYear:   2008,
		SeasonNumbers: []int{1, 2},
		EpisodeCounts: map[int]int{1: 7, 2: 13},
	}

	evidence := Evidence{
		ParsedTitle:         "Breaking Bad",
		WorkFolderTitle:     "Breaking Bad",
		CategoryHint:        "series",
		ParsedYear:          2008,
		ParsedSeason:        1,
		ParsedEpisode:       1,
		ParserEpisodeStrong: true,
		ParserConfidence:    0.9,
		Group: GroupContext{
			Size:             7,
			EpisodicSiblings: 7,
			EpisodeNumbers:   []int{1, 2, 3, 4, 5, 6, 7},
			ConsensusTitle:   "breaking bad",
			ConsensusVotes:   7,
			SeasonFolder:     true,
		},
	}

	candidate := ScoreCandidate(work, evidence)

	if len(candidate.Breakdown.Items) == 0 {
		t.Fatal("no evidence items were recorded")
	}
	for _, item := range candidate.Breakdown.Items {
		if item.Label == "" {
			t.Errorf("an evidence item has no label: %+v", item)
		}
	}
	if candidate.Score < ThresholdAutoAttach {
		t.Errorf("a fully matching file scored only %.0f; items=%+v", candidate.Score, candidate.Breakdown.Items)
	}
	if candidate.Decision != DecisionAuto {
		t.Errorf("decision = %q, want auto_attach", candidate.Decision)
	}
}

// TestExactTitleMatchOutranksWeakSimilarity documents the ranking order.
func TestExactTitleMatchOutranksWeakSimilarity(t *testing.T) {
	evidence := Evidence{
		ParsedTitle:      "Silo",
		WorkFolderTitle:  "Silo",
		ParserConfidence: 0.8,
	}

	exact := ScoreCandidate(Work{ID: 1, TitleEN: "Silo", MediaType: "series"}, evidence)
	similar := ScoreCandidate(Work{ID: 2, TitleEN: "The Silent Sea", MediaType: "series"}, evidence)

	if exact.Score <= similar.Score {
		t.Errorf("exact match (%.0f) did not outrank an unrelated title (%.0f)", exact.Score, similar.Score)
	}
}
