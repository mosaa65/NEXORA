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
	"time"

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

	s.serveCataloguePath(w, r, path)
}

// handleFilePreview returns a cached FFmpeg frame for timeline hovering. The
// requested timestamp is quantized to ten seconds, so one short hover creates
// one reusable image instead of running FFmpeg for every cursor movement.
func (s *Server) handleFilePreview(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}
	second, err := strconv.Atoi(r.URL.Query().Get("at"))
	if err != nil || second < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at must be a non-negative second"})
		return
	}
	second = (second / 10) * 10
	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}
	relative := filepath.ToSlash(filepath.Join("previews", strconv.FormatInt(fileID, 10), fmt.Sprintf("%d.jpg", second)))
	outputPath := filepath.Join(s.config.AssetImageDir, filepath.FromSlash(relative))
	if _, err := os.Stat(outputPath); err != nil {
		if _, err := s.processor.GenerateThumbnail(r.Context(), path, outputPath, time.Duration(second)*time.Second); err != nil {
			writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error()})
			return
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.Redirect(w, r, "/assets/images/"+relative, http.StatusTemporaryRedirect)
}

func (s *Server) serveMediaPath(w http.ResponseWriter, r *http.Request, path string) {
	if !s.mediaPathAllowed(path) {
		writeStreamError(w, http.StatusForbidden, "media path is outside configured roots")
		return
	}

	s.serveMediaFile(w, r, path)
}

// serveCataloguePath streams a file whose path was resolved from the catalogue
// database (GetVideoFilePath), not from user-supplied query input. Catalogued
// paths are already validated at scan time and may legitimately live outside
// the configured MediaRoots (e.g. network shares listed as disks). Re-validating
// them here caused HTTP 403 failures for playlist/stream-by-id and for the copy
// bridge fetching /api/stream/file/{id}; the catalogue is the source of truth.
func (s *Server) serveCataloguePath(w http.ResponseWriter, r *http.Request, path string) {
	s.serveMediaFile(w, r, path)
}

func (s *Server) serveMediaFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeStreamError(w, status, "media file is not available: "+err.Error())
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeStreamError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if info.IsDir() {
		writeStreamError(w, http.StatusBadRequest, "path must point to a file")
		return
	}

	// Go's mime table has no entry for several containers a media library really
	// holds (.mkv, .ts, .m2ts, .avi, .wmv, .flv). Without a type the browser guesses,
	// and a wrong guess on a media element surfaces as a generic decode failure
	// instead of a playable stream. TypeByExtension stays authoritative for tab
	// where it is more specific (the MP4 family), and the known container map
	// fills the gaps it leaves.
	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else if container := mediaContentType(filepath.Ext(path)); container != "" {
		w.Header().Set("Content-Type", container)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	// http.ServeContent already answers HEAD correctly: it writes the headers,
	// including Content-Length and the range advertisement, and no body.
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// mediaContentType maps the video containers the scanner indexes to the MIME type
// a browser needs in order to even try decoding the file.
func mediaContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".mkv":
		return "video/x-matroska"
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".ts":
		return "video/mp2t"
	case ".m2ts", ".mts":
		return "video/mp2t"
	}
	return ""
}

// writeStreamError answers a media request with the media's own content type.
//
// A player or a copy bridge that asks a stream endpoint for bytes and receives a
// JSON body labelled application/json reports it as an unreadable source, which is
// a worse diagnosis than no bytes at all. The body is still plain text so curl and
// the bridge can read the reason, and the status carries the meaning.
func writeStreamError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, message, status)
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
