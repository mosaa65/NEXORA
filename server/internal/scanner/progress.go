package scanner

import (
	"sync"
	"sync/atomic"
	"time"
)

// ScanStatus is the lifecycle of a scan session. It is persisted so that a
// server restart can discover a scan that never finished instead of assuming
// the library is consistent.
type ScanStatus string

const (
	StatusRunning   ScanStatus = "running"
	StatusCompleted ScanStatus = "completed"
	StatusFailed    ScanStatus = "failed"
	StatusCancelled ScanStatus = "cancelled"
	StatusPartial   ScanStatus = "partial"
)

// RootStatus is the independent health of a single media root. Root isolation
// is the reason one unplugged disk does not stop the other disks: each root
// owns its own status and is scanned on its own.
type RootStatus string

const (
	RootPending     RootStatus = "pending"
	RootAvailable   RootStatus = "available"
	RootScanning    RootStatus = "scanning"
	RootCompleted   RootStatus = "completed"
	RootUnavailable RootStatus = "unavailable"
	RootError       RootStatus = "error"
	RootOffline     RootStatus = "offline"
)

// Progress is a live, structured snapshot of a running scan. Every field is
// real counter data, never a guess, so the UI can display a truthful report.
type Progress struct {
	ScanID              string       `json:"scanId"`
	RootID              string       `json:"rootId,omitempty"`
	Root                string       `json:"root,omitempty"`
	Status              ScanStatus   `json:"status"`
	RootStatus          RootStatus   `json:"rootStatus,omitempty"`
	StartedAt           time.Time    `json:"startedAt"`
	FinishedAt          *time.Time   `json:"finishedAt,omitempty"`
	Duration            Duration     `json:"duration"`
	Directories         int64        `json:"directoriesVisited"`
	FilesSeen           int64        `json:"filesSeen"`
	VideoCandidates     int64        `json:"videoCandidates"`
	Accepted            int64        `json:"acceptedMedia"`
	Rejected            int64        `json:"rejectedFiles"`
	NewFiles            int64        `json:"newFiles"`
	ModifiedFiles       int64        `json:"modifiedFiles"`
	RemovedFiles        int64        `json:"removedFiles"`
	RenamedFiles        int64        `json:"renamedFiles"`
	UnchangedFiles      int64        `json:"unchangedFiles"`
	ParseFailures       int64        `json:"parseFailures"`
	LowConfidence       int64        `json:"lowConfidenceItems"`
	PermissionErrors    int64        `json:"permissionErrors"`
	FilesystemErrors    int64        `json:"filesystemErrors"`
	ArtworkFound        int64        `json:"artworkFound"`
	ArtworkMissing      int64        `json:"artworkMissing"`
	DuplicateCandidates int64        `json:"duplicateCandidates"`
	TotalBytes          int64        `json:"totalBytes"`
	SkippedBytes        int64        `json:"skippedBytes"`
	RootsCompleted      int          `json:"rootsCompleted"`
	RootsFailed         int          `json:"rootsFailed"`
	RootsUnavailable    int          `json:"rootsUnavailable"`
	Throughput          Throughput   `json:"throughput"`
	Errors              []ErrorCount `json:"errors,omitempty"`
	TopProblems         []Problem    `json:"topProblems,omitempty"`
}

// Duration renders the elapsed time in a stable, human-readable shape.
type Duration struct {
	Seconds float64 `json:"seconds"`
	Human   string  `json:"human"`
}

// Throughput reports measured scan speed, not an estimate.
type Throughput struct {
	FilesPerSecond float64 `json:"filesPerSecond"`
	BytesPerSecond float64 `json:"bytesPerSecond"`
	MBPerSecond    float64 `json:"mbPerSecond"`
}

// Problem is one actionable finding in the scan report. The owner-facing report
// must explain what happened, not merely count it.
type Problem struct {
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Count   int64  `json:"count"`
	Affects string `json:"affectsDataQuality"`
	Urgency string `json:"urgency"`
	Code    string `json:"code,omitempty"`
}

// counter is the lock-free accumulator shared by all scan goroutines. It is a
// plain atomic bundle: no mutex on the hot path, and no unbounded growth since
// nothing is stored per file.
type counter struct {
	directories         atomic.Int64
	filesSeen           atomic.Int64
	videoCandidates     atomic.Int64
	accepted            atomic.Int64
	rejected            atomic.Int64
	newFiles            atomic.Int64
	modifiedFiles       atomic.Int64
	removedFiles        atomic.Int64
	renamedFiles        atomic.Int64
	unchangedFiles      atomic.Int64
	parseFailures       atomic.Int64
	lowConfidence       atomic.Int64
	permissionErrors    atomic.Int64
	filesystemErrors    atomic.Int64
	artworkFound        atomic.Int64
	artworkMissing      atomic.Int64
	duplicateCandidates atomic.Int64
	totalBytes          atomic.Int64
	skippedBytes        atomic.Int64
	rootsCompleted      atomic.Int64
	rootsFailed         atomic.Int64
	rootsUnavailable    atomic.Int64
}

// ProgressTracker exposes a consistent view of a running scan. It coalesces
// per-file updates into atomics and only materialises a Progress struct when
// somebody asks, which keeps a million-file scan cheap to observe.
type ProgressTracker struct {
	scanID     string
	started    time.Time
	mu         sync.Mutex
	root       string
	rootID     string
	status     ScanStatus
	rootStates map[string]RootStatus
	counts     counter
	errors     *ErrorSink
	finished   *time.Time
	problems   []Problem
}

// NewProgressTracker creates a tracker for a scan session.
func NewProgressTracker(scanID string, errorSink *ErrorSink) *ProgressTracker {
	if errorSink == nil {
		errorSink = NewErrorSink(50)
	}
	return &ProgressTracker{
		scanID:     scanID,
		started:    time.Now().UTC(),
		status:     StatusRunning,
		rootStates: make(map[string]RootStatus),
		errors:     errorSink,
	}
}

// ScanID returns the identifier of the tracked session.
func (p *ProgressTracker) ScanID() string { return p.scanID }

// Errors exposes the underlying sink so scan loops can record failures.
func (p *ProgressTracker) Errors() *ErrorSink { return p.errors }

// SetRootStatus records the independent state of one root.
func (p *ProgressTracker) SetRootStatus(root string, status RootStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rootStates[root] = status
}

// RootStatuses returns a copy of the per-root states.
func (p *ProgressTracker) RootStatuses() map[string]RootStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]RootStatus, len(p.rootStates))
	for root, status := range p.rootStates {
		out[root] = status
	}
	return out
}

// Counts returns the live counter bundle. Callers mutate through the returned
// pointer, which is safe because every field is atomic.
func (p *ProgressTracker) Counts() *counter { return &p.counts }

// AddProblem appends an actionable finding to the report.
func (p *ProgressTracker) AddProblem(problem Problem) {
	if problem.Title == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.problems = append(p.problems, problem)
}

// Finish marks the session terminal and stamps the end time.
func (p *ProgressTracker) Finish(status ScanStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
	now := time.Now().UTC()
	p.finished = &now
	// Mirror a permission-error spike into the actionable problem list once, so
	// the report explains the failure instead of only counting it.
	if denied := p.errors.Count(ErrPermissionDenied); denied > 0 && !p.hasProblemLocked("permission_denied") {
		p.problems = append(p.problems, Problem{
			Title:   "مجلدات غير قابلة للقراءة (Permission denied)",
			Detail:  "تعذّر الدخول إلى بعض المجلدات أثناء الفحص؛ بقية الفحص أُكملت بنجاح.",
			Count:   int64(denied),
			Affects: "لا، الفهرسة استمرت",
			Urgency: "تحقق من صلاحيات المستخدم على هذه المسارات",
			Code:    string(ErrPermissionDenied),
		})
	}
}

func (p *ProgressTracker) hasProblemLocked(code string) bool {
	for _, existing := range p.problems {
		if existing.Code == code {
			return true
		}
	}
	return false
}

// Snapshot renders the current state as structured, UI-ready data.
func (p *ProgressTracker) Snapshot() Progress {
	p.mu.Lock()
	status := p.status
	finished := p.finished
	problems := make([]Problem, len(p.problems))
	copy(problems, p.problems)
	p.mu.Unlock()

	end := time.Now().UTC()
	if finished != nil {
		end = *finished
	}
	elapsed := end.Sub(p.started)
	seconds := elapsed.Seconds()

	filesProcessed := p.counts.newFiles.Load() + p.counts.modifiedFiles.Load() + p.counts.renamedFiles.Load() + p.counts.unchangedFiles.Load()
	bytes := p.counts.totalBytes.Load()

	progress := Progress{
		ScanID:              p.scanID,
		Status:              status,
		StartedAt:           p.started,
		FinishedAt:          finished,
		Duration:            Duration{Seconds: seconds, Human: humanDuration(elapsed)},
		Directories:         p.counts.directories.Load(),
		FilesSeen:           p.counts.filesSeen.Load(),
		VideoCandidates:     p.counts.videoCandidates.Load(),
		Accepted:            p.counts.accepted.Load(),
		Rejected:            p.counts.rejected.Load(),
		NewFiles:            p.counts.newFiles.Load(),
		ModifiedFiles:       p.counts.modifiedFiles.Load(),
		RemovedFiles:        p.counts.removedFiles.Load(),
		RenamedFiles:        p.counts.renamedFiles.Load(),
		UnchangedFiles:      p.counts.unchangedFiles.Load(),
		ParseFailures:       p.counts.parseFailures.Load(),
		LowConfidence:       p.counts.lowConfidence.Load(),
		PermissionErrors:    p.counts.permissionErrors.Load(),
		FilesystemErrors:    p.counts.filesystemErrors.Load(),
		ArtworkFound:        p.counts.artworkFound.Load(),
		ArtworkMissing:      p.counts.artworkMissing.Load(),
		DuplicateCandidates: p.counts.duplicateCandidates.Load(),
		TotalBytes:          bytes,
		SkippedBytes:        p.counts.skippedBytes.Load(),
		RootsCompleted:      int(p.counts.rootsCompleted.Load()),
		RootsFailed:         int(p.counts.rootsFailed.Load()),
		RootsUnavailable:    int(p.counts.rootsUnavailable.Load()),
		Errors:              p.errors.SortedCounts(),
		TopProblems:         problems,
	}
	if seconds > 0 {
		progress.Throughput = Throughput{
			FilesPerSecond: float64(filesProcessed) / seconds,
			BytesPerSecond: float64(bytes) / seconds,
			MBPerSecond:    float64(bytes) / seconds / (1024 * 1024),
		}
	}
	return progress
}

// Duration returns elapsed time since the session started.
func (p *ProgressTracker) Elapsed() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished != nil {
		return p.finished.Sub(p.started)
	}
	return time.Since(p.started)
}

func humanDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	d = d.Round(time.Second)
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	switch {
	case hours > 0:
		return itoa(hours) + "h " + itoa(minutes) + "m " + itoa(seconds) + "s"
	case minutes > 0:
		return itoa(minutes) + "m " + itoa(seconds) + "s"
	default:
		return itoa(seconds) + "s"
	}
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
