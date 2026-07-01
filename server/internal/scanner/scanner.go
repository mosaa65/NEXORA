package scanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Workers    int
	Extensions []string
}

type Scanner struct {
	extensions map[string]struct{}
	workers    int
}

type FileInfo struct {
	Path      string     `json:"path"`
	Size      int64      `json:"size"`
	ModTime   time.Time  `json:"modTime"`
	Parsed    ParsedName `json:"parsed"`
	Extension string     `json:"extension"`
}

var defaultVideoExtensions = []string{
	".mp4",
	".mkv",
	".avi",
	".mov",
	".wmv",
	".m4v",
	".webm",
	".ts",
	".m2ts",
}

func New(options Options) *Scanner {
	extensions := options.Extensions
	if len(extensions) == 0 {
		extensions = defaultVideoExtensions
	}

	extensionSet := make(map[string]struct{}, len(extensions))
	for _, extension := range extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		extensionSet[extension] = struct{}{}
	}

	workers := options.Workers
	if workers <= 0 {
		workers = 4
	}

	return &Scanner{extensions: extensionSet, workers: workers}
}

func (s *Scanner) Scan(ctx context.Context, roots []string) ([]FileInfo, error) {
	files := make([]FileInfo, 0, 1024)
	err := s.Walk(ctx, roots, func(file FileInfo) error {
		files = append(files, file)
		return nil
	})
	return files, err
}

func (s *Scanner) Walk(ctx context.Context, roots []string, emit func(FileInfo) error) error {
	if len(roots) == 0 {
		return errors.New("at least one media root is required")
	}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if err := s.walkRoot(ctx, root, emit); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scanner) IsVideoFile(path string) bool {
	_, ok := s.extensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

func (s *Scanner) ParsePath(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	if info.IsDir() {
		return FileInfo{}, fs.ErrInvalid
	}
	return s.fileInfo(path, info), nil
}

func (s *Scanner) walkRoot(ctx context.Context, root string, emit func(FileInfo) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() || !s.IsVideoFile(path) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		return emit(s.fileInfo(path, info))
	})
}

func (s *Scanner) fileInfo(path string, info fs.FileInfo) FileInfo {
	return FileInfo{
		Path:      path,
		Size:      info.Size(),
		ModTime:   info.ModTime().UTC(),
		Parsed:    ParseFileName(filepath.Base(path)),
		Extension: strings.ToLower(filepath.Ext(path)),
	}
}
