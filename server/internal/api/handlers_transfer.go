package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"nexora/server/internal/transfer"
)

func (s *Server) handleOpenFileLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileID int64  `json:"file_id"`
		Path   string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid payload"})
		return
	}

	filePath := req.Path
	if filePath == "" && req.FileID > 0 {
		var err error
		filePath, err = s.repository.GetVideoFilePath(r.Context(), req.FileID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "file not found in database"})
			return
		}
	}
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file path is required"})
		return
	}

	cleanPath, err := filepath.Abs(filePath)
	if err != nil {
		cleanPath = filePath
	}

	switch runtime.GOOS {
	case "windows":
		if _, err := os.Stat(cleanPath); err == nil {
			_ = exec.Command("cmd", "/c", "explorer", "/select,", cleanPath).Run()
		} else {
			dir := filepath.Dir(cleanPath)
			_ = os.MkdirAll(dir, 0o755)
			_ = exec.Command("cmd", "/c", "start", "", dir).Run()
		}
	case "darwin":
		_ = exec.Command("open", "-R", cleanPath).Run()
	default:
		_ = exec.Command("xdg-open", filepath.Dir(cleanPath)).Run()
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": cleanPath})
}

func (s *Server) handleTransferDevices(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusOK, map[string]any{"devices": []transfer.Device{}, "count": 0})
		return
	}
	devices, err := s.transfer.ListDevices(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"devices": devices,
		"count":   len(devices),
	})
}

func (s *Server) handleTransferDeviceApps(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusOK, map[string]any{"apps": []transfer.DeviceApp{}})
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	apps, err := s.transfer.ListDeviceApps(r.Context(), deviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(apps), "apps": apps})
}

func (s *Server) handleTransferDeviceAppFolders(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusOK, map[string]any{"folders": []transfer.AppFolder{}})
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	bundleID := r.URL.Query().Get("bundle_id")
	folders, err := s.transfer.ListAppFolders(r.Context(), deviceID, bundleID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(folders), "folders": folders})
}

// handleTransferBrowse lists a remote folder lazily (Phase 3). Query params:
// device_id, path, device_type (optional), bundle_id (iOS app, optional).
func (s *Server) handleTransferBrowse(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []transfer.RemoteEntry{}})
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	path := r.URL.Query().Get("path")
	deviceType := transfer.DeviceType(r.URL.Query().Get("device_type"))
	bundleID := r.URL.Query().Get("bundle_id")
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "device_id مطلوب"})
		return
	}
	entries, err := s.transfer.ListDevicePath(r.Context(), deviceID, path, deviceType, bundleID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []transfer.RemoteEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "count": len(entries), "entries": entries})
}

// handleTransferMkdir creates a folder (and parents) on a device (Phase 3).
func (s *Server) handleTransferMkdir(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "خدمة نقل USB غير مفعلة"})
		return
	}
	var req struct {
		DeviceID   string `json:"device_id"`
		Path       string `json:"path"`
		DeviceType string `json:"device_type"`
		BundleID   string `json:"bundle_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "بيانات غير صالحة"})
		return
	}
	if req.DeviceID == "" || req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "device_id و path مطلوبان"})
		return
	}
	if err := s.transfer.CreateDeviceFolder(r.Context(), req.DeviceID, req.Path, transfer.DeviceType(req.DeviceType), req.BundleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": req.Path})
}

// handleTransferEject safely unmounts and ejects a USB drive
func (s *Server) handleTransferEject(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "خدمة نقل USB غير مفعلة"})
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "بيانات غير صالحة"})
		return
	}
	if req.DeviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "معرف الجهاز مطلوب"})
		return
	}
	if err := s.transfer.EjectDevice(r.Context(), req.DeviceID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "تم إخراج القرص بأمان"})
}

func (s *Server) handleTransferCopy(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "خدمة نقل USB غير مفعلة"})
		return
	}

	var req struct {
		DeviceID     string   `json:"device_id"`
		SourcePath   string   `json:"source_path"`
		SourcePaths  []string `json:"source_paths"`
		FileID       int64    `json:"file_id"`
		FileIDs      []int64  `json:"file_ids"`
		TargetApp    string   `json:"target_app"`
		SubFolder    string   `json:"sub_folder"`
		TargetFolder string   `json:"target_folder"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "بيانات غير صالحة"})
		return
	}

	if len(req.FileIDs) > 0 {
		for _, fid := range req.FileIDs {
			if fid <= 0 {
				continue
			}
			path, err := s.repository.GetVideoFilePath(r.Context(), fid)
			if err == nil && strings.TrimSpace(path) != "" {
				req.SourcePaths = append(req.SourcePaths, path)
			}
		}
		if len(req.SourcePaths) > 0 && strings.TrimSpace(req.SourcePath) == "" {
			req.SourcePath = req.SourcePaths[0]
		}
	} else if req.FileID > 0 {
		path, err := s.repository.GetVideoFilePath(r.Context(), req.FileID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "تعذر العثور على ملف الفيديو بالمعرف المحدد"})
			return
		}
		req.SourcePath = path
		if len(req.SourcePaths) == 0 {
			req.SourcePaths = []string{path}
		}
	}
	if strings.TrimSpace(req.SourcePath) == "" && len(req.SourcePaths) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "مسار الملف المصدر مطلوب"})
		return
	}

	job, err := s.transfer.StartCopy(r.Context(), transfer.CopyRequest{
		DeviceID:     req.DeviceID,
		SourcePath:   req.SourcePath,
		SourcePaths:  req.SourcePaths,
		FileID:       req.FileID,
		TargetApp:    req.TargetApp,
		SubFolder:    req.SubFolder,
		TargetFolder: req.TargetFolder,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (s *Server) handleTransferJobsList(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusOK, map[string]any{"jobs": []*transfer.TransferJob{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": s.transfer.ListJobs()})
}

func (s *Server) handleTransferJobGet(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "job not found"})
		return
	}
	id := r.PathValue("id")
	job, exists := s.transfer.GetJob(id)
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "لم يتم العثور على مهمة النسخ المطلوبة"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleTransferJobCancel(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "transfer service unavailable"})
		return
	}
	id := r.PathValue("id")
	if ok := s.transfer.CancelJob(id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "تعذر إلغاء المهمة أو أنها غير موجودة"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "تم إلغاء مهمة النسخ بنجاح"})
}

// handleTransferEvents streams live transfer events to the client as Server-Sent Events.
func (s *Server) handleTransferEvents(w http.ResponseWriter, r *http.Request) {
	if s.transfer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "transfer service unavailable"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming غير مدعوم"})
		return
	}

	stream, unsubscribe := s.transfer.SubscribeEvents(256)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Seed client with current snapshot
	if raw, err := json.Marshal(transfer.TransferEvent{Type: transfer.EventDevices, Devices: s.transfer.SnapshotDevices()}); err == nil {
		fmt.Fprintf(w, "event: devices\ndata: %s\n\n", raw)
	}
	if jobs := s.transfer.ListJobs(); jobs != nil {
		if raw, err := json.Marshal(transfer.TransferEvent{Type: transfer.EventJobs, Jobs: jobs}); err == nil {
			fmt.Fprintf(w, "event: jobs\ndata: %s\n\n", raw)
		}
	}
	flusher.Flush()

	ctx := r.Context()
	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-stream:
			raw, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, raw)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
