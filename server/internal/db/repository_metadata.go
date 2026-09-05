package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"nexora/server/internal/metadata"
	"nexora/server/internal/search"
)

func (r *Repository) ListShowcases(ctx context.Context, opts ShowcaseOptions) (*ShowcaseResult, error) {
	if opts.Limit <= 0 || opts.Limit > 12 {
		opts.Limit = 6
	}
	categorySlug := strings.TrimSpace(opts.CategorySlug)
	result := &ShowcaseResult{Context: strings.TrimSpace(opts.Context), Slides: make([]ShowcaseSlide, 0, opts.Limit)}

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.slug, COALESCE(c.title_ar, ''), c.title_en, COALESCE(c.description_ar, ''), COALESCE(c.description_en, ''),
			COALESCE(c.artwork_path, ''), c.artwork_position, c.accent,
			COUNT(ci.media_item_id)::int, COALESCE(c.target_category_slug, ''), c.target_filters::text
		FROM collections c
		LEFT JOIN collection_items ci ON ci.collection_id = c.id
		WHERE c.is_active = true
			AND ($1 = '' OR c.target_category_slug = '' OR c.target_category_slug = $1)
		GROUP BY c.id
		ORDER BY c.priority DESC, c.id DESC
		LIMIT $2`, categorySlug, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("list showcase collections: %w", err)
	}
	for rows.Next() {
		var slug, titleAR, titleEN, descriptionAR, descriptionEN, artworkPath, artworkPosition, accent, targetCategory, filtersText string
		var itemCount int
		if err := rows.Scan(&slug, &titleAR, &titleEN, &descriptionAR, &descriptionEN, &artworkPath, &artworkPosition, &accent, &itemCount, &targetCategory, &filtersText); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan showcase collection: %w", err)
		}
		result.Slides = append(result.Slides, ShowcaseSlide{
			ID: "collection-" + slug, Kind: "collection", TitleAR: titleAR, TitleEN: titleEN,
			DescriptionAR: descriptionAR, DescriptionEN: descriptionEN, ArtworkPath: artworkPath,
			ArtworkPosition: artworkPosition, Accent: accent, ItemCount: itemCount,
			Target: &ShowcaseTarget{Category: targetCategory, Filters: json.RawMessage(filtersText)},
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate showcase collections: %w", err)
	}
	rows.Close()

	remaining := opts.Limit - len(result.Slides)
	if remaining <= 0 {
		return result, nil
	}
	mediaResult, err := r.ListMediaItems(ctx, ListMediaOptions{CategorySlug: categorySlug, Sort: "rating", Limit: remaining})
	if err != nil {
		return nil, err
	}
	for _, item := range mediaResult.Items {
		artwork := item.BannerPath
		if artwork == "" {
			artwork = item.PosterPath
		}
		result.Slides = append(result.Slides, ShowcaseSlide{
			ID: fmt.Sprintf("media-%d", item.ID), Kind: "featured", MediaID: item.ID,
			TitleAR: item.TitleAR, TitleEN: item.TitleEN, DescriptionAR: item.PlotAR,
			DescriptionEN: item.PlotEN, ArtworkPath: artwork, ArtworkPosition: "center center",
			Accent: "violet", Type: item.Type, Status: item.Status, ReleaseYear: item.ReleaseYear,
			Rating: item.Rating, BestResolution: item.BestResolution, Genres: item.Genres,
		})
	}

	return result, nil
}

func (r *Repository) UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error) {
	var genresArray any
	if len(meta.Genres) > 0 {
		genresArray = pq.Array(meta.Genres)
	}

	poster := meta.CachedPosterPath
	if poster == "" {
		poster = meta.PosterPath
	}
	banner := meta.CachedBannerPath
	if banner == "" {
		banner = meta.BannerPath
	}
	titleAR, titleEN, plotAR, plotEN := localizedMetadataFields(meta)
	metadataFacets := metadataFacetsJSON(meta)
	status := metadataStatus(meta)

	result, err := r.db.ExecContext(ctx, `
		UPDATE media_items
		SET
			title_ar = COALESCE(NULLIF($1, ''), title_ar),
			title_en = COALESCE(NULLIF($2, ''), title_en),
			plot_ar = COALESCE(NULLIF($3, ''), plot_ar),
			plot_en = COALESCE(NULLIF($4, ''), plot_en),
			release_year = COALESCE(NULLIF($5, 0), release_year),
			rating = COALESCE(NULLIF($6, 0.0), rating),
			poster_path = COALESCE(NULLIF($7, ''), poster_path),
			banner_path = COALESCE(NULLIF($8, ''), banner_path),
			genres = COALESCE($9, genres),
			metadata_provider = COALESCE(NULLIF($10, ''), metadata_provider),
			metadata_external_id = COALESCE(NULLIF($11, ''), metadata_external_id),
			metadata_facets = COALESCE($12::jsonb, metadata_facets),
			status = COALESCE(NULLIF($13, ''), status),
			content_rating = COALESCE(NULLIF($14, ''), content_rating),
			metadata_fetched_at = CASE WHEN NULLIF($10, '') IS NULL THEN metadata_fetched_at ELSE CURRENT_TIMESTAMP END,
			metadata_expires_at = CASE WHEN NULLIF($10, '') IS NULL THEN metadata_expires_at ELSE CURRENT_TIMESTAMP + INTERVAL '30 days' END
		WHERE id = $15;
	`, titleAR, titleEN, plotAR, plotEN, meta.ReleaseYear, meta.Rating, poster, banner, genresArray, meta.Provider, meta.ExternalID, metadataFacets, status, meta.ContentRating, id)
	if err != nil {
		return nil, fmt.Errorf("update media metadata: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return nil, sql.ErrNoRows
	}
	if len(meta.RawPayload) > 0 && meta.Provider != "" && meta.ExternalID != "" {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO metadata_snapshots (media_item_id, provider, external_id, locale, raw_payload, expires_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, CURRENT_TIMESTAMP + INTERVAL '30 days')
			ON CONFLICT (media_item_id, provider, locale)
			DO UPDATE SET external_id = EXCLUDED.external_id,
				raw_payload = EXCLUDED.raw_payload,
				fetched_at = CURRENT_TIMESTAMP,
				expires_at = EXCLUDED.expires_at
		`, id, meta.Provider, meta.ExternalID, firstNonEmptyLocale(meta.Locale), string(meta.RawPayload)); err != nil {
			return nil, fmt.Errorf("cache metadata snapshot: %w", err)
		}
		// The movie document is the authoritative place where TMDB declares its
		// belongs_to_collection relation. Persist the relation immediately so all
		// browser reads remain database-only. Full collection details are fetched
		// later by the dedicated, opt-in sync job.
		if err := r.syncCollectionFromMetadata(ctx, id, meta); err != nil {
			return nil, fmt.Errorf("sync provider collection: %w", err)
		}
		if err := r.syncCreditsFromMetadata(ctx, id, meta); err != nil {
			return nil, fmt.Errorf("sync provider credits: %w", err)
		}
	}

	// Fetch single document for search reindexing
	return r.GetSearchDocument(ctx, id)
}

func (r *Repository) syncCollectionFromMetadata(ctx context.Context, mediaID int64, meta metadata.Result) error {
	if meta.Provider != "tmdb" || len(meta.RawPayload) == 0 {
		return nil
	}
	var raw struct {
		Collection *struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			PosterPath   string `json:"poster_path"`
			BackdropPath string `json:"backdrop_path"`
		} `json:"belongs_to_collection"`
	}
	if err := json.Unmarshal(meta.RawPayload, &raw); err != nil || raw.Collection == nil || raw.Collection.ID <= 0 {
		return nil
	}
	c := raw.Collection
	slug := fmt.Sprintf("tmdb-collection-%d", c.ID)
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO provider_collections (provider,external_id,kind,slug,title_ar,title_en,poster_path,backdrop_path)
		VALUES ('tmdb',$1,'movie_collection',$2,
			CASE WHEN $3 LIKE 'ar%%' THEN $4 ELSE NULL END,
			CASE WHEN $3 LIKE 'ar%%' THEN $5 ELSE $4 END,
			NULL,NULL)
		ON CONFLICT (provider,external_id) DO UPDATE SET
			title_ar=COALESCE(EXCLUDED.title_ar,provider_collections.title_ar),
			title_en=CASE WHEN $3 LIKE 'ar%%' THEN provider_collections.title_en ELSE COALESCE(NULLIF(EXCLUDED.title_en,''),provider_collections.title_en) END,
			poster_path=COALESCE(EXCLUDED.poster_path,provider_collections.poster_path),
			backdrop_path=COALESCE(EXCLUDED.backdrop_path,provider_collections.backdrop_path),
			updated_at=CURRENT_TIMESTAMP
		RETURNING id`, fmt.Sprint(c.ID), slug, strings.ToLower(meta.Locale), c.Name, c.Name).Scan(&id)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO media_collection_links (media_item_id,collection_id,source,verified_at)
		VALUES ($1,$2,'tmdb',CURRENT_TIMESTAMP)
		ON CONFLICT (media_item_id,collection_id) DO UPDATE SET verified_at=EXCLUDED.verified_at`, mediaID, id)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE provider_collections pc SET local_item_count=(SELECT COUNT(*) FROM media_collection_links WHERE collection_id=pc.id), updated_at=CURRENT_TIMESTAMP WHERE pc.id=$1`, id)
	return err
}

func (r *Repository) syncCreditsFromMetadata(ctx context.Context, mediaID int64, meta metadata.Result) error {
	if meta.Provider != "tmdb" || len(meta.RawPayload) == 0 {
		return nil
	}
	type castCredit struct {
		ID                 int64   `json:"id"`
		CreditID           string  `json:"credit_id"`
		Name               string  `json:"name"`
		KnownForDepartment string  `json:"known_for_department"`
		LocalProfilePath   string  `json:"local_profile_path"`
		Popularity         float64 `json:"popularity"`
		Character          string  `json:"character"`
		Order              int     `json:"order"`
		Roles              []struct {
			Character string `json:"character"`
		} `json:"roles"`
	}
	type crewCredit struct {
		ID                 int64   `json:"id"`
		CreditID           string  `json:"credit_id"`
		Name               string  `json:"name"`
		KnownForDepartment string  `json:"known_for_department"`
		LocalProfilePath   string  `json:"local_profile_path"`
		Popularity         float64 `json:"popularity"`
		Job                string  `json:"job"`
		Department         string  `json:"department"`
		Jobs               []struct {
			Job        string `json:"job"`
			Department string `json:"department"`
		} `json:"jobs"`
	}
	var raw struct {
		Credits struct {
			Cast []castCredit `json:"cast"`
			Crew []crewCredit `json:"crew"`
		} `json:"credits"`
		AggregateCredits struct {
			Cast []castCredit `json:"cast"`
			Crew []crewCredit `json:"crew"`
		} `json:"aggregate_credits"`
	}
	if err := json.Unmarshal(meta.RawPayload, &raw); err != nil {
		return nil
	}
	upsertPerson := func(id int64, name, department, profile string, popularity float64) (int64, error) {
		if id <= 0 || strings.TrimSpace(name) == "" {
			return 0, nil
		}
		slug := fmt.Sprintf("tmdb-person-%d", id)
		var personID int64
		err := r.db.QueryRowContext(ctx, `INSERT INTO people (provider,external_id,slug,name_ar,name_en,known_for_department,profile_path,popularity,metadata_fetched_at,metadata_expires_at) VALUES ('tmdb',$1,$2,CASE WHEN $3 LIKE 'ar%%' THEN $4 ELSE NULL END,CASE WHEN $3 LIKE 'ar%%' THEN $5 ELSE $4 END,NULLIF($6,''),NULLIF($7,''),NULLIF($8,0),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP + INTERVAL '30 days') ON CONFLICT (provider,external_id) DO UPDATE SET name_ar=COALESCE(EXCLUDED.name_ar,people.name_ar),name_en=CASE WHEN $3 LIKE 'ar%%' THEN people.name_en ELSE COALESCE(NULLIF(EXCLUDED.name_en,''),people.name_en) END,known_for_department=COALESCE(NULLIF(EXCLUDED.known_for_department,''),people.known_for_department),profile_path=COALESCE(EXCLUDED.profile_path,people.profile_path),popularity=COALESCE(EXCLUDED.popularity,people.popularity),updated_at=CURRENT_TIMESTAMP RETURNING id`, fmt.Sprint(id), slug, strings.ToLower(meta.Locale), name, name, department, profile, popularity).Scan(&personID)
		return personID, err
	}
	casts := raw.Credits.Cast
	crews := raw.Credits.Crew
	if len(casts) == 0 {
		casts = raw.AggregateCredits.Cast
		crews = raw.AggregateCredits.Crew
	}
	for index, cast := range casts {
		if index >= 12 {
			break
		}
		creditID := cast.CreditID
		if creditID == "" {
			creditID = fmt.Sprintf("aggregate-cast-%d", cast.ID)
		}
		character := cast.Character
		if character == "" && len(cast.Roles) > 0 {
			character = cast.Roles[0].Character
		}
		// Only a locally cached profile is retained here. The source image path
		// is still present in the raw snapshot for a future explicit refresh.
		profile := cast.LocalProfilePath
		personID, err := upsertPerson(cast.ID, cast.Name, cast.KnownForDepartment, profile, cast.Popularity)
		if err != nil {
			return err
		}
		if personID == 0 {
			continue
		}
		_, err = r.db.ExecContext(ctx, `INSERT INTO media_credits (media_item_id,person_id,provider,provider_credit_id,credit_kind,character_name,billing_order,source,verified_at) VALUES ($1,$2,'tmdb',NULLIF($3,''),'cast',NULLIF($4,''),$5,'tmdb',CURRENT_TIMESTAMP) ON CONFLICT (media_item_id,provider,provider_credit_id) WHERE provider_credit_id IS NOT NULL DO UPDATE SET person_id=EXCLUDED.person_id,character_name=EXCLUDED.character_name,billing_order=EXCLUDED.billing_order,verified_at=EXCLUDED.verified_at,updated_at=CURRENT_TIMESTAMP`, mediaID, personID, creditID, character, cast.Order)
		if err != nil {
			return err
		}
	}
	for _, crew := range crews {
		job, department := crew.Job, crew.Department
		if job == "" && len(crew.Jobs) > 0 {
			job, department = crew.Jobs[0].Job, crew.Jobs[0].Department
		}
		if !strings.EqualFold(job, "Director") {
			continue
		}
		creditID := crew.CreditID
		if creditID == "" {
			creditID = fmt.Sprintf("aggregate-crew-%d-%s", crew.ID, strings.ToLower(job))
		}
		profile := crew.LocalProfilePath
		personID, err := upsertPerson(crew.ID, crew.Name, crew.KnownForDepartment, profile, crew.Popularity)
		if err != nil {
			return err
		}
		if personID == 0 {
			continue
		}
		_, err = r.db.ExecContext(ctx, `INSERT INTO media_credits (media_item_id,person_id,provider,provider_credit_id,credit_kind,job,department,source,verified_at) VALUES ($1,$2,'tmdb',NULLIF($3,''),'crew',NULLIF($4,''),NULLIF($5,''),'tmdb',CURRENT_TIMESTAMP) ON CONFLICT (media_item_id,provider,provider_credit_id) WHERE provider_credit_id IS NOT NULL DO UPDATE SET person_id=EXCLUDED.person_id,job=EXCLUDED.job,department=EXCLUDED.department,verified_at=EXCLUDED.verified_at,updated_at=CURRENT_TIMESTAMP`, mediaID, personID, creditID, job, department)
		if err != nil {
			return err
		}
	}
	_, err := r.db.ExecContext(ctx, `UPDATE people p SET local_media_count=(SELECT COUNT(DISTINCT media_item_id) FROM media_credits WHERE person_id=p.id), updated_at=CURRENT_TIMESTAMP WHERE p.provider='tmdb'`)
	return err
}

// SyncCatalogRelationsFromSnapshots rebuilds all derived local relations from
// previously cached TMDB documents. This is used after introducing the graph
// tables and can be re-run safely after an import or an upgrade.
func (r *Repository) SyncCatalogRelationsFromSnapshots(ctx context.Context) (*CatalogRelationSyncResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT media_item_id, provider, external_id, locale, raw_payload
		FROM metadata_snapshots
		WHERE provider='tmdb'
		ORDER BY media_item_id, CASE WHEN locale LIKE 'en%' THEN 0 ELSE 1 END, locale`)
	if err != nil {
		return nil, fmt.Errorf("list metadata snapshots for relation sync: %w", err)
	}
	defer rows.Close()

	result := &CatalogRelationSyncResult{}
	for rows.Next() {
		var mediaID int64
		var provider, externalID, locale string
		var raw []byte
		if err := rows.Scan(&mediaID, &provider, &externalID, &locale, &raw); err != nil {
			return nil, err
		}
		meta := metadata.Result{Provider: provider, ExternalID: externalID, Locale: locale, RawPayload: raw}
		if err := r.syncCollectionFromMetadata(ctx, mediaID, meta); err != nil {
			return nil, fmt.Errorf("sync collection for media %d: %w", mediaID, err)
		}
		if err := r.syncCreditsFromMetadata(ctx, mediaID, meta); err != nil {
			return nil, fmt.Errorf("sync credits for media %d: %w", mediaID, err)
		}
		result.SnapshotsProcessed++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if _, err := r.db.ExecContext(ctx, `
		UPDATE provider_collections pc
		SET local_item_count=(SELECT COUNT(*) FROM media_collection_links mcl WHERE mcl.collection_id=pc.id), updated_at=CURRENT_TIMESTAMP`); err != nil {
		return nil, fmt.Errorf("refresh collection counts: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `
		UPDATE people p
		SET local_media_count=(SELECT COUNT(DISTINCT media_item_id) FROM media_credits mc WHERE mc.person_id=p.id), updated_at=CURRENT_TIMESTAMP
		WHERE p.provider='tmdb'`); err != nil {
		return nil, fmt.Errorf("refresh people counts: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_collection_links`).Scan(&result.CollectionsLinked); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_credits`).Scan(&result.CreditsLinked); err != nil {
		return nil, err
	}
	return result, nil
}

// SaveProviderCollectionMetadata stores the expensive collection response and
// its locally cached artwork. It is called only by enrichment, never by GET.
func (r *Repository) SaveProviderCollectionMetadata(ctx context.Context, meta metadata.CollectionResult) error {
	if meta.Provider != "tmdb" || strings.TrimSpace(meta.ExternalID) == "" || strings.TrimSpace(meta.Title) == "" {
		return nil
	}
	var collectionID int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO provider_collections (provider,external_id,kind,slug,title_ar,title_en,overview_ar,overview_en,poster_path,backdrop_path,parts_count,rating,metadata_fetched_at,metadata_expires_at)
		VALUES ('tmdb',$1,'movie_collection',$2,CASE WHEN $3 LIKE 'ar%%' THEN $4 ELSE NULL END,CASE WHEN $3 LIKE 'ar%%' THEN NULL ELSE $4 END,CASE WHEN $3 LIKE 'ar%%' THEN NULLIF($5,'') ELSE NULL END,CASE WHEN $3 LIKE 'ar%%' THEN NULL ELSE NULLIF($5,'') END,NULLIF($6,''),NULLIF($7,''),$8,NULLIF($9,0),CURRENT_TIMESTAMP,CURRENT_TIMESTAMP + INTERVAL '30 days')
		ON CONFLICT (provider,external_id) DO UPDATE SET
			title_ar=COALESCE(EXCLUDED.title_ar,provider_collections.title_ar),
			title_en=CASE WHEN $3 LIKE 'ar%%' THEN provider_collections.title_en ELSE EXCLUDED.title_en END,
			overview_ar=COALESCE(EXCLUDED.overview_ar,provider_collections.overview_ar),
			overview_en=CASE WHEN $3 LIKE 'ar%%' THEN provider_collections.overview_en ELSE COALESCE(EXCLUDED.overview_en,provider_collections.overview_en) END,
			poster_path=COALESCE(NULLIF(EXCLUDED.poster_path,''),provider_collections.poster_path),
			backdrop_path=COALESCE(NULLIF(EXCLUDED.backdrop_path,''),provider_collections.backdrop_path),
			parts_count=GREATEST(EXCLUDED.parts_count,provider_collections.parts_count),rating=COALESCE(NULLIF(EXCLUDED.rating,0),provider_collections.rating),metadata_fetched_at=CURRENT_TIMESTAMP,metadata_expires_at=EXCLUDED.metadata_expires_at,updated_at=CURRENT_TIMESTAMP
		RETURNING id`, meta.ExternalID, "tmdb-collection-"+meta.ExternalID, strings.ToLower(meta.Locale), meta.Title, meta.Overview, firstNonEmpty(meta.CachedPosterPath, meta.PosterPath), firstNonEmpty(meta.CachedBackdropPath, meta.BackdropPath), meta.PartsCount, meta.Rating).Scan(&collectionID)
	if err != nil {
		return fmt.Errorf("upsert collection metadata: %w", err)
	}
	if len(meta.RawPayload) > 0 {
		_, err = r.db.ExecContext(ctx, `INSERT INTO collection_metadata_snapshots (collection_id,provider,external_id,locale,raw_payload,expires_at) VALUES ($1,'tmdb',$2,$3,$4::jsonb,CURRENT_TIMESTAMP + INTERVAL '30 days') ON CONFLICT (collection_id,provider,locale) DO UPDATE SET raw_payload=EXCLUDED.raw_payload,fetched_at=CURRENT_TIMESTAMP,expires_at=EXCLUDED.expires_at`, collectionID, meta.ExternalID, firstNonEmptyLocale(meta.Locale), string(meta.RawPayload))
		if err != nil {
			return err
		}
	}
	for order, externalID := range meta.PartExternalIDs {
		_, err = r.db.ExecContext(ctx, `UPDATE media_collection_links mcl SET tmdb_order=$1,verified_at=CURRENT_TIMESTAMP FROM media_items mi WHERE mcl.collection_id=$2 AND mi.id=mcl.media_item_id AND mi.metadata_provider='tmdb' AND mi.metadata_external_id=$3`, order+1, collectionID, externalID)
		if err != nil {
			return err
		}
	}
	return nil
}

// localizedMetadataFields prevents a locale-specific TMDB response from
// overwriting the other title/plot column. This is important for records where
// TMDB falls back to a third language when Arabic text is unavailable.
func localizedMetadataFields(meta metadata.Result) (titleAR, titleEN, plotAR, plotEN string) {
	locale := strings.ToLower(strings.TrimSpace(meta.Locale))
	if strings.HasPrefix(locale, "ar") {
		return meta.Title, "", meta.Overview, ""
	}
	return "", meta.Title, "", meta.Overview
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

// metadataFacetsJSON extracts stable, filterable English attributes from the
// complete TMDB document. The full source document remains in
// metadata_snapshots; this small JSONB projection is indexed on media_items.
func metadataFacetsJSON(meta metadata.Result) any {
	if !strings.HasPrefix(strings.ToLower(meta.Locale), "en") || len(meta.RawPayload) == 0 {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(meta.RawPayload, &raw); err != nil {
		return nil
	}
	mediaType := "movie"
	if raw["name"] != nil && raw["title"] == nil {
		mediaType = "tv"
	}
	facets := map[string]any{
		"provider":              meta.Provider,
		"external_id":           meta.ExternalID,
		"type":                  mediaType,
		"title":                 raw["title"],
		"original_title":        raw["original_title"],
		"original_language":     raw["original_language"],
		"release_date":          raw["release_date"],
		"runtime":               raw["runtime"],
		"status":                raw["status"],
		"content_rating":        meta.ContentRating,
		"genre_ids":             meta.GenreIDs,
		"number_of_seasons":     raw["number_of_seasons"],
		"number_of_episodes":    raw["number_of_episodes"],
		"episode_run_time":      raw["episode_run_time"],
		"adult":                 raw["adult"],
		"popularity":            raw["popularity"],
		"vote_average":          raw["vote_average"],
		"vote_count":            raw["vote_count"],
		"genres":                raw["genres"],
		"keywords":              raw["keywords"],
		"production_companies":  raw["production_companies"],
		"production_countries":  raw["production_countries"],
		"spoken_languages":      raw["spoken_languages"],
		"belongs_to_collection": raw["belongs_to_collection"],
	}
	encoded, err := json.Marshal(facets)
	if err != nil {
		return nil
	}
	return string(encoded)
}

// metadataStatus turns TMDB's provider status into one of the UI's stable
// database values. It is saved during enrichment, so browsing stays offline.
func metadataStatus(meta metadata.Result) string {
	if len(meta.RawPayload) == 0 {
		return ""
	}
	var raw struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(meta.RawPayload, &raw) != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(raw.Status)) {
	case "released", "ended":
		return "completed"
	case "returning series", "in production", "post production", "planned", "rumored", "pilot":
		return "ongoing"
	case "canceled", "cancelled":
		return "cancelled"
	default:
		return ""
	}
}

func (r *Repository) GetMetadataSnapshot(ctx context.Context, mediaItemID int64, locale string) (*MetadataSnapshot, error) {
	locale = firstNonEmptyLocale(locale)
	var snapshot MetadataSnapshot
	var payload string
	err := r.db.QueryRowContext(ctx, `
		SELECT provider, external_id, locale, raw_payload::text, fetched_at, expires_at
		FROM metadata_snapshots
		WHERE media_item_id = $1 AND locale = $2
	`, mediaItemID, locale).Scan(&snapshot.Provider, &snapshot.ExternalID, &snapshot.Locale, &payload, &snapshot.FetchedAt, &snapshot.ExpiresAt)
	if err != nil {
		return nil, err
	}
	snapshot.Payload = json.RawMessage(payload)
	return &snapshot, nil
}

func (r *Repository) SaveSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, snapshots []metadata.SeasonResult) error {
	for _, snapshot := range snapshots {
		if len(snapshot.RawPayload) == 0 || snapshot.Provider == "" || snapshot.ExternalID == "" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO season_metadata_snapshots (media_item_id, provider, external_id, season_number, locale, raw_payload, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, CURRENT_TIMESTAMP + INTERVAL '30 days')
			ON CONFLICT (media_item_id, provider, season_number, locale)
			DO UPDATE SET external_id = EXCLUDED.external_id,
				raw_payload = EXCLUDED.raw_payload,
				fetched_at = CURRENT_TIMESTAMP,
				expires_at = EXCLUDED.expires_at
		`, mediaItemID, snapshot.Provider, snapshot.ExternalID, snapshot.SeasonNumber, firstNonEmptyLocale(snapshot.Locale), string(snapshot.RawPayload)); err != nil {
			return fmt.Errorf("save season metadata snapshot: %w", err)
		}
	}
	return nil
}

func (r *Repository) GetSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, locale string) ([]SeasonMetadataSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT provider, external_id, locale, season_number, raw_payload::text, fetched_at, expires_at
		FROM season_metadata_snapshots
		WHERE media_item_id = $1 AND locale = $2
		ORDER BY season_number ASC
	`, mediaItemID, firstNonEmptyLocale(locale))
	if err != nil {
		return nil, fmt.Errorf("query season metadata snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]SeasonMetadataSnapshot, 0)
	for rows.Next() {
		var snapshot SeasonMetadataSnapshot
		var payload string
		if err := rows.Scan(&snapshot.Provider, &snapshot.ExternalID, &snapshot.Locale, &snapshot.SeasonNumber, &payload, &snapshot.FetchedAt, &snapshot.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan season metadata snapshot: %w", err)
		}
		snapshot.Payload = json.RawMessage(payload)
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func firstNonEmptyLocale(locale string) string {
	if strings.TrimSpace(locale) == "" {
		return "en-US"
	}
	return locale
}

