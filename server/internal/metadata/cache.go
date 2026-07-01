package metadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeFileChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func cacheRemoteImage(ctx context.Context, client *http.Client, imageURL, imageDir, provider, key string) (string, error) {
	if imageURL == "" || imageDir == "" {
		return "", nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return "", fmt.Errorf("download image %s: status %d", imageURL, response.StatusCode)
	}

	extension := strings.ToLower(filepath.Ext(request.URL.Path))
	if extension == "" || len(extension) > 5 {
		extension = ".jpg"
	}

	dir := filepath.Join(imageDir, provider)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	fileName := unsafeFileChars.ReplaceAllString(key, "_") + extension
	destination := filepath.Join(dir, fileName)

	file, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		return "", err
	}

	return filepath.ToSlash(destination), nil
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
