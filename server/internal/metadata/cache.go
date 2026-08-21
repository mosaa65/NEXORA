package metadata

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var unsafeFileChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// CacheImageResult contains the outcome of caching an image
type CacheImageResult struct {
	URL        string // The web-accessible path (e.g. /assets/images/... or remote https://image.tmdb.org/...)
	LocalPath  string // Physical path on disk if saved
	Bytes      int64  // Bytes downloaded
	SHA256     string // SHA-256 checksum
	IsRemote   bool   // True if served directly from CDN without local storage
}

// cacheRemoteImage downloads and saves an image atomically, or returns CDN URL directly if in remote mode
func cacheRemoteImage(ctx context.Context, client *http.Client, imageURL, imageDir, provider, key string, mode ImageMode) (CacheImageResult, error) {
	if imageURL == "" {
		return CacheImageResult{}, nil
	}

	// In remote-only mode, do not download to disk; serve CDN URL directly
	if mode == ImageModeRemote {
		return CacheImageResult{
			URL:      imageURL,
			IsRemote: true,
		}, nil
	}

	if imageDir == "" {
		return CacheImageResult{URL: imageURL, IsRemote: true}, nil
	}

	dir := filepath.Join(imageDir, provider)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, err
	}

	response, err := client.Do(request)
	if err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return CacheImageResult{URL: imageURL, IsRemote: true}, fmt.Errorf("download image %s: status %d", imageURL, response.StatusCode)
	}

	extension := strings.ToLower(filepath.Ext(request.URL.Path))
	if extension == "" || len(extension) > 5 {
		extension = ".jpg"
	}

	fileName := unsafeFileChars.ReplaceAllString(key, "_") + extension
	destination := filepath.Join(dir, fileName)

	// Check if file already exists and has size
	if stat, err := os.Stat(destination); err == nil && stat.Size() > 0 {
		return CacheImageResult{
			URL:       "/assets/images/" + provider + "/" + fileName,
			LocalPath: destination,
			Bytes:     0, // Cached locally already, zero bandwidth consumed
			IsRemote:  false,
		}, nil
	}

	// Atomic save: write to temporary file first then rename
	tempFile, err := os.CreateTemp(dir, "temp_img_*"+extension)
	if err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hasher)
	written, err := io.Copy(multiWriter, response.Body)
	if err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, fmt.Errorf("write image %s: %w", imageURL, err)
	}

	if err := tempFile.Sync(); err != nil {
		return CacheImageResult{URL: imageURL, IsRemote: true}, err
	}
	_ = tempFile.Close()

	// Atomic replacement
	if err := os.Rename(tempPath, destination); err != nil {
		// Fallback for Windows if destination already exists
		_ = os.Remove(destination)
		if renameErr := os.Rename(tempPath, destination); renameErr != nil {
			return CacheImageResult{URL: imageURL, IsRemote: true}, renameErr
		}
	}

	return CacheImageResult{
		URL:       "/assets/images/" + provider + "/" + fileName,
		LocalPath: destination,
		Bytes:     written,
		SHA256:    fmt.Sprintf("%x", hasher.Sum(nil)),
		IsRemote:  false,
	}, nil
}

func yearFromDate(input string) int {
	if len(input) < 4 {
		return 0
	}
	var year int
	if _, err := fmt.Sscanf(input[:4], "%d", &year); err != nil {
		return 0
	}
	return year
}

func parseTimeOrNow(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Now().UTC()
}
