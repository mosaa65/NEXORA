package copybridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nexora/server/internal/transfer"
)

type Server struct {
	cfg Config
	svc *Service
	mux *http.ServeMux
}

func NewServer(cfg Config, svc *Service) http.Handler {
	server := &Server{
		cfg: cfg,
		svc: svc,
		mux: http.NewServeMux(),
	}
	server.routes()
	return server.withMiddleware(server.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/transfer/devices", s.handleTransferDevices)
	s.mux.HandleFunc("GET /api/transfer/device-apps", s.handleTransferDeviceApps)
	s.mux.HandleFunc("GET /api/transfer/device-app-folders", s.handleTransferDeviceAppFolders)
	s.mux.HandleFunc("GET /api/transfer/browse", s.handleTransferBrowse)
	s.mux.HandleFunc("POST /api/transfer/mkdir", s.handleTransferMkdir)
	s.mux.HandleFunc("POST /api/transfer/eject", s.handleTransferEject)
	s.mux.HandleFunc("POST /api/transfer/copy", s.handleTransferCopy)
	s.mux.HandleFunc("GET /api/transfer/jobs", s.handleTransferJobsList)
	s.mux.HandleFunc("GET /api/transfer/job/{id}", s.handleTransferJobGet)
	s.mux.HandleFunc("POST /api/transfer/cancel/{id}", s.handleTransferJobCancel)
	s.mux.HandleFunc("GET /api/transfer/events", s.handleTransferEvents)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "nexora-copy-bridge",
		"addr":    s.cfg.HTTPAddr,
	})
}

func (s *Server) handleTransferDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.svc.ListDevices(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
}

func (s *Server) handleTransferDeviceApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.svc.ListDeviceApps(r.Context(), r.URL.Query().Get("device_id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps, "count": len(apps)})
}

func (s *Server) handleTransferDeviceAppFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := s.svc.ListAppFolders(r.Context(), r.URL.Query().Get("device_id"), r.URL.Query().Get("bundle_id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": folders, "count": len(folders)})
}

func (s *Server) handleTransferBrowse(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	path := r.URL.Query().Get("path")
	deviceType := transfer.DeviceType(r.URL.Query().Get("device_type"))
	bundleID := r.URL.Query().Get("bundle_id")
	if deviceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "device_id مطلوب"})
		return
	}
	entries, err := s.svc.ListDevicePath(r.Context(), deviceID, path, deviceType, bundleID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []transfer.RemoteEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "entries": entries, "count": len(entries)})
}

func (s *Server) handleTransferMkdir(w http.ResponseWriter, r *http.Request) {
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
	if err := s.svc.CreateDeviceFolder(r.Context(), req.DeviceID, req.Path, transfer.DeviceType(req.DeviceType), req.BundleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": req.Path})
}

func (s *Server) handleTransferEject(w http.ResponseWriter, r *http.Request) {
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
	if err := s.svc.EjectDevice(r.Context(), req.DeviceID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "تم إخراج القرص بأمان"})
}

func (s *Server) handleTransferCopy(w http.ResponseWriter, r *http.Request) {
	var req CopyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "بيانات غير صالحة"})
		return
	}
	job, err := s.svc.StartCopy(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (s *Server) handleTransferJobsList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"jobs": s.svc.ListJobs()})
}

func (s *Server) handleTransferJobGet(w http.ResponseWriter, r *http.Request) {
	job, ok := s.svc.GetJob(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "لم يتم العثور على مهمة النسخ المطلوبة"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleTransferJobCancel(w http.ResponseWriter, r *http.Request) {
	if ok := s.svc.CancelJob(r.PathValue("id")); !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "تعذر إلغاء المهمة أو أنها غير موجودة"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "تم إلغاء مهمة النسخ بنجاح"})
}

func (s *Server) handleTransferEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming غير مدعوم"})
		return
	}

	stream, unsubscribe := s.svc.SubscribeEvents(256)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if raw, err := json.Marshal(transfer.TransferEvent{Type: transfer.EventDevices, Devices: s.svc.SnapshotDevices()}); err == nil {
		fmt.Fprintf(w, "event: devices\ndata: %s\n\n", raw)
	}
	if raw, err := json.Marshal(transfer.TransferEvent{Type: transfer.EventJobs, Jobs: s.svc.ListJobs()}); err == nil {
		fmt.Fprintf(w, "event: jobs\ndata: %s\n\n", raw)
	}
	flusher.Flush()

	ctx := r.Context()
	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-stream:
			raw, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, raw)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := corsAllow(origin, s.cfg.CORSOrigins)

		w.Header().Set("Vary", "Origin")
		if allowed != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowed)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}

		if r.Method == http.MethodOptions {
			if allowed == "" {
				writeJSON(w, http.StatusForbidden, map[string]any{"error": "منشأ الطلب غير مصرح به للوصول إلى خدمة NEXORA Copy Bridge"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if allowed == "" && origin != "" {
			// Block cross-origin requests from unapproved pages so a malicious
			// local webpage cannot trigger copy/eject/mkdir on this machine's
			// USB devices. Requests without an Origin header (curl, services,
			// same-machine tools) keep working.
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "منشأ الطلب غير مصرح به للوصول إلى خدمة NEXORA Copy Bridge"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// corsAllow decides whether a browser Origin may talk to the bridge. It returns
// the Access-Control-Allow-Origin header value, or "" if the origin must be
// rejected. Loopback origins are always accepted; an explicit configured list
// takes precedence; when no list is configured the default policy also accepts
// private/LAN origins so a NEXORA page served from the LAN IP works out of the
// box without environment setup.
func corsAllow(origin string, configured []string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	for _, raw := range configured {
		if strings.TrimSpace(raw) == "*" {
			return "*"
		}
	}
	if isLoopbackOrigin(origin) {
		return origin
	}
	if containsConfiguredOrigin(configured, origin) {
		return origin
	}
	if len(configured) == 0 && isLANOrigin(origin) {
		return origin
	}
	return ""
}

func containsConfiguredOrigin(configured []string, origin string) bool {
	for _, raw := range configured {
		entry := normalizeOrigin(raw)
		if entry == "*" {
			return true
		}
		if entry != "" && entry == normalizeOrigin(origin) {
			return true
		}
	}
	return false
}

func normalizeOrigin(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimRight(value, "/")
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		// Accept bare host[:port] entries by trying http/https; exact matches
		// only, so an explicit "192.168.1.10:8080" covers either scheme.
		if strings.HasPrefix(value, "*") {
			return "*"
		}
		u, err := url.Parse("http://" + value)
		if err != nil {
			return ""
		}
		value = u.Scheme + "://" + u.Host
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + strings.ToLower(parsed.Host)
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		host == "127.0.0.1" || host == "::1" || host == "[::1]"
}

// isLANOrigin matches private and link-local address space (RFC 1918,
// RFC 6598, RFC 4193 ULA, link-local). This is intentionally broader than an
// exact allowlist for the unconfigured default, so a NEXORA web UI served from
// the LAN IP keeps working, while public/internet-origin web pages remain
// blocked.
func isLANOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := parsed.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLoopback() || ip.IsUnspecified()
	}
	return false
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
