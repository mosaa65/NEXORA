package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"nexora/server/internal/db"
)

// resolutionQueueItem is the shape the admin review screen consumes. It carries
// the detected metadata, why resolution refused to decide, and the ranked
// candidates with their itemised evidence, so an operator can choose rather than
// retype.
type resolutionQueueItem struct {
	ID                 int64   `json:"id"`
	FilePath           string  `json:"file_path"`
	RelativePath       string  `json:"relative_path,omitempty"`
	OriginalFilename   string  `json:"original_filename,omitempty"`
	Size               int64   `json:"size"`
	DetectedTitle      string  `json:"detected_title,omitempty"`
	DetectedCategory   string  `json:"detected_category,omitempty"`
	DetectedMediaType  string  `json:"detected_media_type,omitempty"`
	DetectedSeason     int     `json:"detected_season,omitempty"`
	DetectedEpisode    int     `json:"detected_episode,omitempty"`
	DetectedYear       int     `json:"detected_year,omitempty"`
	ParserConfidence   float64 `json:"parser_confidence"`
	ResolverConfidence float64 `json:"resolver_confidence"`
	Reason             string  `json:"reason"`
	ReasonDetail       string  `json:"reason_detail,omitempty"`
	Candidates         any     `json:"candidates"`
	State              string  `json:"state"`
	CreatedAt          string  `json:"created_at"`
}

// handleResolutionQueue lists files that entity resolution refused to decide.
//
// These are the files that would previously have become works named "01" or a
// watermark. They are surfaced for an operator instead of being guessed at.
//
// Query parameters:
//
//	reason  filter by reason code (ambiguous_work, unknown_media_type, ...)
//	limit   page size, default 100, maximum 500
func (s *Server) handleResolutionQueue(w http.ResponseWriter, r *http.Request) {
	reason := strings.TrimSpace(r.URL.Query().Get("reason"))

	limit := 100
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	items, err := s.repository.ListResolutionQueue(r.Context(), reason, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	counts, err := s.repository.CountResolutionQueue(r.Context())
	if err != nil {
		counts = map[string]int{}
	}

	// Decode the stored candidate JSON so the client receives structured
	// evidence rather than a string it would have to parse itself.
	view := make([]resolutionQueueItem, 0, len(items))
	for _, item := range items {
		view = append(view, resolutionQueueItem{
			ID:                 item.ID,
			FilePath:           item.FilePath,
			RelativePath:       item.RelativePath,
			OriginalFilename:   item.OriginalFilename,
			Size:               item.Size,
			DetectedTitle:      item.DetectedTitle,
			DetectedCategory:   item.DetectedCategory,
			DetectedMediaType:  item.DetectedMediaType,
			DetectedSeason:     item.DetectedSeason,
			DetectedEpisode:    item.DetectedEpisode,
			DetectedYear:       item.DetectedYear,
			ParserConfidence:   item.ParserConfidence,
			ResolverConfidence: item.ResolverConfidence,
			Reason:             item.Reason,
			ReasonDetail:       item.ReasonDetail,
			Candidates:         decodeCandidateJSON(item.CandidatesJSON),
			State:              item.State,
			CreatedAt:          item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"byReason": counts,
		"returned": len(view),
		"items":    view,
	})
}

// handleResolutionDecide applies an operator decision to a queued file.
//
// The decision is recorded for auditability and, when safe, promoted into a
// durable alias so the same ambiguity never appears again. This is the mechanism
// that makes the library learn without an LLM.
func (s *Server) handleResolutionDecide(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid queue item id"})
		return
	}

	var request struct {
		Action     string `json:"action"`
		WorkID     int64  `json:"work_id,omitempty"`
		Season     int    `json:"season,omitempty"`
		Episode    int    `json:"episode,omitempty"`
		NewTitle   string `json:"new_title,omitempty"`
		LearnAlias bool   `json:"learn_alias"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	action := strings.TrimSpace(request.Action)
	switch action {
	case "attach", "create", "ignore", "mark_movie", "move_season":
	// Allowed: the full set the resolution pipeline understands.
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "action must be one of: attach, create, ignore, mark_movie, move_season",
		})
		return
	}

	// A missing work id on an attach action is a client error, and catching it
	// here gives a clearer message than the repository's own validation.
	decision := db.ResolutionDecision{
		Action:     action,
		WorkID:     request.WorkID,
		Season:     request.Season,
		Episode:    request.Episode,
		NewTitle:   strings.TrimSpace(request.NewTitle),
		LearnAlias: request.LearnAlias,
		DecidedBy:  s.adminIdentity(r),
	}

	if err := s.repository.ApplyResolutionDecision(r.Context(), id, decision); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"itemId":  id,
		"action":  action,
		"learned": request.LearnAlias,
	})
}

// handleResolutionStats reports how much of the library still needs a human, per
// reason. It is the number an operator watches to know whether the library is
// converging.
func (s *Server) handleResolutionStats(w http.ResponseWriter, r *http.Request) {
	counts, err := s.repository.CountResolutionQueue(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	// Ranked so the operator sees the most common cause first.
	type reasonCount struct {
		Reason string `json:"reason"`
		Count  int    `json:"count"`
	}
	ranked := make([]reasonCount, 0, len(counts))
	for reason, count := range counts {
		ranked = append(ranked, reasonCount{Reason: reason, Count: count})
	}
	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].Count > ranked[j-1].Count; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"byReason": ranked,
	})
}

// decodeCandidateJSON turns the stored candidate payload into structured data.
//
// A malformed payload degrades to an empty list rather than failing the whole
// response: the queue must stay readable even if one row is corrupt.
func decodeCandidateJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []any{}
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return []any{}
	}
	return decoded
}

// adminIdentity records who made a resolution decision.
//
// The credential in use is the configured admin account name, because the
// project has a single operator role today. The source address is appended so a
// decision taken from a different workstation is still distinguishable in the
// audit trail.
func (s *Server) adminIdentity(r *http.Request) string {
	name := s.config.AdminUser
	if name == "" {
		name = "admin"
	}
	if ip := clientIP(r); ip != "" {
		return name + "@" + ip
	}
	return name
}
