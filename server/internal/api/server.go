package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
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

type Server struct {
	config     config.Config
	repository repository
	scanner    *scanner.Scanner
	search     searchClient
	mux        *http.ServeMux
}

func NewServer(config config.Config, repository repository, scannerService *scanner.Scanner, searchClient searchClient) http.Handler {
	server := &Server{
		config:     config,
		repository: repository,
		scanner:    scannerService,
		search:     searchClient,
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
