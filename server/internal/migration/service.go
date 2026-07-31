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
	"strconv"
	"sort"
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
	Sources []string `json:"sources"`
	Target  string   `json:"target"`
}

type CopyItem struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Bytes    int64  `json:"bytes"`
	Checksum string `json:"checksum"`
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
		checksum, copied, err := copyWithChecksum(ctx, source, target)
		if err != nil {
			return result, err
		}
		result.TotalBytes += info.Size()
		result.CompletedBytes += copied
		result.Items = append(result.Items, CopyItem{
			Source:   source,
			Target:   target,
			Bytes:    copied,
			Checksum: checksum,
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

func copyWithChecksum(ctx context.Context, source, target string) (string, int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", 0, err
	}

	in, err := os.Open(source)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()

	out, err := os.Create(target)
	if err != nil {
		return "", 0, err
	}
	defer out.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(out, hasher), in)
	if err != nil {
		return "", 0, err
	}
	if err := out.Sync(); err != nil {
		return "", 0, err
	}
	select {
	case <-ctx.Done():
		return "", 0, ctx.Err()
	default:
	}
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
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
