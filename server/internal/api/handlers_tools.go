package api

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"nexora/server/internal/migration"
)

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
	redisOK, redisStatus := s.cache.RedisStatus(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"database": health,
		"cache": map[string]any{
			"redisOk":     redisOK,
			"redisStatus": redisStatus,
		},
	})
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

func (s *Server) handleMediaVerify(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FileID int64  `json:"fileId,omitempty"`
		Path   string `json:"path,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if request.FileID < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId must be a positive integer"})
		return
	}
	if request.FileID > 0 {
		path, err := s.repository.GetVideoFilePath(r.Context(), request.FileID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		request.Path = path
	}
	if strings.TrimSpace(request.Path) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId or path is required"})
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
	if request.FileID == 0 {
		if id, lookupErr := s.repository.GetVideoFileIDByPath(r.Context(), request.Path); lookupErr == nil {
			request.FileID = id
		}
	}
	if request.FileID > 0 {
		if err := s.repository.UpdateVideoVerification(r.Context(), request.FileID, result); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
			return
		}
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

func (s *Server) handleCorruptedFiles(w http.ResponseWriter, r *http.Request) {
	files, err := s.repository.ListCorruptedFiles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(files), "files": files})
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

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repository.GetDashboardStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleQualityReport(w http.ResponseWriter, r *http.Request) {
	if s.quality == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "quality service not available"})
		return
	}

	report, err := s.quality.GenerateReport(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleDisksList(w http.ResponseWriter, r *http.Request) {
	disks, err := s.repository.ListDisks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// If no disks recorded yet, auto-scan
	if len(disks) == 0 && s.diskManager != nil {
		if scanned, scanErr := s.diskManager.ScanDisks(r.Context()); scanErr == nil && len(scanned) > 0 {
			_ = s.repository.SaveDisks(r.Context(), scanned)
			disks = scanned
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(disks),
		"disks": disks,
	})
}

func (s *Server) handleDisksScan(w http.ResponseWriter, r *http.Request) {
	if s.diskManager == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "disk manager not available"})
		return
	}

	disks, err := s.diskManager.ScanDisks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.SaveDisks(r.Context(), disks); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "disks": disks})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(disks),
		"disks": disks,
	})
}
