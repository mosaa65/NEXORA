package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nexora/server/internal/db"
)

// --- Category Handlers ---

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.repository.ListCategories(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":      len(categories),
		"categories": categories,
	})
}

func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		NameAR string `json:"name_ar"`
		NameEN string `json:"name_en"`
		Slug   string `json:"slug"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	cat, err := s.repository.CreateCategory(r.Context(), request.NameAR, request.NameEN, request.Slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

func (s *Server) handleCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "category id must be a positive integer"})
		return
	}

	var request struct {
		NameAR string `json:"name_ar"`
		NameEN string `json:"name_en"`
		Slug   string `json:"slug"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.UpdateCategory(r.Context(), id, request.NameAR, request.NameEN, request.Slug); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "category id must be a positive integer"})
		return
	}

	if err := s.repository.DeleteCategory(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_id": id})
}

// --- Collection Handlers ---

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	items, err := s.repository.ListCollections(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"collections": items})
}
func (s *Server) handleCollectionSave(w http.ResponseWriter, r *http.Request) {
	var req db.CollectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	var id int64
	if raw := r.PathValue("id"); raw != "" {
		var ok bool
		id, ok = parsePositiveID(raw)
		if !ok {
			writeJSON(w, 400, map[string]any{"error": "invalid collection id"})
			return
		}
	}
	item, err := s.repository.SaveCollection(r.Context(), id, req)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) handleCollectionDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, 400, map[string]any{"error": "invalid collection id"})
		return
	}
	if err := s.repository.DeleteCollection(r.Context(), id); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- Franchise (Provider Collection) Handlers ---

// handleFranchises is intentionally database-only. TMDB is contacted only by
// the enrich workflow; browsing a franchise rail is safe on an offline LAN.
func (s *Server) handleFranchises(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	items, err := s.repository.ListProviderCollections(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"franchises": items})
}

func (s *Server) handleFranchise(w http.ResponseWriter, r *http.Request) {
	item, err := s.repository.GetProviderCollection(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleFranchiseMedia(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	collection, result, err := s.repository.ListProviderCollectionMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	_, parts, partsErr := s.repository.ListProviderCollectionParts(r.Context(), r.PathValue("slug"))
	if partsErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": partsErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"franchise": collection, "total": result.Total, "items": result.Items, "parts": parts})
}

func (s *Server) handleFranchiseAdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "franchise id must be positive"})
		return
	}
	var update db.CatalogEntityAdminUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid update payload"})
		return
	}
	if err := s.repository.UpdateProviderCollectionAdmin(r.Context(), id, update); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

// handleFranchiseRefresh is the explicit admin control for an existing
// collection. It fetches en-US and ar-SA once, caches fields and images, then
// leaves all visitor-facing routes fully local.
func (s *Server) handleFranchiseRefresh(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "franchise id must be positive"})
		return
	}
	collection, err := s.repository.GetProviderCollectionByID(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	warnings := s.refreshProviderCollectionMetadata(r.Context(), collection.ExternalID)
	if len(warnings) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "some collection fields could not be refreshed", "warnings": warnings})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "external_id": collection.ExternalID})
}

// handleFranchiseRefreshMissing upgrades pre-existing lightweight collection
// links in a deliberate batch. It is never reached by customer browsing.
func (s *Server) handleFranchiseRefreshMissing(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	candidates, err := s.repository.ListProviderCollectionRefreshCandidates(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	refreshed, warnings := make([]string, 0, len(candidates)), make([]string, 0)
	for _, candidate := range candidates {
		issues := s.refreshProviderCollectionMetadata(r.Context(), candidate.ExternalID)
		if len(issues) > 0 {
			warnings = append(warnings, candidate.ExternalID+": "+strings.Join(issues, "; "))
			continue
		}
		refreshed = append(refreshed, candidate.ExternalID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "refreshed": refreshed, "warnings": warnings})
}

// --- People Handlers ---

// handlePeople and its detail endpoints are backed exclusively by media_credits
// and people tables populated during enrichment or the local rebuild action.
func (s *Server) handlePeople(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	people, err := s.repository.ListPeople(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"people": people})
}

func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	person, err := s.repository.GetPerson(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, person)
}

func (s *Server) handlePersonMedia(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	person, result, err := s.repository.ListPersonMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"person": person, "total": result.Total, "items": result.Items})
}

func (s *Server) handlePersonAdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "person id must be positive"})
		return
	}
	var update db.CatalogEntityAdminUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid update payload"})
		return
	}
	if err := s.repository.UpdatePersonAdmin(r.Context(), id, update); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

// --- Catalog Sync ---

// handleCatalogRelationSync deliberately uses only cached TMDB snapshots. It
// backfills imported libraries without another network request or an API key.
func (s *Server) handleCatalogRelationSync(w http.ResponseWriter, r *http.Request) {
	result, err := s.repository.SyncCatalogRelationsFromSnapshots(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

// --- Showcase Handlers ---

func (s *Server) handleShowcases(w http.ResponseWriter, r *http.Request) {
	limit := 6
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	result, err := s.repository.ListShowcases(r.Context(), db.ShowcaseOptions{
		Context:      strings.TrimSpace(r.URL.Query().Get("context")),
		CategorySlug: strings.TrimSpace(r.URL.Query().Get("category")),
		Limit:        limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- Smart Hub Handlers ---

func (s *Server) handleSmartHubs(w http.ResponseWriter, r *http.Request) {
	hubs, err := s.repository.ListSmartHubs(r.Context(), strings.TrimSpace(r.URL.Query().Get("scope")))
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hubs": hubs})
}
func (s *Server) handleSmartHub(w http.ResponseWriter, r *http.Request) {
	hub, err := s.repository.GetSmartHub(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, hub)
}
func (s *Server) handleSmartHubMedia(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	result, hub, err := s.repository.ListSmartHubMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Offset: offset, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hub": hub, "total": result.Total, "limit": result.Limit, "offset": result.Offset, "items": result.Items})
}
func (s *Server) handleSmartHubsAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := s.repository.ListSmartHubsAdmin(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hubs": items})
}
func (s *Server) handleSmartHubSave(w http.ResponseWriter, r *http.Request) {
	var req db.SmartHubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	item, err := s.repository.SaveSmartHub(r.Context(), r.PathValue("slug"), req)
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) handleSmartHubCreate(w http.ResponseWriter, r *http.Request) {
	var req db.SmartHubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	item, err := s.repository.SaveSmartHub(r.Context(), "", req)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
