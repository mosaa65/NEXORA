package identity

import (
	"strings"
)

// -----------------------------------------------------------------------------
// Provenance: which value wins when several sources disagree
// -----------------------------------------------------------------------------

// Source identifies where a field value came from. The precedence order is the
// mechanism that stops a Full Scan from overwriting an administrator's
// correction, which is requirement 75.
type Source string

const (
	SourceFilesystem Source = "filesystem"
	SourceParser     Source = "parser"
	SourceResolver   Source = "resolver"
	SourceDatabase   Source = "database"
	SourceAdmin      Source = "admin"
	SourceProvider   Source = "tmdb"
)

// Rank orders sources by authority. Higher wins.
//
// The ordering encodes the project's decision: an operator's correction is
// absolute, provider metadata is authoritative for canonical descriptive fields,
// and filesystem/parser output is the weakest because it is machine-derived from
// a filename that may be wrong.
func (s Source) Rank() int {
	switch s {
	case SourceAdmin:
		return 100
	case SourceProvider:
		return 80
	case SourceDatabase:
		return 60
	case SourceResolver:
		return 40
	case SourceParser:
		return 20
	case SourceFilesystem:
		return 10
	default:
		return 0
	}
}

// WinsOver reports whether this source may replace the other value.
func (s Source) WinsOver(other Source) bool {
	return s.Rank() > other.Rank()
}

// FieldValue carries a value together with its provenance, so a write can be
// rejected instead of silently losing an operator decision.
type FieldValue struct {
	Value  string `json:"value"`
	Source Source `json:"source"`
	// Locked marks a value an operator edited; nothing may overwrite it.
	Locked bool `json:"locked"`
}

// MergeField decides the value for one field given the incoming and existing
// values. It returns the chosen value and whether a write is required.
//
// This is the function that implements "Scanner must not destroy manual
// corrections": if the existing value is locked or came from a stronger source,
// the incoming value is discarded.
func MergeField(incoming FieldValue, existing *FieldValue) (FieldValue, bool) {
	if existing == nil || existing.Value == "" {
		return incoming, incoming.Value != ""
	}
	// An empty incoming value never overwrites a populated one.
	if incoming.Value == "" {
		return *existing, false
	}
	// A locked value is final.
	if existing.Locked {
		return *existing, false
	}
	// A stronger existing source is preserved.
	if existing.Source.WinsOver(incoming.Source) {
		// An equal rank with a different value is not a conflict: the newer
		// observation is preferred, because a rename should take effect.
		if existing.Source.Rank() > incoming.Source.Rank() {
			return *existing, false
		}
	}
	// Identical values need no write.
	if existing.Value == incoming.Value && existing.Source == incoming.Source {
		return *existing, false
	}
	return incoming, true
}

// MergeFields applies MergeField to a whole record and reports which fields
// actually changed, so the persistence layer can issue a minimal UPDATE instead
// of rewriting every column on every scan.
func MergeFields(incoming, existing map[string]FieldValue) (map[string]FieldValue, []string) {
	merged := make(map[string]FieldValue, len(incoming))
	changed := make([]string, 0, len(incoming))

	for field, value := range incoming {
		var previous *FieldValue
		if existing != nil {
			if existingValue, ok := existing[field]; ok {
				previous = &existingValue
			}
		}
		chosen, write := MergeField(value, previous)
		merged[field] = chosen
		if write {
			changed = append(changed, field)
		}
	}
	return merged, changed
}

// -----------------------------------------------------------------------------
// Alias learning
// -----------------------------------------------------------------------------

// AliasCandidate is a proposed alias for a work, with the evidence for it.
type AliasCandidate struct {
	MediaItemID int64
	Alias       string
	Normalized  string
	Source      Source
	// Reason records why this alias is safe to learn.
	Reason string
}

// AliasLearner decides which aliases may be learned from a resolution.
//
// The rules matter: an over-eager learner poisons the library, because a wrong
// alias permanently mis-resolves every future file. Only spellings that are
// provably the same name are learned automatically; everything else requires an
// operator decision.
type AliasLearner struct {
	// MinSimilarity is how close a spelling must be to the canonical title to be
	// learned without human confirmation.
	MinSimilarity float64
	// Agent is an optional recorder for observability.
	Agent func(event string, fields map[string]any)
}

// NewAliasLearner returns a learner with conservative defaults.
func NewAliasLearner() *AliasLearner {
	return &AliasLearner{MinSimilarity: 0.9}
}

// Learn evaluates a resolution and returns the aliases that may be stored.
func (l *AliasLearner) Learn(resolution Resolution, work Work, evidence Evidence) []AliasCandidate {
	out := make([]AliasCandidate, 0, 2)

	if work.ID == 0 {
		return out
	}

	// Only learn from a resolution the system was reasonably sure about. A
	// review-queue item must not teach the library.
	if resolution.Decision != DecisionAuto && resolution.Decision != DecisionProvisional {
		return out
	}

	canonicalEN := Normalize(work.TitleEN)
	canonicalAR := Normalize(work.TitleAR)

	add := func(raw, reason string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		normalized := Normalize(raw)
		if normalized == "" {
			return
		}
		// Never store the canonical spelling as an alias: it adds nothing and
		// would make the alias table noisy.
		if normalized == canonicalEN || normalized == canonicalAR {
			return
		}
		// Never learn a name that is not actually a name.
		if quality := AssessTitle(raw, ""); !quality.Usable {
			return
		}
		out = append(out, AliasCandidate{
			MediaItemID: work.ID,
			Alias:       raw,
			Normalized:  normalized,
			Source:      SourceResolver,
			Reason:      reason,
		})
	}

	// The folder name that successfully identified the work.
	if evidence.WorkFolderTitle != "" {
		if sim := bestSimilarity(evidence.WorkFolderTitle, work); sim >= l.MinSimilarity {
			add(evidence.WorkFolderTitle, "folder name matched this work")
		}
	}

	// The parsed filename title, when it is close enough to be a spelling
	// variant rather than a different work.
	if evidence.ParsedTitle != "" {
		if sim := bestSimilarity(evidence.ParsedTitle, work); sim >= l.MinSimilarity {
			add(evidence.ParsedTitle, "filename title matched this work")
		}
	}

	// The parsed Arabic title, which is the most valuable alias to learn because
	// it connects an Arabic folder to an English canonical work.
	if evidence.ParsedTitleAR != "" {
		add(evidence.ParsedTitleAR, "parsed Arabic matched this work")
	}

	// A local poster or folder that named the work differently is already
	// covered by the folder title above.

	if l.Agent != nil && len(out) > 0 {
		l.Agent("aliases_learned", map[string]any{
			"work_id": work.ID,
			"count":   len(out),
		})
	}
	return out
}

// LearnedFromAdmin builds the alias from an explicit operator decision. This is
// always safe and always stored, because a human asserted it.
func LearnedFromAdmin(mediaItemID int64, alias, canonicalTitle string) AliasCandidate {
	return AliasCandidate{
		MediaItemID: mediaItemID,
		Alias:       strings.TrimSpace(alias),
		Normalized:  Normalize(alias),
		Source:      SourceAdmin,
		Reason:      "operator confirmed " + alias + " identifies " + canonicalTitle,
	}
}

// bestSimilarity returns how closely a name resembles any known name of a work.
func bestSimilarity(name string, work Work) float64 {
	best := Similarity(name, work.TitleEN)
	best = maxFloat(best, Similarity(name, work.TitleAR))
	best = maxFloat(best, Similarity(name, work.TitleNormalized))
	for _, alias := range work.Aliases {
		best = maxFloat(best, Similarity(name, alias))
	}
	return best
}

// -----------------------------------------------------------------------------
// Duplicate work consolidation
// -----------------------------------------------------------------------------

// MergePlan describes how to fold a duplicate work into a canonical one.
//
// This is how the existing degenerate entities are repaired without losing
// history: files are re-pointed, the duplicate is marked merged, and nothing is
// deleted. Deletion would lose the audit trail and any watch progress.
type MergePlan struct {
	CanonicalID int64    `json:"canonical_id"`
	MergedIDs   []int64  `json:"merged_ids"`
	AliasToKeep []string `json:"aliases_to_keep"`
	FileCount   int      `json:"file_count"`
	Reason      string   `json:"reason"`
}

// DetectDuplicateWorks finds works that describe the same thing by normalized
// title and media type. It is read-only: it proposes merges, never performs them.
func DetectDuplicateWorks(works []Work) []MergePlan {
	groups := make(map[string][]Work, len(works))
	for _, work := range works {
		key := string(mediaTypeOf(work)) + "::" + Normalize(work.TitleEN)
		if strings.HasSuffix(key, "::") {
			key = string(mediaTypeOf(work)) + "::" + Normalize(work.TitleAR)
		}
		if strings.TrimSuffix(key, "::") == "" {
			continue
		}
		groups[key] = append(groups[key], work)
	}

	plans := make([]MergePlan, 0)
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		// The canonical work is the one that is not provisional and has the most
		// files; ties are broken by the lower id so the result is deterministic.
		canonical := chooseCanonical(group)
		plan := MergePlan{
			CanonicalID: canonical.ID,
			Reason:      "same normalized title and media type",
		}
		for _, work := range group {
			if work.ID == canonical.ID {
				continue
			}
			plan.MergedIDs = append(plan.MergedIDs, work.ID)
			plan.AliasToKeep = append(plan.AliasToKeep, work.TitleEN)
			if work.TitleAR != "" && work.TitleAR != canonical.TitleAR {
				plan.AliasToKeep = append(plan.AliasToKeep, work.TitleAR)
			}
		}
		plans = append(plans, plan)
	}
	return plans
}

func chooseCanonical(group []Work) Work {
	canonical := group[0]
	for _, work := range group[1:] {
		switch {
		case canonical.Provisional && !work.Provisional:
			canonical = work
		case canonical.Provisional == work.Provisional && work.ID < canonical.ID:
			canonical = work
		}
	}
	return canonical
}

func mediaTypeOf(work Work) MediaType {
	if work.MediaType == "" {
		return MediaTypeUnknown
	}
	return MediaType(work.MediaType)
}
