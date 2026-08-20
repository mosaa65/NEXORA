package scanner

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchKeepsMissingRootPendingUntilContextCancelled(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "removable-media")
	watcher := NewEventWatcher(New(Options{}), false)
	watcher.retryInterval = 5 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := watcher.Watch(ctx, []string{missingRoot}, func(Event) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Watch error = %v, want context deadline", err)
	}
}
