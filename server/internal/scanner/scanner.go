package scanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type scanCandidate struct {
	path  string
	entry fs.DirEntry
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
	if emit == nil {
		return errors.New("file callback is required")
	}

	cleanRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		if root = strings.TrimSpace(root); root != "" {
			cleanRoots = append(cleanRoots, root)
		}
	}
	if len(cleanRoots) == 0 {
		return errors.New("at least one media root is required")
	}

	// Walking a large directory tree is I/O bound. Directory discovery and
	// file metadata parsing therefore run independently, while emit stays on
	// one goroutine. The latter keeps existing repository callbacks safe: they
	// can append or write to a database without adding their own locking.
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan scanCandidate, s.workers*2)
	results := make(chan FileInfo, s.workers*2)

	var errOnce sync.Once
	var firstErr error
	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	var walkers sync.WaitGroup
	for _, root := range cleanRoots {
		walkers.Add(1)
		go func(root string) {
			defer walkers.Done()
			if err := s.walkRoot(workCtx, root, jobs); err != nil && !errors.Is(err, context.Canceled) {
				setErr(err)
			}
		}(root)
	}
	go func() {
		walkers.Wait()
		close(jobs)
	}()

	var workers sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				info, err := job.entry.Info()
				if err != nil {
					setErr(err)
					return
				}
				file := s.fileInfo(job.path, info)
				select {
				case results <- file:
				case <-workCtx.Done():
					return
				}
			}
		}()
	}
	go func() {
		workers.Wait()
		close(results)
	}()

	for file := range results {
		if err := emit(file); err != nil {
			setErr(err)
		}
	}

	if firstErr != nil {
		return firstErr
	}
	if err := ctx.Err(); err != nil {
		return err
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

func (s *Scanner) walkRoot(ctx context.Context, root string, jobs chan<- scanCandidate) error {
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

		select {
		case jobs <- scanCandidate{path: path, entry: entry}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (s *Scanner) fileInfo(path string, info fs.FileInfo) FileInfo {
	return FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().UTC(),
		// Folder names often carry the canonical bilingual title and season;
		// relying on the filename alone loses that information for real-world
		// libraries such as "Series/Title/Season 01/E01.mkv".
		Parsed:    ParseFilePath(path),
		Extension: strings.ToLower(filepath.Ext(path)),
	}
}
