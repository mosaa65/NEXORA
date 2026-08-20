package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"nexora/server/internal/scanner"
)

type Service struct {
	extensions map[string]struct{}
}

type Options struct {
	Extensions []string
}

type PreviewRequest struct {
	Root string `json:"root"`
}

type PreviewEntry struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	Kind         string `json:"kind"`
	Action       string `json:"action"`
	Reason       string `json:"reason,omitempty"`
	Exists       bool   `json:"exists"`
	Size         int64  `json:"size"`
	Checksum     string `json:"checksum,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	Title        string `json:"title,omitempty"`
	Episode      int    `json:"episode,omitempty"`
	Season       int    `json:"season,omitempty"`
	SuggestedDir string `json:"suggestedDir,omitempty"`
}

type PreviewResult struct {
	Root           string         `json:"root"`
	Entries        []PreviewEntry `json:"entries"`
	KeepCount      int            `json:"keepCount"`
	MoveCount      int            `json:"moveCount"`
	RenameCount    int            `json:"renameCount"`
	DuplicateCount int            `json:"duplicateCount"`
}

type CopyRequest struct {
	Sources      []string `json:"sources"`
	Target       string   `json:"target"`
	RemoveSource bool     `json:"removeSource,omitempty"`
}

type CopyItem struct {
	Source        string `json:"source"`
	Target        string `json:"target"`
	Bytes         int64  `json:"bytes"`
	Checksum      string `json:"checksum"`
	Resumed       bool   `json:"resumed"`
	AlreadyCopied bool   `json:"alreadyCopied"`
	SourceRemoved bool   `json:"sourceRemoved"`
}

type CopyResult struct {
	Items          []CopyItem `json:"items"`
	TotalBytes     int64      `json:"totalBytes"`
	CompletedBytes int64      `json:"completedBytes"`
}

func New(options Options) *Service {
	extensions := options.Extensions
	if len(extensions) == 0 {
		extensions = []string{".mp4", ".mkv", ".avi", ".mov", ".m4v", ".webm"}
	}
	set := make(map[string]struct{}, len(extensions))
	for _, extension := range extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		set[extension] = struct{}{}
	}
	return &Service{extensions: set}
}

func (s *Service) Preview(ctx context.Context, root string) (PreviewResult, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return PreviewResult{}, errors.New("root is required")
	}

	entries := make([]PreviewEntry, 0, 256)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if entry.IsDir() || !s.isMediaFile(path) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		parsed := parseMediaPlacement(path)
		target := buildSuggestedTarget(root, parsed, filepath.Base(path))
		action, reason := classifyPreviewAction(path, target)
		entries = append(entries, PreviewEntry{
			Source:       path,
			Target:       target,
			Kind:         parsed.kind,
			Action:       action,
			Reason:       reason,
			Exists:       false,
			Size:         info.Size(),
			Title:        parsed.title,
			Season:       parsed.season,
			Episode:      parsed.episode,
			Resolution:   parsed.resolution,
			SuggestedDir: filepath.Dir(target),
		})
		return nil
	})
	if err != nil {
		return PreviewResult{}, fmt.Errorf("preview migration: %w", err)
	}

	targetCounts := make(map[string]int)
	fingerprintCounts := make(map[string]int)
	for _, entry := range entries {
		targetCounts[normalizePath(entry.Target)]++
		fingerprintCounts[entryFingerprint(entry)]++
	}
	for index := range entries {
		if fingerprintCounts[entryFingerprint(entries[index])] > 1 && entries[index].Action != "keep" {
			entries[index].Action = "duplicate"
			entries[index].Reason = "another file has the same name, episode, extension and size"
			continue
		}
		if targetCounts[normalizePath(entries[index].Target)] > 1 && entries[index].Action != "keep" {
			entries[index].Action = "duplicate"
			entries[index].Reason = "another file resolves to the same destination"
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Source) < strings.ToLower(entries[j].Source)
	})
	result := PreviewResult{Root: root, Entries: entries}
	for _, entry := range entries {
		switch entry.Action {
		case "keep":
			result.KeepCount++
		case "move":
			result.MoveCount++
		case "rename":
			result.RenameCount++
		case "duplicate":
			result.DuplicateCount++
		}
	}
	return result, nil
}

func entryFingerprint(entry PreviewEntry) string {
	base := strings.ToLower(filepath.Base(entry.Source))
	return strings.Join([]string{
		normalizePath(entry.Target),
		base,
		strconv.FormatInt(entry.Size, 10),
		strings.ToLower(filepath.Ext(entry.Source)),
	}, "|")
}

func (s *Service) Copy(ctx context.Context, request CopyRequest) (CopyResult, error) {
	if len(request.Sources) == 0 {
		return CopyResult{}, errors.New("sources are required")
	}
	if strings.TrimSpace(request.Target) == "" {
		return CopyResult{}, errors.New("target is required")
	}

	if err := os.MkdirAll(request.Target, 0o755); err != nil {
		return CopyResult{}, err
	}

	var result CopyResult
	for _, source := range request.Sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			return result, err
		}
		if info.IsDir() {
			continue
		}

		target := filepath.Join(request.Target, filepath.Base(source))
		checksum, copied, resumed, alreadyCopied, err := copyWithChecksum(ctx, source, target)
		if err != nil {
			return result, err
		}
		removed := false
		if request.RemoveSource {
			if err := os.Remove(source); err != nil {
				return result, fmt.Errorf("remove verified source %q: %w", source, err)
			}
			removed = true
		}
		result.TotalBytes += info.Size()
		result.CompletedBytes += info.Size()
		result.Items = append(result.Items, CopyItem{
			Source:        source,
			Target:        target,
			Bytes:         copied,
			Checksum:      checksum,
			Resumed:       resumed,
			AlreadyCopied: alreadyCopied,
			SourceRemoved: removed,
		})
	}
	return result, nil
}

type parsedPlacement struct {
	kind       string
	title      string
	season     int
	episode    int
	resolution string
}

func parseMediaPlacement(path string) parsedPlacement {
	parsed := scanner.ParseFileName(filepath.Base(path))
	placement := parsedPlacement{
		kind:       "movie",
		title:      parsed.Title,
		season:     parsed.SeasonNumber,
		episode:    parsed.EpisodeNumber,
		resolution: parsed.Resolution,
	}
	if parsed.IsEpisode {
		placement.kind = "series"
	}
	lower := strings.ToLower(path)
	if strings.Contains(lower, "anime") || strings.Contains(path, "أنمي") {
		placement.kind = "anime"
	}
	if placement.title == "" {
		placement.title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return placement
}

func buildSuggestedTarget(root string, parsed parsedPlacement, baseName string) string {
	group := "Movies"
	if parsed.kind == "series" || parsed.kind == "anime" {
		group = filepath.Join("Series", safePathComponent(parsed.title))
		if parsed.season > 0 {
			group = filepath.Join(group, fmt.Sprintf("Season %02d", parsed.season))
		}
		return filepath.Join(root, group, baseName)
	}
	if parsed.title != "" {
		group = filepath.Join(group, safePathComponent(parsed.title))
	}
	return filepath.Join(root, group, baseName)
}

func classifyPreviewAction(source, target string) (string, string) {
	sourceDir := normalizePath(filepath.Dir(source))
	targetDir := normalizePath(filepath.Dir(target))
	sourceBase := strings.ToLower(filepath.Base(source))
	targetBase := strings.ToLower(filepath.Base(target))

	switch {
	case sourceDir == targetDir && sourceBase == targetBase:
		return "keep", "source already matches target"
	case sourceDir == targetDir && sourceBase != targetBase:
		return "rename", "same folder, different file name"
	case sourceDir != targetDir && sourceBase == targetBase:
		return "move", "file will be moved to organized folder"
	default:
		return "move", "file will be moved and normalized"
	}
}

func normalizePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	return strings.ToLower(filepath.Clean(absolute))
}

func (s *Service) isMediaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := s.extensions[ext]
	return ok
}

const partialCopySuffix = ".nexora-part"

// copyWithChecksum writes to a private partial file, resumes it when possible,
// and atomically publishes the destination only after both file checksums
// match. This deliberately avoids a rename/move until the original is known
// to be intact at the destination.
func copyWithChecksum(ctx context.Context, source, target string) (checksum string, size int64, resumed bool, alreadyCopied bool, err error) {
	sourcePath, err := filepath.Abs(source)
	if err != nil {
		return "", 0, false, false, fmt.Errorf("resolve source path: %w", err)
	}
	targetPath, err := filepath.Abs(target)
	if err != nil {
		return "", 0, false, false, fmt.Errorf("resolve target path: %w", err)
	}
	if normalizePath(sourcePath) == normalizePath(targetPath) {
		return "", 0, false, false, errors.New("source and target must be different files")
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", 0, false, false, err
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", 0, false, false, err
	}
	if sourceInfo.IsDir() {
		return "", 0, false, false, errors.New("source must be a file")
	}

	sourceChecksum, err := checksumFile(ctx, sourcePath)
	if err != nil {
		return "", 0, false, false, fmt.Errorf("checksum source: %w", err)
	}

	if targetChecksum, targetErr := checksumFile(ctx, targetPath); targetErr == nil {
		if targetChecksum != sourceChecksum {
			return "", 0, false, false, fmt.Errorf("target already exists with different checksum: %s", targetPath)
		}
		return sourceChecksum, sourceInfo.Size(), false, true, nil
	} else if !errors.Is(targetErr, os.ErrNotExist) {
		return "", 0, false, false, fmt.Errorf("checksum target: %w", targetErr)
	}

	partialPath := targetPath + partialCopySuffix
	partialInfo, partialErr := os.Stat(partialPath)
	if partialErr != nil && !errors.Is(partialErr, os.ErrNotExist) {
		return "", 0, false, false, partialErr
	}
	resumeAt := int64(0)
	if partialErr == nil {
		if partialInfo.Size() > sourceInfo.Size() {
			return "", 0, false, false, fmt.Errorf("partial copy is larger than source: %s", partialPath)
		}
		resumeAt = partialInfo.Size()
		resumed = resumeAt > 0
	}

	in, err := os.Open(sourcePath)
	if err != nil {
		return "", 0, false, false, err
	}
	defer in.Close()
	if _, err := in.Seek(resumeAt, io.SeekStart); err != nil {
		return "", 0, false, false, err
	}

	out, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", 0, false, false, err
	}
	outClosed := false
	defer func() {
		if !outClosed {
			_ = out.Close()
		}
	}()
	if _, err := out.Seek(resumeAt, io.SeekStart); err != nil {
		return "", 0, false, false, err
	}
	if err := out.Truncate(resumeAt); err != nil {
		return "", 0, false, false, err
	}

	if _, err := copyWithContext(ctx, out, in); err != nil {
		return "", 0, false, false, err
	}
	if err := out.Sync(); err != nil {
		return "", 0, false, false, err
	}
	if err := out.Close(); err != nil {
		return "", 0, false, false, err
	}
	outClosed = true

	partialChecksum, err := checksumFile(ctx, partialPath)
	if err != nil {
		return "", 0, false, false, fmt.Errorf("checksum copied file: %w", err)
	}
	if partialChecksum != sourceChecksum {
		_ = os.Remove(partialPath)
		return "", 0, false, false, errors.New("copied file checksum does not match source")
	}
	if err := os.Rename(partialPath, targetPath); err != nil {
		return "", 0, false, false, fmt.Errorf("publish verified copy: %w", err)
	}
	return sourceChecksum, sourceInfo.Size(), resumed, false, nil
}

func checksumFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := copyWithContext(ctx, hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 1024*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			written, writeErr := destination.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

func safePathComponent(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "Unknown"
	}
	input = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, input)
	return input
}
