package transfer

import "sync"

// TransferEventType distinguishes the kinds of live events pushed to SSE
// subscribers of the transfer service.
type TransferEventType string

const (
	// EventJobs is a full snapshot of the jobs list (sent on subscribe).
	EventJobs TransferEventType = "jobs"
	// EventJob is a single job update (progress, phase, status...).
	EventJob TransferEventType = "job"
	// EventDevices is a full snapshot of the currently connected devices
	// (sent on subscribe and whenever the device set changes).
	EventDevices TransferEventType = "devices"
)

// TransferEvent is a single push event delivered to real-time subscribers
// (e.g. an SSE endpoint). Only one payload field is populated per type.
type TransferEvent struct {
	Type    TransferEventType `json:"type"`
	Job     *TransferJob      `json:"job,omitempty"`
	Jobs    []*TransferJob    `json:"jobs,omitempty"`
	Devices []Device          `json:"devices,omitempty"`
}

// eventBroker fans TransferEvents out to all registered subscribers. Subscriber
// channels are buffered and events are dropped (never blocked) when a slow
// subscriber falls behind, which keeps live progress flowing for everyone else.
type eventBroker struct {
	mu   sync.RWMutex
	subs map[chan TransferEvent]struct{}
}

func newEventBroker() *eventBroker {
	return &eventBroker{subs: make(map[chan TransferEvent]struct{})}
}

// subscribe registers a new subscriber. The returned cancel function removes
// and closes the channel; callers must invoke it to release the subscription.
func (b *eventBroker) subscribe(buffer int) (<-chan TransferEvent, func()) {
	ch := make(chan TransferEvent, buffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}

func (b *eventBroker) publish(ev TransferEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- ev:
		default: // drop for slow/backlogged subscriber
		}
	}
}
