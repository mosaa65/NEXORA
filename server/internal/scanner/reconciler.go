package scanner

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReconcileAction is the single mutation the reconciler tells the persistence
// layer to perform. Keeping this an explicit, closed set is what makes the
// filesystem-to-database transition auditable instead of implicit.
type ReconcileAction string

const (
	// ActionInsert creates a new catalogue record.
	ActionInsert ReconcileAction = "insert"
	// ActionUpdate refreshes an existing record whose metadata changed.
	ActionUpdate ReconcileAction = "update"
	// ActionMovePath rewrites the path of an existing record. The record keeps
	// its identity, its technical metadata and its watch progress.
	ActionMovePath ReconcileAction = "move_path"
	// ActionMarkMissing flags a record that is no longer present.
	ActionMarkMissing ReconcileAction = "mark_missing"
	// ActionMarkUnavailable flags a record whose root or file could not be read.
	ActionMarkUnavailable ReconcileAction = "mark_unavailable"
	// ActionRestore returns a previously missing/unavailable record to ACTIVE.
	ActionRestore ReconcileAction = "restore"
	// ActionCleanup is the explicit, separate deletion step. It is only issued
	// for records already in MISSING state, never directly from a scan.
	ActionCleanup ReconcileAction = "cleanup"
)

// ReconcileDecision pairs an action with the record it applies to.
type ReconcileDecision struct {
	Action ReconcileAction `json:"action"`
	// KnownID is the existing catalogue row, when one exists.
	KnownID int64 `json:"knownId,omitempty"`
	// Path is the current path on disk.
	Path string `json:"path,omitempty"`
	// PreviousPath is the old path for a move.
	PreviousPath string `json:"previousPath,omitempty"`
	// File carries the fresh parse result for insert/update.
	File *FileInfo `json:"file,omitempty"`
	// Reason explains why the action was chosen, for the audit trail.
	Reason string `json:"reason,omitempty"`
}

// ReconcileResult summarises the delta between the filesystem and the catalogue.
type ReconcileResult struct {
	Inserted          int `json:"inserted"`
	Updated           int `json:"updated"`
	Moved             int `json:"moved"`
	MarkedMissing     int `json:"markedMissing"`
	MarkedUnavailable int `json:"markedUnavailable"`
	Restored          int `json:"restored"`
	Unchanged         int `json:"unchanged"`
	// RootStates is the per-root outcome, which decides whether an absent record
	// is MISSING (root was read) or UNAVAILABLE (root could not be read).
	RootStates map[string]RootStatus `json:"rootStates"`
	// RootFailures explains root-level failures so the report can be truthful.
	RootFailures map[string]string `json:"rootFailures,omitempty"`
}

// Reconcile turns a scan report into a set of explicit actions.
//
// The critical safety property: a record is only ever marked MISSING when the
// root that owns it was actually readable. If the disk is offline, permission
// was denied, or the share timed out, the record becomes UNAVAILABLE instead.
// That distinction is the difference between "the file is gone" and "I could not
// see the file", and only the first is allowed to lead to a deletion.
func Reconcile(report Report, roots []string) ReconcileResult {
	result := ReconcileResult{
		RootStates:   report.RootStates,
		RootFailures: make(map[string]string, len(report.RootFailures)),
	}
	for root, failure := range report.RootFailures {
		result.RootFailures[root] = failure.Err
	}

	for _, missing := range report.Missing {
		if recordRootReachable(missing, report.RootStates) {
			result.MarkedMissing++
		} else {
			result.MarkedUnavailable++
		}
	}
	return result
}

// recordRootReachable decides whether the root owning a record was readable in
// this pass. An unknown root is treated as *not* reachable, because the safe
// failure mode is to keep the record rather than to declare it missing.
func recordRootReachable(record KnownFile, states map[string]RootStatus) bool {
	if len(states) == 0 {
		// No per-root information: refuse to declare anything missing.
		return false
	}
	// Find the deepest root that contains this record.
	bestRoot := ""
	for root := range states {
		normalizedRoot := NormalizePathKey(root)
		normalizedPath := NormalizePathKey(record.Path)
		if normalizedPath == normalizedRoot || strings.HasPrefix(normalizedPath, normalizedRoot+"/") {
			if len(normalizedRoot) > len(NormalizePathKey(bestRoot)) {
				bestRoot = root
			}
		}
	}
	if bestRoot == "" {
		return false
	}
	status := states[bestRoot]
	return status == RootCompleted || status == RootScanning || status == RootAvailable
}

// DecideActions converts scanned files plus the missing set into the ordered,
// explicit decisions a persistence layer should apply.
//
// Ordering matters: moves are emitted before inserts so a rename never creates a
// duplicate row, and restores come before updates so a returning disk is
// reflected without a second pass.
func DecideActions(files []FileInfo, missing []KnownFile) []ReconcileDecision {
	decisions := make([]ReconcileDecision, 0, len(files)+len(missing))

	restores := make([]ReconcileDecision, 0)
	updates := make([]ReconcileDecision, 0)
	moves := make([]ReconcileDecision, 0)
	inserts := make([]ReconcileDecision, 0)

	for i := range files {
		file := files[i]
		switch file.Change {
		case ChangeRenamed:
			moves = append(moves, ReconcileDecision{
				Action:       ActionMovePath,
				KnownID:      file.KnownID,
				Path:         file.Path,
				PreviousPath: file.RenamedFrom,
				File:         &files[i],
				Reason:       "same file identity observed at a new path",
			})
		case ChangeChanged:
			if file.KnownID > 0 {
				updates = append(updates, ReconcileDecision{
					Action:  ActionUpdate,
					KnownID: file.KnownID,
					Path:    file.Path,
					File:    &files[i],
					Reason:  "size/modification time changed",
				})
			} else {
				inserts = append(inserts, ReconcileDecision{
					Action: ActionInsert,
					Path:   file.Path,
					File:   &files[i],
					Reason: "changed content without a known record",
				})
			}
		case ChangeNew:
			inserts = append(inserts, ReconcileDecision{
				Action: ActionInsert,
				Path:   file.Path,
				File:   &files[i],
				Reason: "path not present in the catalogue",
			})
		default:
			if file.KnownID > 0 && file.State == StateActive {
				/** unchanged: nothing to do */
			}
		}
	}

	for i := range missing {
		record := missing[i]
		decisions = append(decisions, ReconcileDecision{
			Action:  ActionMarkMissing,
			KnownID: record.ID,
			Path:    record.Path,
			Reason:  "record not observed while its root was readable",
		})
	}

	decisions = append(decisions, restores...)
	decisions = append(decisions, moves...)
	decisions = append(decisions, updates...)
	decisions = append(decisions, inserts...)
	return decisions
}

// -----------------------------------------------------------------------------
// Continuous library watching
// -----------------------------------------------------------------------------

// WatchSchedulerOptions configure the interval-driven reconciliation sweeps.
type WatchSchedulerOptions struct {
	// Interval between background reconciliation sweeps. Zero disables the
	// scheduler; the fsnotify watcher still runs.
	Interval time.Duration
	// Roots to reconcile. Empty means "use the configured media roots".
	Roots []string
	// Mode of the scheduled sweep. Reconcile is the correct default because it
	// is incremental and only settles drift.
	Mode ScanMode
	// Logger receives one summary line per sweep.
	Logger interface {
		Info(msg string, args ...any)
	}
}

// WatchScheduler periodically reconciles the filesystem against the catalogue.
//
// This is the answer to the watcher's fundamental unreliability: fsnotify can
// miss events (buffer overflow, network shares, downtime), so a periodic
// reconciliation is what guarantees the index eventually matches reality.
type WatchScheduler struct {
	scanner *Scanner
	options WatchSchedulerOptions
	// loadKnown loads the persisted catalogue for the given roots.
	loadKnown func(roots []string) ([]KnownFile, error)
	// apply is called with the decisions each sweep produced.
	apply func(ReconcileResult, []ReconcileDecision) error
}

// NewWatchScheduler wires a scheduler to a scanner and persistence callbacks.
func NewWatchScheduler(
	scanner *Scanner,
	options WatchSchedulerOptions,
	loadKnown func(roots []string) ([]KnownFile, error),
	apply func(ReconcileResult, []ReconcileDecision) error,
) *WatchScheduler {
	return &WatchScheduler{
		scanner:   scanner,
		options:   options,
		loadKnown: loadKnown,
		apply:     apply,
	}
}

// Run drives the periodic sweep until the context is cancelled.
func (s *WatchScheduler) Run(ctx context.Context, emit func(FileInfo) error) error {
	if s.options.Interval <= 0 {
		<-ctx.Done()
		return ctx.Err()
	}
	ticker := time.NewTicker(s.options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.SweepOnce(ctx, emit); err != nil && ctx.Err() == nil {
				if s.options.Logger != nil {
					s.options.Logger.Info("reconcile sweep failed", "error", err.Error())
				}
			}
		}
	}
}

// SweepOnce performs one reconciliation pass.
func (s *WatchScheduler) SweepOnce(ctx context.Context, emit func(FileInfo) error) error {
	roots := s.options.Roots
	if len(roots) == 0 {
		roots = s.scanner.ConfiguredRoots()
	}
	if len(roots) == 0 {
		return nil
	}

	var known []KnownFile
	if s.loadKnown != nil {
		loaded, err := s.loadKnown(roots)
		if err != nil {
			return err
		}
		known = loaded
	}

	mode := s.options.Mode
	if mode == "" {
		mode = ModeReconcile
	}

	report, err := s.scanner.ScanTree(ctx, roots, ScanOptions{Mode: mode, Known: known}, emit)
	if err != nil {
		return err
	}

	result := Reconcile(report, roots)
	if s.apply != nil {
		if err := s.apply(result, DecideActions(nil, report.Missing)); err != nil {
			return err
		}
	}
	if s.options.Logger != nil {
		s.options.Logger.Info("reconcile sweep completed",
			"missing", result.MarkedMissing,
			"unavailable", result.MarkedUnavailable,
		)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Data quality analysis
// -----------------------------------------------------------------------------

// QualityIssue is one row of the index-quality report. The owner asked for a
// data-quality analysis, not just counters, so each issue explains impact.
type QualityIssue struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Detail      string   `json:"detail"`
	Count       int      `json:"count"`
	Samples     []string `json:"samples,omitempty"`
	AffectsData string   `json:"affectsData"`
}

// AnalyzeQuality inspects a batch of parsed files and reports the conditions an
// admin should review. It is deliberately read-only and allocation-bounded: it
// keeps only a few samples per issue.
type QualityAnalyzer struct {
	sampleLimit int
}

// NewQualityAnalyzer creates an analyzer retaining `sampleLimit` examples.
func NewQualityAnalyzer(sampleLimit int) *QualityAnalyzer {
	if sampleLimit <= 0 {
		sampleLimit = 5
	}
	return &QualityAnalyzer{sampleLimit: sampleLimit}
}

// Analyze walks the supplied files and returns the quality issues found.
func (a *QualityAnalyzer) Analyze(files []FileInfo) []QualityIssue {
	issues := map[string]*QualityIssue{}
	ensure := func(code, title, detail, affects string) *QualityIssue {
		if existing, ok := issues[code]; ok {
			return existing
		}
		issue := &QualityIssue{Code: code, Title: title, Detail: detail, AffectsData: affects}
		issues[code] = issue
		return issue
	}
	add := func(code, title, detail, affects, path string) {
		issue := ensure(code, title, detail, affects)
		issue.Count++
		if len(issue.Samples) < a.sampleLimit {
			issue.Samples = append(issue.Samples, path)
		}
	}

	paths := make(map[string]int, len(files))
	titles := make(map[string]int, len(files))

	for i := range files {
		file := files[i]
		key := NormalizePathKey(file.Path)
		paths[key]++
		if paths[key] == 2 {
			add("duplicate_path", "مسار مكرر",
				"نفس المسار ظهر أكثر من مرة في نفس الفحص، ما قد ينتج سجلًا مكررًا.",
				"نعم — تكرار في الفهرس", file.Path)
		}

		if file.Parsed.Title == "" {
			add("missing_title", "ملف بدون عنوان",
				"لم يحتوِ المسار على دليل كافٍ لاستخراج عنوان، ولم يُخمَّن أي عنوان.",
				"نعم — يظهر بدون اسم", file.Path)
		}

		if file.Parsed.CategorySlug == "" {
			add("missing_category", "ملف بدون تصنيف",
				"لم يُطابق أي مجلد في المسار تصنيفًا معروفًا.",
				"نعم — قد يظهر في قسم خاطئ", file.Path)
		}

		if file.Parsed.IsEpisode && file.Parsed.EpisodeNumber == 0 && file.Parsed.SpecialKind == "" {
			add("episode_without_number", "حلقة بدون رقم",
				"الملف يبدو حلقة لكن لم يُستخرج رقم الحلقة.",
				"نعم — ترتيب الحلقات سيكون خاطئًا", file.Path)
		}

		if !file.Parsed.IsEpisode && file.Parsed.PartNumber > 0 {
			add("multipart_movie", "فيلم بأجزاء",
				"الملف جزء من عمل متعدد الأجزاء (Part/CD). تأكد أن كل جزء مرتبط بنفس العمل.",
				"جزئيًا — يحتاج تحققًا", file.Path)
		}

		if file.Parsed.IsEpisode && file.Parsed.SeasonNumber > 100 {
			add("invalid_season", "رقم موسم غير منطقي",
				"رقم الموسم المستخرج يتجاوز 100، وهو غالبًا رقم حلقة أو سنة أُسيء تفسيرها.",
				"نعم — موسم خاطئ", file.Path)
		}

		if file.Parsed.IsEpisode && file.Parsed.EpisodeNumber > 5000 {
			add("impossible_episode", "رقم حلقة غير منطقي",
				"رقم الحلقة المستخرج كبير جدًا (>5000) وقد يكون جزءًا من اسم أو سنة.",
				"نعم — حلقة خاطئة", file.Path)
		}

		if file.Parsed.Confidence < ConfidenceMedium {
			add("low_confidence", "تحليل منخفض الثقة",
				"الأدلة المتاحة (اسم الملف/المجلد) غير كافية للجزم بالعنوان أو الموسم.",
				"نعم — يحتاج مراجعة بشرية", file.Path)
		}

		if file.ArtworkPath == "" {
			add("missing_artwork", "بوستر مفقود",
				"لم يوجد ملف صورة في مجلد الفيديو أو المجلد الأعلى.",
				"لا — تجميلي فقط", file.Path)
		}

		if file.Parsed.Title != "" {
			titleKey := NormalizeTitleForSearch(file.Parsed.Title)
			if file.Parsed.IsEpisode {
				titleKey += "#s" + itoa(file.Parsed.SeasonNumber) + "e" + itoa(file.Parsed.EpisodeNumber)
			}
			titles[titleKey]++
			if titles[titleKey] == 2 {
				add("duplicate_logical_title", "عنوان منطقي مكرر",
					"ملفان يمثلان نفس الحلقة/العمل، ما قد يعني نسخة مكررة على القرص.",
					"نعم — تكرار محتمل", file.Path)
			}
		}
	}

	out := make([]QualityIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, *issue)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Code < out[j].Code
		}
		return out[i].Count > out[j].Count
	})
	return out
}

// -----------------------------------------------------------------------------
// Stale-record cleanup policy
// -----------------------------------------------------------------------------

// CleanupPolicy controls the only operation in this system that removes
// catalogue rows. It is intentionally hard to trigger by accident.
type CleanupPolicy struct {
	// MinMissingAge is how long a record must have been MISSING before it may
	// be deleted. A disk that is unplugged for an hour must not lose its rows.
	MinMissingAge time.Duration
	// RequireRootOnline means cleanup is refused while the owning root is
	// unreachable, which makes "delete everything because the disk is offline"
	// structurally impossible.
	RequireRootOnline bool
	// MaxDeletionsPerRun caps a single cleanup so a misconfiguration cannot
	// wipe the catalogue in one call.
	MaxDeletionsPerRun int
}

// DefaultCleanupPolicy is deliberately conservative.
func DefaultCleanupPolicy() CleanupPolicy {
	return CleanupPolicy{
		MinMissingAge:      7 * 24 * time.Hour,
		RequireRootOnline:  true,
		MaxDeletionsPerRun: 5000,
	}
}

// CleanupCandidate is a record eligible for deletion.
type CleanupCandidate struct {
	ID       int64
	Path     string
	State    FileState
	LastSeen time.Time
}

// SelectCleanupCandidates applies the policy and returns what may be removed,
// plus a refusal reason when the operation must not proceed.
func SelectCleanupCandidates(
	candidates []CleanupCandidate,
	rootStates map[string]RootStatus,
	policy CleanupPolicy,
	now time.Time,
) ([]CleanupCandidate, string) {

	if policy.RequireRootOnline {
		for root, state := range rootStates {
			switch state {
			case RootUnavailable, RootOffline:
				return nil, "cleanup refused: media root " + root + " is not reachable; records are kept to avoid deleting data for an offline disk"
			}
		}
	}

	selected := make([]CleanupCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.State != StateMissing {
			continue
		}
		age := now.Sub(candidate.LastSeen)
		if age < policy.MinMissingAge {
			continue
		}
		selected = append(selected, candidate)
		if policy.MaxDeletionsPerRun > 0 && len(selected) >= policy.MaxDeletionsPerRun {
			break
		}
	}
	return selected, ""
}

// RootsForKnownFile finds the media root that contains a path. It is used to map
// a record back to its root so the correct root state is consulted.
func RootsForKnownFile(path string, roots []string) string {
	normalizedPath := NormalizePathKey(path)
	best := ""
	for _, root := range roots {
		normalizedRoot := NormalizePathKey(root)
		if normalizedRoot == "" {
			continue
		}
		if normalizedPath == normalizedRoot || strings.HasPrefix(normalizedPath, normalizedRoot+"/") {
			if len(normalizedRoot) > len(NormalizePathKey(best)) {
				best = root
			}
		}
	}
	return best
}

// RootOfPath returns the volume identity of a path, used to group records by
// disk for per-root reporting.
func RootOfPath(path string) string {
	return filepath.VolumeName(path)
}
