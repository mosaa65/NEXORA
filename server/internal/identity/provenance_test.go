package identity

import (
	"testing"
)

// -----------------------------------------------------------------------------
// Requirement 75: a Full Scan must not destroy manual corrections
// -----------------------------------------------------------------------------

// TestAdminValueSurvivesScannerRescan is the explicit guarantee. An operator who
// renamed "Silo" to "Silo (2023)" must not have it reverted by the next scan.
func TestAdminValueSurvivesScannerRescan(t *testing.T) {
	// The operator's correction is stored with admin provenance and locked.
	existing := map[string]FieldValue{
		"title": {Value: "Silo (2023)", Source: SourceAdmin, Locked: true},
	}

	// The scanner arrives with the raw parser guess.
	incoming := map[string]FieldValue{
		"title": {Value: "Silo", Source: SourceParser},
	}

	merged, changed := MergeFields(incoming, existing)

	if merged["title"].Value != "Silo (2023)" {
		t.Errorf("title = %q, want the operator value to survive", merged["title"].Value)
	}
	if len(changed) != 0 {
		t.Errorf("a locked field was scheduled for writing: %v", changed)
	}
}

// TestProviderValueBeatsParserButNotAdmin documents the precedence order.
func TestProviderValueBeatsParserButNotAdmin(t *testing.T) {
	tests := []struct {
		name     string
		incoming FieldValue
		existing FieldValue
		want     string
		write    bool
	}{
		{
			name:     "parser may not overwrite provider",
			incoming: FieldValue{Value: "Breaking.Bad", Source: SourceParser},
			existing: FieldValue{Value: "Breaking Bad", Source: SourceProvider},
			want:     "Breaking Bad",
			write:    false,
		},
		{
			name:     "admin may overwrite provider",
			incoming: FieldValue{Value: "Operator Choice", Source: SourceAdmin},
			existing: FieldValue{Value: "Breaking Bad", Source: SourceProvider},
			want:     "Operator Choice",
			write:    true,
		},
		{
			name:     "provider may overwrite parser",
			incoming: FieldValue{Value: "Breaking Bad", Source: SourceProvider},
			existing: FieldValue{Value: "Breaking.Bad", Source: SourceParser},
			want:     "Breaking Bad",
			write:    true,
		},
		{
			name:     "empty incoming never erases a value",
			incoming: FieldValue{Value: "", Source: SourceAdmin},
			existing: FieldValue{Value: "Keep Me", Source: SourceParser},
			want:     "Keep Me",
			write:    false,
		},
		{
			name:     "resolver may refine a parser guess",
			incoming: FieldValue{Value: "Silo", Source: SourceResolver},
			existing: FieldValue{Value: "Silo الموسم", Source: SourceParser},
			want:     "Silo",
			write:    true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			existing := testCase.existing
			merged, changed := MergeFields(
				map[string]FieldValue{"field": testCase.incoming},
				map[string]FieldValue{"field": existing},
			)
			if got := merged["field"].Value; got != testCase.want {
				t.Errorf("value = %q, want %q", got, testCase.want)
			}
			shouldWrite := len(changed) > 0
			if shouldWrite != testCase.write {
				t.Errorf("write = %v, want %v", shouldWrite, testCase.write)
			}
		})
	}
}

// TestSourceRankingIsTotal ensures precedence is a strict order with no ties
// that could make behaviour depend on map iteration.
func TestSourceRankingIsTotal(t *testing.T) {
	ordered := []Source{
		SourceFilesystem, SourceParser, SourceResolver,
		SourceDatabase, SourceProvider, SourceAdmin,
	}
	for i := 1; i < len(ordered); i++ {
		if ordered[i].Rank() <= ordered[i-1].Rank() {
			t.Errorf("%s (%d) must outrank %s (%d)",
				ordered[i], ordered[i].Rank(), ordered[i-1], ordered[i-1].Rank())
		}
	}
}

// -----------------------------------------------------------------------------
// Requirement 71: rule-based library learning
// -----------------------------------------------------------------------------

// TestAliasLearningAcceptsSpellingVariantsOnly verifies the learner stores
// spellings of the same name and rejects anything that could poison the library.
func TestAliasLearningAcceptsSpellingVariantsOnly(t *testing.T) {
	learner := NewAliasLearner()
	work := Work{
		ID:              9,
		TitleEN:         "Breaking Bad",
		TitleAR:         "بريكنغ باد",
		TitleNormalized: "breaking bad",
	}

	// A separator-only variant normalizes to the canonical title, so there is
	// nothing new to learn: storing it would only add noise. This is the
	// correct outcome, not a miss.
	redundant := learner.Learn(Resolution{Decision: DecisionAuto}, work, Evidence{
		WorkFolderTitle: "Breaking_Bad",
		ParsedTitle:     "Breaking Bad",
	})
	if len(redundant) != 0 {
		t.Errorf("a separator-only variant was stored as a new alias: %+v", redundant)
	}

	// A genuinely different spelling is the valuable alias, because it links a
	// second name to the canonical work. "Breaking Bad - بطل العالم" is the real
	// bilingual-folder shape, and "BreakingBad" is the no-space compact shape.
	aliases := learner.Learn(Resolution{Decision: DecisionAuto}, work, Evidence{
		WorkFolderTitle: "BreakingBad",
		ParsedTitle:     "BreakingBad",
	})

	if len(aliases) == 0 {
		t.Fatalf("a genuine spelling variant was not learned (best similarity was below threshold)")
	}
	for _, alias := range aliases {
		if alias.Normalized == "" {
			t.Errorf("learned alias %q has no normalized form", alias.Alias)
		}
		if alias.Source != SourceResolver {
			t.Errorf("auto-learned alias has source %q, want resolver", alias.Source)
		}
		if alias.Reason == "" {
			t.Errorf("learned alias %q has no reason", alias.Alias)
		}
	}
}

// TestAliasLearningIgnoresReviewItems ensures an uncertain resolution cannot
// teach the library, which is how a bad alias would spread.
func TestAliasLearningIgnoresReviewItems(t *testing.T) {
	learner := NewAliasLearner()
	work := Work{ID: 9, TitleEN: "Silo"}

	for _, decision := range []Decision{DecisionReview, DecisionProvisional} {
		aliases := learner.Learn(Resolution{Decision: decision}, work, Evidence{
			WorkFolderTitle: "Silo",
			ParsedTitle:     "Silo",
		})
		if len(aliases) != 0 {
			t.Errorf("decision %q learned %d aliases; uncertain resolutions must not teach", decision, len(aliases))
		}
	}
}

// TestAliasLearningRejectsUnrelatedNames prevents a different show from being
// recorded as an alias of this one.
func TestAliasLearningRejectsUnrelatedNames(t *testing.T) {
	learner := NewAliasLearner()
	work := Work{ID: 9, TitleEN: "Silo", TitleAR: "سايلو"}

	aliases := learner.Learn(Resolution{Decision: DecisionAuto}, work, Evidence{
		WorkFolderTitle: "Severance",
		ParsedTitle:     "Planet Earth",
	})

	if len(aliases) != 0 {
		t.Errorf("unrelated names were learned as aliases: %+v", aliases)
	}
}

// TestAdminAliasIsAlwaysStored verifies an explicit decision is never second
// guessed, because a human asserted it.
func TestAdminAliasIsAlwaysStored(t *testing.T) {
	alias := LearnedFromAdmin(9, "ون بيس", "One Piece")

	if alias.Source != SourceAdmin {
		t.Errorf("source = %q, want admin", alias.Source)
	}
	if alias.Normalized != Normalize("ون بيس") {
		t.Errorf("normalized = %q, want %q", alias.Normalized, Normalize("ون بيس"))
	}
	if alias.Reason == "" {
		t.Error("an operator decision must record its reasoning")
	}
}

// -----------------------------------------------------------------------------
// Duplicate work detection
// -----------------------------------------------------------------------------

// TestDetectDuplicateWorksFindsRealDuplicates reproduces the actual database
// problem: "Fate Stay Night" and "Fate_Stay_Night" are one work.
func TestDetectDuplicateWorksFindsRealDuplicates(t *testing.T) {
	works := []Work{
		{ID: 1, TitleEN: "Fate Stay Night", MediaType: "series", Provisional: true},
		{ID: 2, TitleEN: "Fate_Stay_Night", MediaType: "series", Provisional: true},
		{ID: 3, TitleEN: "One Piece", MediaType: "series"},
	}

	plans := DetectDuplicateWorks(works)
	if len(plans) != 1 {
		t.Fatalf("found %d merge plans, want 1: %+v", len(plans), plans)
	}
	if plans[0].CanonicalID != 1 {
		t.Errorf("canonical = %d, want the lower id 1 for determinism", plans[0].CanonicalID)
	}
	if len(plans[0].MergedIDs) != 1 || plans[0].MergedIDs[0] != 2 {
		t.Errorf("merged ids = %v, want [2]", plans[0].MergedIDs)
	}
}

// TestDetectDuplicateWorksPrefersNonProvisional verifies the canonical choice
// favours a real entity over a provisional one.
func TestDetectDuplicateWorksPrefersNonProvisional(t *testing.T) {
	works := []Work{
		{ID: 10, TitleEN: "Silo", MediaType: "series", Provisional: true},
		{ID: 20, TitleEN: "Silo", MediaType: "series", Provisional: false},
	}

	plans := DetectDuplicateWorks(works)
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %+v", plans)
	}
	if plans[0].CanonicalID != 20 {
		t.Errorf("canonical = %d, want the non-provisional work 20", plans[0].CanonicalID)
	}
}

// TestDetectDuplicateWorksSeparatesMediaTypes ensures a film and a series with
// the same name are not merged.
func TestDetectDuplicateWorksSeparatesMediaTypes(t *testing.T) {
	works := []Work{
		{ID: 1, TitleEN: "Fargo", MediaType: "movie"},
		{ID: 2, TitleEN: "Fargo", MediaType: "series"},
	}

	if plans := DetectDuplicateWorks(works); len(plans) != 0 {
		t.Errorf("a film and a series with the same name were merged: %+v", plans)
	}
}
