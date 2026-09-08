package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"nexora/server/internal/search"
)

func (r *Repository) ListProviderCollections(ctx context.Context, limit int) ([]ProviderCollection, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,slug,provider,external_id,kind,COALESCE(title_ar,''),title_en,
			COALESCE(overview_ar,''),COALESCE(overview_en,''),COALESCE(poster_path,''),COALESCE(backdrop_path,''),
			parts_count,COALESCE(rating,0),(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=provider_collections.id),is_featured,is_hidden
		FROM provider_collections
		WHERE is_hidden=false AND (SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=provider_collections.id) >= 1
		ORDER BY is_featured DESC,sort_priority DESC,(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=provider_collections.id) DESC,title_en ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list provider collections: %w", err)
	}
	defer rows.Close()
	items := make([]ProviderCollection, 0)
	for rows.Next() {
		var item ProviderCollection
		if err := rows.Scan(&item.ID, &item.Slug, &item.Provider, &item.ExternalID, &item.Kind, &item.TitleAR, &item.TitleEN, &item.OverviewAR, &item.OverviewEN, &item.PosterPath, &item.BackdropPath, &item.PartsCount, &item.Rating, &item.LocalItemCount, &item.IsFeatured, &item.IsHidden); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetProviderCollection(ctx context.Context, slug string) (*ProviderCollection, error) {
	var item ProviderCollection
	err := r.db.QueryRowContext(ctx, `SELECT id,slug,provider,external_id,kind,COALESCE(title_ar,''),title_en,COALESCE(overview_ar,''),COALESCE(overview_en,''),COALESCE(poster_path,''),COALESCE(backdrop_path,''),parts_count,COALESCE(rating,0),(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=provider_collections.id),is_featured,is_hidden FROM provider_collections WHERE slug=$1 AND is_hidden=false`, strings.TrimSpace(slug)).Scan(&item.ID, &item.Slug, &item.Provider, &item.ExternalID, &item.Kind, &item.TitleAR, &item.TitleEN, &item.OverviewAR, &item.OverviewEN, &item.PosterPath, &item.BackdropPath, &item.PartsCount, &item.Rating, &item.LocalItemCount, &item.IsFeatured, &item.IsHidden)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetProviderCollectionByID(ctx context.Context, id int64) (*ProviderCollection, error) {
	var item ProviderCollection
	err := r.db.QueryRowContext(ctx, `SELECT id,slug,provider,external_id,kind,COALESCE(title_ar,''),title_en,COALESCE(overview_ar,''),COALESCE(overview_en,''),COALESCE(poster_path,''),COALESCE(backdrop_path,''),parts_count,COALESCE(rating,0),(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=provider_collections.id),is_featured,is_hidden FROM provider_collections WHERE id=$1`, id).Scan(&item.ID, &item.Slug, &item.Provider, &item.ExternalID, &item.Kind, &item.TitleAR, &item.TitleEN, &item.OverviewAR, &item.OverviewEN, &item.PosterPath, &item.BackdropPath, &item.PartsCount, &item.Rating, &item.LocalItemCount, &item.IsFeatured, &item.IsHidden)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ProviderCollectionNeedsRefresh prevents repeated collection downloads while
// ensuring older lightweight collection links are upgraded on first refresh.
func (r *Repository) ProviderCollectionNeedsRefresh(ctx context.Context, externalID string) (bool, error) {
	var needs bool
	err := r.db.QueryRowContext(ctx, `SELECT NOT EXISTS (SELECT 1 FROM collection_metadata_snapshots cms JOIN provider_collections pc ON pc.id=cms.collection_id WHERE pc.provider='tmdb' AND pc.external_id=$1 AND cms.locale LIKE 'en%' AND (cms.expires_at IS NULL OR cms.expires_at > CURRENT_TIMESTAMP))`, strings.TrimSpace(externalID)).Scan(&needs)
	return needs, err
}

// ListProviderCollectionRefreshCandidates is used by an explicit admin
// maintenance action. It skips collections already carrying a fresh full
// English snapshot, so it never re-downloads data unnecessarily.
func (r *Repository) ListProviderCollectionRefreshCandidates(ctx context.Context, limit int) ([]ProviderCollection, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT pc.id,pc.slug,pc.provider,pc.external_id,pc.kind,COALESCE(pc.title_ar,''),pc.title_en,
			COALESCE(pc.overview_ar,''),COALESCE(pc.overview_en,''),COALESCE(pc.poster_path,''),COALESCE(pc.backdrop_path,''),
			pc.parts_count,COALESCE(pc.rating,0),(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=pc.id),pc.is_featured,pc.is_hidden
		FROM provider_collections pc
		WHERE pc.provider='tmdb' AND NOT EXISTS (
			SELECT 1 FROM collection_metadata_snapshots cms
			WHERE cms.collection_id=pc.id AND cms.locale LIKE 'en%' AND (cms.expires_at IS NULL OR cms.expires_at > CURRENT_TIMESTAMP)
		)
		ORDER BY pc.updated_at ASC,pc.id ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ProviderCollection, 0)
	for rows.Next() {
		var item ProviderCollection
		if err := rows.Scan(&item.ID, &item.Slug, &item.Provider, &item.ExternalID, &item.Kind, &item.TitleAR, &item.TitleEN, &item.OverviewAR, &item.OverviewEN, &item.PosterPath, &item.BackdropPath, &item.PartsCount, &item.Rating, &item.LocalItemCount, &item.IsFeatured, &item.IsHidden); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpdateProviderCollectionAdmin(ctx context.Context, id int64, update CatalogEntityAdminUpdate) error {
	result, err := r.db.ExecContext(ctx, `UPDATE provider_collections SET is_featured=COALESCE($1,is_featured),is_hidden=COALESCE($2,is_hidden),sort_priority=COALESCE($3,sort_priority),updated_at=CURRENT_TIMESTAMP WHERE id=$4`, nullableBool(update.IsFeatured), nullableBool(update.IsHidden), nullableInt(update.SortPriority), id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ListProviderCollectionMedia(ctx context.Context, slug string, opts ListMediaOptions) (*ProviderCollection, *MediaListResult, error) {
	collection, err := r.GetProviderCollection(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT media_item_id FROM media_collection_links WHERE collection_id=$1 ORDER BY tmdb_order NULLS LAST, media_item_id`, collection.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, collection.LocalItemCount)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return collection, &MediaListResult{Limit: opts.Limit, Offset: opts.Offset, Items: []search.MediaDocument{}}, nil
	}
	opts.IDs = ids
	result, err := r.ListMediaItems(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	// SQL's generic catalogue ordering is useful for filters, but franchise
	// membership should retain TMDB's part order when no explicit sort is chosen.
	if strings.TrimSpace(opts.Sort) == "" {
		order := make(map[int64]int, len(ids))
		for index, id := range ids {
			order[id] = index
		}
		sort.SliceStable(result.Items, func(i, j int) bool { return order[result.Items[i].ID] < order[result.Items[j].ID] })
	}
	return collection, result, nil
}

// ListProviderCollectionParts returns locally stored TMDB collection members,
// marking each one as local or pending. It never contacts TMDB while browsing.
func (r *Repository) ListProviderCollectionParts(ctx context.Context, slug string) (*ProviderCollection, []ProviderCollectionPart, error) {
	collection, err := r.GetProviderCollection(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	var englishRaw []byte
	err = r.db.QueryRowContext(ctx, `SELECT raw_payload FROM collection_metadata_snapshots WHERE collection_id=$1 AND locale LIKE 'en%' ORDER BY fetched_at DESC LIMIT 1`, collection.ID).Scan(&englishRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return collection, []ProviderCollectionPart{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	type snapshotPart struct {
		ID              int64   `json:"id"`
		Title           string  `json:"title"`
		ReleaseDate     string  `json:"release_date"`
		PosterPath      string  `json:"poster_path"`
		LocalPosterPath string  `json:"local_poster_path"`
		VoteAverage     float64 `json:"vote_average"`
		Overview        string  `json:"overview"`
	}
	var englishSnapshot struct {
		Parts []snapshotPart `json:"parts"`
	}
	if err := json.Unmarshal(englishRaw, &englishSnapshot); err != nil {
		return nil, nil, err
	}
	parts := make([]ProviderCollectionPart, 0, len(englishSnapshot.Parts))
	byExternal := make(map[string]int, len(englishSnapshot.Parts))
	for _, part := range englishSnapshot.Parts {
		item := ProviderCollectionPart{ExternalID: fmt.Sprint(part.ID), Title: part.Title, TitleEN: part.Title, Rating: part.VoteAverage, Overview: part.Overview, OverviewEN: part.Overview, PosterPath: firstNonEmpty(part.LocalPosterPath, part.PosterPath)}
		if len(part.ReleaseDate) >= 4 {
			item.Year, _ = strconv.Atoi(part.ReleaseDate[:4])
		}
		byExternal[item.ExternalID] = len(parts)
		parts = append(parts, item)
	}
	var arabicRaw []byte
	if err := r.db.QueryRowContext(ctx, `SELECT raw_payload FROM collection_metadata_snapshots WHERE collection_id=$1 AND locale LIKE 'ar%' ORDER BY fetched_at DESC LIMIT 1`, collection.ID).Scan(&arabicRaw); err == nil {
		var arabicSnapshot struct {
			Parts []snapshotPart `json:"parts"`
		}
		if json.Unmarshal(arabicRaw, &arabicSnapshot) == nil {
			for _, part := range arabicSnapshot.Parts {
				index, found := byExternal[fmt.Sprint(part.ID)]
				if !found {
					continue
				}
				if strings.TrimSpace(part.Title) != "" {
					parts[index].TitleAR = part.Title
					parts[index].Title = part.Title
				}
				if strings.TrimSpace(part.Overview) != "" {
					parts[index].OverviewAR = part.Overview
					parts[index].Overview = part.Overview
				}
			}
		}
	}
	if len(parts) == 0 {
		return collection, parts, nil
	}
	ids := make([]string, 0, len(parts))
	for index := range parts {
		ids = append(ids, parts[index].ExternalID)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,metadata_external_id FROM media_items WHERE metadata_provider='tmdb' AND metadata_external_id = ANY($1::text[])`, pq.Array(ids))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var mediaID int64
		var externalID string
		if err := rows.Scan(&mediaID, &externalID); err != nil {
			return nil, nil, err
		}
		if index, ok := byExternal[externalID]; ok {
			parts[index].Local = true
			parts[index].MediaID = mediaID
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return collection, parts, nil
}

// ListPeople exposes only people with a useful local relationship and a
// featured TMDB billing position. This keeps the directory aligned with the
// highlighted cast shown on a media-details page rather than listing every
// background credit in the library.
func (r *Repository) ListPeople(ctx context.Context, limit int) ([]Person, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id,slug,provider,external_id,COALESCE(name_ar,''),name_en,
			COALESCE(known_for_department,''),COALESCE(profile_path,''),COALESCE(popularity,0),
			local_media_count,is_featured,is_hidden
		FROM people
		WHERE is_hidden=false AND local_media_count >= 2
			AND EXISTS (
				SELECT 1 FROM media_credits mc
				WHERE mc.person_id=people.id AND mc.credit_kind='cast'
					AND mc.billing_order BETWEEN 0 AND 23
			)
		ORDER BY is_featured DESC, local_media_count DESC, popularity DESC, name_en ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()
	people := make([]Person, 0)
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.ID, &person.Slug, &person.Provider, &person.ExternalID, &person.NameAR, &person.NameEN, &person.KnownForDepartment, &person.ProfilePath, &person.Popularity, &person.LocalMediaCount, &person.IsFeatured, &person.IsHidden); err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}

func (r *Repository) GetPerson(ctx context.Context, slug string) (*Person, error) {
	var person Person
	err := r.db.QueryRowContext(ctx, `
		SELECT id,slug,provider,external_id,COALESCE(name_ar,''),name_en,
			COALESCE(known_for_department,''),COALESCE(profile_path,''),COALESCE(popularity,0),
			local_media_count,is_featured,is_hidden
		FROM people WHERE slug=$1 AND is_hidden=false`, strings.TrimSpace(slug)).Scan(
		&person.ID, &person.Slug, &person.Provider, &person.ExternalID, &person.NameAR, &person.NameEN,
		&person.KnownForDepartment, &person.ProfilePath, &person.Popularity, &person.LocalMediaCount, &person.IsFeatured, &person.IsHidden)
	if err != nil {
		return nil, err
	}
	return &person, nil
}

func (r *Repository) UpdatePersonAdmin(ctx context.Context, id int64, update CatalogEntityAdminUpdate) error {
	result, err := r.db.ExecContext(ctx, `UPDATE people SET is_featured=COALESCE($1,is_featured),is_hidden=COALESCE($2,is_hidden),sort_priority=COALESCE($3,sort_priority),updated_at=CURRENT_TIMESTAMP WHERE id=$4`, nullableBool(update.IsFeatured), nullableBool(update.IsHidden), nullableInt(update.SortPriority), id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ListPersonMedia(ctx context.Context, slug string, opts ListMediaOptions) (*Person, *MediaListResult, error) {
	person, err := r.GetPerson(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT media_item_id FROM media_credits
		WHERE person_id=$1 ORDER BY media_item_id`, person.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, person.LocalMediaCount)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return person, &MediaListResult{Limit: opts.Limit, Offset: opts.Offset, Items: []search.MediaDocument{}}, nil
	}
	opts.IDs = ids
	result, err := r.ListMediaItems(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	return person, result, nil
}


func (r *Repository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	// Total media
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_items`).Scan(&stats.TotalMedia)

	// Total files and size
	var totalFiles sql.NullInt64
	var totalSize sql.NullInt64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM video_files`).Scan(&totalFiles, &totalSize)
	if totalFiles.Valid {
		stats.TotalFiles = totalFiles.Int64
	}
	if totalSize.Valid {
		stats.TotalStorageBytes = totalSize.Int64
	}

	// Categories summary
	categories, err := r.ListCategories(ctx)
	if err == nil {
		stats.Categories = categories
	}

	// Missing episodes count
	missing, err := r.ListMissingEpisodes(ctx)
	if err == nil {
		stats.MissingEpisodesCount = len(missing)
	}

	// Duplicate groups count
	duplicates, err := r.ListDuplicateGroups(ctx)
	if err == nil {
		stats.DuplicatesCount = len(duplicates)
	}
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_files WHERE verification_status = 'corrupted'`).Scan(&stats.CorruptedFilesCount)

	// Disks
	disks, err := r.ListDisks(ctx)
	if err == nil {
		stats.Disks = disks
	}

	return &stats, nil
}

func (r *Repository) ListDisks(ctx context.Context) ([]StorageDisk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, disk_letter, COALESCE(disk_label, ''), total_space, free_space, is_active, last_scanned
		FROM storage_disks
		ORDER BY disk_letter ASC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list disks: %w", err)
	}
	defer rows.Close()

	disks := make([]StorageDisk, 0)
	for rows.Next() {
		var disk StorageDisk
		var lastScanned sql.NullTime
		if err := rows.Scan(&disk.ID, &disk.DiskLetter, &disk.DiskLabel, &disk.TotalSpace, &disk.FreeSpace, &disk.IsActive, &lastScanned); err != nil {
			return nil, fmt.Errorf("scan disk: %w", err)
		}
		if lastScanned.Valid {
			disk.LastScanned = &lastScanned.Time
		}
		disk.UsedSpace = disk.TotalSpace - disk.FreeSpace
		if disk.TotalSpace > 0 {
			disk.UsedPercent = float64(disk.UsedSpace) / float64(disk.TotalSpace) * 100.0
		}
		disks = append(disks, disk)
	}
	return disks, nil
}

func (r *Repository) SaveDisks(ctx context.Context, disks []StorageDisk) error {
	for _, disk := range disks {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO storage_disks (disk_letter, disk_label, total_space, free_space, is_active, last_scanned)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
			ON CONFLICT (disk_letter)
			DO UPDATE SET
				disk_label = EXCLUDED.disk_label,
				total_space = EXCLUDED.total_space,
				free_space = EXCLUDED.free_space,
				is_active = EXCLUDED.is_active,
				last_scanned = CURRENT_TIMESTAMP;
		`, disk.DiskLetter, disk.DiskLabel, disk.TotalSpace, disk.FreeSpace, disk.IsActive)
		if err != nil {
			return fmt.Errorf("save disk %s: %w", disk.DiskLetter, err)
		}
	}
	return nil
}

type UpdateMediaRequest struct {
	TitleAR      string   `json:"title_ar"`
	TitleEN      string   `json:"title_en"`
	PlotAR       string   `json:"plot_ar"`
	PlotEN       string   `json:"plot_en"`
	ReleaseYear  int      `json:"release_year"`
	Rating       float64  `json:"rating"`
	PosterPath   string   `json:"poster_path"`
	BannerPath   string   `json:"banner_path"`
	Genres       []string `json:"genres"`
	Type         string   `json:"type"`
	CategorySlug string   `json:"category_slug"`
}

type CreateMediaRequest struct {
	TitleAR      string   `json:"title_ar"`
	TitleEN      string   `json:"title_en"`
	Type         string   `json:"type"`
	CategorySlug string   `json:"category_slug"`
	PlotAR       string   `json:"plot_ar"`
	PlotEN       string   `json:"plot_en"`
	ReleaseYear  int      `json:"release_year"`
	Rating       float64  `json:"rating"`
	PosterPath   string   `json:"poster_path"`
	BannerPath   string   `json:"banner_path"`
	Genres       []string `json:"genres"`
}


// CleanAndSyncAllGenres is a library-wide maintenance job that cleans duplicate
// tags from the genres array and populates content_rating from metadata_snapshots.
func (r *Repository) CleanAndSyncAllGenres(ctx context.Context) (int, error) {
	// 1. First run SQL cleanup for all items
	_, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET genres = (
			SELECT array_agg(DISTINCT cleaned_tag)
			FROM (
				SELECT CASE
					WHEN tag ILIKE 'animation' THEN 'أنمي'
					WHEN tag ILIKE 'action' THEN 'أكشن'
					WHEN tag ILIKE 'adventure' THEN 'مغامرة'
					WHEN tag ILIKE 'comedy' THEN 'كوميديا'
					WHEN tag ILIKE 'drama' THEN 'دراما'
					WHEN tag ILIKE 'crime' THEN 'جريمة'
					WHEN tag ILIKE 'documentary' THEN 'وثائقي'
					WHEN tag ILIKE 'fantasy' THEN 'فانتازيا'
					WHEN tag ILIKE 'family' THEN 'عائلي'
					WHEN tag ILIKE 'horror' THEN 'رعب'
					WHEN tag ILIKE 'history' THEN 'تاريخي'
					WHEN tag ILIKE 'mystery' THEN 'غموض'
					WHEN tag ILIKE 'romance' THEN 'رومانسي'
					WHEN tag ILIKE 'science fiction' OR tag ILIKE 'sci-fi' THEN 'خيال علمي'
					WHEN tag ILIKE 'thriller' THEN 'إثارة'
					WHEN tag ILIKE 'war' THEN 'حرب'
					WHEN tag ILIKE 'music' THEN 'موسيقى'
					WHEN tag ILIKE 'western' THEN 'ويسترن'
					WHEN tag ILIKE 'kids' THEN 'أطفال'
					ELSE tag
				END AS cleaned_tag
				FROM unnest(genres) AS tag
				WHERE tag IS NOT NULL AND tag != ''
			) cleaned_sub
			WHERE cleaned_tag IS NOT NULL AND cleaned_tag != ''
		)
		WHERE genres IS NOT NULL AND array_length(genres, 1) > 0;
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup genres SQL: %w", err)
	}

	// 2. Extract content_rating and genres from snapshots
	rows, err := r.db.QueryContext(ctx, `
		SELECT ms.media_item_id, mi.type, ms.raw_payload::text
		FROM metadata_snapshots ms
		JOIN media_items mi ON mi.id = ms.media_item_id
		WHERE ms.locale = 'en-US' OR ms.locale = 'ar-SA'
		ORDER BY ms.media_item_id ASC, CASE WHEN ms.locale = 'en-US' THEN 1 ELSE 2 END ASC;
	`)
	if err != nil {
		return 0, fmt.Errorf("query snapshots for sync: %w", err)
	}
	defer rows.Close()

	type itemPayload struct {
		MediaID   int64
		MediaType string
		Payload   []byte
	}
	seenMap := make(map[int64]bool)
	var toProcess []itemPayload
	for rows.Next() {
		var p itemPayload
		var payloadStr string
		if err := rows.Scan(&p.MediaID, &p.MediaType, &payloadStr); err == nil {
			if !seenMap[p.MediaID] {
				seenMap[p.MediaID] = true
				p.Payload = []byte(payloadStr)
				toProcess = append(toProcess, p)
			}
		}
	}

	updatedCount := 0
	for _, item := range toProcess {
		var doc struct {
			Genres []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"genres"`
			ReleaseDates struct {
				Results []struct {
					ISO31661     string `json:"iso_3166_1"`
					ReleaseDates []struct {
						Certification string `json:"certification"`
					} `json:"release_dates"`
				} `json:"results"`
			} `json:"release_dates"`
			ContentRatings struct {
				Results []struct {
					ISO31661 string `json:"iso_3166_1"`
					Rating   string `json:"rating"`
				} `json:"results"`
			} `json:"content_ratings"`
		}
		if json.Unmarshal(item.Payload, &doc) != nil {
			continue
		}

		contentRating := ""
		if item.MediaType == "movie" {
			for _, c := range doc.ReleaseDates.Results {
				if strings.EqualFold(c.ISO31661, "US") {
					for _, rd := range c.ReleaseDates {
						if cert := strings.TrimSpace(rd.Certification); cert != "" {
							contentRating = cert
							break
						}
					}
				}
				if contentRating != "" {
					break
				}
			}
		} else {
			for _, c := range doc.ContentRatings.Results {
				if strings.EqualFold(c.ISO31661, "US") {
					if rating := strings.TrimSpace(c.Rating); rating != "" {
						contentRating = rating
						break
					}
				}
			}
		}

		if contentRating != "" {
			_, _ = r.db.ExecContext(ctx, `
				UPDATE media_items
				SET content_rating = $1,
				    metadata_facets = jsonb_set(COALESCE(metadata_facets, '{}'::jsonb), '{content_rating}', to_jsonb($1::text))
				WHERE id = $2;
			`, contentRating, item.MediaID)
			updatedCount++
		}
	}

	return len(seenMap), nil
}
