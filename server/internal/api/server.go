package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/migration"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type repository interface {
	Health(ctx context.Context) (db.Health, error)
	IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error)
	ListCategories(ctx context.Context) ([]db.CategorySummary, error)
	ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error)
	ListVideoFiles(ctx context.Context, mediaItemID int64) ([]db.VideoFile, error)
	GetVideoFilePath(ctx context.Context, id int64) (string, error)
	GetVideoFileIDByPath(ctx context.Context, path string) (int64, error)
	UpdateVideoTechnicalDetails(ctx context.Context, id int64, details media.InspectResult) error
	ListDuplicateGroups(ctx context.Context) ([]db.DuplicateGroup, error)
	ListMissingEpisodes(ctx context.Context) ([]db.MissingEpisode, error)
	CalculateChecksums(ctx context.Context, mediaItemID int64) (db.ChecksumResult, error)
}

type searchClient interface {
	IndexDocuments(ctx context.Context, documents []search.MediaDocument) (search.SyncResult, error)
	SearchDocuments(ctx context.Context, query string, limit int, filter string) (search.SearchResult, error)
}

type metadataService interface {
	Lookup(ctx context.Context, query metadata.Query) (metadata.Result, error)
}

type mediaProcessor interface {
	Verify(ctx context.Context, path string) (media.VerifyResult, error)
	Inspect(ctx context.Context, path string) (media.InspectResult, error)
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string, at time.Duration) (string, error)
}

type migrationService interface {
	Preview(ctx context.Context, root string) (migration.PreviewResult, error)
	Copy(ctx context.Context, request migration.CopyRequest) (migration.CopyResult, error)
}

type Server struct {
	config     config.Config
	repository repository
	scanner    *scanner.Scanner
	search     searchClient
	metadata   metadataService
	processor  mediaProcessor
	migration  migrationService
	mux        *http.ServeMux
}

func NewServer(
	config config.Config,
	repository repository,
	scannerService *scanner.Scanner,
	searchClient searchClient,
	metadataService metadataService,
	processor mediaProcessor,
	migrationService migrationService,
) http.Handler {
	server := &Server{
		config:     config,
		repository: repository,
		scanner:    scannerService,
		search:     searchClient,
		metadata:   metadataService,
		processor:  processor,
		migration:  migrationService,
		mux:        http.NewServeMux(),
	}
	server.routes()
	return server.withMiddleware(server.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/categories", s.handleCategories)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/media/{id}/files", s.handleMediaFiles)
	s.mux.HandleFunc("GET /api/library/duplicates", s.handleDuplicates)
	s.mux.HandleFunc("GET /api/library/missing-episodes", s.handleMissingEpisodes)
	s.mux.HandleFunc("GET /api/scan", s.handleScan)
	s.mux.HandleFunc("POST /api/ingest", s.handleIngest)
	s.mux.HandleFunc("POST /api/index", s.handleIndex)
	s.mux.HandleFunc("POST /api/search/sync", s.handleSearchSync)
	s.mux.HandleFunc("POST /api/metadata/lookup", s.handleMetadataLookup)
	s.mux.HandleFunc("POST /api/media/verify", s.handleMediaVerify)
	s.mux.HandleFunc("POST /api/media/inspect", s.handleMediaInspect)
	s.mux.HandleFunc("POST /api/media/thumbnail", s.handleThumbnail)
	s.mux.HandleFunc("POST /api/media/checksums", s.handleChecksums)
	s.mux.HandleFunc("POST /api/migration/preview", s.handleMigrationPreview)
	s.mux.HandleFunc("POST /api/migration/copy", s.handleMigrationCopy)
	s.mux.HandleFunc("GET /api/stream", s.handleStream)
	s.mux.HandleFunc("GET /api/stream/file/{id}", s.handleStreamByID)
}

type indexResult struct {
	Roots         []string          `json:"roots"`
	Scanned       int               `json:"scanned"`
	Imported      int               `json:"imported"`
	Inspected     int               `json:"inspected"`
	InspectFailed int               `json:"inspectFailed"`
	SearchSync    search.SyncResult `json:"searchSync"`
	Warnings      []string          `json:"warnings,omitempty"`
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
	files, err := s.scanner.Scan(r.Context(), request.Roots)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	result := indexResult{Roots: request.Roots, Scanned: len(files), Warnings: []string{}}
	ingested, err := s.repository.IngestScannedFiles(r.Context(), files)
	result.Imported = ingested.Imported
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}
	for _, file := range files {
		id, err := s.repository.GetVideoFileIDByPath(r.Context(), file.Path)
		if err != nil {
			result.InspectFailed++
			result.Warnings = append(result.Warnings, "could not locate indexed file: "+file.Path)
			continue
		}
		details, err := s.processor.Inspect(r.Context(), file.Path)
		if err != nil {
			result.InspectFailed++
			result.Warnings = append(result.Warnings, "could not inspect: "+file.Path)
			continue
		}
		if err := s.repository.UpdateVideoTechnicalDetails(r.Context(), id, details); err != nil {
			result.InspectFailed++
			result.Warnings = append(result.Warnings, "could not save inspection: "+file.Path)
			continue
		}
		result.Inspected++
	}
	documents, err := s.repository.ListSearchDocuments(r.Context(), 10000)
	if err != nil {
		result.Warnings = append(result.Warnings, "search sync skipped: "+err.Error())
	} else if syncResult, err := s.search.IndexDocuments(r.Context(), documents); err != nil {
		result.Warnings = append(result.Warnings, "search sync skipped: "+err.Error())
	} else {
		result.SearchSync = syncResult
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMediaInspect(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FileID int64 `json:"fileId"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if request.FileID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId must be a positive integer"})
		return
	}
	path, err := s.repository.GetVideoFilePath(r.Context(), request.FileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}
	details, err := s.processor.Inspect(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error()})
		return
	}
	if err := s.repository.UpdateVideoTechnicalDetails(r.Context(), request.FileID, details); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) handleDuplicates(w http.ResponseWriter, r *http.Request) {
	groups, err := s.repository.ListDuplicateGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(groups), "groups": groups})
}

func (s *Server) handleMissingEpisodes(w http.ResponseWriter, r *http.Request) {
	missing, err := s.repository.ListMissingEpisodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(missing), "episodes": missing})
}

func (s *Server) handleChecksums(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MediaItemID int64 `json:"mediaItemId,omitempty"`
	}
	if r.Body != http.NoBody {
		if err := decodeJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
	}
	if request.MediaItemID < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "mediaItemId must be positive"})
		return
	}
	result, err := s.repository.CalculateChecksums(r.Context(), request.MediaItemID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health, err := s.repository.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":    false,
			"error": err.Error(),
			"time":  time.Now().UTC(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "database": health})
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

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.repository.ListCategories(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":      len(categories),
		"categories": categories,
	})
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

	files, err := s.scanner.Scan(r.Context(), roots)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.repository.IngestScannedFiles(r.Context(), files)
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

func (s *Server) handleMediaVerify(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(request.Path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}

	result, err := s.processor.Verify(r.Context(), request.Path)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path       string `json:"path"`
		OutputPath string `json:"outputPath,omitempty"`
		Second     int    `json:"second,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(request.Path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}
	if request.OutputPath == "" {
		base := strings.TrimSuffix(filepath.Base(request.Path), filepath.Ext(request.Path))
		request.OutputPath = filepath.Join(s.config.AssetImageDir, "thumbnails", safeFileName(base)+".jpg")
	}

	outputPath, err := s.processor.GenerateThumbnail(r.Context(), request.Path, request.OutputPath, time.Duration(request.Second)*time.Second)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"thumbnailPath": outputPath})
}

func (s *Server) handleMigrationPreview(w http.ResponseWriter, r *http.Request) {
	var request migration.PreviewRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(request.Root) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "root is required"})
		return
	}
	if !s.mediaPathAllowed(request.Root) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration root is outside configured media roots"})
		return
	}
	result, err := s.migration.Preview(r.Context(), request.Root)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMigrationCopy(w http.ResponseWriter, r *http.Request) {
	var request migration.CopyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(request.Target) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration target is outside configured media roots"})
		return
	}
	for _, source := range request.Sources {
		if !s.mediaPathAllowed(source) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration source is outside configured media roots"})
			return
		}
	}
	result, err := s.migration.Copy(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	s.serveMediaPath(w, r, path)
}

func (s *Server) handleStreamByID(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	s.serveMediaPath(w, r, path)
}

func (s *Server) serveMediaPath(w http.ResponseWriter, r *http.Request, path string) {
	if !s.mediaPathAllowed(path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}

	file, err := os.Open(path)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "path must point to a file"})
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) mediaPathAllowed(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if len(s.config.MediaRoots) == 0 {
		return true
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absolutePath = strings.ToLower(filepath.Clean(absolutePath))

	for _, root := range s.config.MediaRoots {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		absoluteRoot = strings.ToLower(filepath.Clean(absoluteRoot))
		if absolutePath == absoluteRoot || strings.HasPrefix(absolutePath, absoluteRoot+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

var unsafeFileName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeFileName(input string) string {
	input = strings.Trim(unsafeFileName.ReplaceAllString(input, "_"), "._-")
	if input == "" {
		return "thumbnail"
	}
	return input
}

func escapeFilterValue(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}

func parsePositiveID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}
