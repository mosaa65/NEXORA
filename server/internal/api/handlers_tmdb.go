package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexora/server/internal/db"
	"nexora/server/internal/metadata"
)

func (s *Server) handleTMDBSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleTMDBQueueGet(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.repository.ListTMDBQueue(r.Context(), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleTMDBUsageHistory(w http.ResponseWriter, r *http.Request) {
	days := 90
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			days = parsed
		}
	}
	history, err := s.repository.GetTMDBUsageHistory(r.Context(), days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": history})
}

func (s *Server) handleTMDBQueueCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MediaItemID int64 `json:"media_item_id"`
		Priority    int   `json:"priority"`
	}
	if err := decodeJSON(r, &request); err != nil || request.MediaItemID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media_item_id must be positive"})
		return
	}
	if err := s.repository.EnqueueTMDBRefresh(r.Context(), request.MediaItemID, request.Priority); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "media_item_id": request.MediaItemID})
}

func (s *Server) handleTMDBQueueCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "queue id must be positive"})
		return
	}
	if err := s.repository.CancelTMDBQueueJob(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleTMDBCandidates(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	if title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "title is required"})
		return
	}
	year := 0
	if raw := r.URL.Query().Get("year"); raw != "" {
		year, _ = strconv.Atoi(raw)
	}
	candidates, err := s.metadata.SearchCandidates(r.Context(), metadata.Query{
		Title:    title,
		Type:     strings.TrimSpace(r.URL.Query().Get("type")),
		Year:     year,
		Language: "en-US",
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}

func (s *Server) handleTMDBSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var settings metadata.TMDBSettings
	if err := decodeJSON(r, &settings); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.SaveTMDBSettings(r.Context(), settings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Sync into in-memory service
	s.metadata.SetTMDBSettings(settings)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "settings": settings})
}

func (s *Server) handleTMDBStatsGet(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repository.GetTMDBUsageSummary(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleTMDBModulesGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	modules := metadata.GetModuleList(settings.Modules)
	writeJSON(w, http.StatusOK, map[string]any{
		"fetch_mode": settings.FetchMode,
		"image_mode": settings.ImageMode,
		"modules":    modules,
	})
}

func (s *Server) handleTMDBModulesUpdate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FetchMode *metadata.FetchMode    `json:"fetch_mode,omitempty"`
		ImageMode *metadata.ImageMode    `json:"image_mode,omitempty"`
		Modules   *metadata.ModuleConfig `json:"modules,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	current, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if request.FetchMode != nil {
		current.ApplyProfile(*request.FetchMode)
	}
	if request.ImageMode != nil {
		current.ImageMode = *request.ImageMode
	}
	if request.Modules != nil {
		current.Modules = *request.Modules
		current.FetchMode = metadata.FetchModeCustom
	}

	if err := s.repository.SaveTMDBSettings(r.Context(), *current); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.metadata.SetTMDBSettings(*current)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"settings": current,
		"modules":  metadata.GetModuleList(current.Modules),
	})
}

func (s *Server) handleTMDBTestConnection(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	remoteConfig, err := s.metadata.FetchTMDBConfiguration(r.Context())
	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
			RequestKind:  "test_connection",
			Endpoint:     "/configuration",
			StatusCode:   502,
			DurationMS:   int(latency),
			ErrorMessage: err.Error(),
		})
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"connected": false,
			"error":     err.Error(),
			"latencyMs": latency,
		})
		return
	}

	_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
		RequestKind: "test_connection",
		Endpoint:    "/configuration",
		StatusCode:  200,
		DurationMS:  int(latency),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"connected":     true,
		"latencyMs":     latency,
		"configuration": remoteConfig,
		"checkedAt":     time.Now().UTC(),
	})
}

func (s *Server) handleTMDBConfigurationGet(w http.ResponseWriter, r *http.Request) {
	remoteConfig, err := s.metadata.FetchTMDBConfiguration(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, remoteConfig)
}

func (s *Server) handleTMDBPreviewGet(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	settings, _ := s.repository.GetTMDBSettings(r.Context())
	if settings == nil {
		def := metadata.DefaultSettings()
		settings = &def
	}

	// Calculate estimated requests & bytes
	estimatedRequests := 2 // Search + Details EN
	estimatedBytes := int64(350 * 1024)
	if settings.ImageMode == metadata.ImageModeLocal || settings.ImageMode == metadata.ImageModeHybrid {
		estimatedBytes += 250 * 1024 // Poster + Backdrop download
	}
	if settings.Modules.MaxCastImages > 0 && settings.ImageMode == metadata.ImageModeLocal {
		estimatedBytes += int64(settings.Modules.MaxCastImages * 30 * 1024)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"media_id":               mediaID,
		"title_en":               item.TitleEN,
		"title_ar":               item.TitleAR,
		"type":                   item.Type,
		"fetch_mode":             settings.FetchMode,
		"image_mode":             settings.ImageMode,
		"estimatedRequests":      estimatedRequests,
		"estimatedBytes":         estimatedBytes,
		"estimatedSizeFormatted": fmt.Sprintf("%.2f MB", float64(estimatedBytes)/(1024.0*1024.0)),
	})
}
