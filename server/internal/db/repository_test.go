package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheLocalArtwork(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := t.TempDir()
	sourceImage := filepath.Join(sourceDir, "poster.jpg")

	// Create dummy source image
	if err := os.WriteFile(sourceImage, []byte("fake-jpeg-binary-content"), 0o644); err != nil {
		t.Fatalf("create source image: %v", err)
	}

	repo := NewRepository(nil)
	repo.SetAssetImageDir(tempDir)

	// 1. First cache attempt should copy image and return local URL
	cachedURL := repo.CacheLocalArtwork(sourceImage)
	if cachedURL == "" {
		t.Fatal("expected cached URL, got empty string")
	}

	expectedPrefix := "/assets/images/local/local_"
	if len(cachedURL) < len(expectedPrefix) || cachedURL[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("unexpected cached URL format: %s", cachedURL)
	}

	// 2. Empty source should return empty string
	if empty := repo.CacheLocalArtwork(""); empty != "" {
		t.Errorf("expected empty string for empty source, got: %s", empty)
	}

	// 3. Fallback when assetImageDir is empty
	repoNoAssetDir := NewRepository(nil)
	fallbackURL := repoNoAssetDir.CacheLocalArtwork(sourceImage)
	if fallbackURL == "" {
		t.Fatal("expected stream URL fallback, got empty string")
	}
}

func TestRepositoryStructInitialization(t *testing.T) {
	repo := NewRepository(nil)
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
	repo.SetAssetImageDir("custom/dir")
	if repo.assetImageDir != "custom/dir" {
		t.Errorf("expected assetImageDir to be 'custom/dir', got: %s", repo.assetImageDir)
	}
}
