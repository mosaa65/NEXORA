package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWalkScansAllRootsAndSerializesEmit(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	for _, file := range []string{
		filepath.Join(rootA, "one.mp4"),
		filepath.Join(rootA, "nested", "two.mkv"),
		filepath.Join(rootB, "three.avi"),
		filepath.Join(rootB, "ignore.txt"),
	} {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("media"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := New(Options{Workers: 3})
	var mu sync.Mutex
	paths := make(map[string]struct{})
	inEmit := false
	err := scanner.Walk(context.Background(), []string{rootA, rootB}, func(file FileInfo) error {
		mu.Lock()
		defer mu.Unlock()
		if inEmit {
			t.Fatal("emit must not be called concurrently")
		}
		inEmit = true
		defer func() { inEmit = false }()
		paths[filepath.Base(file.Path)] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(paths) != 3 {
		t.Fatalf("found %d media files, want 3: %#v", len(paths), paths)
	}
}

func TestWalkStopsWhenEmitterReturnsError(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one.mp4", "two.mp4"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("media"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want := errors.New("stop ingest")
	err := New(Options{Workers: 2}).Walk(context.Background(), []string{root}, func(FileInfo) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Walk error = %v, want %v", err, want)
	}
}

func TestParsePathUsesFolderContext(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Series", "Better Call Saul", "Season 02", "S02E03.1080p.mkv")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("media"), 0o644); err != nil {
		t.Fatal(err)
	}

	file, err := New(Options{}).ParsePath(path)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}
	if file.Parsed.Title != "Better Call Saul" {
		t.Errorf("title = %q, want folder title", file.Parsed.Title)
	}
	if file.Parsed.SeasonNumber != 2 || file.Parsed.EpisodeNumber != 3 {
		t.Errorf("season/episode = %d/%d, want 2/3", file.Parsed.SeasonNumber, file.Parsed.EpisodeNumber)
	}
}
