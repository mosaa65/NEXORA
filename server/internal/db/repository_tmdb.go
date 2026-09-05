package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"nexora/server/internal/metadata"
)

type TMDBLogEntry struct {
	MediaItemID      *int64
	RequestKind      string
	Endpoint         string
	StatusCode       int
	BytesDownloaded  int64
	ImagesDownloaded int
	DurationMS       int
	ErrorMessage     string
}

type TMDBQueueJob struct {
	ID          int64     `json:"id"`
	MediaItemID int64     `json:"media_item_id"`
	Priority    int       `json:"priority"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *Repository) GetTMDBSettings(ctx context.Context) (*metadata.TMDBSettings, error) {
	var s metadata.TMDBSettings
	var modulesRaw, remoteConfigRaw []byte
	var fetchMode, imageMode, prefLang, fallbackLang, incImgLang string
	var posterSize, backdropSize, profileSize, stillSize string
	var dailyBandwidthMB int64
	var enableRateLimitDelay bool
	var rateLimitReqPerSec int
	var autoRefreshEnabled, refreshOnOpen bool
	var refreshIntervalDays, refreshStaleDays, queueMaxConcurrent int
	var updatedAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT
			fetch_mode, image_mode, preferred_language, fallback_language, include_image_language,
			daily_bandwidth_mb, enable_rate_limit_delay, rate_limit_requests_per_sec,
			poster_size, backdrop_size, profile_size, still_size,
			auto_refresh_enabled, refresh_interval_days, refresh_on_open, refresh_stale_days, queue_max_concurrent,
			modules, remote_config, updated_at
		FROM tmdb_settings
		WHERE id = 1;
	`).Scan(
		&fetchMode, &imageMode, &prefLang, &fallbackLang, &incImgLang,
		&dailyBandwidthMB, &enableRateLimitDelay, &rateLimitReqPerSec,
		&posterSize, &backdropSize, &profileSize, &stillSize,
		&autoRefreshEnabled, &refreshIntervalDays, &refreshOnOpen, &refreshStaleDays, &queueMaxConcurrent,
		&modulesRaw, &remoteConfigRaw, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			defaults := metadata.DefaultSettings()
			_ = r.SaveTMDBSettings(ctx, defaults)
			return &defaults, nil
		}
		return nil, fmt.Errorf("get tmdb settings: %w", err)
	}

	s.FetchMode = metadata.FetchMode(fetchMode)
	s.ImageMode = metadata.ImageMode(imageMode)
	s.PreferredLanguage = prefLang
	s.FallbackLanguage = fallbackLang
	s.IncludeImageLanguage = incImgLang
	s.DailyBandwidthMB = dailyBandwidthMB
	s.EnableRateLimitDelay = enableRateLimitDelay
	s.RateLimitRequestsPerSec = rateLimitReqPerSec
	s.PosterSize = posterSize
	s.BackdropSize = backdropSize
	s.ProfileSize = profileSize
	s.StillSize = stillSize
	s.AutoRefreshEnabled = autoRefreshEnabled
	s.RefreshIntervalDays = refreshIntervalDays
	s.RefreshOnOpen = refreshOnOpen
	s.RefreshStaleDays = refreshStaleDays
	s.QueueMaxConcurrent = queueMaxConcurrent
	s.UpdatedAt = updatedAt

	if len(modulesRaw) > 0 {
		_ = json.Unmarshal(modulesRaw, &s.Modules)
	}

	return &s, nil
}

func (r *Repository) SaveTMDBSettings(ctx context.Context, s metadata.TMDBSettings) error {
	modulesJSON, err := json.Marshal(s.Modules)
	if err != nil {
		return fmt.Errorf("marshal modules: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tmdb_settings (
			id, fetch_mode, image_mode, preferred_language, fallback_language, include_image_language,
			daily_bandwidth_mb, enable_rate_limit_delay, rate_limit_requests_per_sec,
			poster_size, backdrop_size, profile_size, still_size, auto_refresh_enabled, refresh_interval_days, refresh_on_open, refresh_stale_days, queue_max_concurrent, modules, updated_at
		) VALUES (
			1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18::jsonb, CURRENT_TIMESTAMP
		)
		ON CONFLICT (id) DO UPDATE SET
			fetch_mode = EXCLUDED.fetch_mode,
			image_mode = EXCLUDED.image_mode,
			preferred_language = EXCLUDED.preferred_language,
			fallback_language = EXCLUDED.fallback_language,
			include_image_language = EXCLUDED.include_image_language,
			daily_bandwidth_mb = EXCLUDED.daily_bandwidth_mb,
			enable_rate_limit_delay = EXCLUDED.enable_rate_limit_delay,
			rate_limit_requests_per_sec = EXCLUDED.rate_limit_requests_per_sec,
			poster_size = EXCLUDED.poster_size,
			backdrop_size = EXCLUDED.backdrop_size,
			profile_size = EXCLUDED.profile_size,
			still_size = EXCLUDED.still_size,
			modules = EXCLUDED.modules,
			updated_at = CURRENT_TIMESTAMP;
	`,
		string(s.FetchMode), string(s.ImageMode), s.PreferredLanguage, s.FallbackLanguage, s.IncludeImageLanguage,
		s.DailyBandwidthMB, s.EnableRateLimitDelay, s.RateLimitRequestsPerSec,
		s.PosterSize, s.BackdropSize, s.ProfileSize, s.StillSize, s.AutoRefreshEnabled, s.RefreshIntervalDays, s.RefreshOnOpen, s.RefreshStaleDays, s.QueueMaxConcurrent, string(modulesJSON),
	)

	if err != nil {
		return fmt.Errorf("save tmdb settings: %w", err)
	}
	return nil
}

func (r *Repository) EnqueueTMDBRefresh(ctx context.Context, mediaID int64, priority int) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tmdb_refresh_queue (media_item_id,priority) VALUES ($1,$2) ON CONFLICT DO NOTHING`, mediaID, priority)
	return err
}

func (r *Repository) EnqueueStaleTMDBRefreshes(ctx context.Context, staleDays, limit int) error {
	if staleDays < 1 {
		staleDays = 30
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO tmdb_refresh_queue (media_item_id,priority) SELECT id,0 FROM media_items WHERE metadata_provider IS NULL OR metadata_fetched_at IS NULL OR metadata_fetched_at < CURRENT_TIMESTAMP - ($1 * INTERVAL '1 day') ORDER BY metadata_fetched_at NULLS FIRST,id LIMIT $2 ON CONFLICT DO NOTHING`, staleDays, limit)
	return err
}

func (r *Repository) EnqueueTMDBRefreshIfStale(ctx context.Context, mediaID int64, staleDays int) error {
	if staleDays < 1 {
		staleDays = 7
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO tmdb_refresh_queue (media_item_id,priority) SELECT id,20 FROM media_items WHERE id=$1 AND (metadata_provider IS NULL OR metadata_fetched_at IS NULL OR metadata_fetched_at < CURRENT_TIMESTAMP - ($2 * INTERVAL '1 day')) ON CONFLICT DO NOTHING`, mediaID, staleDays)
	return err
}

func (r *Repository) ListTMDBQueue(ctx context.Context, limit int) ([]TMDBQueueJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,media_item_id,priority,status,attempts,COALESCE(last_error,''),scheduled_at,created_at FROM tmdb_refresh_queue ORDER BY CASE status WHEN 'running' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END,priority DESC,scheduled_at,id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]TMDBQueueJob, 0)
	for rows.Next() {
		var job TMDBQueueJob
		if err := rows.Scan(&job.ID, &job.MediaItemID, &job.Priority, &job.Status, &job.Attempts, &job.LastError, &job.ScheduledAt, &job.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) CancelTMDBQueueJob(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE tmdb_refresh_queue SET status='cancelled',finished_at=CURRENT_TIMESTAMP WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ClaimTMDBQueueJob(ctx context.Context) (*TMDBQueueJob, error) {
	var job TMDBQueueJob
	err := r.db.QueryRowContext(ctx, `UPDATE tmdb_refresh_queue SET status='running',attempts=attempts+1,started_at=CURRENT_TIMESTAMP WHERE id=(SELECT id FROM tmdb_refresh_queue WHERE status='pending' AND scheduled_at <= CURRENT_TIMESTAMP ORDER BY priority DESC,scheduled_at,id FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING id,media_item_id,priority,status,attempts,COALESCE(last_error,''),scheduled_at,created_at`).Scan(&job.ID, &job.MediaItemID, &job.Priority, &job.Status, &job.Attempts, &job.LastError, &job.ScheduledAt, &job.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) FinishTMDBQueueJob(ctx context.Context, id int64, succeeded bool, message string) error {
	if succeeded {
		_, err := r.db.ExecContext(ctx, `UPDATE tmdb_refresh_queue SET status='succeeded',last_error=NULLIF($1,''),finished_at=CURRENT_TIMESTAMP WHERE id=$2`, message, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE tmdb_refresh_queue SET status=CASE WHEN attempts < 3 THEN 'pending' ELSE 'failed' END,last_error=NULLIF($1,''),scheduled_at=CASE WHEN attempts < 3 THEN CURRENT_TIMESTAMP + (POWER(2, attempts) * INTERVAL '1 minute') ELSE scheduled_at END,finished_at=CASE WHEN attempts < 3 THEN NULL ELSE CURRENT_TIMESTAMP END WHERE id=$2`, message, id)
	return err
}

func (r *Repository) LogTMDBUsage(ctx context.Context, entry TMDBLogEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tmdb_usage_log (
			media_item_id, request_kind, endpoint, status_code, bytes_downloaded, images_downloaded, duration_ms, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''));
	`, entry.MediaItemID, entry.RequestKind, entry.Endpoint, entry.StatusCode, entry.BytesDownloaded, entry.ImagesDownloaded, entry.DurationMS, entry.ErrorMessage)
	return err
}

func (r *Repository) GetTMDBUsageSummary(ctx context.Context) (*metadata.TMDBUsageSummary, error) {
	summary := &metadata.TMDBUsageSummary{}

	// 1. Overall & monthly request counts and total bytes
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(bytes_downloaded), 0),
			COALESCE(SUM(images_downloaded), 0),
			MAX(created_at)
		FROM tmdb_usage_log;
	`).Scan(&summary.TotalRequests, &summary.TotalBytesDownloaded, &summary.TotalImagesDownloaded, &summary.LastRequestAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("query total tmdb stats: %w", err)
	}

	// 2. Today's stats (since midnight UTC)
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(bytes_downloaded), 0),
			COALESCE(SUM(images_downloaded), 0)
		FROM tmdb_usage_log
		WHERE created_at >= CURRENT_DATE;
	`).Scan(&summary.RequestsToday, &summary.BytesToday, &summary.ImagesToday)

	summary.MBToday = float64(summary.BytesToday) / (1024.0 * 1024.0)

	// 3. This month's stats
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tmdb_usage_log
		WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE);
	`).Scan(&summary.RequestsThisMonth)

	// 4. Enriched vs Pending Media items count
	_ = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN metadata_provider IS NOT NULL AND metadata_provider != '' THEN 1 END),
			COUNT(CASE WHEN metadata_provider IS NULL OR metadata_provider = '' THEN 1 END)
		FROM media_items;
	`).Scan(&summary.EnrichedMediaCount, &summary.PendingMediaCount)

	// 5. Get configured quota
	settings, err := r.GetTMDBSettings(ctx)
	if err == nil && settings != nil {
		summary.DailyQuotaMB = settings.DailyBandwidthMB
		if summary.DailyQuotaMB > 0 {
			summary.DailyQuotaUsedPercent = (summary.MBToday / float64(summary.DailyQuotaMB)) * 100.0
			if summary.DailyQuotaUsedPercent > 100.0 {
				summary.DailyQuotaUsedPercent = 100.0
			}
		}
	}

	return summary, nil
}

func (r *Repository) GetTMDBUsageHistory(ctx context.Context, days int) ([]metadata.TMDBUsageDay, error) {
	if days < 1 || days > 730 {
		days = 90
	}
	rows, err := r.db.QueryContext(ctx, `SELECT created_at::date,COUNT(*),COALESCE(SUM(bytes_downloaded),0),COALESCE(SUM(images_downloaded),0),COUNT(*) FILTER (WHERE status_code BETWEEN 200 AND 299),COUNT(*) FILTER (WHERE status_code < 200 OR status_code >= 300) FROM tmdb_usage_log WHERE created_at >= CURRENT_DATE - ($1 * INTERVAL '1 day') GROUP BY created_at::date ORDER BY created_at::date DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]metadata.TMDBUsageDay, 0)
	for rows.Next() {
		var day time.Time
		var item metadata.TMDBUsageDay
		if err := rows.Scan(&day, &item.Requests, &item.BytesDownloaded, &item.ImagesDownloaded, &item.Successful, &item.Failed); err != nil {
			return nil, err
		}
		item.Day = day.Format("2006-01-02")
		item.MBDownloaded = float64(item.BytesDownloaded) / (1024 * 1024)
		history = append(history, item)
	}
	return history, rows.Err()
}
