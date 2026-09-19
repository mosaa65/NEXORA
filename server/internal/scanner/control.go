package scanner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ScanControlState is the operator-facing lifecycle of a scan.
//
// It is deliberately separate from ScanStatus: a scan can be RUNNING while a
// pause has been requested but not yet taken effect. Collapsing the two would
// make the UI claim a scan is paused while workers are still finishing.
type ScanControlState string

const (
	// ControlRunning means work is flowing.
	ControlRunning ScanControlState = "running"
	// ControlPausing means a pause was requested and workers are draining their
	// current item. No new item is started.
	ControlPausing ScanControlState = "pausing"
	// ControlPaused means no worker is mid-item; the scan is suspended and its
	// state is safe to leave for minutes or hours.
	ControlPaused ScanControlState = "paused"
	// ControlResuming means work is flowing again.
	ControlResuming ScanControlState = "resuming"
	// ControlCancelled is terminal.
	ControlCancelled ScanControlState = "cancelled"
)

// WorkerRole is what a worker is responsible for. It is reported per worker so
// the UI can show what the system is actually doing rather than a count.
type WorkerRole string

const (
	RoleDiscovery   WorkerRole = "discovery"
	RoleMetadata    WorkerRole = "metadata"
	RolePersistence WorkerRole = "persistence"
	RoleIdle        WorkerRole = "idle"
)

// WorkerSnapshot is the live state of one worker.
//
// The design requires per-worker visibility: "Workers = 8" tells an operator
// nothing, while "worker 3: PARSER ACTIVE" tells them exactly where the work is.
type WorkerSnapshot struct {
	ID          int        `json:"id"`
	Role        WorkerRole `json:"role"`
	Active      bool       `json:"active"`
	CurrentPath string     `json:"currentPath,omitempty"`
	Processed   int64      `json:"processed"`
	// ItemStartedAt lets the UI show how long the current item has been running,
	// which is how a stuck file becomes visible.
	ItemStartedAt *time.Time `json:"itemStartedAt,omitempty"`
	IdleSince     *time.Time `json:"idleSince,omitempty"`
}

// scanControl coordinates cooperative pause, resume and cancellation across the
// whole pipeline.
//
// Pause is COOPERATIVE by design, not a hard stop:
//
//   - no new work item is started while paused;
//   - an item already in flight is allowed to finish, so the database never
//     receives a half-written row;
//   - the counters, cursor and per-root state stay exactly as they were, so a
//     resume continues from where it stopped.
//
// Cancel remains a separate operation with different semantics: it ends the scan
// and reports CANCELLED. Reusing cancel for pause would make a temporary stop
// indistinguishable from an abandoned run.
type scanControl struct {
	mu        sync.Mutex
	state     ScanControlState
	resume    chan struct{}
	cancelled bool
	// workers holds the live state of every worker, indexed by worker id.
	workers map[int]*WorkerSnapshot
	// waitingForIdle counts workers currently blocked on a pause, so the UI can
	// show "pausing (3 workers finishing)".
	waitingForIdle atomic.Int64
	// pausedSince records when the scan actually reached the paused state.
	pausedSince *time.Time
}

func newScanControl(workers int) *scanControl {
	control := &scanControl{
		state:   ControlRunning,
		workers: make(map[int]*WorkerSnapshot, workers),
	}
	for i := 0; i < workers; i++ {
		control.workers[i] = &WorkerSnapshot{ID: i, Role: RoleIdle}
	}
	return control
}

// State returns the current control state.
func (c *scanControl) State() ScanControlState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Pause requests a pause.
//
// It returns the state after the request, which is PAUSING while workers are
// still finishing their current item. Calling Pause on an already paused scan is
// a no-op rather than an error, so a retried request from the UI is harmless.
func (c *scanControl) Pause() ScanControlState {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case ControlRunning, ControlResuming:
		c.state = ControlPausing
		// A fresh channel signals resume; closing it would release every waiter but
		// could not be reused for a second pause.
		c.resume = make(chan struct{})
	case ControlPausing, ControlPaused:
		// Already pausing or paused: nothing to do.
	}
	return c.state
}

// Resume lifts a pause.
func (c *scanControl) Resume() ScanControlState {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.state {
	case ControlPausing, ControlPaused:
		c.state = ControlResuming
		if c.resume != nil {
			close(c.resume)
			c.resume = nil
		}
		c.pausedSince = nil
	}
	return c.state
}

// Cancel marks the scan cancelled and releases any paused workers.
func (c *scanControl) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelled = true
	c.state = ControlCancelled
	if c.resume != nil {
		close(c.resume)
		c.resume = nil
	}
}

// Cancelled reports whether cancellation was requested.
func (c *scanControl) Cancelled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cancelled
}

// wait blocks while the scan is paused.
//
// It is called immediately BEFORE a worker takes a new item, never in the middle
// of one. That is what makes the pause safe: no partial item is ever abandoned.
// It returns false when the context ended or the scan was cancelled, telling the
// caller to stop.
func (c *scanControl) wait(ctx context.Context) bool {
	for {
		c.mu.Lock()
		state := c.state
		resume := c.resume
		cancelled := c.cancelled
		if state == ControlPausing {
			// The first worker to arrive marks the scan fully paused once every
			// worker has been observed waiting.
			c.state = ControlPaused
			now := time.Now()
			c.pausedSince = &now
		}
		c.mu.Unlock()

		if cancelled {
			return false
		}
		if ctx.Err() != nil {
			return false
		}
		if state != ControlPausing && state != ControlPaused {
			// Running or resuming: clear the idle marker and proceed.
			c.waitingForIdle.Add(-1)
			return true
		}

		c.waitingForIdle.Add(1)
		select {
		case <-resume:
		case <-ctx.Done():
			c.waitingForIdle.Add(-1)
			return false
		}
	}
}

// markWorker records what a worker is doing right now.
func (c *scanControl) markWorker(id int, role WorkerRole, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	worker, exists := c.workers[id]
	if !exists {
		worker = &WorkerSnapshot{ID: id}
		c.workers[id] = worker
	}
	if role == RoleIdle {
		worker.Active = false
		worker.Role = RoleIdle
		worker.CurrentPath = ""
		if worker.IdleSince == nil {
			now := time.Now()
			worker.IdleSince = &now
		}
		return
	}
	worker.Active = true
	worker.Role = role
	worker.CurrentPath = path
	worker.Processed++
	now := time.Now()
	worker.ItemStartedAt = &now
	worker.IdleSince = nil
}

// WorkerStates returns a stable, ordered snapshot of every worker.
func (c *scanControl) WorkerStates() []WorkerSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]WorkerSnapshot, 0, len(c.workers))
	for _, worker := range c.workers {
		copyWorker := *worker
		out = append(out, copyWorker)
	}
	// Order by id so the UI list does not reshuffle on every poll.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// ControlSnapshot is the pause/worker view exposed to the API.
type ControlSnapshot struct {
	State         ScanControlState `json:"state"`
	Workers       []WorkerSnapshot `json:"workers"`
	ActiveWorkers int              `json:"activeWorkers"`
	IdleWorkers   int              `json:"idleWorkers"`
	Draining      int64            `json:"workersDrainingCurrentItem"`
	PausedFor     string           `json:"pausedFor,omitempty"`
}

// Snapshot renders the current control state for the API.
func (c *scanControl) Snapshot() ControlSnapshot {
	c.mu.Lock()
	state := c.state
	pausedSince := c.pausedSince
	c.mu.Unlock()

	workers := c.WorkerStates()
	snapshot := ControlSnapshot{State: state, Workers: workers}
	for _, worker := range workers {
		if worker.Active {
			snapshot.ActiveWorkers++
		} else {
			snapshot.IdleWorkers++
		}
	}
	snapshot.Draining = c.waitingForIdle.Load()
	if pausedSince != nil {
		snapshot.PausedFor = humanDuration(time.Since(*pausedSince))
	}
	return snapshot
}
