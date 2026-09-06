package transfer

import (
	"context"
	"sync"
)

// Scheduler routes jobs to per-device workers. Each device has exactly one
// worker whose queue is consumed sequentially (PerDeviceConcurrency is fixed
// at 1 by config), while different devices run concurrently.
//
// Workers are created lazily on first use and kept for the lifetime of the
// engine. Re-submitting a job to the same device appends to that worker's
// queue.
type Scheduler struct {
	engine *TransferEngine

	mu      sync.Mutex
	workers map[string]*DeviceWorker
}

func NewScheduler(engine *TransferEngine) *Scheduler {
	return &Scheduler{
		engine:  engine,
		workers: make(map[string]*DeviceWorker),
	}
}

// Submit hands a job to the worker responsible for its destination device.
func (s *Scheduler) Submit(job *TransferJobV2) error {
	s.mu.Lock()
	key := s.workerKey(job.Destination)
	w := s.workers[key]
	if w == nil {
		w = newDeviceWorker(s.engine, key, job.Destination.DeviceID)
		s.workers[key] = w
	}
	s.mu.Unlock()
	return w.Enqueue(job)
}

// workerKey groups jobs by device so jobs targeting the same physical device
// share one worker (and never run simultaneously against the same device /
// AFC session).
func (s *Scheduler) workerKey(d TransferDestination) string {
	if d.DeviceID != "" {
		return "device:" + d.DeviceID
	}
	return "device:" + string(d.DeviceType)
}

// Stop cancels all running workers and clears the registry. It is called when
// the engine shuts down.
func (s *Scheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	workers := make([]*DeviceWorker, 0, len(s.workers))
	for _, w := range s.workers {
		workers = append(workers, w)
	}
	s.mu.Unlock()
	for _, w := range workers {
		w.Stop()
	}
}
