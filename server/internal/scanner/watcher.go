package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type EventKind string

const (
	EventCreated  EventKind = "created"
	EventModified EventKind = "modified"
	EventRemoved  EventKind = "removed"
)

type Event struct {
	Kind EventKind `json:"kind"`
	Path string    `json:"path"`
	File *FileInfo `json:"file,omitempty"`
}

type EventWatcher struct {
	scanner   *Scanner
	recursive bool
}

func NewEventWatcher(scanner *Scanner, recursive bool) *EventWatcher {
	return &EventWatcher{scanner: scanner, recursive: recursive}
}

func (w *EventWatcher) Watch(ctx context.Context, roots []string, handle func(Event) error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, root := range roots {
		if err := w.addRoot(watcher, root); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-watcher.Errors:
			if err != nil {
				return err
			}
		case event := <-watcher.Events:
			if err := w.handleFSNotifyEvent(watcher, event, handle); err != nil {
				return err
			}
		}
	}
}

func (w *EventWatcher) addRoot(watcher *fsnotify.Watcher, root string) error {
	if !w.recursive {
		return watcher.Add(root)
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func (w *EventWatcher) handleFSNotifyEvent(watcher *fsnotify.Watcher, event fsnotify.Event, handle func(Event) error) error {
	if event.Op&fsnotify.Remove == fsnotify.Remove {
		return handle(Event{Kind: EventRemoved, Path: event.Name})
	}

	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return nil
	}

	info, err := os.Stat(event.Name)
	if err != nil {
		return nil
	}

	if info.IsDir() {
		if w.recursive && event.Op&fsnotify.Create == fsnotify.Create {
			return w.addRoot(watcher, event.Name)
		}
		return nil
	}

	if !w.scanner.IsVideoFile(event.Name) {
		return nil
	}

	file := w.scanner.fileInfo(event.Name, info)
	kind := EventModified
	if event.Op&fsnotify.Create == fsnotify.Create {
		kind = EventCreated
	}
	return handle(Event{Kind: kind, Path: event.Name, File: &file})
}
