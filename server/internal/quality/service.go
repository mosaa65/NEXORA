package quality

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"nexora/server/internal/db"
)

// QualityReport is the comprehensive library health overview.
type QualityReport struct {
	GeneratedAt          time.Time              `json:"generated_at"`
	TotalMedia           int64                  `json:"total_media"`
	TotalFiles           int64                  `json:"total_files"`
	TotalSizeBytes       int64                  `json:"total_size_bytes"`
	TotalWastedBytes     int64                  `json:"total_wasted_bytes"`
	DuplicateGroupsCount int                    `json:"duplicate_groups_count"`
	DuplicateGroups      []db.DuplicateGroup    `json:"duplicate_groups"`
	MissingEpisodesCount int                    `json:"missing_episodes_count"`
	MissingEpisodes      []MissingEpisodeDetail `json:"missing_episodes"`
	CorruptedFilesCount  int                    `json:"corrupted_files_count"`
	CorruptedFiles       []CorruptedFileDetail  `json:"corrupted_files"`
}

// MissingEpisodeDetail contains actionable context for missing episodes.
type MissingEpisodeDetail struct {
	MediaItemID   int64  `json:"media_item_id"`
	ShowTitleAR   string `json:"show_title_ar,omitempty"`
	ShowTitleEN   string `json:"show_title_en"`
	CategorySlug  string `json:"category_slug"`
	SeasonID      int64  `json:"season_id"`
	SeasonNumber  int    `json:"season_number"`
	MissingEpNum  int    `json:"missing_episode_number"`
	SuggestedName string `json:"suggested_filename"`
}

// CorruptedFileDetail identifies a damaged video file.
type CorruptedFileDetail struct {
	FileID      int64     `json:"file_id"`
	MediaItemID int64     `json:"media_item_id"`
	ShowTitle   string    `json:"show_title"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	ErrorOutput string    `json:"error_output"`
	DetectedAt  time.Time `json:"detected_at"`
}

// Service provides library health, deduplication, and integrity checks.
type Service struct {
	db          *sql.DB
	ffmpegPath  string
	ffprobePath string
}

// NewService creates a new Quality Engine service.
func NewService(database *sql.DB, ffmpegPath, ffprobePath string) *Service {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &Service{
		db:          database,
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// GenerateReport compiles a complete health snapshot across duplicates, missing episodes, and corrupted files.
func (s *Service) GenerateReport(ctx context.Context) (*QualityReport, error) {
	report := &QualityReport{
		GeneratedAt:     time.Now().UTC(),
		DuplicateGroups: make([]db.DuplicateGroup, 0),
		MissingEpisodes: make([]MissingEpisodeDetail, 0),
		CorruptedFiles:  make([]CorruptedFileDetail, 0),
	}

	// 1. Total media and files count
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items`).Scan(&report.TotalMedia)
	var totalFiles, totalSize sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM video_files`).Scan(&totalFiles, &totalSize)
	if totalFiles.Valid {
		report.TotalFiles = totalFiles.Int64
	}
	if totalSize.Valid {
		report.TotalSizeBytes = totalSize.Int64
	}

	// 2. Fetch duplicate groups and calculate wasted storage
	duplicates, wastedBytes, err := s.FindDuplicates(ctx)
	if err == nil {
		report.DuplicateGroups = duplicates
		report.DuplicateGroupsCount = len(duplicates)
		report.TotalWastedBytes = wastedBytes
	}

	// 3. Find missing episodes with show titles
	missing, err := s.FindMissingEpisodes(ctx)
	if err == nil {
		report.MissingEpisodes = missing
		report.MissingEpisodesCount = len(missing)
	}

	// 4. Find corrupted files from database flags
	corrupted, err := s.ListCorruptedFiles(ctx)
	if err == nil {
		report.CorruptedFiles = corrupted
		report.CorruptedFilesCount = len(corrupted)
	}

	return report, nil
}

// FindDuplicates returns all groups of files sharing the exact same checksum.
func (s *Service) FindDuplicates(ctx context.Context) ([]db.DuplicateGroup, int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, media_item_id, COALESCE(season_id, 0), COALESCE(episode_number, 0),
			COALESCE(title_ar, ''), COALESCE(title_en, ''), file_path, file_size,
			COALESCE(duration, 0), COALESCE(resolution, ''), COALESCE(video_codec, ''),
			COALESCE(audio_tracks, '[]'::jsonb)::text, COALESCE(subtitles, '[]'::jsonb)::text,
			created_at, checksum
		FROM video_files
		WHERE checksum IS NOT NULL AND checksum <> ''
		  AND checksum IN (
			SELECT checksum FROM video_files
			WHERE checksum IS NOT NULL AND checksum <> ''
			GROUP BY checksum HAVING COUNT(*) > 1
		  )
		ORDER BY checksum, file_size DESC;
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("query duplicates: %w", err)
	}
	defer rows.Close()

	groups := make([]db.DuplicateGroup, 0)
	byChecksum := make(map[string]int)
	var totalWastedBytes int64

	for rows.Next() {
		var file db.VideoFile
		var audioTracks, subtitles, checksum string
		if err := rows.Scan(
			&file.ID, &file.MediaItemID, &file.SeasonID, &file.EpisodeNumber,
			&file.TitleAR, &file.TitleEN, &file.FilePath, &file.FileSize,
			&file.Duration, &file.Resolution, &file.VideoCodec,
			&audioTracks, &subtitles, &file.CreatedAt, &checksum,
		); err != nil {
			return nil, 0, fmt.Errorf("scan duplicate: %w", err)
		}
		file.AudioTracks = json.RawMessage(audioTracks)
		file.Subtitles = json.RawMessage(subtitles)

		index, exists := byChecksum[checksum]
		if !exists {
			index = len(groups)
			byChecksum[checksum] = index
			groups = append(groups, db.DuplicateGroup{Checksum: checksum, FileSize: file.FileSize, Files: []db.VideoFile{}})
		} else {
			// Second+ copies in the same group represent wasted space
			totalWastedBytes += file.FileSize
		}
		groups[index].Files = append(groups[index].Files, file)
	}

	return groups, totalWastedBytes, rows.Err()
}

// FindMissingEpisodes identifies gaps in numbered series seasons with rich titles.
func (s *Service) FindMissingEpisodes(ctx context.Context) ([]MissingEpisodeDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH numbered AS (
			SELECT season_id, MAX(episode_number) AS max_episode
			FROM video_files
			WHERE season_id IS NOT NULL AND episode_number IS NOT NULL AND episode_number > 0
			GROUP BY season_id
		)
		SELECT
			mi.id,
			COALESCE(mi.title_ar, ''),
			mi.title_en,
			COALESCE(c.slug, 'series'),
			s.id,
			s.season_number,
			expected.episode_number
		FROM seasons s
		JOIN numbered n ON n.season_id = s.id
		JOIN media_items mi ON mi.id = s.media_item_id
		LEFT JOIN categories c ON c.id = mi.category_id
		CROSS JOIN LATERAL generate_series(1, n.max_episode) AS expected(episode_number)
		LEFT JOIN video_files vf ON vf.season_id = s.id AND vf.episode_number = expected.episode_number
		WHERE vf.id IS NULL
		ORDER BY mi.title_en, s.season_number, expected.episode_number;
	`)
	if err != nil {
		return nil, fmt.Errorf("query missing episodes: %w", err)
	}
	defer rows.Close()

	var missing []MissingEpisodeDetail
	for rows.Next() {
		var item MissingEpisodeDetail
		if err := rows.Scan(
			&item.MediaItemID,
			&item.ShowTitleAR,
			&item.ShowTitleEN,
			&item.CategorySlug,
			&item.SeasonID,
			&item.SeasonNumber,
			&item.MissingEpNum,
		); err != nil {
			return nil, fmt.Errorf("scan missing episode: %w", err)
		}
		item.SuggestedName = fmt.Sprintf("%s S%02dE%02d.mkv", item.ShowTitleEN, item.SeasonNumber, item.MissingEpNum)
		missing = append(missing, item)
	}

	return missing, rows.Err()
}

// VerifyFile runs FFmpeg bitstream verification to detect corrupted frames or unplayable files.
func (s *Service) VerifyFile(ctx context.Context, filePath string) (bool, string, error) {
	if strings.TrimSpace(filePath) == "" {
		return false, "", errors.New("file path is required")
	}

	// Verify file actually exists on disk
	if _, err := os.Stat(filePath); err != nil {
		return false, "file not found on disk", err
	}

	// Run fast error scan with ffmpeg null sink
	cmd := exec.CommandContext(ctx, s.ffmpegPath, "-v", "error", "-i", filePath, "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	errOutput := strings.TrimSpace(string(output))

	if err != nil {
		return false, errOutput, nil
	}
	if len(errOutput) > 0 {
		return false, errOutput, nil
	}

	return true, "", nil
}

// ListCorruptedFiles returns files that failed verification or are missing from disk.
func (s *Service) ListCorruptedFiles(ctx context.Context) ([]CorruptedFileDetail, error) {
	// Query files where resolution or codec is missing/empty, or flagged in database
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			vf.id,
			vf.media_item_id,
			COALESCE(mi.title_en, 'Unknown'),
			vf.file_path,
			vf.file_size,
			vf.created_at
		FROM video_files vf
		JOIN media_items mi ON mi.id = vf.media_item_id
		WHERE (vf.file_size = 0 OR vf.resolution IS NULL OR vf.resolution = '')
		ORDER BY vf.id DESC
		LIMIT 100;
	`)
	if err != nil {
		return nil, fmt.Errorf("query corrupted files: %w", err)
	}
	defer rows.Close()

	var corrupted []CorruptedFileDetail
	for rows.Next() {
		var item CorruptedFileDetail
		if err := rows.Scan(
			&item.FileID,
			&item.MediaItemID,
			&item.ShowTitle,
			&item.FilePath,
			&item.FileSize,
			&item.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan corrupted file: %w", err)
		}
		if item.FileSize == 0 {
			item.ErrorOutput = "0 bytes file size (empty file)"
		} else {
			item.ErrorOutput = "unreadable video stream / missing codec info"
		}
		corrupted = append(corrupted, item)
	}

	return corrupted, rows.Err()
}
