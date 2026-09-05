package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nexora/server/internal/media"
)

// handleStreamImage streams a local image file safely with ETag & long-term caching.
func (s *Server) handleStreamImage(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if filePath == "" {
		http.Error(w, "missing image path", http.StatusBadRequest)
		return
	}
	if !s.mediaPathAllowed(filePath) {
		http.Error(w, "forbidden path", http.StatusForbidden)
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "error reading image stat", http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".jfif", ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".bmp":
		contentType = "image/bmp"
	case ".svg":
		contentType = "image/svg+xml"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", fmt.Sprintf("\"%x-%x\"", stat.ModTime().Unix(), stat.Size()))
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
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

func (s *Server) handleFileSubtitles(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	subs := media.FindExternalSubtitles(path)
	for i := range subs {
		subs[i].Path = fmt.Sprintf("/api/stream/file/%d/subtitles/%d", fileID, subs[i].Index)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"file_id":   fileID,
		"count":     len(subs),
		"subtitles": subs,
	})
}

func (s *Server) handleFileSubtitleStream(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}
	subIndex, err := strconv.Atoi(r.PathValue("subId"))
	if err != nil || subIndex <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subId must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	subs := media.FindExternalSubtitles(path)
	var targetSub *media.SubtitleInfo
	for _, sub := range subs {
		if sub.Index == subIndex {
			targetSub = &sub
			break
		}
	}
	if targetSub == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "subtitle not found"})
		return
	}

	file, err := os.Open(targetSub.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if strings.ToLower(filepath.Ext(targetSub.Path)) == ".srt" {
		if err := media.ConvertSRTToWebVTT(file, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, _ = io.Copy(w, file)
	}
}

func (s *Server) mediaPathAllowed(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	// If no roots are configured (e.g. test environment), allow safe local relative paths without traversal
	if len(s.config.MediaRoots) == 0 && s.config.AssetImageDir == "" {
		clean := filepath.Clean(path)
		return !strings.HasPrefix(clean, "..") && !filepath.IsAbs(clean)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absolutePath = strings.ToLower(filepath.Clean(absolutePath))

	allowedRoots := make([]string, 0, len(s.config.MediaRoots)+1)
	allowedRoots = append(allowedRoots, s.config.MediaRoots...)
	if s.config.AssetImageDir != "" {
		allowedRoots = append(allowedRoots, s.config.AssetImageDir)
	}

	if len(allowedRoots) == 0 {
		return false
	}

	for _, root := range allowedRoots {
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
