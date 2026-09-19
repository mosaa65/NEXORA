package scanner

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ScanOptions are the per-run knobs for a scan session. They are separate from
// Scanner.Options because a caller may scan the same Scanner with different
// intent (full vs incremental vs recovery).
type ScanOptions struct {
	// Mode selects what the run is for.
	Mode ScanMode
	// Known is the persisted catalogue used for change detection.
	Known []KnownFile
	// EmitUnchanged forwards unchanged files to the callback as well. The
	// default is false: incremental scans must not re-write the whole library.
	EmitUnchanged bool
	// OnProgress, when set, is called periodically with a real progress
	// snapshot. It is never called per file.
	OnProgress func(Progress)
	// ProgressInterval overrides how often OnProgress fires.
	ProgressInterval time.Duration
	// Logger receives one aggregated line per interval instead of one per file.
	Logger *slog.Logger
	// Control, when set, exposes cooperative pause/resume/cancel and live
	// per-worker state for the duration of the run. The scan attaches its
	// internal controller to it so the API can drive the run while it is going.
	Control *ScanControlHandle
}

// ScanControlHandle is the API-facing handle for a running scan.
//
// The scanner owns the controller; the caller owns the handle. That direction
// keeps the pause logic inside the pipeline (where the safety rules live) while
// still letting an HTTP handler reach it.
type ScanControlHandle struct {
	mu      sync.Mutex
	control *scanControl
}

// attach binds the scanner's controller to this handle.
func (h *ScanControlHandle) attach(control *scanControl) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.control = control
}

// Snapshot returns the current control state and worker list, or a zero value
// when the scan has not started yet.
func (h *ScanControlHandle) Snapshot() ControlSnapshot {
	h.mu.Lock()
	control := h.control
	h.mu.Unlock()
	if control == nil {
		return ControlSnapshot{State: ControlRunning}
	}
	return control.Snapshot()
}

// Pause requests a cooperative pause and reports the resulting state.
func (h *ScanControlHandle) Pause() ScanControlState {
	h.mu.Lock()
	control := h.control
	h.mu.Unlock()
	if control == nil {
		return ControlRunning
	}
	return control.Pause()
}

// Resume lifts a pause and reports the resulting state.
func (h *ScanControlHandle) Resume() ScanControlState {
	h.mu.Lock()
	control := h.control
	h.mu.Unlock()
	if control == nil {
		return ControlRunning
	}
	return control.Resume()
}

// Cancel marks the scan cancelled and releases paused workers.
func (h *ScanControlHandle) Cancel() {
	h.mu.Lock()
	control := h.control
	h.mu.Unlock()
	if control == nil {
		return
	}
	control.Cancel()
}

// ScanMode declares the intent of a scan, which changes how the result is
// interpreted.
type ScanMode string

const (
	// ModeFull builds a complete inventory and re-parses everything.
	ModeFull ScanMode = "full"
	// ModeIncremental emits only new/changed/renamed files.
	ModeIncremental ScanMode = "incremental"
	// ModeReconcile is an incremental scan whose primary output is the delta
	// between the filesystem and the catalogue.
	ModeReconcile ScanMode = "reconcile"
	// ModeRecovery is used when a previously unavailable root returns. It
	// re-checks that root only, which is why a re-plugged disk does not force a
	// library-wide rescan.
	ModeRecovery ScanMode = "recovery"
)

// Report is the full structured outcome of a scan session.
type Report struct {
	Progress   Progress              `json:"progress"`
	RootStates map[string]RootStatus `json:"rootStates"`
	// Missing lists indexed paths under the scanned roots that were not seen.
	// They are candidates for MISSING, never automatic deletions.
	Missing []KnownFile `json:"-"`
	// RootFailures explains which roots failed and why.
	RootFailures map[string]ScanError `json:"rootFailures,omitempty"`
	// ErrorSamples are bounded examples behind the aggregated counters.
	ErrorSamples []ScanError `json:"errorSamples,omitempty"`
}

// ScanTree performs a scan session and returns its report. Files are streamed to
// `emit`; nothing is accumulated in memory beyond bounded queues and the
// identity index of already-known files.
func (s *Scanner) ScanTree(ctx context.Context, roots []string, options ScanOptions, emit func(FileInfo) error) (Report, error) {
	if emit == nil {
		return Report{}, errors.New("file callback is required")
	}
	cleanRoots, err := normalizeRoots(roots)
	if err != nil {
		return Report{}, err
	}
	return s.runPipeline(ctx, cleanRoots, options, emit)
}

// runPipeline is the heart of the scanner:
//
//	discovery goroutines -> candidates (bounded) -> metadata workers -> results (bounded) -> emit (single)
//
// Every stage is bounded. Discovery feeds a fixed-size channel, so a fast
// walker cannot outrun the worker pool and grow memory without limit. The
// single-threaded emit keeps the persistence contract simple and safe.
func (s *Scanner) runPipeline(ctx context.Context, roots []string, options ScanOptions,
	emit func(FileInfo) error) (Report, error) {

	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tracker := NewProgressTracker(newScanID(), NewErrorSink(s.options.SampleErrorLimit))
	index := newIdentityIndex(options.Known)
	artwork := newArtworkResolver()
	// The group collector accumulates per-directory structure so a folder of
	// bare episode numbers can be recognised as one season. It is bounded: see
	// groupCollector's cap and eviction.
	groups := newGroupCollector(s.options.GroupTrackingCap)

	candidates := make(chan Visit, s.queueSize)
	results := make(chan FileInfo, s.queueSize)

	d := newDiscovery(s.options.Discovery, candidates)

	rootWorkers := s.options.RootWorkers
	if rootWorkers <= 0 {
		rootWorkers = len(roots)
	}
	if rootWorkers > len(roots) {
		rootWorkers = len(roots)
	}
	if rootWorkers < 1 {
		rootWorkers = 1
	}

	// Cooperative pause, resume and cancellation, plus per-worker visibility.
	// The control is shared by every worker so a pause applies to the whole
	// pipeline rather than to one goroutine.
	control := newScanControl(s.workers + rootWorkers)
	if options.Control != nil {
		options.Control.attach(control)
	}

	var walkers sync.WaitGroup
	rootQueue := make(chan string, len(roots))
	for _, root := range roots {
		rootQueue <- root
	}
	close(rootQueue)

	for i := 0; i < rootWorkers; i++ {
		workerID := i
		walkers.Add(1)
		go func() {
			defer walkers.Done()
			defer control.markWorker(workerID, RoleIdle, "")
			for root := range rootQueue {
				// Pause is checked before starting a new root, never mid-directory.
				if !control.wait(workCtx) {
					return
				}
				if workCtx.Err() != nil {
					return
				}
				control.markWorker(workerID, RoleDiscovery, root)
				d.walkRoot(workCtx, root, tracker)
			}
		}()
	}
	go func() {
		walkers.Wait()
		close(candidates)
	}()

	var workers sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		workerID := rootWorkers + i
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer control.markWorker(workerID, RoleIdle, "")
			for visit := range candidates {
				// The pause gate sits BEFORE an item is taken, so a paused scan never
				// abandons a half-processed file.
				if !control.wait(workCtx) {
					return
				}
				if workCtx.Err() != nil {
					return
				}
				control.markWorker(workerID, RoleMetadata, visit.Path)
				file, ok := s.processCandidate(visit, index, artwork, groups, tracker, options)
				if !ok {
					continue
				}
				control.markWorker(workerID, RolePersistence, file.Path)
				select {
				case results <- file:
				case <-workCtx.Done():
					return
				}
			}
		}()
	}
	go func() {
		workers.Wait()
		close(results)
	}()

	progressStop := make(chan struct{})
	if options.OnProgress != nil || options.Logger != nil {
		interval := options.ProgressInterval
		if interval <= 0 {
			interval = s.options.ProgressInterval
		}
		if interval <= 0 {
			interval = 5 * time.Second
		}
		go s.reportProgress(workCtx, progressStop, interval, tracker, options)
	}
	defer close(progressStop)

	// Emit runs on this goroutine only, so the caller needs no locking.
	var emitErr error
	for file := range results {
		if emitErr != nil {
			// Keep draining so the workers can exit, but stop persisting.
			continue
		}
		if err := emit(file); err != nil {
			emitErr = err
			cancel()
		}
	}
	if emitErr != nil {
		tracker.Finish(StatusFailed)
		return s.buildReport(tracker, index, roots), emitErr
	}

	if err := ctx.Err(); err != nil {
		tracker.Finish(StatusCancelled)
		return s.buildReport(tracker, index, roots), err
	}

	tracker.Finish(classifyOverallStatus(tracker))
	return s.buildReport(tracker, index, roots), nil
}

// reportProgress emits one aggregated snapshot per interval. A million-file
// scan must never produce a million log lines.
func (s *Scanner) reportProgress(ctx context.Context, stop <-chan struct{}, interval time.Duration,
	tracker *ProgressTracker, options ScanOptions) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			snapshot := tracker.Snapshot()
			if options.OnProgress != nil {
				options.OnProgress(snapshot)
			}
			if options.Logger != nil {
				options.Logger.Info("scan progress",
					slog.String("scan_id", snapshot.ScanID),
					slog.Int64("directories", snapshot.Directories),
					slog.Int64("files_seen", snapshot.FilesSeen),
					slog.Int64("accepted", snapshot.Accepted),
					slog.Int64("new", snapshot.NewFiles),
					slog.Int64("changed", snapshot.ModifiedFiles),
					slog.Int64("unchanged", snapshot.UnchangedFiles),
					slog.Int("error_kinds", len(snapshot.Errors)),
					slog.Float64("files_per_sec", snapshot.Throughput.FilesPerSecond),
				)
			}
		}
	}
}

// classifyOverallStatus turns per-root outcomes into the session status.
func classifyOverallStatus(tracker *ProgressTracker) ScanStatus {
	states := tracker.RootStatuses()
	if len(states) == 0 {
		return StatusCompleted
	}
	failed := 0
	unavailable := 0
	for _, state := range states {
		switch state {
		case RootError:
			failed++
		case RootUnavailable, RootOffline:
			unavailable++
		}
	}
	if failed == len(states) {
		return StatusFailed
	}
	if failed > 0 || unavailable > 0 {
		return StatusPartial
	}
	return StatusCompleted
}

// processCandidate turns one discovered file into a FileInfo, applying the
// change decision. It returns false when the file should not be emitted.
func (s *Scanner) processCandidate(visit Visit, index *identityIndex, artwork *artworkResolver,
	groups *groupCollector, tracker *ProgressTracker, options ScanOptions) (FileInfo, bool) {

	counts := tracker.Counts()
	counts.videoCandidates.Add(1)

	info, err := visit.Entry.Info()
	if err != nil {
		// A file that vanished between listing and stat is normal on a busy
		// share. Record it and move on; never fail the scan.
		tracker.Errors().AddError(err, visit.Path, visit.Root, "stat")
		counts.filesystemErrors.Add(1)
		return FileInfo{}, false
	}
	if info.IsDir() {
		return FileInfo{}, false
	}
	if !s.IsVideoFile(visit.Path) {
		counts.rejected.Add(1)
		return FileInfo{}, false
	}
	counts.accepted.Add(1)

	file := s.fileInfo(visit.Path, info, visit.Root, artwork)

	// Record this file's contribution to its directory structure, then publish
	// the accumulated summary onto the file. Publishing the whole-directory
	// summary (rather than only the files seen so far) is what makes batch
	// resolution possible: every file carries the neighbourhood's evidence.
	groups.Observe(visit.Path, file.Parsed, file.Context.SeasonNumber > 0)
	groups.publish(&file)

	change := index.classify(visit.Path, file.Fingerprint, true)
	file.Change = change.Kind
	if change.Known != nil {
		file.KnownID = change.Known.ID
	}
	if change.RenamedFrom != "" {
		file.RenamedFrom = change.RenamedFrom
	}

	switch change.Kind {
	case ChangeNew:
		counts.newFiles.Add(1)
		file.State = StatePendingScan
	case ChangeChanged:
		counts.modifiedFiles.Add(1)
		file.State = StateChanged
	case ChangeRenamed:
		counts.renamedFiles.Add(1)
		file.State = StateRenamed
	default:
		counts.unchangedFiles.Add(1)
		file.State = StateActive
	}

	counts.totalBytes.Add(file.Size)
	if file.ArtworkPath != "" {
		counts.artworkFound.Add(1)
	} else {
		counts.artworkMissing.Add(1)
	}
	if file.Parsed.Confidence < ConfidenceMedium {
		counts.lowConfidence.Add(1)
	}

	// Incremental runs skip unchanged files entirely: no re-parse downstream,
	// no database write, no search re-index.
	if change.Kind == ChangeUnchanged && !options.EmitUnchanged && options.Mode != ModeFull {
		counts.skippedBytes.Add(file.Size)
		return FileInfo{}, false
	}
	return file, true
}

// buildReport assembles the structured report including reconciliation inputs
// and the actionable problem list.
func (s *Scanner) buildReport(tracker *ProgressTracker, index *identityIndex, roots []string) Report {
	missing := index.missing(roots)
	if len(missing) > 0 {
		tracker.AddProblem(Problem{
			Title:   "سجلات مفقودة من الأقراص",
			Detail:  "سجلات موجودة في الفهرس ولم تُشاهد أثناء الفحص. تُصنَّف MISSING ولا تُحذف تلقائيًا لأن القرص قد يكون مفصولًا أو القراءة فشلت مؤقتًا.",
			Count:   int64(len(missing)),
			Affects: "نعم — تحتاج مراجعة قبل أي تنظيف",
			Urgency: "تأكد من توصيل كل الأقراص ثم شغّل عملية تنظيف صريحة",
			Code:    string(StateMissing),
		})
	}

	progress := tracker.Snapshot()
	if progress.LowConfidence > 0 {
		tracker.AddProblem(Problem{
			Title:   "عناصر بثقة تحليل منخفضة",
			Detail:  "أسماء ملفات أو مجلدات لا تحمل دليلًا كافيًا على العنوان أو الموسم، فتُركت بدون تخمين.",
			Count:   progress.LowConfidence,
			Affects: "نعم — قد يظهر عنوان أو موسم غير دقيق",
			Urgency: "راجعها من واجهة الإدارة وأصلح أسماء المجلدات",
			Code:    "low_confidence",
		})
	}
	if denied := progress.PermissionErrors; denied > 0 {
		tracker.AddProblem(Problem{
			Title:   "مجلدات غير قابلة للقراءة",
			Detail:  "تعذّر الدخول إلى بعض المجلدات؛ تم تخطيها وإكمال الفحص لباقي المسارات.",
			Count:   denied,
			Affects: "لا — الفهرس سليم لكنه ناقص لهذه المسارات",
			Urgency: "راجع صلاحيات المستخدم على هذه المجلدات",
			Code:    string(ErrPermissionDenied),
		})
	}
	progress = tracker.Snapshot()

	return Report{
		Progress:     progress,
		RootStates:   tracker.RootStatuses(),
		Missing:      missing,
		ErrorSamples: tracker.Errors().Samples(),
	}
}
