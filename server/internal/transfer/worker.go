package transfer

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// DeviceWorker serialises all jobs destined for a single physical device so
// that only one transfer ever touches that device (and its AFC session) at a
// time. Different devices run concurrently via their own workers.
type DeviceWorker struct {
	engine *TransferEngine
	key    string

	queue chan *TransferJobV2
	done  chan struct{}

	mu      sync.Mutex
	running bool

	lastProgressCheck time.Time
	lastTransferred   int64
	smoothedSpeed     float64
}

func newDeviceWorker(engine *TransferEngine, key, deviceID string) *DeviceWorker {
	return &DeviceWorker{
		engine: engine,
		key:    key,
		queue:  make(chan *TransferJobV2, 32),
		done:   make(chan struct{}),
	}
}

// Enqueue appends a job to this device's queue and starts the processor
// goroutine on the first enqueued job.
func (w *DeviceWorker) Enqueue(job *TransferJobV2) error {
	select {
	case w.queue <- job:
	default:
		return fmt.Errorf("جهاز مشغول: قائمة انتظار %s ممتلئة", w.key)
	}

	w.mu.Lock()
	if !w.running {
		w.running = true
		go w.process()
	}
	w.mu.Unlock()
	return nil
}

// Stop signals the processor to drain and stop accepting new work.
func (w *DeviceWorker) Stop() {
	close(w.done)
}

// process consumes the queue sequentially, one job at a time.
func (w *DeviceWorker) process() {
	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	for {
		select {
		case <-w.done:
			return
		case job := <-w.queue:
			if job == nil {
				continue
			}
			w.processJob(job)
			if isTerminal(job) {
				w.engine.endTransfer(job.ID)
			}
		}
	}
}

// processJob runs a single multi-file job end to end: connect, ensure
// destination, copy every file, verify, then mark complete.
func (w *DeviceWorker) processJob(job *TransferJobV2) {
	jobCtx, cancel := context.WithCancel(context.Background())
	w.engine.registerCancel(job.ID, cancel)
	defer func() {
		cancel()
		w.engine.unregisterCancel(job.ID)
	}()

	job.Status = StatusProcessing
	job.Phase = PhasePreparing
	w.engine.beginTransfer(job)
	w.engine.NotifyUpdate(job)

	backend, err := w.engine.backendFor(job.Destination)
	if err != nil {
		w.fail(job, err, nil)
		return
	}
	defer backend.Close()

	job.Phase = PhaseConnecting
	if err := backend.Connect(jobCtx, Device{ID: job.Destination.DeviceID, Type: job.Destination.DeviceType}, job.Destination); err != nil {
		w.fail(job, err, backend)
		return
	}

	// Ensure the destination directory exists before copying anything.
	job.Phase = PhaseCheckingDestination
	if err := backend.Mkdir(jobCtx, job.Destination.RemotePath); err != nil {
		_ = jobCtx.Err()
		w.fail(job, err, backend)
		return
	}

	w.lastProgressCheck = time.Now()
	w.lastTransferred = 0
	w.smoothedSpeed = 0

	for i := range job.Files {
		file := &job.Files[i]
		job.CurrentFileIndex = i
		job.CurrentFile = filepath.Base(file.SourcePath)

		if file.Status == StatusCompleted {
			continue
		}

		err := w.copyFile(jobCtx, backend, job, file)
		if err != nil {
			w.fail(job, err, backend)
			return
		}

		file.Status = StatusCompleted
		now := time.Now()
		file.CompletedAt = &now
	}

	now := time.Now()
	job.CompletedAt = &now
	job.Phase = PhaseCompleted
	job.Status = StatusCompleted
	job.Progress = 100
	job.TransferredBytes = job.TotalBytes
	job.ETA = 0
	job.SpeedBps = 0
	w.engine.NotifyUpdate(job)
}

// copyFile handles a single file: retry policy, streaming copy and verify.
func (w *DeviceWorker) copyFile(ctx context.Context, backend TransferBackend, job *TransferJobV2, file *TransferFile) error {
	srcInfo, err := statSource(file.SourcePath)
	if err != nil {
		return WrapTransferError(ErrSourceNotFound("تعذر قراءة الملف المصدر: "+err.Error()), err)
	}
	file.Size = srcInfo.Size()

	dest := joinRemotePath(job.Destination.RemotePath, filepath.Base(file.SourcePath))
	file.DestinationPath = dest

	job.Phase = PhaseCopying

	opts := PutOptions{
		BufferSize: w.engine.config.BufferSize,
		Overwrite:  false,
		OnProgress: func(transferred int64) {
			w.onFileProgress(job, file, transferred)
		},
	}

	if job.TotalBytes == 0 {
		job.TotalBytes = sumFileSizes(job.Files)
	}

	attempt := 0
	var lastErr error
	for attempt <= w.engine.config.MaxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		opts.ResumeOffset = remoteResumeOffset(ctx, backend, dest, file.Size)

		err = backend.Put(ctx, file.SourcePath, dest, opts)
		if err == nil {
			// Verification before declaring this file done.
			if verr := w.engine.verify.Verify(ctx, backend, dest, file.Size); verr != nil {
				lastErr = verr
			} else {
				return nil
			}
		} else {
			lastErr = err
		}

		if IsCancellation(lastErr) {
			return lastErr
		}
		if !IsRetryable(lastErr) {
			return lastErr
		}

		attempt++
		if attempt <= w.engine.config.MaxRetries {
			job.Status = StatusRetrying
			job.Phase = PhaseRetrying
			job.RetryCount = attempt
			w.engine.NotifyUpdate(job)
			time.Sleep(retryBackoff(w.engine.config, attempt))
		}
	}
	return lastErr
}

// onFileProgress updates overall job progress (sum of completed files plus the
// current file's real byte count).
func (w *DeviceWorker) onFileProgress(job *TransferJobV2, file *TransferFile, transferred int64) {
	file.Transferred = transferred
	job.CurrentFile = filepath.Base(file.SourcePath)

	var done int64
	for i := range job.Files {
		f := &job.Files[i]
		if f.Status == StatusCompleted || i < job.CurrentFileIndex {
			done += f.Size
		}
	}
	done += transferred
	if done > job.TotalBytes {
		done = job.TotalBytes
	}
	job.TransferredBytes = done
	if job.TotalBytes > 0 {
		job.Progress = float64(done) / float64(job.TotalBytes) * 100
	}

	now := time.Now()
	if w.lastProgressCheck.IsZero() {
		w.lastProgressCheck = now
		w.lastTransferred = done
	} else if dur := now.Sub(w.lastProgressCheck); dur >= 500*time.Millisecond {
		deltaBytes := done - w.lastTransferred
		if deltaBytes >= 0 {
			instantSpeed := float64(deltaBytes) / dur.Seconds()
			if w.smoothedSpeed <= 0 {
				w.smoothedSpeed = instantSpeed
			} else {
				w.smoothedSpeed = 0.7*w.smoothedSpeed + 0.3*instantSpeed
			}
			job.SpeedBps = int64(w.smoothedSpeed)
			remaining := job.TotalBytes - done
			if job.SpeedBps > 0 && remaining > 0 {
				job.ETA = time.Duration(float64(remaining)/float64(job.SpeedBps)) * time.Second
			} else {
				job.ETA = 0
			}
		}
		w.lastProgressCheck = now
		w.lastTransferred = done
		w.engine.NotifyUpdate(job)
	}
}

// fail records a terminal failure for a job.
func (w *DeviceWorker) fail(job *TransferJobV2, err error, backend TransferBackend) {
	now := time.Now()
	job.CompletedAt = &now

	switch {
	case IsCancellation(err):
		job.Error = &TransferError{Code: CodeUserCancelled, Message: "تم إلغاء عملية النسخ"}
		job.Status = StatusCancelled
		job.Phase = PhaseCancelled
	case AsTransferError(err) != nil:
		job.Error = AsTransferError(err)
		job.Status = StatusFailed
		job.Phase = PhaseFailed
	default:
		job.Error = &TransferError{Code: CodeAFCIOError, Message: err.Error()}
		job.Status = StatusFailed
		job.Phase = PhaseFailed
	}
	w.engine.NotifyUpdate(job)
}

// ---------------------------------------------------------------------------
// small pure helpers
// ---------------------------------------------------------------------------

func isTerminal(job *TransferJobV2) bool {
	return job.Status == StatusCompleted || job.Status == StatusFailed || job.Status == StatusCancelled
}

func sumFileSizes(files []TransferFile) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

func retryBackoff(cfg TransferConfig, attempt int) time.Duration {
	wait := cfg.RetryInitialDelay
	for i := 1; i < attempt; i++ {
		wait *= 2
	}
	if wait > cfg.RetryMaxDelay {
		wait = cfg.RetryMaxDelay
	}
	return wait
}
