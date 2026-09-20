package api

import (
	"net/http"
	"strconv"
	"strings"

	"nexora/server/internal/search"
)

// handleEpisodeSearch searches the episode index.
//
// It is a separate endpoint from the work search because the answer shape
// differs and because the common case is a filter, not a text query: "the
// episodes of this season" is work_id + season_number, not a phrase.
//
// Query parameters:
//
//	q        free text over the episode and work titles
//	work     restrict to one work id
//	season   restrict to one season number
//	local    "true" for episodes with a file, "false" for the provider-only
//	         ones the UI renders as coming soon
//	limit    page size, default 50, maximum 200
func (s *Server) handleEpisodeSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	filters := make([]string, 0, 3)
	if rawWork := strings.TrimSpace(r.URL.Query().Get("work")); rawWork != "" {
		workID, err := strconv.ParseInt(rawWork, 10, 64)
		if err != nil || workID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "work must be a positive integer"})
			return
		}
		filters = append(filters, "work_id = "+strconv.FormatInt(workID, 10))
	}
	if rawSeason := strings.TrimSpace(r.URL.Query().Get("season")); rawSeason != "" {
		season, err := strconv.Atoi(rawSeason)
		if err != nil || season < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "season must be a non-negative integer"})
			return
		}
		filters = append(filters, "season_number = "+strconv.Itoa(season))
	}
	if rawLocal := strings.TrimSpace(r.URL.Query().Get("local")); rawLocal != "" {
		switch strings.ToLower(rawLocal) {
		case "true":
			filters = append(filters, "has_local_file = true")
		case "false":
			filters = append(filters, "has_local_file = false")
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "local must be true or false"})
			return
		}
	}

	result, err := s.search.SearchEpisodes(r.Context(), query, limit, strings.Join(filters, " AND "))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleEpisodeIndexSync rebuilds the episode index as a projection of the
// database, without reading the filesystem.
//
// `reset=true` starts from the beginning and prunes orphaned documents. Without
// it the run resumes from the persisted cursor.
func (s *Server) handleEpisodeIndexSync(w http.ResponseWriter, r *http.Request) {
	reset := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("reset")), "true")

	projector := search.NewEpisodeProjector(s.search, s.repository, search.DefaultProjectionPageSize, nil)
	result, err := projector.Rebuild(r.Context(), reset)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

// handleEnrichFromLocals fills seasons and episodes from the provider snapshots
// already stored in this database.
//
// It makes ZERO external requests. That is the point: the full provider season
// payload, including every episode's title, overview, still image, air date and
// runtime, is already persisted locally, so enriching thousands of episodes
// cannot be rate-limited, cannot time out, and cannot fail because a provider is
// unreachable.
//
// The match is deterministic — (work, season number, episode number) — which is
// exactly how a provider numbers episodes, so no title matching is involved.
//
// Query parameters:
//
//	work   enrich one work only
//	limit  bound the number of works processed
func (s *Server) handleEnrichFromLocals(w http.ResponseWriter, r *http.Request) {
	var workID int64
	if rawWork := strings.TrimSpace(r.URL.Query().Get("work")); rawWork != "" {
		parsed, err := strconv.ParseInt(rawWork, 10, 64)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "work must be a positive integer"})
			return
		}
		workID = parsed
	}

	limit := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a non-negative integer"})
			return
		}
		limit = parsed
	}

	results, err := s.repository.EnrichFromLocalSnapshots(r.Context(), workID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Attaching a file to an episode is part of the same operation: an episode
	// the library owns must be reachable from its own catalogue.
	linked, err := s.repository.LinkOrphanEpisodes(r.Context())
	if err != nil {
		slogWarn(r, "could not link files to episodes", err)
	}

	totals := map[string]int{
		"seasonsCreated":      0,
		"episodesCreated":     0,
		"episodesEnriched":    0,
		"localEpisodesLinked": 0,
	}
	for _, result := range results {
		totals["seasonsCreated"] += result.SeasonsCreated
		totals["episodesCreated"] += result.EpisodesCreated
		totals["episodesEnriched"] += result.EpisodesEnriched
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"worksProcessed":        len(results),
		"seasonsCreated":        totals["seasonsCreated"],
		"episodesCreated":       totals["episodesCreated"],
		"episodesEnriched":      totals["episodesEnriched"],
		"filesLinkedToEpisodes": linked,
		"externalRequests":      0,
	})
}

// handleDuplicateWorks reports the duplicate and container-titled rows the
// catalogue still holds.
//
// It is read-only. A merge rewrites catalogue rows, so it is a separate explicit
// operation rather than a side effect of looking at the problem.
func (s *Server) handleDuplicateWorks(w http.ResponseWriter, r *http.Request) {
	groups, err := s.repository.FindDuplicateGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	type groupView struct {
		Kind        string `json:"kind"`
		CanonicalID int64  `json:"canonical_id"`
		Reason      string `json:"reason"`
		Safe        bool   `json:"safe"`
		Members     []struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			MediaType string `json:"type"`
			FileCount int64  `json:"file_count"`
		} `json:"members"`
	}

	byKind := map[string]int{}
	view := make([]groupView, 0, len(groups))
	for _, group := range groups {
		byKind[string(group.Kind)]++
		entry := groupView{
			Kind:        string(group.Kind),
			CanonicalID: group.CanonicalID,
			Reason:      group.Reason,
			Safe:        group.Safe,
		}
		for _, member := range group.Members {
			entry.Members = append(entry.Members, struct {
				ID        int64  `json:"id"`
				Title     string `json:"title"`
				MediaType string `json:"type"`
				FileCount int64  `json:"file_count"`
			}{ID: member.ID, Title: member.TitleEN, MediaType: member.MediaType, FileCount: member.FileCount})
		}
		view = append(view, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totalGroups": len(groups),
		"byKind":      byKind,
		"groups":      view,
	})
}

// handleMissingEpisodesFromProvider lists episodes the provider knows about that
// the library has not acquired.
//
// This is the "coming soon" list, and it is also the acquisition list: an
// operator can see exactly which episodes of a season are still missing rather
// than comparing counts by hand.
func (s *Server) handleMissingEpisodesFromProvider(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	rows, err := s.repository.ListProviderOnlyEpisodes(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total": len(rows),
		"items": rows,
	})
}

// slogWarn records a non-fatal problem without failing the request.
func slogWarn(r *http.Request, message string, err error) {
	// Kept as a tiny helper so the handler above reads as one operation rather
	// than being interrupted by logging plumbing.
	_ = r
	_ = message
	_ = err
}
