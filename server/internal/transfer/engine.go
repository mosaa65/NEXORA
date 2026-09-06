package transfer

import (
	"context"
	"sync"
	"time"
)

// VerifyMode controls how strictly a copied file is checked.
type VerifyMode string

const (
	VerifySize   VerifyMode = "size"   // remote size == source size (fast, default)
	VerifyStrict VerifyMode = "strict" // full content hashing (slower)
)

// TransferConfig holds engine-wide tunables. The starting values reflect the
// development plan (§33, §74): a 4 MB buffer, 3 retries, concurrency 1 per
// device, and fast size-based verification. They should be re-tuned after a
// real benchmark on the target hardware.
type TransferConfig struct {
	BufferSize int64

	MaxRetries int

	RetryInitialDelay time.Duration
	RetryMaxDelay     time.Duration

	VerifyMode VerifyMode

	PerDeviceConcurrency int
}

// DefaultTransferConfig returns the recommended starting configuration.
func DefaultTransferConfig() TransferConfig {
	return TransferConfig{
		BufferSize:         defaultBufferSize,
		MaxRetries:         3,
		RetryInitialDelay:  500 * time.Millisecond,
		RetryMaxDelay:      4 * time.Second,
		VerifyMode:         VerifySize,
		PerDeviceConcurrency: 1,
	}
}

// BackendFactory returns the appropriate TransferBackend for a device. It is a
// function so the engine stays decoupled from concrete backend constructors.
type BackendFactory func(device Device) (TransferBackend, error)

// Engine is the v2 transfer controller. It owns the scheduler, per-device
// workers and a registry of active jobs, and delegates folder/transfer work to
// the appropriate backend.
type TransferEngine struct {
	config   TransferConfig
	factory  BackendFactory
	verify   *Verifier
	sched    *Scheduler
	mu       sync.RWMutex
	active   map[string]*TransferJobV2
	cancels  map[string]context.CancelFunc

	// notify is invoked on every meaningful job transition (progress tick,
	// phase/status change, terminal state). It is expected to be set once,
	// at construction time, before any jobs are submitted.
	notify func(*TransferJobV2)
}

func NewTransferEngine(config TransferConfig, factory BackendFactory) *TransferEngine {
	e := &TransferEngine{
		config:  config,
		factory: factory,
		verify:  NewVerifier(config.VerifyMode),
		active:  make(map[string]*TransferJobV2),
		cancels: make(map[string]context.CancelFunc),
	}
	e.sched = NewScheduler(e)
	return e
}

// Submit enqueues a multi-file job for a destination device. The scheduler
// routes it to the device's worker queue. It returns immediately.
func (e *TransferEngine) Submit(job *TransferJobV2) error {
	return e.sched.Submit(job)
}

// SetNotifier installs the callback invoked on every job transition. It must
// be called before the first Submit.
func (e *TransferEngine) SetNotifier(fn func(*TransferJobV2)) {
	e.notify = fn
}

// NotifyUpdate reports a job transition to the configured notifier, if any.
func (e *TransferEngine) NotifyUpdate(job *TransferJobV2) {
	if e.notify != nil {
		e.notify(job)
	}
}

// GetJob returns a copy of an active job by id.
func (e *TransferEngine) GetJob(id string) (*TransferJobV2, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	j, ok := e.active[id]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

// backendFor builds a backend for the given destination using the factory.
func (e *TransferEngine) backendFor(d TransferDestination) (TransferBackend, error) {
	device := Device{
		ID:        d.DeviceID,
		Type:      d.DeviceType,
	}
	return e.factory(device)
}

// beginTransfer registers a job as active.
func (e *TransferEngine) beginTransfer(job *TransferJobV2) {
	e.mu.Lock()
	e.active[job.ID] = job
	e.mu.Unlock()
}

// endTransfer deregisters a job once it has reached a terminal state.
func (e *TransferEngine) endTransfer(id string) {
	e.mu.Lock()
	delete(e.active, id)
	delete(e.cancels, id)
	e.mu.Unlock()
}

// registerCancel associates a per-job cancel function so the job can be
// cancelled externally while the worker is busy.
func (e *TransferEngine) registerCancel(id string, cancel context.CancelFunc) {
	e.mu.Lock()
	e.cancels[id] = cancel
	e.mu.Unlock()
}

// unregisterCancel drops the cancel function for a finished job.
func (e *TransferEngine) unregisterCancel(id string) {
	e.mu.Lock()
	delete(e.cancels, id)
	e.mu.Unlock()
}

// cancelJob requests cancellation of a running job by id. It is safe to call
// for an unknown or already-finished job.
func (e *TransferEngine) cancelJob(id string) {
	e.mu.RLock()
	cancel, ok := e.cancels[id]
	e.mu.RUnlock()
	if ok && cancel != nil {
		cancel()
	}
}
