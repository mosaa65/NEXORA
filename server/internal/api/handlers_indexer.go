package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"nexora/server/internal/db"
	"nexora/server/internal/metadata"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

// scanGuard allows exactly one scan at a time and exposes its live progress and
// cancellation. Two concurrent scans would race on the same rows and could
// double-insert during a full scan, so this is a correctness guard rather than a
// rate limit.
//
// The zero value is usable: every method lazily initialises the mutex so a
// Server constructed directly (as tests do) behaves identically to one built by
// NewServer.
type scanGuard struct {
	initOnce sync.Once
	mu       sync.Mutex
	running  bool
	scanID   string
	cancelFn context.CancelFunc
	progress *scanner.Progress
	// control exposes cooperative pause/resume and per-worker state for the
	// running scan. It is nil before a scan starts and after it ends.
	control *scanner.ScanControlHandle
}

func newScanGuard() *scanGuard { return &scanGuard{} }

// ensure makes the zero value safe to use.
func (g *scanGuard) ensure() *scanGuard {
	if g == nil {
		return nil
	}
	return g
}

// tryAcquire claims the scan slot, reporting false when one is already running.
func (g *scanGuard) tryAcquire() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running {
		return false
	}
	g.running = true
	return true
}

// release frees the slot and clears the cached progress.
func (g *scanGuard) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.running = false
	g.scanID = ""
	g.cancelFn = nil
	g.progress = nil
}

// setProgress records the latest snapshot so the status endpoint can report it.
func (g *scanGuard) setProgress(scanID string, cancel context.CancelFunc, progress scanner.Progress) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.scanID = scanID
	g.cancelFn = cancel
	g.progress = &progress
}

// snapshot returns the live progress when a scan is running.
func (g *scanGuard) snapshot() (scanner.Progress, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.running || g.progress == nil {
		return scanner.Progress{}, false
	}
	return *g.progress, true
}

// currentScanID returns the identifier of the running scan.
func (g *scanGuard) currentScanID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.scanID
}

// cancel requests cancellation of the running scan.
//
// Cancel is deliberately a different operation from pause: it ends the scan and
// reports CANCELLED, while pause suspends it and keeps every counter so a resume
// continues from the same point.
func (g *scanGuard) cancel() bool {
	g.mu.Lock()
	control := g.control
	cancelFn := g.cancelFn
	g.mu.Unlock()

	// Release any worker blocked on a pause first, otherwise a paused scan would
	// never observe the cancellation.
	if control != nil {
		control.Cancel()
	}
	if cancelFn == nil {
		return false
	}
	cancelFn()
	return true
}

// setControl records the control handle for the running scan.
func (g *scanGuard) setControl(control *scanner.ScanControlHandle) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.control = control
}

// controlSnapshot reports the pause state and per-worker state.
func (g *scanGuard) controlSnapshot() (scanner.ControlSnapshot, bool) {
	g.mu.Lock()
	control := g.control
	running := g.running
	g.mu.Unlock()
	if control == nil || !running {
		return scanner.ControlSnapshot{}, false
	}
	return control.Snapshot(), true
}

// pause requests a cooperative pause of the running scan.
func (g *scanGuard) pause() (scanner.ScanControlState, bool) {
	g.mu.Lock()
	control := g.control
	running := g.running
	g.mu.Unlock()
	if control == nil || !running {
		return "", false
	}
	return control.Pause(), true
}

// resume lifts a pause on the running scan.
func (g *scanGuard) resume() (scanner.ScanControlState, bool) {
	g.mu.Lock()
	control := g.control
	running := g.running
	g.mu.Unlock()
	if control == nil || !running {
		return "", false
	}
	return control.Resume(), true
}

const (
	ingestBatchSize  = 256
	maxIndexWarnings = 100
)

// indexResult is the response of a scan run. It keeps the fields the existing
// admin UI reads (scanned/imported/inspected/searchSync) and adds the full
// structured report so the UI can show real progress and real problems.
type indexResult struct {
	Roots       []string          `json:"roots"`
	ScanID      string            `json:"scanId"`
	Mode        string            `json:"mode"`
	Status      string            `json:"status"`
	Report      *scanner.Progress `json:"report,omitempty"`
	RootStates  map[string]string `json:"rootStates,omitempty"`
	Missing     int               `json:"missing,omitempty"`
	Unavailable int               `json:"unavailable,omitempty"`

	Scanned       int `json:"scanned"`
	Imported      int `json:"imported"`
	Inserted      int `json:"inserted"`
	Updated       int `json:"updated"`
	Moved         int `json:"moved"`
	Unchanged     int `json:"unchanged"`
	Skipped       int `json:"skipped"`
	Inspected     int `json:"inspected"`
	InspectFailed int `json:"inspectFailed"`

	// Entity-resolution outcome. These are the numbers that say whether the
	// library stayed clean: how many files attached to an existing work, how
	// many needed a human, and how many aliases the run taught the library.
	WorksCreated    int `json:"worksCreated"`
	QueuedForReview int `json:"queuedForReview"`
	Unresolved      int `json:"unresolved"`
	AliasesLearned  int `json:"aliasesLearned"`

	// Search projection detail. Paging and resume are reported because they are
	// what removes the previous silent 10,000-document ceiling.
	ProjectionPages   int  `json:"projectionPages,omitempty"`
	ProjectionResumed bool `json:"projectionResumed,omitempty"`

	SearchSync search.SyncResult `json:"searchSync"`
	Warnings   []string          `json:"warnings,omitempty"`
}

func (r *indexResult) addWarning(message string) {
	if len(r.Warnings) < maxIndexWarnings {
		r.Warnings = append(r.Warnings, message)
		return
	}
	if len(r.Warnings) == maxIndexWarnings {
		r.Warnings = append(r.Warnings, "additional indexing warnings were omitted")
	}
}

// handleIndex runs the library indexing pipeline.
//
// The flow is deliberately staged, and persistence is separated from discovery:
//
//	scan (streamed, bounded, per-root isolated)
//	  -> batch upsert only for new/changed/renamed files
//	  -> optional technical inspection (never a full re-probe of the library)
//	  -> search index sync
//
// Incremental runs skip unchanged files entirely, which is what makes a
// 100k-file library re-scan in seconds instead of hours.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
		// Mode is "full" or "incremental"; empty defaults to incremental.
		Mode string `json:"mode"`
		// Inspect controls ffprobe inspection of new/changed files.
		Inspect *bool `json:"inspect"`
		// SyncSearch controls the Meilisearch sync at the end of the run.
		SyncSearch *bool `json:"syncSearch"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}

	mode := scanner.ModeIncremental
	if strings.EqualFold(strings.TrimSpace(request.Mode), string(scanner.ModeFull)) {
		mode = scanner.ModeFull
	}
	inspect := mode == scanner.ModeFull
	if request.Inspect != nil {
		inspect = *request.Inspect
	}
	syncSearch := true
	if request.SyncSearch != nil {
		syncSearch = *request.SyncSearch
	}

	result := indexResult{Roots: request.Roots, Mode: string(mode), Warnings: []string{}}

	// A Server built directly (as tests do) may not have a guard yet.
	if s.scanGuard == nil {
		s.scanGuard = newScanGuard()
	}

	// Only one scan may run at a time: two concurrent scans would race on the
	// same rows and could double-insert during a full scan.
	if !s.scanGuard.tryAcquire() {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "a scan is already running",
			"scanId": s.scanGuard.currentScanID(),
		})
		return
	}
	defer s.scanGuard.release()

	scanID := newScanSessionID()
	result.ScanID = scanID
	if err := s.repository.StartScanSession(r.Context(), scanID, string(mode)); err != nil {
		result.addWarning("could not record scan session: " + err.Error())
	}

	report, err := s.runScan(r.Context(), scanRequest{
		roots:      request.Roots,
		mode:       mode,
		inspect:    inspect,
		scanID:     scanID,
		result:     &result,
		rootStates: map[string]string{},
	})
	if err != nil {
		if finishErr := s.repository.FinishScanSession(context.WithoutCancel(r.Context()), scanID, string(scanner.StatusFailed), report.Progress); finishErr != nil {
			result.addWarning("could not finalize scan session: " + finishErr.Error())
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}

	result.Status = string(report.Progress.Status)
	result.Report = &report.Progress
	result.Missing = len(report.Missing)
	if len(report.RootStates) > 0 {
		result.RootStates = make(map[string]string, len(report.RootStates))
		for root, state := range report.RootStates {
			result.RootStates[root] = string(state)
		}
	}

	if err := s.repository.FinishScanSession(context.WithoutCancel(r.Context()), scanID, string(report.Progress.Status), report.Progress); err != nil {
		result.addWarning("could not finalize scan session: " + err.Error())
	}

	if syncSearch {
		// A paged projection, not a single limited read. The previous sync read at
		// most 10,000 documents and silently stopped, so a larger library kept a
		// permanently incomplete index with no error to notice.
		projector := search.NewProjector(s.search, s.repository, search.DefaultProjectionPageSize, nil)
		projection, err := projector.Rebuild(r.Context(), false)
		if err != nil {
			result.addWarning("search projection incomplete: " + err.Error())
		}
		result.SearchSync = search.SyncResult{Indexed: projection.Documents, TaskUID: projection.TaskUID}
		result.ProjectionPages = projection.Pages
		result.ProjectionResumed = projection.Resumed
	}

	writeJSON(w, http.StatusOK, result)
}

// scanRequest carries the parameters of one scan run.
type scanRequest struct {
	roots      []string
	mode       scanner.ScanMode
	inspect    bool
	scanID     string
	result     *indexResult
	rootStates map[string]string
}

// runScan executes the pipeline and persists its results in bounded batches.
func (s *Server) runScan(ctx context.Context, request scanRequest) (scanner.Report, error) {
	// Load the persisted catalogue for the scanned roots so the scanner can skip
	// unchanged files and infer renames.
	known, err := s.repository.ListKnownFiles(ctx, request.roots)
	if err != nil {
		request.result.addWarning("could not load known files: " + err.Error())
	}

	// Every write goes through the shared resolution session, so the scan sees the
	// same candidate set and learned aliases the watcher and scheduler use, and a
	// work created by one batch is visible to the next.
	if s.resolution == nil {
		s.resolution = db.NewResolutionSession()
	}
	// Force a reload at the start of a run so entities created since the last
	// scan are candidates for this one.
	s.resolution.Invalidate()

	batch := make([]scanner.FileInfo, 0, ingestBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		// The resolution-driven path: a physical file is matched against existing
		// entities before anything is created, and an ambiguous file is queued for
		// review instead of inventing a work.
		stats, err := s.resolution.Ingest(ctx, s.repository, batch)
		request.result.Inserted += stats.Resolved + stats.Created
		request.result.Updated += stats.Provisional
		request.result.Imported += stats.FilesAttached
		request.result.QueuedForReview += stats.QueuedForReview
		request.result.WorksCreated += stats.Created
		request.result.AliasesLearned += stats.AliasesLearned
		request.result.Unresolved += stats.Unresolved
		if stats.Failed > 0 {
			request.result.addWarning(fmt.Sprintf("%d file(s) in the batch could not be persisted", stats.Failed))
		}
		if err != nil {
			return err
		}
		if request.inspect {
			s.inspectBatch(ctx, batch, request.result)
		}
		batch = batch[:0]
		return nil
	}

	// Publish live progress so /api/scan/status can report a real snapshot.
	scanCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()
	s.scanGuard.setProgress(request.scanID, cancelScan, scanner.Progress{
		ScanID:    request.scanID,
		Status:    scanner.StatusRunning,
		StartedAt: time.Now().UTC(),
	})

	// The control handle lets the operator pause, resume and cancel this run, and
	// lets the UI see what every worker is doing. Pause is cooperative: a worker
	// finishing its current file is allowed to complete, so no half-written row
	// ever reaches the database.
	controlHandle := &scanner.ScanControlHandle{}
	s.scanGuard.setControl(controlHandle)

	report, err := s.scanner.ScanTree(scanCtx, request.roots, scanner.ScanOptions{
		Mode:    request.mode,
		Known:   known,
		Control: controlHandle,
		OnProgress: func(progress scanner.Progress) {
			s.scanGuard.setProgress(request.scanID, cancelScan, progress)
		},
	}, func(file scanner.FileInfo) error {
		request.result.Scanned++
		batch = append(batch, file)
		if len(batch) == cap(batch) {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return report, err
	}

	// Reconciliation: records under a readable root that were not observed are
	// marked MISSING. Records under an unreadable root stay untouched, so an
	// unplugged disk never loses its catalogue entries.
	if len(report.Missing) > 0 {
		missingPaths := make([]string, 0, len(report.Missing))
		unavailablePaths := make([]string, 0)
		for _, record := range report.Missing {
			root := scanner.RootsForKnownFile(record.Path, request.roots)
			state := report.RootStates[root]
			if state == scanner.RootCompleted || state == scanner.RootScanning || state == scanner.RootAvailable {
				missingPaths = append(missingPaths, record.Path)
			} else {
				unavailablePaths = append(unavailablePaths, record.Path)
			}
		}
		if len(missingPaths) > 0 {
			if _, err := s.repository.MarkFilesState(ctx, missingPaths, scanner.StateMissing, request.scanID); err != nil {
				request.result.addWarning("could not mark missing files: " + err.Error())
			}
		}
		if len(unavailablePaths) > 0 {
			if _, err := s.repository.MarkFilesState(ctx, unavailablePaths, scanner.StateUnavailable, request.scanID); err != nil {
				request.result.addWarning("could not mark unavailable files: " + err.Error())
			}
		}
	}

	return report, nil
}

// inspectBatch runs ffprobe for a batch of freshly ingested files and stores the
// technical details. It is bounded and only ever applied to new/changed files,
// which is what prevents a full library re-probe on every scan.
func (s *Server) inspectBatch(ctx context.Context, batch []scanner.FileInfo, result *indexResult) {
	for i := range batch {
		if ctx.Err() != nil {
			return
		}
		file := batch[i]
		id, err := s.repository.GetVideoFileIDByPath(ctx, file.Path)
		if err != nil {
			result.InspectFailed++
			result.addWarning("could not locate indexed file: " + file.Path)
			continue
		}
		details, err := s.processor.Inspect(ctx, file.Path)
		if err != nil {
			result.InspectFailed++
			result.addWarning("could not inspect: " + file.Path)
			continue
		}
		if err := s.repository.UpdateVideoTechnicalDetails(ctx, id, details); err != nil {
			result.InspectFailed++
			result.addWarning("could not save inspection: " + file.Path)
			continue
		}
		result.Inspected++
	}
}

// newScanSessionID produces a sortable scan identifier.
func newScanSessionID() string {
	return fmt.Sprintf("scan-%d", time.Now().UTC().UnixNano())
}

// handleScanStatus reports the running scan's live progress, its pause state and
// a per-worker breakdown, or the most recent completed session when idle.
//
// The worker list is what turns "8 workers" into information: an operator can
// see which worker is discovering, which is parsing, which is persisting and
// which is idle, together with the file each one is on.
func (s *Server) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	if progress, ok := s.scanGuard.snapshot(); ok {
		payload := map[string]any{
			"running":  true,
			"progress": progress,
		}
		if control, controlOK := s.scanGuard.controlSnapshot(); controlOK {
			payload["control"] = control
			payload["workers"] = control.Workers
			payload["activeWorkers"] = control.ActiveWorkers
			payload["idleWorkers"] = control.IdleWorkers
			payload["scanState"] = control.State
		}
		writeJSON(w, http.StatusOK, payload)
		return
	}
	session, err := s.repository.LatestScanProgress(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"running": false,
		"latest":  session,
	})
}

// handleScanCancel requests cancellation of the running scan. The scanner
// finishes its current work and leaves the catalogue consistent.
func (s *Server) handleScanCancel(w http.ResponseWriter, r *http.Request) {
	cancelled := s.scanGuard.cancel()
	writeJSON(w, http.StatusAccepted, map[string]any{"cancelled": cancelled})
}

// handleScanPause requests a cooperative pause.
//
// The response reports the state AFTER the request, which is normally "pausing":
// workers finish the file they are on and then stop taking new work. The scan
// only becomes "paused" once no worker is mid-item, so the UI must show the
// transition rather than claiming an immediate stop.
//
// Pause is not a substitute for Cancel: a paused scan keeps its counters, its
// cursor and its per-root state, and Resuming continues from the same point.
func (s *Server) handleScanPause(w http.ResponseWriter, r *http.Request) {
	state, ok := s.scanGuard.pause()
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "no scan is running"})
		return
	}

	payload := map[string]any{"state": state}
	if snapshot, snapshotOK := s.scanGuard.controlSnapshot(); snapshotOK {
		payload["control"] = snapshot
		payload["workers"] = snapshot.Workers
		payload["workersDraining"] = snapshot.Draining
	}
	writeJSON(w, http.StatusAccepted, payload)
}

// handleScanResume lifts a pause and continues from the same point.
func (s *Server) handleScanResume(w http.ResponseWriter, r *http.Request) {
	state, ok := s.scanGuard.resume()
	if !ok {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "no scan is running"})
		return
	}

	payload := map[string]any{"state": state}
	if snapshot, snapshotOK := s.scanGuard.controlSnapshot(); snapshotOK {
		payload["control"] = snapshot
		payload["workers"] = snapshot.Workers
	}
	writeJSON(w, http.StatusAccepted, payload)
}

// handleScanWorkers reports only the worker breakdown, for a UI that polls the
// worker panel more often than the full status.
func (s *Server) handleScanWorkers(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := s.scanGuard.controlSnapshot()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"running":   false,
			"workers":   []any{},
			"scanState": "idle",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"running":       true,
		"scanState":     snapshot.State,
		"workers":       snapshot.Workers,
		"activeWorkers": snapshot.ActiveWorkers,
		"idleWorkers":   snapshot.IdleWorkers,
		"draining":      snapshot.Draining,
		"pausedFor":     snapshot.PausedFor,
	})
}

// handleInterruptedScans lists scans that never completed, which is how the owner
// learns the index may need a reconciliation pass after an unclean shutdown.
func (s *Server) handleInterruptedScans(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.repository.InterruptedScanSessions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

type indexPreviewItem struct {
	Title         string               `json:"title"`
	TitleAR       string               `json:"title_ar"`
	TitleEN       string               `json:"title_en"`
	Type          string               `json:"type"`
	CategorySlug  string               `json:"category_slug"`
	OriginTags    []string             `json:"origin_tags"`
	ReleaseYear   int                  `json:"release_year"`
	PosterPath    string               `json:"poster_path,omitempty"`
	RawPosterPath string               `json:"raw_poster_path,omitempty"`
	TotalFiles    int                  `json:"total_files"`
	TotalSize     int64                `json:"total_size"`
	Seasons       []indexPreviewSeason `json:"seasons"`
}

type indexPreviewSeason struct {
	SeasonNumber int                `json:"season_number"`
	Title        string             `json:"title"`
	EpisodeCount int                `json:"episode_count"`
	Episodes     []indexPreviewFile `json:"episodes"`
}

type indexPreviewFile struct {
	Path          string `json:"path"`
	EpisodeNumber int    `json:"episode_number"`
	Size          int64  `json:"size"`
	Resolution    string `json:"resolution"`
}

// handleIndexPreview generates a non-destructive dry-run preview of what would be indexed.
func (s *Server) handleIndexPreview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}

	type mediaGroupKey struct {
		titleKey string
		mType    string
	}

	groups := make(map[mediaGroupKey]*indexPreviewItem)
	var groupOrder []mediaGroupKey
	totalScanned := 0

	err := s.scanner.Walk(r.Context(), request.Roots, func(file scanner.FileInfo) error {
		totalScanned++
		categorySlug, mediaType := db.ClassifyMedia(file.Path, file.Parsed)
		title := file.Parsed.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
		}
		titleEN := file.Parsed.TitleEN
		if titleEN == "" {
			titleEN = title
		}
		titleAR := file.Parsed.TitleAR
		if titleAR == "" {
			titleAR = title
		}

		key := mediaGroupKey{
			titleKey: strings.ToLower(titleEN),
			mType:    mediaType,
		}

		group, exists := groups[key]
		if !exists {
			localPosterURL := ""
			if file.ArtworkPath != "" {
				localPosterURL = s.repository.CacheLocalArtwork(file.ArtworkPath)
			}
			originTags := scanner.DetectOriginTagsFromPath(file.Path)
			if originTags == nil {
				originTags = []string{}
			}

			group = &indexPreviewItem{
				Title:         title,
				TitleAR:       titleAR,
				TitleEN:       titleEN,
				Type:          mediaType,
				CategorySlug:  categorySlug,
				OriginTags:    originTags,
				ReleaseYear:   file.Parsed.ReleaseYear,
				PosterPath:    localPosterURL,
				RawPosterPath: file.ArtworkPath,
				Seasons:       []indexPreviewSeason{},
			}
			groups[key] = group
			groupOrder = append(groupOrder, key)
		}

		group.TotalFiles++
		group.TotalSize += file.Size
		if group.PosterPath == "" && file.ArtworkPath != "" {
			group.PosterPath = s.repository.CacheLocalArtwork(file.ArtworkPath)
			group.RawPosterPath = file.ArtworkPath
		}

		seasonNum := file.Parsed.SeasonNumber
		if seasonNum <= 0 {
			seasonNum = 1
		}

		// Find or add season
		var targetSeason *indexPreviewSeason
		for idx := range group.Seasons {
			if group.Seasons[idx].SeasonNumber == seasonNum {
				targetSeason = &group.Seasons[idx]
				break
			}
		}
		if targetSeason == nil {
			group.Seasons = append(group.Seasons, indexPreviewSeason{
				SeasonNumber: seasonNum,
				Title:        fmt.Sprintf("Season %02d", seasonNum),
				Episodes:     []indexPreviewFile{},
			})
			targetSeason = &group.Seasons[len(group.Seasons)-1]
		}

		targetSeason.EpisodeCount++
		targetSeason.Episodes = append(targetSeason.Episodes, indexPreviewFile{
			Path:          file.Path,
			EpisodeNumber: file.Parsed.EpisodeNumber,
			Size:          file.Size,
			Resolution:    file.Parsed.Resolution,
		})

		return nil
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	resultItems := make([]*indexPreviewItem, 0, len(groupOrder))
	for _, k := range groupOrder {
		resultItems = append(resultItems, groups[k])
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roots":      request.Roots,
		"totalFiles": totalScanned,
		"totalMedia": len(resultItems),
		"mediaItems": resultItems,
	})
}

// handleClassifyOrigins repairs country tags from the user's existing folder
// hierarchy. It is deliberately local-only: no title matching or third-party
// request can overwrite a folder such as "مسلسلات/عربي".
func (s *Server) handleClassifyOrigins(w http.ResponseWriter, r *http.Request) {
	updated, err := s.repository.ClassifyOriginsFromPaths(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Country tags are also searchable, so refresh the local search index.
	// The projection is paged, so this scales past the old 10,000 ceiling.
	projector := search.NewProjector(s.search, s.repository, search.DefaultProjectionPageSize, nil)
	if _, err := projector.Rebuild(r.Context(), true); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"updated": updated,
			"warning": "origin tags updated but search index refresh failed: " + err.Error(),
		})
		return
	}
	_ = updated
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}

// handleSearchSync rebuilds the search index as a projection of PostgreSQL.
//
// Query parameter `reset=true` starts from the beginning, which is the
// "drop the index and rebuild it from the database" operation. Without it the
// run RESUMES from the persisted cursor, so an interrupted rebuild continues.
//
// Nothing here reads the filesystem: the index is derived, and rebuilding it
// never re-scans a single media file.
func (s *Server) handleSearchSync(w http.ResponseWriter, r *http.Request) {
	reset := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("reset")), "true")

	projector := search.NewProjector(s.search, s.repository, search.DefaultProjectionPageSize, nil)
	result, err := projector.Rebuild(r.Context(), reset)
	if err != nil {
		// Report the progress already made alongside the error, so the operator
		// knows how far the index got rather than only that it failed.
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 24
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	filterParts := make([]string, 0, 2)
	if rawType := strings.TrimSpace(r.URL.Query().Get("type")); rawType != "" {
		filterParts = append(filterParts, `type = "`+escapeFilterValue(rawType)+`"`)
	}
	if rawCategory := strings.TrimSpace(r.URL.Query().Get("category")); rawCategory != "" {
		filterParts = append(filterParts, `category_slug = "`+escapeFilterValue(rawCategory)+`"`)
	}

	result, err := s.search.SearchDocuments(r.Context(), query, limit, strings.Join(filterParts, " AND "))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMetadataLookup(w http.ResponseWriter, r *http.Request) {
	var request metadata.Query
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.metadata.Lookup(r.Context(), request)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, metadata.ErrNotConfigured) {
			status = http.StatusFailedDependency
		}
		if errors.Is(err, metadata.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}
