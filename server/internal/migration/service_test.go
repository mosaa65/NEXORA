package migration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationPreviewAndCopy(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	movieFile := filepath.Join(sourceDir, "Inception.2010.1080p.mkv")
	if err := os.WriteFile(movieFile, []byte("fake video data for movie"), 0644); err != nil {
		t.Fatalf("failed to write movie file: %v", err)
	}

	seriesFile := filepath.Join(sourceDir, "Attack.on.Titan.S04E05.1080p.mp4")
	if err := os.WriteFile(seriesFile, []byte("fake video data for anime"), 0644); err != nil {
		t.Fatalf("failed to write anime file: %v", err)
	}

	svc := New(Options{})
	ctx := context.Background()

	preview, err := svc.Preview(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	if len(preview.Entries) != 2 {
		t.Fatalf("expected 2 entries in preview, got %d", len(preview.Entries))
	}

	copyReq := CopyRequest{
		Sources: []string{movieFile, seriesFile},
		Target:  targetDir,
	}

	copyResult, err := svc.Copy(ctx, copyReq)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if len(copyResult.Items) != 2 {
		t.Fatalf("expected 2 copied items, got %d", len(copyResult.Items))
	}

	for _, item := range copyResult.Items {
		if item.Checksum == "" {
			t.Errorf("expected checksum for item %s, got empty string", item.Target)
		}
		if _, err := os.Stat(item.Target); err != nil {
			t.Errorf("target file not created: %s", item.Target)
		}
	}
}

func TestCopyResumesPartialFileAndCanRemoveVerifiedSource(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	payload := bytes.Repeat([]byte("nexora-media"), 200000)
	source := filepath.Join(sourceDir, "archive.mkv")
	if err := os.WriteFile(source, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, filepath.Base(source))
	if err := os.WriteFile(target+partialCopySuffix, payload[:len(payload)/2], 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := New(Options{}).Copy(context.Background(), CopyRequest{
		Sources:      []string{source},
		Target:       targetDir,
		RemoveSource: true,
	})
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	if len(result.Items) != 1 || !result.Items[0].Resumed || !result.Items[0].SourceRemoved {
		t.Fatalf("unexpected copy result: %#v", result.Items)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("published file does not match source content")
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source must be removed only after verification, stat error: %v", err)
	}
	if _, err := os.Stat(target + partialCopySuffix); !os.IsNotExist(err) {
		t.Fatalf("partial file must be published or removed, stat error: %v", err)
	}
}

func TestCopyRefusesToOverwriteDifferentDestination(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "movie.mp4")
	target := filepath.Join(targetDir, "movie.mp4")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("existing different file"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := New(Options{}).Copy(context.Background(), CopyRequest{Sources: []string{source}, Target: targetDir})
	if err == nil {
		t.Fatal("Copy must refuse to overwrite an unrelated destination")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "existing different file" {
		t.Fatal("existing destination was modified")
	}
}
