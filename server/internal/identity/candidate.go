package identity

import (
	"sort"
	"strings"
)

// -----------------------------------------------------------------------------
// Evidence
// -----------------------------------------------------------------------------

// Evidence is everything the resolver knows about one file before it decides
// which work it belongs to. Collecting it separately from scoring is what makes
// the decision auditable: every score can be traced to a named piece of proof.
type Evidence struct {
	// File facts.
	Path         string
	OriginalName string
	FileSize     int64

	// Path hierarchy, outermost first, e.g.
	// ["Media", "مسلسلات", "تركي", "Silo", "الموسم الثالث"].
	Segments []string

	// Parser output.
	ParsedTitle         string
	ParsedTitleAR       string
	ParsedTitleEN       string
	ParsedYear          int
	ParsedSeason        int
	ParsedEpisode       int
	ParsedPart          int
	ParsedResolution    string
	ParserCategory      string
	ParserConfidence    float64
	ParserIsEpisode     bool
	ParserEpisodeStrong bool

	// ProviderIdentity is "provider:externalID" when the FILE itself carries a
	// provider identity — for example a folder named after a TMDB id, or metadata
	// already recorded for this exact path. It is empty in normal scanning, which
	// is the expected case; it exists so a provider id on the entity can never be
	// mistaken for evidence about the file.
	ProviderIdentity string
	// Folder-derived context.
	WorkFolderTitle    string // the folder most likely naming the work
	SeasonFolderNumber int
	SeasonFolderTitle  string
	OriginTag          string
	CategoryHint       string

	// Group context: what the surrounding files look like. This is the evidence
	// that lets a folder of "01.mkv .. 04.mkv" be recognised as one season even
	// though no single filename is informative.
	Group GroupContext
}

// GroupContext summarises the directory a file lives in. Resolving a file in
// isolation is the main cause of invention; seeing the sibling set is what lets
// the resolver be confident.
type GroupContext struct {
	// Total candidate media files discovered in the same directory.
	Size int
	// How many siblings carry a strong episode marker (S01E01, الحلقة 4, "04").
	EpisodicSiblings int
	// How many distinct plausible episode numbers the group covers.
	EpisodeNumbers []int
	// How many siblings carry a release year, i.e. look like standalone movies.
	YearSiblings int
	// True when the directory itself is named like a season.
	SeasonFolder bool
	// The most common non-empty parsed title among siblings. A title shared by
	// many files is far more trustworthy than one file's guess.
	ConsensusTitle string
	// How many siblings agree with ConsensusTitle.
	ConsensusVotes int
}

// EpisodeRange is the span of episode numbers observed in the group.
type EpisodeRange struct {
	Low        int
	High       int
	Contiguous bool
}

// ContiguousEpisodeRange reports the span of episode numbers observed in the
// group. A contiguous run starting at 1 is strong series evidence.
func (g GroupContext) ContiguousEpisodeRange() EpisodeRange {
	if len(g.EpisodeNumbers) == 0 {
		return EpisodeRange{}
	}
	numbers := append([]int(nil), g.EpisodeNumbers...)
	sort.Ints(numbers)

	low, high := numbers[0], numbers[len(numbers)-1]
	span := high - low + 1
	if low != 1 || span <= 0 {
		return EpisodeRange{Low: low, High: high}
	}
	// Require the range to be densely covered, allowing a small number of gaps
	// so one missing episode does not destroy a confident group.
	coverage := float64(len(numbers)) / float64(span)
	return EpisodeRange{Low: low, High: high, Contiguous: coverage >= 0.75 && len(numbers) >= 3}
}

// -----------------------------------------------------------------------------
// Candidate and scoring
// -----------------------------------------------------------------------------

// Work is the existing logical entity a file may attach to. It is a plain value
// so the resolver has no dependency on the database layer.
type Work struct {
	ID              int64
	TitleEN         string
	TitleAR         string
	TitleNormalized string
	MediaType       string // movie | series | anime | unknown
	CategorySlug    string
	ReleaseYear     int
	Aliases         []string
	OriginTag       string
	Provider        string
	ExternalID      string
	// Existing season numbers, used for season-compatibility evidence.
	SeasonNumbers []int
	// Existing episode numbers per season, used to detect contradictory evidence
	// like a file claiming episode 400 of a 10-episode season.
	EpisodeCounts map[int]int
	// Provisional marks an entity created by the resolver before enrichment.
	Provisional bool
	// MetadataLocked means an operator has edited it and the scanner must not
	// change its canonical fields.
	MetadataLocked bool
}

// ScoreBreakdown is the itemised reasoning behind a candidate's score. The admin
// review UI shows exactly this, so a decision is never a bare number.
type ScoreBreakdown struct {
	Items []ScoreItem
	Total float64
}

// ScoreItem is one named contribution to a match score.
type ScoreItem struct {
	Label  string  `json:"label"`
	Points float64 `json:"points"`
	Detail string  `json:"detail,omitempty"`
}

func (b *ScoreBreakdown) add(label string, points float64, detail string) {
	b.Items = append(b.Items, ScoreItem{Label: label, Points: points, Detail: detail})
	b.Total += points
}

// cap limits the total to a ceiling without discarding the itemised evidence, so
// the review screen still shows exactly why a candidate scored what it did.
func (b *ScoreBreakdown) cap(ceiling float64) {
	if b.Total <= ceiling {
		return
	}
	b.Items = append(b.Items, ScoreItem{
		Label:  "scored_capped",
		Points: ceiling - b.Total,
		Detail: "capped: structural similarity without a name match cannot identify a work",
	})
	b.Total = ceiling
}

// Weights are the evidence weights from the design. They are named constants
// rather than inline numbers so a tuning change is explicit and reviewable.
const (
	WeightExactTitle       = 40
	WeightFolderTitle      = 25
	WeightAliasMatch       = 20
	WeightSeasonCompatible = 10
	WeightYearCompatible   = 10
	WeightCategoryCompat   = 10
	WeightEpisodeRelation  = 20
	WeightGroupConsensus   = 18
	WeightGroupSeasonShape = 14
	WeightProviderIdentity = 45
	WeightContradiction    = -30
	WeightTypeConflict     = -25
	WeightUnmatchedQuality = -12
)

// Decision is the resolver's verdict for candidate.
type Decision string

const (
	// DecisionAuto attaches with no human involvement.
	DecisionAuto Decision = "auto_attach"
	// DecisionProvisional attaches but flags the entity for review, because the
	// match is plausible yet not certain.
	DecisionProvisional Decision = "attach_provisional"
	// DecisionReview means no automatic decision is safe; a human decides.
	DecisionReview Decision = "needs_review"
	// DecisionCreate creates a new provisional work. It is only allowed when the
	// evidence is strong enough that inventing an entity is better than losing
	// the file, and even then the entity is marked provisional.
	DecisionCreate Decision = "create_provisional"
)

// Thresholds encode "never create a new work too quickly". They are deliberately
// high for creation: an unresolved file is recoverable, a polluted library is not.
const (
	ThresholdAutoAttach  = 70
	ThresholdProvisional = 45
	ThresholdCreateWork  = 75
	ThresholdReview      = 0
)

// Candidate is a scored possible match.
type Candidate struct {
	Work       Work           `json:"-"`
	WorkID     int64          `json:"work_id"`
	Title      string         `json:"title"`
	Score      float64        `json:"score"`
	Breakdown  ScoreBreakdown `json:"breakdown"`
	Decision   Decision       `json:"decision"`
	Season     int            `json:"season,omitempty"`
	Episode    int            `json:"episode,omitempty"`
	Reasons    []string       `json:"reasons,omitempty"`
	TitleMatch float64        `json:"title_match"`
}

// ScoreCandidate evaluates one existing work against the collected evidence.
//
// The result is additive and explainable: every point comes from a named piece
// of evidence, and contradictory evidence subtracts. This is what makes the
// resolver predictable and debuggable, unlike an opaque similarity number.
func ScoreCandidate(work Work, evidence Evidence) Candidate {
	breakdown := ScoreBreakdown{}
	candidate := Candidate{Work: work, WorkID: work.ID, Title: displayTitle(work)}

	filenameTitle := evidence.ParsedTitle
	// Compare against the best of every name we have for this work.
	bestMatch := 0.0
	matchedAgainst := ""
	for _, name := range append([]string{work.TitleEN, work.TitleAR, work.TitleNormalized}, work.Aliases...) {
		if strings.TrimSpace(name) == "" {
			continue
		}
		score := Similarity(filenameTitle, name)
		if score > bestMatch {
			bestMatch = score
			matchedAgainst = name
		}
	}
	candidate.TitleMatch = bestMatch

	switch {
	case bestMatch >= 0.995:
		breakdown.add("exact_title", WeightExactTitle, "filename title equals "+matchedAgainst)
	case bestMatch >= 0.9:
		breakdown.add("near_exact_title", WeightAliasMatch, "filename title closely matches "+matchedAgainst)
	case bestMatch >= 0.7:
		breakdown.add("partial_title", WeightAliasMatch*0.6, "filename title partially matches "+matchedAgainst)
	case bestMatch >= 0.5:
		breakdown.add("weak_title", WeightAliasMatch*0.25, "filename title weakly resembles "+matchedAgainst)
	}

	// Folder title evidence. A folder named after the show is the strongest
	// signal in a real library, and it survives a useless filename.
	if evidence.WorkFolderTitle != "" {
		folderMatch := 0.0
		for _, name := range append([]string{work.TitleEN, work.TitleAR}, work.Aliases...) {
			if strings.TrimSpace(name) == "" {
				continue
			}
			if score := Similarity(evidence.WorkFolderTitle, name); score > folderMatch {
				folderMatch = score
			}
		}
		if folderMatch >= 0.9 {
			breakdown.add("folder_title", WeightFolderTitle, "folder "+evidence.WorkFolderTitle+" names this work")
		} else if folderMatch >= 0.7 {
			breakdown.add("folder_title_partial", WeightFolderTitle*0.6, "folder "+evidence.WorkFolderTitle+" resembles this work")
		}
	}

	// Explicit alias hit, which is the learned library memory in action.
	for _, alias := range work.Aliases {
		if alias == "" {
			continue
		}
		if Normalize(alias) == Normalize(filenameTitle) {
			breakdown.add("alias_match", WeightAliasMatch, "learned alias "+alias)
			break
		}
	}

	// Provider identity is definitive ONLY when the file actually carries the
	// same provider identity.
	//
	// It must never be awarded merely because the candidate entity happens to
	// have a TMDB id. Doing that gave every enriched work +45 points against
	// every file, which made unrelated titles tie at the same score and pushed
	// them all into review. The identity is only evidence when both sides agree.
	if work.Provider != "" && work.ExternalID != "" && evidence.ProviderIdentity != "" {
		if evidence.ProviderIdentity == work.Provider+":"+work.ExternalID {
			breakdown.add("provider_identity", WeightProviderIdentity,
				work.Provider+" id "+work.ExternalID+" confirmed by the file name")
		} else {
			breakdown.add("provider_conflict", WeightContradiction*0.8,
				"file identifies a different provider title than "+work.Provider+" id "+work.ExternalID)
		}
	}

	// Season compatibility. A season that already exists is supportive; a file
	// claiming a season the work cannot plausibly have is contradictory.
	if evidence.ParsedSeason > 0 {
		if containsInt(work.SeasonNumbers, evidence.ParsedSeason) {
			breakdown.add("season_known", WeightSeasonCompatible, "season already exists on this work")
		} else if len(work.SeasonNumbers) > 0 {
			breakdown.add("season_new", WeightSeasonCompatible*0.3, "season is new for this work")
		}
	}

	// Year compatibility. A two-year tolerance avoids penalising release-year
	// differences between regional releases of the same title.
	if evidence.ParsedYear > 0 && work.ReleaseYear > 0 {
		diff := evidence.ParsedYear - work.ReleaseYear
		if diff < 0 {
			diff = -diff
		}
		switch {
		case diff == 0:
			breakdown.add("year_exact", WeightYearCompatible, "release year matches")
		case diff <= 1:
			breakdown.add("year_close", WeightYearCompatible*0.7, "release year is within a year")
		case diff <= 2:
			breakdown.add("year_near", WeightYearCompatible*0.4, "release year is within two years")
		default:
			breakdown.add("year_conflict", WeightContradiction, "release year differs by "+itoa(diff))
		}
	}

	// Category compatibility.
	if evidence.CategoryHint != "" && work.CategorySlug != "" {
		if evidence.CategoryHint == work.CategorySlug {
			breakdown.add("category_match", WeightCategoryCompat, "category "+work.CategorySlug+" matches")
		} else if categoriesCompatible(evidence.CategoryHint, work.CategorySlug) {
			breakdown.add("category_related", WeightCategoryCompat*0.5, "categories are related")
		} else {
			breakdown.add("category_conflict", WeightContradiction*0.7, "category "+evidence.CategoryHint+" conflicts with "+work.CategorySlug)
		}
	}

	// Origin compatibility: a Turkish folder should not attach to an anime work.
	if evidence.OriginTag != "" && work.OriginTag != "" {
		if evidence.OriginTag == work.OriginTag {
			breakdown.add("origin_match", WeightCategoryCompat*0.6, "origin "+work.OriginTag+" matches")
		} else {
			breakdown.add("origin_conflict", WeightContradiction*0.5, "origin differs")
		}
	}

	// Existing episode relationship: if the surrounding group's episode numbers
	// fit inside the work's known episode set, this is a strong signal.
	if season := evidence.ParsedSeason; season > 0 {
		if known, exists := work.EpisodeCounts[season]; exists && known > 0 {
			if evidence.ParsedEpisode > 0 && evidence.ParsedEpisode <= known {
				breakdown.add("episode_in_range", WeightEpisodeRelation, "episode fits the known season size")
			} else if evidence.ParsedEpisode > known*3 && known < 50 {
				// A huge episode number against a small known season is a parser
				// mistake, not a new episode.
				breakdown.add("episode_out_of_range", WeightContradiction,
					"episode "+itoa(evidence.ParsedEpisode)+" exceeds the known season size "+itoa(known))
			}
		}
	}

	// Group consensus: the neighbourhood agreeing on a title is strong evidence
	// that survives one bad filename.
	if evidence.Group.ConsensusTitle != "" && evidence.Group.ConsensusVotes >= 2 {
		consensusMatch := Similarity(evidence.Group.ConsensusTitle, work.TitleEN)
		consensusMatch = maxFloat(consensusMatch, Similarity(evidence.Group.ConsensusTitle, work.TitleAR))
		for _, alias := range work.Aliases {
			consensusMatch = maxFloat(consensusMatch, Similarity(evidence.Group.ConsensusTitle, alias))
		}
		if consensusMatch >= 0.85 {
			breakdown.add("group_consensus", WeightGroupConsensus,
				itoa(evidence.Group.ConsensusVotes)+" sibling files agree on this title")
		}
	}

	// Season-shape evidence: a folder holding a contiguous episode run is a
	// series, so a series work should win over a movie work.
	if episodeRange := evidence.Group.ContiguousEpisodeRange(); episodeRange.Contiguous {
		if isSeriesType(work.MediaType) {
			breakdown.add("group_series_shape", WeightGroupSeasonShape,
				"sibling episodes form a contiguous run "+itoa(episodeRange.Low)+"-"+itoa(episodeRange.High))
		} else if work.MediaType == "movie" {
			breakdown.add("group_shape_conflict", WeightTypeConflict,
				"the directory holds an episode run but this work is a movie")
		}
	}

	// A work with no name resemblance at all to anything we observed must not
	// collect a high score from structural coincidence alone.
	//
	// Structural evidence (a matching category, a nearby year) is weak on its
	// own: every film shares the "movies" category and many share a release
	// year. Without a name link, that evidence cannot identify a work, and
	// letting it accumulate is how two unrelated titles ended up tied.
	hasNameEvidence := bestMatch >= 0.5 ||
		evidence.WorkFolderTitle != "" ||
		evidence.Group.ConsensusTitle != "" ||
		evidence.ProviderIdentity != ""
	if !hasNameEvidence {
		breakdown.add("no_name_evidence", WeightUnmatchedQuality,
			"no title or identity evidence connects this work to the file")
	}
	// Structural coincidence alone is not an identification. When nothing names
	// the work, the score is capped well below the attach threshold so the file
	// is reviewed instead of silently attached to an unrelated title.
	if bestMatch < 0.5 {
		breakdown.cap(ThresholdProvisional - 1)
	}

	candidate.Breakdown = breakdown
	candidate.Score = breakdown.Total
	candidate.Decision, candidate.Reasons = decide(breakdown.Total, candidate, evidence)
	return candidate
}

// RankCandidates scores every known work and returns them best first.
func RankCandidates(works []Work, evidence Evidence) []Candidate {
	candidates := make([]Candidate, 0, len(works))
	for _, work := range works {
		if work.ID == 0 {
			continue
		}
		candidates = append(candidates, ScoreCandidate(work, evidence))
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			// Prefer the entity that is not provisional, so a real work wins a tie.
			return !candidates[i].Work.Provisional && candidates[j].Work.Provisional
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

// decide converts a score into a verdict, applying the ambiguity rule: when the
// top two candidates are too close, no automatic decision is safe.
func decide(score float64, candidate Candidate, evidence Evidence) (Decision, []string) {
	reasons := make([]string, 0, 2)

	switch {
	case score >= ThresholdAutoAttach:
		return DecisionAuto, reasons
	case score >= ThresholdProvisional:
		reasons = append(reasons, "evidence supports an attachment but is not conclusive")
		return DecisionProvisional, reasons
	default:
		reasons = append(reasons, "insufficient evidence to attach to this work")
		return DecisionReview, reasons
	}
}

// ResolveAmbiguity is applied after ranking: if the best two candidates are
// within the ambiguity margin, the decision is downgraded to review regardless
// of the absolute score. A confident score means nothing when a rival scores the
// same, which is exactly the "Silo" vs "Silo (2023)" situation.
func ResolveAmbiguity(candidates []Candidate, margin float64) (Candidate, bool) {
	if len(candidates) == 0 {
		return Candidate{}, false
	}
	best := candidates[0]
	if len(candidates) == 1 {
		return best, false
	}
	second := candidates[1]
	if best.Score-second.Score < margin {
		best.Decision = DecisionReview
		best.Reasons = append(best.Reasons, "ambiguous: "+itoa(int(best.Score))+
			" vs "+itoa(int(second.Score))+" for "+second.Title)
		return best, true
	}
	return best, false
}

// AmbiguityMargin is the score gap below which two candidates are considered
// indistinguishable.
const AmbiguityMargin = 8

func displayTitle(work Work) string {
	if work.TitleEN != "" {
		return work.TitleEN
	}
	if work.TitleAR != "" {
		return work.TitleAR
	}
	return work.TitleNormalized
}

func isSeriesType(mediaType string) bool {
	switch mediaType {
	case "series", "anime", "tv", "episodic":
		return true
	}
	return false
}

// categoriesCompatible reports whether two categories describe comparable
// content, so anime and series are not treated as a conflict. This is the
// distinction the design requires: category is not the same axis as media type.
func categoriesCompatible(a, b string) bool {
	if a == b {
		return true
	}
	families := [][]string{
		{"series", "anime", "kids", "documentaries", "plays"},
		{"movies", "documentaries", "plays"},
	}
	for _, family := range families {
		if containsString(family, a) && containsString(family, b) {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
