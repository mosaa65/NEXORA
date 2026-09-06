package db

import (
	"context"
	"fmt"
)

// ListRelatedMedia returns a deduplicated database-only view of TMDB's
// recommendation and similar lists. A local match is resolved by provider ID
// and media kind; titles are presentation data and never identity.
func (r *Repository) ListRelatedMedia(ctx context.Context, mediaID int64, limit int) ([]RelatedMedia, error) {
	if limit <= 0 || limit > 48 {
		limit = 18
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH grouped AS (
			SELECT provider, target_external_id, target_kind,
				CASE WHEN BOOL_OR(relation_type = 'recommendation') THEN 'recommendation' ELSE 'similar' END AS relation_type,
				MIN(provider_rank) AS provider_rank,
				MAX(title_ar) FILTER (WHERE title_ar IS NOT NULL AND title_ar <> '') AS title_ar,
				MAX(title_en) FILTER (WHERE title_en IS NOT NULL AND title_en <> '') AS title_en,
				MAX(original_title) FILTER (WHERE original_title IS NOT NULL AND original_title <> '') AS original_title,
				MAX(overview_ar) FILTER (WHERE overview_ar IS NOT NULL AND overview_ar <> '') AS overview_ar,
				MAX(overview_en) FILTER (WHERE overview_en IS NOT NULL AND overview_en <> '') AS overview_en,
				MAX(poster_path) FILTER (WHERE poster_path IS NOT NULL AND poster_path <> '') AS poster_path,
				MAX(release_year) AS release_year, MAX(rating) AS rating
			FROM media_related_titles WHERE source_media_item_id = $1
			GROUP BY provider, target_external_id, target_kind
		)
		SELECT g.provider, g.target_external_id, g.target_kind, g.relation_type,
			COALESCE(local.title_ar, g.title_ar, ''), COALESCE(local.title_en, g.title_en, ''),
			COALESCE(g.original_title, ''), COALESCE(g.overview_ar, ''), COALESCE(g.overview_en, ''),
			COALESCE(local.poster_path, g.poster_path, ''), COALESCE(local.release_year, g.release_year, 0),
			COALESCE(local.rating, g.rating, 0), COALESCE(local.id, 0), COALESCE(local.type, '')
		FROM grouped g
		LEFT JOIN LATERAL (
			SELECT mi.id, mi.title_ar, mi.title_en, mi.poster_path, mi.release_year, mi.rating, mi.type
			FROM media_items mi
			WHERE mi.metadata_provider = g.provider AND mi.metadata_external_id = g.target_external_id
				AND CASE WHEN mi.type IN ('series', 'anime', 'tv') THEN 'tv' ELSE 'movie' END = g.target_kind
			ORDER BY mi.id LIMIT 1
		) local ON true
		ORDER BY CASE WHEN g.relation_type = 'recommendation' THEN 0 ELSE 1 END, g.provider_rank, g.title_en
		LIMIT $2`, mediaID, limit)
	if err != nil {
		return nil, fmt.Errorf("list related media: %w", err)
	}
	defer rows.Close()

	items := make([]RelatedMedia, 0)
	for rows.Next() {
		var item RelatedMedia
		var localID int64
		if err := rows.Scan(&item.Provider, &item.ExternalID, &item.Kind, &item.RelationType,
			&item.TitleAR, &item.TitleEN, &item.OriginalTitle, &item.OverviewAR, &item.OverviewEN,
			&item.PosterPath, &item.ReleaseYear, &item.Rating, &localID, &item.LocalMediaType); err != nil {
			return nil, err
		}
		if localID > 0 {
			item.Local, item.LocalMediaID = true, localID
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
