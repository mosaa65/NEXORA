package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"nexora/server/internal/search"
)

func (r *Repository) GetSearchDocument(ctx context.Context, id int64) (*search.MediaDocument, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			mi.id,
			mi.title_ar,
			mi.title_en,
			mi.type,
			mi.plot_ar,
			mi.plot_en,
			mi.release_year,
			mi.rating,
			mi.poster_path,
			mi.banner_path,
			COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
			COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
			c.slug,
			c.name_ar,
			c.name_en,
			COALESCE(mi.file_count, 0), mi.status, COALESCE(mi.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0), COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0), COALESCE(mi.total_file_size, 0),
			COALESCE(mi.best_resolution, ''), COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, mi.runtime_minutes, 0),
			COALESCE(mi.has_arabic_audio, false), COALESCE(mi.has_arabic_subtitles, false)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE mi.id = $1;
	`, id)

	var doc search.MediaDocument
	var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN, contentRating sql.NullString
	var releaseYear sql.NullInt64
	var rating sql.NullFloat64
	var genresText string
	var summary mediaCardSummary

	if err := row.Scan(
		&doc.ID,
		&titleAR,
		&doc.TitleEN,
		&doc.Type,
		&plotAR,
		&plotEN,
		&releaseYear,
		&rating,
		&posterPath,
		&bannerPath,
		&genresText,
		&contentRating,
		&categorySlug,
		&categoryAR,
		&categoryEN,
		&doc.FileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount, &summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
		&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get search document %d: %w", id, err)
	}

	doc.TitleAR = nullableString(titleAR)
	doc.PlotAR = nullableString(plotAR)
	doc.PlotEN = nullableString(plotEN)
	doc.PosterPath = nullableString(posterPath)
	doc.BannerPath = nullableString(bannerPath)
	doc.ContentRating = nullableString(contentRating)
	doc.CategorySlug = nullableString(categorySlug)
	doc.CategoryAR = nullableString(categoryAR)
	doc.CategoryEN = nullableString(categoryEN)
	if releaseYear.Valid {
		doc.ReleaseYear = int(releaseYear.Int64)
	}
	if rating.Valid {
		doc.Rating = rating.Float64
	}
	doc.Status = summary.Status
	doc.SeasonCount = summary.SeasonCount
	doc.TMDBSeasonCount = summary.TMDBSeasonCount
	doc.TMDBEpisodeCount = summary.TMDBEpisodeCount
	doc.TotalSize = summary.TotalSize
	doc.BestResolution = summary.BestResolution
	doc.RuntimeMinutes = summary.RuntimeMinutes
	doc.HasArabicAudio = summary.HasArabicAudio
	doc.HasArabicSubtitles = summary.HasArabicSubtitles
	if err := json.Unmarshal([]byte(genresText), &doc.Genres); err != nil {
		doc.Genres = nil
	}

	return &doc, nil
}

func (r *Repository) ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			mi.id,
			mi.title_ar,
			mi.title_en,
			mi.type,
			mi.plot_ar,
			mi.plot_en,
			mi.release_year,
			mi.rating,
			mi.poster_path,
			mi.banner_path,
			COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
			COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
			c.slug,
			c.name_ar,
			c.name_en,
			COALESCE(mi.file_count, 0), mi.status, COALESCE(mi.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0), COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0), COALESCE(mi.total_file_size, 0),
			COALESCE(mi.best_resolution, ''), COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, mi.runtime_minutes, 0),
			COALESCE(mi.has_arabic_audio, false), COALESCE(mi.has_arabic_subtitles, false)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		ORDER BY mi.created_at DESC, mi.id DESC
		LIMIT $1;
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query search documents: %w", err)
	}
	defer rows.Close()

	documents := make([]search.MediaDocument, 0)
	for rows.Next() {
		var doc search.MediaDocument
		var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN, contentRating sql.NullString
		var releaseYear sql.NullInt64
		var rating sql.NullFloat64
		var genresText string
		var fileCount int
		var summary mediaCardSummary

		if err := rows.Scan(
			&doc.ID,
			&titleAR,
			&doc.TitleEN,
			&doc.Type,
			&plotAR,
			&plotEN,
			&releaseYear,
			&rating,
			&posterPath,
			&bannerPath,
			&genresText,
			&contentRating,
			&categorySlug,
			&categoryAR,
			&categoryEN,
			&fileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount, &summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
			&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
		); err != nil {
			return nil, fmt.Errorf("scan search document: %w", err)
		}

		doc.TitleAR = nullableString(titleAR)
		doc.PlotAR = nullableString(plotAR)
		doc.PlotEN = nullableString(plotEN)
		doc.PosterPath = nullableString(posterPath)
		doc.BannerPath = nullableString(bannerPath)
		doc.ContentRating = nullableString(contentRating)
		doc.CategorySlug = nullableString(categorySlug)
		doc.CategoryAR = nullableString(categoryAR)
		doc.CategoryEN = nullableString(categoryEN)
		if releaseYear.Valid {
			doc.ReleaseYear = int(releaseYear.Int64)
		}
		if rating.Valid {
			doc.Rating = rating.Float64
		}
		doc.FileCount = fileCount
		doc.Status = summary.Status
		doc.SeasonCount = summary.SeasonCount
		doc.TMDBSeasonCount = summary.TMDBSeasonCount
		doc.TMDBEpisodeCount = summary.TMDBEpisodeCount
		doc.TotalSize = summary.TotalSize
		doc.BestResolution = summary.BestResolution
		doc.RuntimeMinutes = summary.RuntimeMinutes
		doc.HasArabicAudio = summary.HasArabicAudio
		doc.HasArabicSubtitles = summary.HasArabicSubtitles
		if err := json.Unmarshal([]byte(genresText), &doc.Genres); err != nil {
			doc.Genres = nil
		}

		documents = append(documents, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search documents: %w", err)
	}

	return documents, nil
}




func nullableString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func (r *Repository) GetMediaItem(ctx context.Context, id int64) (*MediaItemDetail, error) {
	var item MediaItemDetail
	var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN, contentRating sql.NullString
	var releaseYear sql.NullInt64
	var rating sql.NullFloat64
	var genresText string
	var categoryID sql.NullInt64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			mi.id,
			mi.category_id,
			mi.title_ar,
			mi.title_en,
			mi.type,
			mi.plot_ar,
			mi.plot_en,
			mi.release_year,
			mi.rating,
			mi.poster_path,
			mi.banner_path,
			COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
			COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
			mi.status,
			mi.created_at,
			c.slug,
			c.name_ar,
			c.name_en
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE mi.id = $1
	`, id).Scan(
		&item.ID,
		&categoryID,
		&titleAR,
		&item.TitleEN,
		&item.Type,
		&plotAR,
		&plotEN,
		&releaseYear,
		&rating,
		&posterPath,
		&bannerPath,
		&genresText,
		&contentRating,
		&item.Status,
		&item.CreatedAt,
		&categorySlug,
		&categoryAR,
		&categoryEN,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get media item %d: %w", id, err)
	}

	if categoryID.Valid {
		item.CategoryID = categoryID.Int64
	}
	item.TitleAR = nullableString(titleAR)
	item.PlotAR = nullableString(plotAR)
	item.PlotEN = nullableString(plotEN)
	item.PosterPath = nullableString(posterPath)
	item.BannerPath = nullableString(bannerPath)
	item.ContentRating = nullableString(contentRating)
	item.CategorySlug = nullableString(categorySlug)
	item.CategoryAR = nullableString(categoryAR)
	item.CategoryEN = nullableString(categoryEN)
	if releaseYear.Valid {
		item.ReleaseYear = int(releaseYear.Int64)
	}
	if rating.Valid {
		item.Rating = rating.Float64
	}
	_ = json.Unmarshal([]byte(genresText), &item.Genres)

	// Fetch seasons if any
	seasonRows, err := r.db.QueryContext(ctx, `
		SELECT id, season_number, COALESCE(title_ar, ''), COALESCE(title_en, '')
		FROM seasons
		WHERE media_item_id = $1
		ORDER BY season_number ASC;
	`, id)
	if err == nil {
		defer seasonRows.Close()
		seasonsMap := make(map[int64]*SeasonDetail)
		for seasonRows.Next() {
			var s SeasonDetail
			if err := seasonRows.Scan(&s.ID, &s.SeasonNumber, &s.TitleAR, &s.TitleEN); err == nil {
				s.Episodes = make([]VideoFile, 0)
				item.Seasons = append(item.Seasons, s)
			}
		}
		for idx := range item.Seasons {
			seasonsMap[item.Seasons[idx].ID] = &item.Seasons[idx]
		}

		// Fetch video files
		files, fileErr := r.ListVideoFiles(ctx, id)
		if fileErr == nil {
			item.FileCount = len(files)
			for i := range files {
				files[i].StreamURL = fmt.Sprintf("/api/stream/file/%d", files[i].ID)
				if files[i].SeasonID > 0 && seasonsMap[files[i].SeasonID] != nil {
					seasonsMap[files[i].SeasonID].Episodes = append(seasonsMap[files[i].SeasonID].Episodes, files[i])
				} else {
					item.Files = append(item.Files, files[i])
				}
			}
		}
	}

	return &item, nil
}

func (r *Repository) ListMediaItems(ctx context.Context, opts ListMediaOptions) (*MediaListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 10000 {
		opts.Limit = 1000
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if strings.TrimSpace(opts.CategorySlug) != "" {
		slug := strings.TrimSpace(opts.CategorySlug)
		if slug == "kids" {
			whereClauses = append(whereClauses, fmt.Sprintf("(c.slug = $%d OR COALESCE(mi.genres::text[], ARRAY[]::text[]) && ARRAY['كرتون','رسوم متحركة','أطفال','ديزني','بيكسار','سبيستون','دريم وركس','animation','Animation']::text[])", argIdx))
			args = append(args, slug)
			argIdx++
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("c.slug = $%d", argIdx))
			args = append(args, slug)
			argIdx++
		}
	}

	if strings.TrimSpace(opts.Type) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.type = $%d", argIdx))
		args = append(args, strings.TrimSpace(opts.Type))
		argIdx++
	}
	if len(opts.IDs) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.id = ANY($%d::bigint[])", argIdx))
		args = append(args, pq.Array(opts.IDs))
		argIdx++
	}
	if len(opts.Types) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.type = ANY($%d::text[])", argIdx))
		args = append(args, pq.Array(opts.Types))
		argIdx++
	}
	if len(opts.Categories) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("c.slug = ANY($%d::text[])", argIdx))
		args = append(args, pq.Array(opts.Categories))
		argIdx++
	}
	if len(opts.TagsAny) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(mi.genres::text[], ARRAY[]::text[]) && $%d::text[]", argIdx))
		args = append(args, pq.Array(opts.TagsAny))
		argIdx++
	}
	if opts.YearFrom > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.release_year >= $%d", argIdx))
		args = append(args, opts.YearFrom)
		argIdx++
	}
	if opts.YearTo > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.release_year <= $%d", argIdx))
		args = append(args, opts.YearTo)
		argIdx++
	}
	if opts.RatingGTE > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("mi.rating >= $%d", argIdx))
		args = append(args, opts.RatingGTE)
		argIdx++
	}

	if strings.TrimSpace(opts.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(opts.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(mi.title_en ILIKE $%d OR mi.title_ar ILIKE $%d)", argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT mi.id)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE %s;
	`, whereSQL)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count media items: %w", err)
	}

	orderBy := "mi.created_at DESC, mi.id DESC"
	switch opts.Sort {
	case "rating":
		orderBy = "mi.rating DESC NULLS LAST, mi.id DESC"
	case "year":
		orderBy = "mi.release_year DESC NULLS LAST, mi.id DESC"
	case "title":
		orderBy = "mi.title_en ASC"
	}

	query := fmt.Sprintf(`
		SELECT
			mi.id,
			mi.title_ar,
			mi.title_en,
			mi.type,
			mi.plot_ar,
			mi.plot_en,
			mi.release_year,
			mi.rating,
			mi.poster_path,
			mi.banner_path,
			COALESCE(array_to_json(mi.genres), '[]'::json)::text AS genres,
			COALESCE(mi.content_rating, mi.metadata_facets->>'content_rating', '') AS content_rating,
			c.slug,
			c.name_ar,
			c.name_en,
			COALESCE(mi.file_count, 0), mi.status, COALESCE(mi.season_count, 0),
			COALESCE(NULLIF(mi.metadata_facets->>'number_of_seasons', '')::int, 0), COALESCE(NULLIF(mi.metadata_facets->>'number_of_episodes', '')::int, 0), COALESCE(mi.total_file_size, 0),
			COALESCE(mi.best_resolution, ''), COALESCE(NULLIF(mi.metadata_facets->>'runtime', '')::int, mi.runtime_minutes, 0),
			COALESCE(mi.has_arabic_audio, false), COALESCE(mi.has_arabic_subtitles, false)
		FROM media_items mi
		LEFT JOIN categories c ON c.id = mi.category_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list media items: %w", err)
	}
	defer rows.Close()

	items := make([]search.MediaDocument, 0)
	for rows.Next() {
		var doc search.MediaDocument
		var titleAR, plotAR, plotEN, posterPath, bannerPath, categorySlug, categoryAR, categoryEN, contentRating sql.NullString
		var releaseYear sql.NullInt64
		var rating sql.NullFloat64
		var genresText string
		var fileCount int
		var summary mediaCardSummary

		if err := rows.Scan(
			&doc.ID,
			&titleAR,
			&doc.TitleEN,
			&doc.Type,
			&plotAR,
			&plotEN,
			&releaseYear,
			&rating,
			&posterPath,
			&bannerPath,
			&genresText,
			&contentRating,
			&categorySlug,
			&categoryAR,
			&categoryEN,
			&fileCount, &summary.Status, &summary.SeasonCount, &summary.TMDBSeasonCount, &summary.TMDBEpisodeCount, &summary.TotalSize, &summary.BestResolution,
			&summary.RuntimeMinutes, &summary.HasArabicAudio, &summary.HasArabicSubtitles,
		); err != nil {
			return nil, fmt.Errorf("scan list item: %w", err)
		}

		doc.TitleAR = nullableString(titleAR)
		doc.PlotAR = nullableString(plotAR)
		doc.PlotEN = nullableString(plotEN)
		doc.PosterPath = nullableString(posterPath)
		doc.BannerPath = nullableString(bannerPath)
		doc.ContentRating = nullableString(contentRating)
		doc.CategorySlug = nullableString(categorySlug)
		doc.CategoryAR = nullableString(categoryAR)
		doc.CategoryEN = nullableString(categoryEN)
		if releaseYear.Valid {
			doc.ReleaseYear = int(releaseYear.Int64)
		}
		if rating.Valid {
			doc.Rating = rating.Float64
		}
		doc.FileCount = fileCount
		doc.Status = summary.Status
		doc.SeasonCount = summary.SeasonCount
		doc.TMDBSeasonCount = summary.TMDBSeasonCount
		doc.TMDBEpisodeCount = summary.TMDBEpisodeCount
		doc.TotalSize = summary.TotalSize
		doc.BestResolution = summary.BestResolution
		doc.RuntimeMinutes = summary.RuntimeMinutes
		doc.HasArabicAudio = summary.HasArabicAudio
		doc.HasArabicSubtitles = summary.HasArabicSubtitles
		_ = json.Unmarshal([]byte(genresText), &doc.Genres)

		items = append(items, doc)
	}

	return &MediaListResult{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
		Items:  items,
	}, nil
}

// ListShowcases returns editorial collections followed by featured local works.
// Both layers are read from PostgreSQL; no remote metadata is contacted here.

func (r *Repository) CreateMediaItem(ctx context.Context, req CreateMediaRequest) (*search.MediaDocument, error) {
	categorySlug := req.CategorySlug
	if categorySlug == "" {
		categorySlug = "movies"
		if req.Type == "series" || req.Type == "anime" {
			categorySlug = req.Type
		}
	}

	var categoryID int64
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, categorySlug).Scan(&categoryID); err != nil {
		return nil, fmt.Errorf("find category %q: %w", categorySlug, err)
	}

	titleEN := strings.TrimSpace(req.TitleEN)
	if titleEN == "" {
		titleEN = req.TitleAR
	}
	if titleEN == "" {
		return nil, errors.New("title is required")
	}

	mediaType := req.Type
	if mediaType == "" {
		mediaType = "movie"
	}

	var genresArray any
	if len(req.Genres) > 0 {
		genresArray = pq.Array(req.Genres)
	}

	var newID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO media_items (
			category_id,
			title_ar,
			title_en,
			type,
			plot_ar,
			plot_en,
			release_year,
			rating,
			poster_path,
			banner_path,
			genres,
			status
		)
		VALUES ($1, NULLIF($2, ''), $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, 0), NULLIF($8, 0.0), NULLIF($9, ''), NULLIF($10, ''), $11, 'completed')
		RETURNING id;
	`, categoryID, req.TitleAR, titleEN, mediaType, req.PlotAR, req.PlotEN, req.ReleaseYear, req.Rating, req.PosterPath, req.BannerPath, genresArray).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("insert media item: %w", err)
	}

	if doc, err := r.GetSearchDocument(ctx, newID); err == nil && doc != nil {
		return doc, nil
	}

	return &search.MediaDocument{
		ID:           newID,
		TitleAR:      req.TitleAR,
		TitleEN:      titleEN,
		Type:         mediaType,
		PlotAR:       req.PlotAR,
		PlotEN:       req.PlotEN,
		ReleaseYear:  req.ReleaseYear,
		Rating:       req.Rating,
		PosterPath:   req.PosterPath,
		BannerPath:   req.BannerPath,
		CategorySlug: categorySlug,
		Genres:       req.Genres,
	}, nil
}

func (r *Repository) UpdateMediaFull(ctx context.Context, id int64, req UpdateMediaRequest) (*search.MediaDocument, error) {
	var categoryID sql.NullInt64
	if req.CategorySlug != "" {
		var catID int64
		if err := r.db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = $1`, req.CategorySlug).Scan(&catID); err == nil {
			categoryID = sql.NullInt64{Int64: catID, Valid: true}
		}
	}

	var genresArray any
	if len(req.Genres) > 0 {
		genresArray = pq.Array(req.Genres)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET
			category_id = COALESCE(NULLIF($1, 0), category_id),
			title_ar = COALESCE(NULLIF($2, ''), title_ar),
			title_en = COALESCE(NULLIF($3, ''), title_en),
			type = COALESCE(NULLIF($4, ''), type),
			plot_ar = COALESCE(NULLIF($5, ''), plot_ar),
			plot_en = COALESCE(NULLIF($6, ''), plot_en),
			release_year = COALESCE(NULLIF($7, 0), release_year),
			rating = COALESCE(NULLIF($8, 0.0), rating),
			poster_path = COALESCE(NULLIF($9, ''), poster_path),
			banner_path = COALESCE(NULLIF($10, ''), banner_path),
			genres = COALESCE($11, genres)
		WHERE id = $12;
	`, categoryID.Int64, req.TitleAR, req.TitleEN, req.Type, req.PlotAR, req.PlotEN, req.ReleaseYear, req.Rating, req.PosterPath, req.BannerPath, genresArray, id)
	if err != nil {
		return nil, fmt.Errorf("update media full: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetSearchDocument(ctx, id)
}

func (r *Repository) DeleteMediaItem(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Delete associated video_files
	if _, err := tx.ExecContext(ctx, `DELETE FROM video_files WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete video files: %w", err)
	}

	// 2. Delete seasons
	if _, err := tx.ExecContext(ctx, `DELETE FROM seasons WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete seasons: %w", err)
	}

	// 3. Delete metadata snapshots
	if _, err := tx.ExecContext(ctx, `DELETE FROM metadata_snapshots WHERE media_item_id = $1`, id); err != nil {
		return fmt.Errorf("delete snapshots: %w", err)
	}

	// 4. Delete media item
	res, err := tx.ExecContext(ctx, `DELETE FROM media_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete media item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}
