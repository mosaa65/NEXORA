package api

import (
	"database/sql"
	"errors"
	"net/http"
)

// handleMediaPlayback returns everything the watch screen needs in one read.
//
// The watch screen previously opened with two calls: the full media detail and a
// paged episode-index search. On a show with hundreds of episodes the second call
// was up to fifty requests, and none of it was needed before the first frame. This
// endpoint answers the playback question directly — which file plays, what order
// surrounds it, what comes next — and leaves the descriptive catalogue work to the
// detail endpoint.
//
// Query parameters:
//
//	file  which item to start from. It accepts either a `video_files` id (what a
//	      stream URL or a player bookmark holds) or an episode id (what the work
//	      details page navigates with). The repository resolves both explicitly.
//	      An unknown value falls back to the catalogue's first file rather than
//	      failing, because a stale link should still play something.
func (s *Server) handleMediaPlayback(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	var wanted int64
	if raw := r.URL.Query().Get("file"); raw != "" {
		parsed, valid := parsePositiveID(raw)
		if !valid {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file must be a positive integer"})
			return
		}
		wanted = parsed
	}

	plan, err := s.repository.GetPlaybackPlan(r.Context(), mediaID, wanted)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, plan)
}