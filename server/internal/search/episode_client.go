package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ConfigureEpisodeIndex creates the episode index and applies its settings.
//
// The episode index is configured separately from the work index because the
// fields differ: an episode has season and episode numbers, an air date and a
// still image, while a work has a plot, genres and a rating. Sharing one
// configuration would mean every settings change had to satisfy both shapes.
func (c *Client) ConfigureEpisodeIndex(ctx context.Context) error {
	body := map[string]string{"uid": EpisodeIndexName, "primaryKey": "id"}
	response, err := c.doJSON(ctx, http.MethodPost, "/indexes", body)
	if err != nil {
		return err
	}
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	response.Body.Close()

	// An existing index is not an error: the settings below still need applying.
	if response.StatusCode >= 300 && !strings.Contains(string(payload), "index_already_exists") {
		return fmt.Errorf("create episode index: status %d: %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}

	settings := map[string]any{
		// Weighted by significance. The episode's own title comes first, then the
		// parent work's name, so "Ozymandias" finds the episode and
		// "Breaking Bad" finds everything from the show.
		"searchableAttributes": []string{
			"episode_title_en",
			"episode_title_ar",
			"episode_title_normalized",
			"work_title_en",
			"work_title_ar",
			"work_title_normalized",
			"overview_en",
			"overview_ar",
		},
		// Filtering is what makes "episodes of work 42" or "season 3 only" a
		// lookup rather than a scan, which is the whole reason this index exists.
		"filterableAttributes": []string{
			"work_id",
			"season_number",
			"episode_number",
			"category_slug",
			"has_local_file",
		},
		"sortableAttributes": []string{
			"season_number",
			"episode_number",
			"air_date",
		},
		"rankingRules": []string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
		// Meilisearch stops paginating at 1000 hits by default. A long-running
		// show exceeds that — the live library has a work with 1,181 episodes —
		// so the default would silently truncate a work's episode list at the
		// 1000th hit. Raised so a whole work can be paged through.
		"pagination": map[string]int{
			"maxTotalHits": 10000,
		},
	}

	path := "/indexes/" + url.PathEscape(EpisodeIndexName) + "/settings"
	response, err = c.doJSON(ctx, http.MethodPatch, path, settings)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("configure episode index: status %d: %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}

// IndexEpisodeDocuments writes episode documents to their own index.
func (c *Client) IndexEpisodeDocuments(ctx context.Context, documents []EpisodeDocument) (SyncResult, error) {
	if len(documents) == 0 {
		return SyncResult{}, nil
	}
	if err := c.ConfigureEpisodeIndex(ctx); err != nil {
		return SyncResult{}, err
	}

	path := "/indexes/" + url.PathEscape(EpisodeIndexName) + "/documents"
	response, err := c.doJSON(ctx, http.MethodPost, path, documents)
	if err != nil {
		return SyncResult{}, err
	}
	defer response.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode >= 300 {
		return SyncResult{}, fmt.Errorf("index episode documents: status %d: %s",
			response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return SyncResult{Indexed: len(documents), TaskUID: taskUIDFrom(payload)}, nil
}

// DeleteEpisodeDocuments removes episode documents by primary key.
func (c *Client) DeleteEpisodeDocuments(ctx context.Context, ids []int64) (SyncResult, error) {
	if len(ids) == 0 {
		return SyncResult{}, nil
	}
	if err := c.ConfigureEpisodeIndex(ctx); err != nil {
		return SyncResult{}, err
	}

	path := "/indexes/" + url.PathEscape(EpisodeIndexName) + "/documents/delete-batch"
	response, err := c.doJSON(ctx, http.MethodPost, path, ids)
	if err != nil {
		return SyncResult{}, err
	}
	defer response.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode >= 300 {
		return SyncResult{}, fmt.Errorf("delete episode documents: status %d: %s",
			response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return SyncResult{Indexed: len(ids), TaskUID: taskUIDFrom(payload)}, nil
}

// EpisodeDocumentIDs returns every primary key in the episode index.
//
// Paginated for the same reason as the work index: assuming one response holds
// the whole index would silently truncate on a large library, and a truncated
// orphan check leaves the missing tail forever unpruned.
func (c *Client) EpisodeDocumentIDs(ctx context.Context) ([]int64, error) {
	if err := c.ConfigureEpisodeIndex(ctx); err != nil {
		return nil, err
	}

	const pageSize = 1000
	const maxPages = 100000
	ids := make([]int64, 0, pageSize)
	for page := 0; page < maxPages; page++ {
		path := fmt.Sprintf("/indexes/%s/documents?fields=id&limit=%d&offset=%d",
			url.PathEscape(EpisodeIndexName), pageSize, page*pageSize)

		response, err := c.doJSON(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(io.LimitReader(response.Body, 8<<20))
		response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("list episode documents: status %d: %s",
				response.StatusCode, strings.TrimSpace(string(payload)))
		}

		var batch struct {
			Results []struct {
				ID int64 `json:"id"`
			} `json:"results"`
		}
		if err := json.Unmarshal(payload, &batch); err != nil {
			return nil, fmt.Errorf("decode episode documents: %w", err)
		}
		if len(batch.Results) == 0 {
			break
		}
		for _, entry := range batch.Results {
			ids = append(ids, entry.ID)
		}
		if len(batch.Results) < pageSize {
			break
		}
		if err := ctx.Err(); err != nil {
			return ids, err
		}
	}
	return ids, nil
}

// SearchEpisodes queries the episode index.
//
// It is separate from the work search because the return shape differs and
// because filtering by work is the common case: "show me the episodes of this
// season" is a filter on work_id and season_number, not a text query.
func (c *Client) SearchEpisodes(ctx context.Context, query string, limit int, filter string) (EpisodeSearchResult, error) {
	return c.SearchEpisodesPage(ctx, query, limit, 0, filter)
}

// SearchEpisodesPage is SearchEpisodes with an offset.
//
// A work with many episodes does not fit in one page: the endpoint caps a page
// at 200 hits, and a long-running show can hold more than that (the live library
// has a work with 23 seasons). Without an offset the caller would silently see
// only the first page, so a paging caller must be able to ask for the rest.
func (c *Client) SearchEpisodesPage(ctx context.Context, query string, limit, offset int, filter string) (EpisodeSearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	payload := map[string]any{
		"q":     query,
		"limit": limit,
	}
	if offset > 0 {
		payload["offset"] = offset
	}
	if strings.TrimSpace(filter) != "" {
		payload["filter"] = filter
	}

	response, err := c.doJSON(ctx, http.MethodPost, "/indexes/"+url.PathEscape(EpisodeIndexName)+"/search", payload)
	if err != nil {
		return EpisodeSearchResult{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return EpisodeSearchResult{}, err
	}
	if response.StatusCode >= 300 {
		return EpisodeSearchResult{}, fmt.Errorf("search episodes: status %d: %s",
			response.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw struct {
		Query              string            `json:"query"`
		EstimatedTotalHits int               `json:"estimatedTotalHits"`
		Hits               []EpisodeDocument `json:"hits"`
		ProcessingTimeMS   int               `json:"processingTimeMs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return EpisodeSearchResult{}, err
	}

	return EpisodeSearchResult{
		Query:              raw.Query,
		EstimatedTotalHits: raw.EstimatedTotalHits,
		Hits:               raw.Hits,
		ProcessingTimeMS:   raw.ProcessingTimeMS,
		Limit:              limit,
		Offset:             offset,
		Filter:             filter,
	}, nil
}

// EpisodeSearchResult is the response of an episode search.
type EpisodeSearchResult struct {
	Query              string            `json:"query"`
	EstimatedTotalHits int               `json:"estimatedTotalHits"`
	Hits               []EpisodeDocument `json:"hits"`
	ProcessingTimeMS   int               `json:"processingTimeMs"`
	Limit              int               `json:"limit"`
	Offset             int               `json:"offset"`
	Filter             string            `json:"filter,omitempty"`
}

// taskUIDFrom extracts the task identifier from a Meilisearch write response.
func taskUIDFrom(payload []byte) string {
	var task struct {
		TaskUID int `json:"taskUid"`
		UID     int `json:"uid"`
	}
	if err := json.Unmarshal(payload, &task); err != nil {
		return ""
	}
	if task.TaskUID != 0 {
		return fmt.Sprintf("%d", task.TaskUID)
	}
	if task.UID != 0 {
		return fmt.Sprintf("%d", task.UID)
	}
	return ""
}
