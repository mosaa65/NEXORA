package api

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"nexora/server/internal/db"
	"nexora/server/internal/metadata"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type indexResult struct {
	Roots         []string          `json:"roots"`
	Scanned       int               `json:"scanned"`
	Imported      int               `json:"imported"`
	Inspected     int               `json:"inspected"`
	InspectFailed int               `json:"inspectFailed"`
	SearchSync    search.SyncResult `json:"searchSync"`
	Warnings      []string          `json:"warnings,omitempty"`
}

const (
	ingestBatchSize  = 128
	maxIndexWarnings = 100
)

func (r *indexResult) addWarning(message string) {
	if len(r.Warnings) < maxIndexWarnings {
		r.Warnings = append(r.Warnings, message)
		return
	}
	if len(r.Warnings) == maxIndexWarnings {
		r.Warnings = append(r.Warnings, "additional indexing warnings were omitted")
	}
}

// handleIndex is the unified first-pass library workflow. Metadata artwork and
// subtitle extraction are deliberately separate jobs because they can require
// external providers or long-running FFmpeg work.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}
	result := indexResult{Roots: request.Roots, Warnings: []string{}}
	batch := make([]scanner.FileInfo, 0, ingestBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ingested, err := s.repository.IngestScannedFiles(r.Context(), batch)
		result.Imported += ingested.Imported
		if err != nil {
			return err
		}
		for _, file := range batch {
			id, err := s.repository.GetVideoFileIDByPath(r.Context(), file.Path)
			if err != nil {
				result.InspectFailed++
				result.addWarning("could not locate indexed file: " + file.Path)
				continue
			}
			details, err := s.processor.Inspect(r.Context(), file.Path)
			if err != nil {
				result.InspectFailed++
				result.addWarning("could not inspect: " + file.Path)
				continue
			}
			if err := s.repository.UpdateVideoTechnicalDetails(r.Context(), id, details); err != nil {
				result.InspectFailed++
				result.addWarning("could not save inspection: " + file.Path)
				continue
			}
			result.Inspected++
		}
		batch = batch[:0]
		return nil
	}
	err := s.scanner.Walk(r.Context(), request.Roots, func(file scanner.FileInfo) error {
		result.Scanned++
		batch = append(batch, file)
		if len(batch) == cap(batch) {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}
	documents, err := s.repository.ListSearchDocuments(r.Context(), 10000)
	if err != nil {
		result.addWarning("search sync skipped: " + err.Error())
	} else if syncResult, err := s.search.IndexDocuments(r.Context(), documents); err != nil {
		result.addWarning("search sync skipped: " + err.Error())
	} else {
		result.SearchSync = syncResult
	}
	writeJSON(w, http.StatusOK, result)
}

type indexPreviewItem struct {
	Title         string               `json:"title"`
	TitleAR       string               `json:"title_ar"`
	TitleEN       string               `json:"title_en"`
	Type          string               `json:"type"`
	CategorySlug  string               `json:"category_slug"`
	OriginTags    []string             `json:"origin_tags"`
	ReleaseYear   int                  `json:"release_year"`
	PosterPath    string               `json:"poster_path,omitempty"`
	RawPosterPath string               `json:"raw_poster_path,omitempty"`
	TotalFiles    int                  `json:"total_files"`
	TotalSize     int64                `json:"total_size"`
	Seasons       []indexPreviewSeason `json:"seasons"`
}

type indexPreviewSeason struct {
	SeasonNumber int                `json:"season_number"`
	Title        string             `json:"title"`
	EpisodeCount int                `json:"episode_count"`
	Episodes     []indexPreviewFile `json:"episodes"`
}

type indexPreviewFile struct {
	Path          string `json:"path"`
	EpisodeNumber int    `json:"episode_number"`
	Size          int64  `json:"size"`
	Resolution    string `json:"resolution"`
}

// handleIndexPreview generates a non-destructive dry-run preview of what would be indexed.
func (s *Server) handleIndexPreview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}

	type mediaGroupKey struct {
		titleKey string
		mType    string
	}

	groups := make(map[mediaGroupKey]*indexPreviewItem)
	var groupOrder []mediaGroupKey
	totalScanned := 0

	err := s.scanner.Walk(r.Context(), request.Roots, func(file scanner.FileInfo) error {
		totalScanned++
		categorySlug, mediaType := db.ClassifyMedia(file.Path, file.Parsed)
		title := file.Parsed.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
		}
		titleEN := file.Parsed.TitleEN
		if titleEN == "" {
			titleEN = title
		}
		titleAR := file.Parsed.TitleAR
		if titleAR == "" {
			titleAR = title
		}

		key := mediaGroupKey{
			titleKey: strings.ToLower(titleEN),
			mType:    mediaType,
		}

		group, exists := groups[key]
		if !exists {
			localPosterURL := ""
			if file.ArtworkPath != "" {
				localPosterURL = s.repository.CacheLocalArtwork(file.ArtworkPath)
			}
			originTags := scanner.DetectOriginTagsFromPath(file.Path)
			if originTags == nil {
				originTags = []string{}
			}

			group = &indexPreviewItem{
				Title:         title,
				TitleAR:       titleAR,
				TitleEN:       titleEN,
				Type:          mediaType,
				CategorySlug:  categorySlug,
				OriginTags:    originTags,
				ReleaseYear:   file.Parsed.ReleaseYear,
				PosterPath:    localPosterURL,
				RawPosterPath: file.ArtworkPath,
				Seasons:       []indexPreviewSeason{},
			}
			groups[key] = group
			groupOrder = append(groupOrder, key)
		}

		group.TotalFiles++
		group.TotalSize += file.Size
		if group.PosterPath == "" && file.ArtworkPath != "" {
			group.PosterPath = s.repository.CacheLocalArtwork(file.ArtworkPath)
			group.RawPosterPath = file.ArtworkPath
		}

		seasonNum := file.Parsed.SeasonNumber
		if seasonNum <= 0 {
			seasonNum = 1
		}

		// Find or add season
		var targetSeason *indexPreviewSeason
		for idx := range group.Seasons {
			if group.Seasons[idx].SeasonNumber == seasonNum {
				targetSeason = &group.Seasons[idx]
				break
			}
		}
		if targetSeason == nil {
			group.Seasons = append(group.Seasons, indexPreviewSeason{
				SeasonNumber: seasonNum,
				Title:        fmt.Sprintf("Season %02d", seasonNum),
				Episodes:     []indexPreviewFile{},
			})
			targetSeason = &group.Seasons[len(group.Seasons)-1]
		}

		targetSeason.EpisodeCount++
		targetSeason.Episodes = append(targetSeason.Episodes, indexPreviewFile{
			Path:          file.Path,
			EpisodeNumber: file.Parsed.EpisodeNumber,
			Size:          file.Size,
			Resolution:    file.Parsed.Resolution,
		})

		return nil
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	resultItems := make([]*indexPreviewItem, 0, len(groupOrder))
	for _, k := range groupOrder {
		resultItems = append(resultItems, groups[k])
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roots":      request.Roots,
		"totalFiles": totalScanned,
		"totalMedia": len(resultItems),
		"mediaItems": resultItems,
	})
}


// handleClassifyOrigins repairs country tags from the user's existing folder
// hierarchy. It is deliberately local-only: no title matching or third-party
// request can overwrite a folder such as "مسلسلات/عربي".
func (s *Server) handleClassifyOrigins(w http.ResponseWriter, r *http.Request) {
	updated, err := s.repository.ClassifyOriginsFromPaths(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Country tags are also searchable, so refresh the local search index.
	if documents, err := s.repository.ListSearchDocuments(r.Context(), 10000); err == nil {
		if _, err := s.search.IndexDocuments(r.Context(), documents); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true, "updated": updated,
				"warning": "origin tags updated but search index refresh failed: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	roots := r.URL.Query()["root"]
	if len(roots) == 0 {
		roots = s.config.MediaRoots
	}
	if len(roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "provide at least one root query parameter or set NEXORA_MEDIA_ROOTS",
		})
		return
	}

	files, err := s.scanner.Scan(r.Context(), roots)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roots": roots,
		"count": len(files),
		"files": files,
	})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	roots := r.URL.Query()["root"]
	if len(roots) == 0 {
		roots = s.config.MediaRoots
	}
	if len(roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "provide at least one root query parameter or set NEXORA_MEDIA_ROOTS",
		})
		return
	}

	result := db.IngestResult{}
	batch := make([]scanner.FileInfo, 0, ingestBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ingested, err := s.repository.IngestScannedFiles(r.Context(), batch)
		result.Imported += ingested.Imported
		if err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	err := s.scanner.Walk(r.Context(), roots, func(file scanner.FileInfo) error {
		result.Scanned++
		batch = append(batch, file)
		if len(batch) == cap(batch) {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "partial": result})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleSearchSync(w http.ResponseWriter, r *http.Request) {
	limit := 1000
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	documents, err := s.repository.ListSearchDocuments(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.search.IndexDocuments(r.Context(), documents)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 24
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	filterParts := make([]string, 0, 2)
	if rawType := strings.TrimSpace(r.URL.Query().Get("type")); rawType != "" {
		filterParts = append(filterParts, `type = "`+escapeFilterValue(rawType)+`"`)
	}
	if rawCategory := strings.TrimSpace(r.URL.Query().Get("category")); rawCategory != "" {
		filterParts = append(filterParts, `category_slug = "`+escapeFilterValue(rawCategory)+`"`)
	}

	result, err := s.search.SearchDocuments(r.Context(), query, limit, strings.Join(filterParts, " AND "))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMetadataLookup(w http.ResponseWriter, r *http.Request) {
	var request metadata.Query
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.metadata.Lookup(r.Context(), request)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, metadata.ErrNotConfigured) {
			status = http.StatusFailedDependency
		}
		if errors.Is(err, metadata.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}
