package identity

import (
	"strings"
)

// -----------------------------------------------------------------------------
// Folder context graph
// -----------------------------------------------------------------------------

// ContextGraph is the meaning extracted from a file's path hierarchy.
//
// Owners organise libraries as Root/Category/Origin/Work/Season/Episode, and
// reading that structure is far more reliable than reading a filename. The graph
// is built once per directory and shared by every file inside it.
type ContextGraph struct {
	// Segments, outermost first, with classification and origin folders removed.
	Segments []string
	// Category slug inferred from the hierarchy.
	Category string
	// Production origin inferred from the hierarchy.
	Origin string
	// WorkTitle is the folder most likely naming the work.
	WorkTitle string
	// WorkTitleDepth is the index in Segments where WorkTitle was found.
	WorkTitleDepth int
	// SeasonNumber when a folder declares one.
	SeasonNumber int
	// SeasonFolderTitle is the season folder name, which sometimes still carries
	// the work name ("Silo الموسم الثالث").
	SeasonFolderTitle string
	// EpisodeMarker when a folder declares an episode ("الحلقة 04").
	EpisodeMarker int
}

// BuildContextGraph reads the hierarchy and labels each level.
//
// It walks from the deepest folder outward and records the first segment that
// looks like a work name, which is the key insight: in
// "Media/مسلسلات/تركي/Silo/الموسم الثالث/الحلقة 04.mkv" the work is the segment
// directly above the season folder, not the file.
func BuildContextGraph(segments []string, analyze func(string) SegmentAnalysis) ContextGraph {
	graph := ContextGraph{Segments: segments}

	// Classify from outermost inward so a deeper category overrides a shallower
	// one, matching how owners nest "Movies/Series/...".
	for index, segment := range segments {
		analysis := analyze(segment)
		if analysis.Category != "" {
			graph.Category = analysis.Category
		}
		if analysis.Origin != "" {
			graph.Origin = analysis.Origin
		}
		_ = index
	}

	// Find the season and episode folders and the work folder.
	// Search from the deepest segment backwards: the closest folder wins.
	workDepth := -1
	for index := len(segments) - 1; index >= 0; index-- {
		analysis := analyze(segments[index])

		if analysis.Category != "" || isStructuralCategory(analysis.Category) {
			continue
		}

		if analysis.Season > 0 && graph.SeasonNumber == 0 {
			graph.SeasonNumber = analysis.Season
			graph.SeasonFolderTitle = segments[index]
			if analysis.TitlePrefix != "" {
				// "Silo الموسم الثالث" names both the work and the season.
				graph.WorkTitle = analysis.TitlePrefix
				graph.WorkTitleDepth = index
				workDepth = index
			}
			continue
		}

		if analysis.EpisodeRectangle > 0 && graph.EpisodeMarker == 0 {
			graph.EpisodeMarker = analysis.EpisodeRectangle
			continue
		}

		if analysis.Origin != "" {
			continue
		}

		if analysis.Title == "" {
			continue
		}

		if workDepth < 0 {
			graph.WorkTitle = analysis.Title
			graph.WorkTitleDepth = index
			workDepth = index
		}
	}

	if graph.WorkTitleDepth < 0 {
		graph.WorkTitle = ""
	}
	return graph
}

// SegmentAnalysis is what the scanner's parser reports about one path segment.
// It is an interface rather than a parser import so this package stays free of
// scanner dependencies and remains independently testable.
type SegmentAnalysis struct {
	Title            string
	TitlePrefix      string
	Season           int
	EpisodeRectangle int
	Category         string
	Origin           string
}

func isStructuralCategory(category string) bool {
	return category == ""
}

// -----------------------------------------------------------------------------
// Resolution
// -----------------------------------------------------------------------------

// Resolution is the complete verdict for one file.
type Resolution struct {
	// Decision is what should happen to the file.
	Decision Decision
	// Work is the entity the file attaches to, when there is one.
	Work *Work
	// NewWorkTitle is set when a new provisional work should be created.
	NewWorkTitle string
	// MediaType is the structural kind decided for the work.
	MediaType MediaType
	// Category is the content category decided from the hierarchy.
	Category   string
	Origin     string
	Season     int
	Episode    int
	EpisodeEnd int
	Part       int

	// Confidence values are separate so the admin UI can show which stage was
	// unsure: the parser may be confident while resolution is not, and vice versa.
	ParserConfidence   float64
	ResolverConfidence float64

	// State is the lifecycle state for the file after resolution.
	State ResolutionState
	// ReasonCode is a machine-readable explanation for queues and metrics.
	ReasonCode string
	// Reason is the human-readable explanation.
	Reason string
	// Candidates are the ranked alternatives, retained for the review queue so an
	// operator can choose instead of retyping.
	Candidates []Candidate
}

// ResolutionState mirrors the database enum, declared here so the resolver has
// no database dependency.
type ResolutionState string

const (
	StateDiscovered        ResolutionState = "discovered"
	StateParsed            ResolutionState = "parsed"
	StateResolved          ResolutionState = "resolved"
	StateUnresolved        ResolutionState = "unresolved"
	StateNeedsReview       ResolutionState = "needs_review"
	StateEnrichmentPending ResolutionState = "enrichment_pending"
	StateEnriched          ResolutionState = "enriched"
	StateIndexed           ResolutionState = "indexed"
	StateError             ResolutionState = "error"
)

// WorkTitleForDisplay returns the best human-readable name for the resolved work,
// preferring the canonical entity over the provisional candidate name. The admin
// review screen shows this next to the candidate list.
func (r Resolution) WorkTitleForDisplay() string {
	if r.Work != nil {
		if r.Work.TitleEN != "" {
			return r.Work.TitleEN
		}
		if r.Work.TitleAR != "" {
			return r.Work.TitleAR
		}
	}
	return r.NewWorkTitle
}

// ResolverInput is everything the resolver needs for one file.
type ResolverInput struct {
	Evidence Evidence
	// Known works, already filtered to plausible matches by the caller when the
	// library is large.
	Known []Work
	// AliasLookup maps a normalized alias to a work id, which is the learned
	// memory that makes repeated resolutions exact.
	AliasLookup map[string]int64
}

// Resolver resolves files to logical entities.
type Resolver struct {
	// AmbiguityMargin is the score gap below which two candidates are treated as
	// indistinguishable. It is configurable so a cautious library can widen it.
	AmbiguityMargin float64
	// AllowProvisionalCreation permits creating a provisional work when the
	// evidence is strong but no entity exists. When false, such files are queued
	// for review instead, which is the safest setting for a new library.
	AllowProvisionalCreation bool
	// MinimumAlphabetTitleLength rejects works whose title is too short to be a
	// real name, which is the guard against "03" becoming a work.
	MinimumAlphabetTitleLength int
}

// NewResolver returns a resolver with the safe defaults from the design.
func NewResolver() *Resolver {
	return &Resolver{
		AmbiguityMargin:            AmbiguityMargin,
		AllowProvisionalCreation:   true,
		MinimumAlphabetTitleLength: 2,
	}
}

// Resolve decides which logical entity a file belongs to.
//
// The order is fixed and deliberate:
//
//  1. Reject a title that is not a title at all (watermark, bare number).
//  2. Decide the media type from structure, not naming.
//  3. Decide season and episode, refusing bare numbers without support.
//  4. Look for an existing entity: exact alias, then scored candidates.
//  5. Only if no entity matches AND the evidence is strong, create a
//     provisional entity.
//  6. Otherwise queue for human review.
//
// Creating a work is the last resort, never the first behaviour.
func (r *Resolver) Resolve(input ResolverInput) Resolution {
	evidence := input.Evidence

	resolution := Resolution{
		ParserConfidence:   evidence.ParserConfidence,
		ResolverConfidence: 0,
		MediaType:          MediaTypeUnknown,
		Category:           evidence.CategoryHint,
		Origin:             evidence.OriginTag,
		Part:               evidence.ParsedPart,
	}

	// ---- Step 1: is the parsed title usable at all?
	parsedQuality := AssessTitle(evidence.ParsedTitle, evidence.OriginalName)
	folderQuality := AssessTitle(evidence.WorkFolderTitle, "")

	// Choosing the candidate name is a real decision, not a fallback chain.
	//
	// A folder is normally the better name because a human wrote it. But some
	// folders name a CONTAINER rather than the work: a franchise folder ("DC"),
	// an actor folder, or a browsing folder ("أعمال" = "works"). In those cases
	// the filename names the actual title.
	//
	// The discriminator is release evidence: a filename carrying a release year
	// or resolution marker is a release name, which is a stronger work name than
	// a one-or-two-word container folder.
	candidateTitle := ""
	titleOrigin := ""
	switch {
	case parsedQuality.Usable && filenameNamesTheWork(evidence):
		candidateTitle = evidence.ParsedTitle
		titleOrigin = "filename"
	case folderQuality.Usable:
		candidateTitle = evidence.WorkFolderTitle
		titleOrigin = "folder"
	case parsedQuality.Usable:
		candidateTitle = evidence.ParsedTitle
		titleOrigin = "filename"
	case evidence.Group.ConsensusTitle != "":
		candidateTitle = evidence.Group.ConsensusTitle
		titleOrigin = "group_consensus"
	}

	if candidateTitle == "" {
		return r.unresolved(resolution, evidence,
			"no_usable_title",
			"no folder or filename provided a usable title ("+firstNonEmpty(parsedQuality.Reason, folderQuality.Reason)+")")
	}

	resolution.NewWorkTitle = candidateTitle

	// ---- Step 2: media type from structure.
	typeEvidence := DetectMediaType(evidence)
	resolution.MediaType = typeEvidence.MediaType

	// ---- Step 3: season and episode.
	season := ResolveSeason(evidence)
	resolution.Season = season.Number
	episode := ResolveEpisode(evidence)
	if episode.Valid {
		resolution.Episode = episode.Number
		resolution.EpisodeEnd = episode.End
	}

	// ---- Step 4: existing entities. Alias lookup first, because a learned
	// alias is exact knowledge rather than a similarity guess.
	normalized := Normalize(candidateTitle)
	if input.AliasLookup != nil && normalized != "" {
		if workID, found := input.AliasLookup[normalized]; found {
			for index := range input.Known {
				if input.Known[index].ID == workID {
					work := input.Known[index]
					resolution.Work = &work
					resolution.Decision = DecisionAuto
					resolution.ResolverConfidence = 0.97
					resolution.State = StateResolved
					resolution.ReasonCode = "alias_hit"
					resolution.Reason = "matched a learned alias for " + displayTitle(work)
					r.fillFromWork(&resolution, work)
					return resolution
				}
			}
		}
	}

	// Score the candidates.
	candidates := RankCandidates(input.Known, evidence)
	resolution.Candidates = candidates

	if len(candidates) == 0 {
		// No entity resembles this file at all.
		return r.noCandidates(resolution, evidence, candidateTitle, titleOrigin, typeEvidence)
	}

	best, ambiguous := ResolveAmbiguity(candidates, r.AmbiguityMargin)
	resolution.ResolverConfidence = normalizeScore(best.Score)

	if ambiguous {
		resolution.Decision = DecisionReview
		resolution.State = StateNeedsReview
		resolution.ReasonCode = "ambiguous_work"
		resolution.Reason = "two works match equally: " + best.Title + " and " + candidates[1].Title
		return resolution
	}

	switch best.Decision {
	case DecisionAuto:
		work := best.Work
		resolution.Work = &work
		resolution.Decision = DecisionAuto
		resolution.State = StateResolved
		resolution.ReasonCode = "auto_attached"
		resolution.Reason = "matched " + best.Title + " with " + itoa(int(best.Score)) + " points"
		r.fillFromWork(&resolution, work)
		return resolution

	case DecisionProvisional:
		work := best.Work
		resolution.Work = &work
		resolution.Decision = DecisionProvisional
		resolution.State = StateNeedsReview
		resolution.ReasonCode = "low_confidence_attachment"
		resolution.Reason = "weak match to " + best.Title + " (" + itoa(int(best.Score)) + " points)"
		r.fillFromWork(&resolution, work)
		return resolution

	default:
		// The best candidate scored too low. Fall through to the creation check,
		// because a genuinely new work may be arriving.
	}

	return r.noStrongCandidate(resolution, evidence, candidateTitle, titleOrigin, typeEvidence)
}

// noCandidates handles a file that resembles nothing in the library.
func (r *Resolver) noCandidates(resolution Resolution, evidence Evidence,
	candidateTitle, titleOrigin string, typeEvidence TypeEvidence) Resolution {

	// A brand new work may only be created when the type is certain and the
	// title is usable. If the type is unknown, the file must be reviewed: this is
	// precisely the case that produced "الكنز ج1 الحلقه" as a work.
	if typeEvidence.MediaType == MediaTypeUnknown {
		resolution.State = StateNeedsReview
		resolution.Decision = DecisionReview
		resolution.ReasonCode = "unknown_media_type"
		resolution.Reason = "could not decide between movie and series; queued for review"
		return resolution
	}
	if !r.AllowProvisionalCreation {
		resolution.State = StateNeedsReview
		resolution.Decision = DecisionReview
		resolution.ReasonCode = "creation_disabled"
		resolution.Reason = "no matching work and provisional creation is disabled"
		return resolution
	}

	resolution.Decision = DecisionCreate
	resolution.State = StateEnrichmentPending
	resolution.ResolverConfidence = creationConfidence(evidence, titleOrigin)
	resolution.ReasonCode = "new_work"
	resolution.Reason = "no existing work matched; created a provisional " +
		string(typeEvidence.MediaType) + " from the " + titleOrigin + " title"
	return resolution
}

// noStrongCandidate handles a file whose best candidate was too weak.
func (r *Resolver) noStrongCandidate(resolution Resolution, evidence Evidence,
	candidateTitle, titleOrigin string, typeEvidence TypeEvidence) Resolution {
	return r.noCandidates(resolution, evidence, candidateTitle, titleOrigin, typeEvidence)
}

// creationConfidence scores how safe it is to invent an entity.
func creationConfidence(evidence Evidence, titleOrigin string) float64 {
	score := 0.4
	switch titleOrigin {
	case "folder":
		score += 0.2
	case "group_consensus":
		score += 0.15
	}
	if evidence.ParserEpisodeStrong {
		score += 0.1
	}
	if evidence.SeasonFolderNumber > 0 {
		score += 0.1
	}
	if evidence.Group.ContiguousEpisodeRange().Contiguous {
		score += 0.15
	}
	if evidence.ParserConfidence >= 0.8 {
		score += 0.05
	}
	if score > 0.95 {
		score = 0.95
	}
	return score
}

// filenameNamesTheWork reports whether the filename is a better work name than
// the folder it sits in.
//
// The rule targets container folders, which are extremely common in real
// libraries and are the reason a naive "folder wins" policy breaks:
//
//	.../Franchises/DC/Part 3 - The Dark Knight Rises.mkv   folder "DC" is a franchise
//	.../مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv  folder "أعمال" is "works"
//
// In both cases the filename carries the title. The discriminator is release
// evidence: a filename with a release year or a resolution marker is a release
// name, and a short folder beside it is almost certainly a container.
func filenameNamesTheWork(evidence Evidence) bool {
	parsed := strings.TrimSpace(evidence.ParsedTitle)
	folder := strings.TrimSpace(evidence.WorkFolderTitle)

	if parsed == "" {
		return false
	}
	// With no folder name there is nothing to prefer the filename over; the
	// normal fallback handles it.
	if folder == "" {
		return true
	}

	// A folder that only classifies or describes structure is a container.
	if isContainerFolderName(folder) {
		return true
	}

	normalizedParsed := Normalize(parsed)
	normalizedFolder := Normalize(folder)

	// A filename carrying release evidence outranks a short container folder.
	hasReleaseEvidence := evidence.ParsedYear > 0 || evidence.ParsedResolution != ""
	if hasReleaseEvidence && len(strings.Fields(folder)) <= 1 {
		return true
	}

	// The filename contains the folder name as a word sequence, so the folder is
	// a prefix of the release name rather than the full title.
	if containsWordSequence(normalizedParsed, normalizedFolder) &&
		len([]rune(normalizedParsed)) > len([]rune(normalizedFolder))+2 {
		return true
	}

	// A short folder beside a materially longer filename is a container. Real
	// libraries are full of these: "Franchises/DC/Part 3 - The Dark Knight
	// Rises.mkv", "Actors/Al Pacino/Serpico.mkv". A one-word folder cannot
	// describe a four-word film title, whereas a four-word folder usually IS the
	// title, which is why the comparison is asymmetric.
	folderWords := len(strings.Fields(normalizedFolder))
	parsedWords := len(strings.Fields(normalizedParsed))
	if folderWords <= 1 && parsedWords >= 3 {
		return true
	}

	// A very short folder (initials, an acronym) is a grouping label rather than
	// a title: "DC", "MCU", "HP". A real single-word work name is normally longer.
	if len([]rune(normalizedFolder)) <= 3 && len([]rune(normalizedParsed)) > len([]rune(normalizedFolder))+3 {
		return true
	}

	return false
}

// containerFolderNames are folder names that describe a container, a browse
// grouping, or a category rather than a work. They must never name a work.
var containerFolderNames = map[string]struct{}{
	// Arabic browse groupings seen in real libraries.
	"اعمال": {}, "أعمال": {}, "افلام": {}, "أفلام": {}, "مسلسلات": {}, "مسلسل": {},
	"مكتبه": {}, "مكتبة": {}, "القسم": {}, "قسم": {}, "الكل": {}, "متنوع": {},
	"افلام ومسلسلات": {}, "اخرى": {}, "أخرى": {}, "اخري": {}, "أخري": {},
	"جديد": {}, "قديمة": {}, "قديم": {}, "متنوعة": {}, "منوعة": {}, "متنوعه": {},
	"franchises": {}, "collection": {}, "collections": {}, "boxset": {}, "box set": {},
	"movies": {}, "films": {}, "series": {}, "tv": {}, "shows": {},
	"library": {}, "media": {}, "video": {}, "videos": {}, "unsorted": {},
	"misc": {}, "other": {}, "others": {}, "extra": {}, "extras": {},
	"featurettes": {}, "bonus": {}, "sample": {}, "samples": {},
}

// IsContainerFolderName reports whether a folder name is a container (a browse
// grouping, a category or a franchise label) rather than a work title.
//
// It is exported because the catalogue repair needs the same rule the resolver
// uses when choosing a title. Two implementations of "what is a container" would
// eventually disagree, and then a repaired row and a freshly ingested row would
// differ for the same folder.
func IsContainerFolderName(folder string) bool { return isContainerFolderName(folder) }

// IsContainerFolderNameForTitle reports whether a folder is a container in the
// wider sense the catalogue repair needs: either the name itself is a container,
// or it begins with one and carries a qualifier ("مسلسلات تركية").
//
// It deliberately mirrors scanner.IsContainerFolderForTitle. The two packages
// cannot import each other, so the rule is written twice and a test asserts the
// same inputs classify the same way in both.

// IsStructuralTitle reports whether a title is only a structural keyword
// ("Season", "Episode", "Part") rather than a name.
func IsStructuralTitle(title string) bool { return isStructuralWord(title) }

// isContainerFolderName reports whether a folder name is a container rather than
// a work title.
func isContainerFolderName(folder string) bool {
	normalized := Normalize(folder)
	if normalized == "" {
		return true
	}
	if _, exists := containerFolderNames[normalized]; exists {
		return true
	}
	// A folder made only of structural words is a container too.
	if isStructuralWord(folder) {
		return true
	}
	return false
}

// unresolved is the terminal path for a file with no usable title.
func (r *Resolver) unresolved(resolution Resolution, evidence Evidence, code, reason string) Resolution {
	resolution.Decision = DecisionReview
	resolution.State = StateUnresolved
	resolution.ReasonCode = code
	resolution.Reason = reason
	resolution.ResolverConfidence = 0
	// Retain candidates even here so the review queue can suggest alternatives.
	if len(evidence.Path) > 0 {
		resolution.Candidates = nil
	}
	return resolution
}

// fillFromWork copies the entity's decisions onto the resolution, so the file
// inherits the work's media type rather than asserting its own.
func (r *Resolver) fillFromWork(resolution *Resolution, work Work) {
	if work.MediaType != "" && work.MediaType != "unknown" {
		resolution.MediaType = MediaType(work.MediaType)
	}
	if work.CategorySlug != "" {
		resolution.Category = work.CategorySlug
	}
	if work.OriginTag != "" {
		resolution.Origin = work.OriginTag
	}
}

// normalizeScore maps a raw evidence score to 0..1 for reporting.
func normalizeScore(score float64) float64 {
	if score <= 0 {
		return 0
	}
	// A perfect match accumulates roughly these weights; clamp at 1.
	const fullMatch = WeightExactTitle + WeightFolderTitle + WeightAliasMatch +
		WeightSeasonCompatible + WeightYearCompatible + WeightCategoryCompat + WeightEpisodeRelation
	value := score / fullMatch
	if value > 1 {
		return 1
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
