package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexora/server/internal/db"
	"nexora/server/internal/metadata"
	"nexora/server/internal/search"
)

func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		if parsed, err := strconv.Atoi(rawOffset); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	opts := db.ListMediaOptions{
		CategorySlug: strings.TrimSpace(r.URL.Query().Get("category")),
		Type:         strings.TrimSpace(r.URL.Query().Get("type")),
		Search:       strings.TrimSpace(r.URL.Query().Get("q")),
		Sort:         strings.TrimSpace(r.URL.Query().Get("sort")),
		Limit:        limit,
		Offset:       offset,
	}

	result, err := s.repository.ListMediaItems(r.Context(), opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMediaDetail(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, item)
	settings := s.metadata.GetTMDBSettings()
	if settings.AutoRefreshEnabled && settings.RefreshOnOpen {
		_ = s.repository.EnqueueTMDBRefreshIfStale(r.Context(), mediaID, settings.RefreshStaleDays)
	}
}

// handleMediaRelated reads the locally persisted provider relationship graph.
// It deliberately does not call TMDB while a customer is browsing details.
func (s *Server) handleMediaRelated(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}
	limit := 18
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 48 {
		limit = 48
	}
	items, err := s.repository.ListRelatedMedia(r.Context(), mediaID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"media_id": mediaID, "items": items})
}

func (s *Server) handleMediaCreate(w http.ResponseWriter, r *http.Request) {
	var request db.CreateMediaRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.CreateMediaItem(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if s.search != nil && doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) handleMediaUpdateFull(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	var request db.UpdateMediaRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.UpdateMediaFull(r.Context(), mediaID, request)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	if s.search != nil && doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	if err := s.repository.DeleteMediaItem(r.Context(), mediaID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_id": mediaID})
}

func (s *Server) handleMediaFiles(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	files, err := s.repository.ListVideoFiles(r.Context(), mediaID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	for index := range files {
		files[index].StreamURL = "/api/stream/file/" + strconv.FormatInt(files[index].ID, 10)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"media_id": mediaID,
		"count":    len(files),
		"files":    files,
	})
}

func (s *Server) handleMediaMetadataSnapshot(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}
	locale := strings.TrimSpace(r.URL.Query().Get("locale"))
	if locale == "" {
		locale = "en-US"
	}
	snapshot, err := s.repository.GetMetadataSnapshot(r.Context(), mediaID, locale)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleMediaSeasonMetadataSnapshots(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}
	locale := strings.TrimSpace(r.URL.Query().Get("locale"))
	if locale == "" {
		locale = "en-US"
	}
	snapshots, err := s.repository.GetSeasonMetadataSnapshots(r.Context(), mediaID, locale)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mediaId": mediaID, "locale": locale, "items": snapshots})
}

func (s *Server) handleMediaEnrich(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	// Sync current settings from DB into metadata service if available
	if dbSettings, err := s.repository.GetTMDBSettings(r.Context()); err == nil && dbSettings != nil {
		s.metadata.SetTMDBSettings(*dbSettings)
	}

	lookupTitle := item.TitleEN
	if lookupTitle == "" {
		lookupTitle = item.TitleAR
	}

	var canonical metadata.Result
	var lookupErr error
	if selectedID := strings.TrimSpace(r.URL.Query().Get("tmdb_id")); selectedID != "" {
		canonical, lookupErr = s.metadata.LookupByExternalID(r.Context(), metadata.Query{Type: item.Type, Language: "en-US"}, selectedID)
	} else {
		canonical, lookupErr = s.metadata.Lookup(r.Context(), metadata.Query{
			Title: lookupTitle, Type: item.Type, Year: item.ReleaseYear, Language: "en-US",
		})
	}
	if lookupErr != nil {
		_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
			MediaItemID:  &mediaID,
			RequestKind:  "lookup",
			Endpoint:     "/search",
			StatusCode:   502,
			DurationMS:   int(time.Since(startTime).Milliseconds()),
			ErrorMessage: lookupErr.Error(),
		})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "metadata lookup failed: " + lookupErr.Error(), "title": lookupTitle, "mediaId": mediaID})
		return
	}

	// The confirmed English search result defines the TMDB ID. Arabic is then
	// fetched by that same ID, never via a second ambiguous title search.
	results := []metadata.Result{canonical}
	var lookupWarnings []string
	if canonical.Provider == "tmdb" {
		arabic, arabicErr := s.metadata.LookupByExternalID(r.Context(), metadata.Query{Type: item.Type, Language: "ar-SA"}, canonical.ExternalID)
		if arabicErr != nil {
			lookupWarnings = append(lookupWarnings, "ar-SA: "+arabicErr.Error())
		} else {
			// Apply the canonical English document first, then enrich the same
			// provider-ID records with Arabic presentation fields.
			results = []metadata.Result{canonical, arabic}
		}
	}

	var doc *search.MediaDocument
	for _, metaResult := range results {
		updated, updateErr := s.repository.UpdateMediaMetadata(r.Context(), mediaID, metaResult)
		if updateErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": updateErr.Error()})
			return
		}
		if updated != nil {
			doc = updated
		}
	}

	// Fetch complete franchise details only during this explicit enrich action.
	// Normal franchise GET requests remain strictly database-only.
	if canonical.Provider == "tmdb" && item.Type == "movie" {
		if collectionID := tmdbCollectionID(canonical.RawPayload); collectionID != "" {
			needsRefresh, needsErr := s.repository.ProviderCollectionNeedsRefresh(r.Context(), collectionID)
			if needsErr != nil {
				lookupWarnings = append(lookupWarnings, "collection cache state: "+needsErr.Error())
			} else if needsRefresh {
				lookupWarnings = append(lookupWarnings, s.refreshProviderCollectionMetadata(r.Context(), collectionID)...)
			}
		}
	}

	if doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	seasonCount := 0
	if canonical.Provider == "tmdb" && isSeriesType(item.Type) {
		for _, seasonNumber := range tmdbSeasonNumbers(canonical.RawPayload) {
			englishSeason, seasonErr := s.metadata.LookupSeasonByExternalID(r.Context(), canonical.ExternalID, seasonNumber, "en-US")
			if seasonErr != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d en-US: %v", seasonNumber, seasonErr))
				continue
			}
			seasonSnapshots := []metadata.SeasonResult{englishSeason}
			arabicSeason, arabicSeasonErr := s.metadata.LookupSeasonByExternalID(r.Context(), canonical.ExternalID, seasonNumber, "ar-SA")
			if arabicSeasonErr != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d ar-SA: %v", seasonNumber, arabicSeasonErr))
			} else {
				seasonSnapshots = append(seasonSnapshots, arabicSeason)
			}
			if err := s.repository.SaveSeasonMetadataSnapshots(r.Context(), mediaID, seasonSnapshots); err != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d cache: %v", seasonNumber, err))
				continue
			}
			seasonCount++
		}
	}

	// Log successful enrich usage
	_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
		MediaItemID:      &mediaID,
		RequestKind:      "enrich",
		Endpoint:         "/details",
		StatusCode:       200,
		BytesDownloaded:  int64(len(canonical.RawPayload)),
		ImagesDownloaded: 2, // Poster + Backdrop
		DurationMS:       int(time.Since(startTime).Milliseconds()),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"metadata":      results,
		"document":      doc,
		"warnings":      lookupWarnings,
		"seasonsCached": seasonCount,
	})
}

func isSeriesType(mediaType string) bool {
	return mediaType == "series" || mediaType == "anime" || mediaType == "tv"
}

func tmdbCollectionID(raw json.RawMessage) string {
	var payload struct {
		Collection *struct {
			ID json.Number `json:"id"`
		} `json:"belongs_to_collection"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil || payload.Collection == nil {
		return ""
	}
	return payload.Collection.ID.String()
}

func (s *Server) refreshProviderCollectionMetadata(ctx context.Context, externalID string) []string {
	warnings := make([]string, 0)
	for _, language := range []string{"en-US", "ar-SA"} {
		collection, err := s.metadata.LookupCollectionByExternalID(ctx, externalID, language)
		if err != nil {
			warnings = append(warnings, "collection "+language+": "+err.Error())
			continue
		}
		if err := s.repository.SaveProviderCollectionMetadata(ctx, collection); err != nil {
			warnings = append(warnings, "collection cache "+language+": "+err.Error())
		}
	}
	return warnings
}

func tmdbSeasonNumbers(raw json.RawMessage) []int {
	var payload struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
		} `json:"seasons"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	numbers := make([]int, 0, len(payload.Seasons))
	for _, season := range payload.Seasons {
		if season.SeasonNumber >= 0 {
			numbers = append(numbers, season.SeasonNumber)
		}
	}
	return numbers
}

func (s *Server) handleMediaMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	var request struct {
		Title       string   `json:"title"`
		Overview    string   `json:"overview"`
		ReleaseYear int      `json:"releaseYear"`
		Rating      float64  `json:"rating"`
		PosterPath  string   `json:"posterPath"`
		BannerPath  string   `json:"bannerPath"`
		Genres      []string `json:"genres"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.UpdateMediaMetadata(r.Context(), mediaID, metadata.Result{
		Title:       request.Title,
		Overview:    request.Overview,
		ReleaseYear: request.ReleaseYear,
		Rating:      request.Rating,
		PosterPath:  request.PosterPath,
		BannerPath:  request.BannerPath,
		Genres:      request.Genres,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "document": doc})
}
