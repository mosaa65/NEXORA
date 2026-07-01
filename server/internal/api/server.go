package api

import (
	"context"
	"errors"
	"encoding/json"
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
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type repository interface {
	Health(ctx context.Context) (db.Health, error)
	IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error)
	ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error)
}

type searchClient interface {
	IndexDocuments(ctx context.Context, documents []search.MediaDocument) (search.SyncResult, error)
}

type metadataService interface {
	Lookup(ctx context.Context, query metadata.Query) (metadata.Result, error)
}

type mediaProcessor interface {
	Verify(ctx context.Context, path string) (media.VerifyResult, error)
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string, at time.Duration) (string, error)
}

type Server struct {
	config     config.Config
	repository repository
	scanner    *scanner.Scanner
	search     searchClient
	metadata   metadataService
	processor  mediaProcessor
	mux        *http.ServeMux
}

func NewServer(
	config config.Config,
	repository repository,
	scannerService *scanner.Scanner,
	searchClient searchClient,
	metadataService metadataService,
	processor mediaProcessor,
) http.Handler {
	server := &Server{
		config:     config,
		repository: repository,
		scanner:    scannerService,
		search:     searchClient,
		metadata:   metadataService,
		processor:  processor,
		mux:        http.NewServeMux(),
	}
	server.routes()
	return server.withMiddleware(server.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/scan", s.handleScan)
	s.mux.HandleFunc("POST /api/ingest", s.handleIngest)
	s.mux.HandleFunc("POST /api/search/sync", s.handleSearchSync)
	s.mux.HandleFunc("POST /api/metadata/lookup", s.handleMetadataLookup)
	s.mux.HandleFunc("POST /api/media/verify", s.handleMediaVerify)
	s.mux.HandleFunc("POST /api/media/thumbnail", s.handleThumbnail)
	s.mux.HandleFunc("GET /api/stream", s.handleStream)
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

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
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
