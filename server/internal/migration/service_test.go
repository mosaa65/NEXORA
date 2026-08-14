package migration

import (
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
