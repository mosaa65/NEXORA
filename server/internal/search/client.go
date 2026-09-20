package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Host   string
	APIKey string
	Index  string
}

type Client struct {
	host       string
	apiKey     string
	index      string
	httpClient *http.Client
}

// MediaDocument is the search projection of one work.
//
// It is deliberately separate from the database model: the shape here is what a
// search engine needs, not what persistence needs. Display fields and search
// fields coexist so the client can render a card from one hit.
type MediaDocument struct {
	ID      int64  `json:"id"`
	TitleAR string `json:"title_ar,omitempty"`
	TitleEN string `json:"title_en"`
	// TitleNormalized is a search-only field. It folds Arabic letter variants,
	// diacritics and Arabic-Indic numerals, so "اسامة" matches "أسامة" and
	// "الحلقة ١" matches "الحلقة 1". The original spelling is never replaced:
	// TitleAR and TitleEN keep it for display.
	TitleNormalized string `json:"title_normalized,omitempty"`
	// AlternateTitles carries every known alias for the work, so a library that
	// learned "ون بيس" as an alias for "One Piece" is findable by either name.
	AlternateTitles    []string `json:"alternate_titles,omitempty"`
	Type               string   `json:"type"`
	PlotAR             string   `json:"plot_ar,omitempty"`
	PlotEN             string   `json:"plot_en,omitempty"`
	ReleaseYear        int      `json:"release_year,omitempty"`
	Rating             float64  `json:"rating,omitempty"`
	PosterPath         string   `json:"poster_path,omitempty"`
	BannerPath         string   `json:"banner_path,omitempty"`
	Genres             []string `json:"genres,omitempty"`
	GenreIDs           []int    `json:"genre_ids,omitempty"`
	ContentRating      string   `json:"content_rating,omitempty"`
	CategorySlug       string   `json:"category_slug,omitempty"`
	CategoryAR         string   `json:"category_ar,omitempty"`
	CategoryEN         string   `json:"category_en,omitempty"`
	FileCount          int      `json:"file_count"`
	Status             string   `json:"status,omitempty"`
	SeasonCount        int      `json:"season_count,omitempty"`
	TMDBSeasonCount    int      `json:"tmdb_season_count,omitempty"`
	TMDBEpisodeCount   int      `json:"tmdb_episode_count,omitempty"`
	TotalSize          int64    `json:"total_size,omitempty"`
	BestResolution     string   `json:"best_resolution,omitempty"`
	RuntimeMinutes     int      `json:"runtime_minutes,omitempty"`
	HasArabicAudio     bool     `json:"has_arabic_audio,omitempty"`
	HasArabicSubtitles bool     `json:"has_arabic_subtitles,omitempty"`
}

type SyncResult struct {
	Indexed int    `json:"indexed"`
	TaskUID string `json:"taskUid,omitempty"`
}

type SearchResult struct {
	Query              string          `json:"query"`
	EstimatedTotalHits int             `json:"estimatedTotalHits"`
	Hits               []MediaDocument `json:"hits"`
	ProcessingTimeMS   int             `json:"processingTimeMs"`
	Limit              int             `json:"limit"`
	Filter             string          `json:"filter,omitempty"`
}

func NewClient(config Config) *Client {
	host := strings.TrimRight(config.Host, "/")
	if host == "" {
		host = "http://127.0.0.1:7700"
	}
	index := config.Index
	if index == "" {
		index = "media_items"
	}
	return &Client{
		host:   host,
		apiKey: config.APIKey,
		index:  index,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) EnsureIndex(ctx context.Context) error {
	body := map[string]string{"uid": c.index, "primaryKey": "id"}
	response, err := c.doJSON(ctx, http.MethodPost, "/indexes", body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusCreated || response.StatusCode == http.StatusAccepted {
		return c.configureSettings(ctx)
	}
	if response.StatusCode == http.StatusBadRequest {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		if strings.Contains(string(payload), "index_already_exists") {
			return c.configureSettings(ctx)
		}
		return fmt.Errorf("create meilisearch index: %s", strings.TrimSpace(string(payload)))
	}
	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("create meilisearch index: status %d: %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return c.configureSettings(ctx)
}

func (c *Client) IndexDocuments(ctx context.Context, documents []MediaDocument) (SyncResult, error) {
	if len(documents) == 0 {
		return SyncResult{}, nil
	}
	if err := c.EnsureIndex(ctx); err != nil {
		return SyncResult{}, err
	}

	path := "/indexes/" + url.PathEscape(c.index) + "/documents"
	response, err := c.doJSON(ctx, http.MethodPost, path, documents)
	if err != nil {
		return SyncResult{}, err
	}
	defer response.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode >= 300 {
		return SyncResult{}, fmt.Errorf("index meilisearch documents: status %d: %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}

	var task struct {
		TaskUID int    `json:"taskUid"`
		UID     int    `json:"uid"`
		Status  string `json:"status"`
	}
	_ = json.Unmarshal(payload, &task)

	taskUID := ""
	if task.TaskUID != 0 {
		taskUID = fmt.Sprintf("%d", task.TaskUID)
	} else if task.UID != 0 {
		taskUID = fmt.Sprintf("%d", task.UID)
	}

	return SyncResult{Indexed: len(documents), TaskUID: taskUID}, nil
}

// DocumentIDs returns every primary key currently held by the index.
//
// It is what makes orphan detection possible: the projector compares this set
// against the set of works that still exist in PostgreSQL, and everything left
// over is a leftover from a deleted or merged row.
//
// Meilisearch paginates this endpoint, so it is read page by page rather than
// assuming one response carries the whole index. A library with a million works
// would otherwise silently truncate at the engine's default page size.
func (c *Client) DocumentIDs(ctx context.Context) ([]int64, error) {
	if err := c.EnsureIndex(ctx); err != nil {
		return nil, err
	}

	const pageSize = 1000
	const maxPages = 100000 // safety bound: 100M documents
	ids := make([]int64, 0, pageSize)
	for page := 0; page < maxPages; page++ {
		path := fmt.Sprintf("/indexes/%s/documents?fields=id&limit=%d&offset=%d",
			url.PathEscape(c.index), pageSize, page*pageSize)

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
			return nil, fmt.Errorf("list meilisearch documents: status %d: %s",
				response.StatusCode, strings.TrimSpace(string(payload)))
		}

		var batch struct {
			Results []struct {
				ID int64 `json:"id"`
			} `json:"results"`
		}
		if err := json.Unmarshal(payload, &batch); err != nil {
			return nil, fmt.Errorf("decode meilisearch documents: %w", err)
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

// DeleteDocuments removes documents by primary key.
//
// Meilisearch deletes by primary key, so removing a merged duplicate or a
// deleted work is a targeted operation rather than a full index rebuild.
func (c *Client) DeleteDocuments(ctx context.Context, ids []int64) (SyncResult, error) {
	if len(ids) == 0 {
		return SyncResult{}, nil
	}
	if err := c.EnsureIndex(ctx); err != nil {
		return SyncResult{}, err
	}

	path := "/indexes/" + url.PathEscape(c.index) + "/documents/delete-batch"
	response, err := c.doJSON(ctx, http.MethodPost, path, ids)
	if err != nil {
		return SyncResult{}, err
	}
	defer response.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode >= 300 {
		return SyncResult{}, fmt.Errorf("delete meilisearch documents: status %d: %s",
			response.StatusCode, strings.TrimSpace(string(payload)))
	}

	var task struct {
		TaskUID int `json:"taskUid"`
		UID     int `json:"uid"`
	}
	_ = json.Unmarshal(payload, &task)
	taskUID := ""
	if task.TaskUID != 0 {
		taskUID = fmt.Sprintf("%d", task.TaskUID)
	} else if task.UID != 0 {
		taskUID = fmt.Sprintf("%d", task.UID)
	}
	return SyncResult{Indexed: len(ids), TaskUID: taskUID}, nil
}

func (c *Client) SearchDocuments(ctx context.Context, query string, limit int, filter string) (SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}

	payload := map[string]any{
		"q":                    query,
		"limit":                limit,
		"attributesToRetrieve": []string{"id", "title_ar", "title_en", "type", "plot_ar", "plot_en", "release_year", "rating", "poster_path", "banner_path", "genres", "category_slug", "category_ar", "category_en", "file_count", "status", "season_count", "tmdb_season_count", "tmdb_episode_count", "total_size", "best_resolution", "runtime_minutes", "has_arabic_audio", "has_arabic_subtitles"},
	}
	if strings.TrimSpace(filter) != "" {
		payload["filter"] = filter
	}

	response, err := c.doJSON(ctx, http.MethodPost, "/indexes/"+url.PathEscape(c.index)+"/search", payload)
	if err != nil {
		return SearchResult{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return SearchResult{}, err
	}
	if response.StatusCode >= 300 {
		return SearchResult{}, fmt.Errorf("search meilisearch documents: status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw struct {
		Query              string          `json:"query"`
		EstimatedTotalHits int             `json:"estimatedTotalHits"`
		Hits               []MediaDocument `json:"hits"`
		ProcessingTimeMS   int             `json:"processingTimeMs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return SearchResult{}, err
	}

	return SearchResult{
		Query:              raw.Query,
		EstimatedTotalHits: raw.EstimatedTotalHits,
		Hits:               raw.Hits,
		ProcessingTimeMS:   raw.ProcessingTimeMS,
		Limit:              limit,
		Filter:             filter,
	}, nil
}

func (c *Client) configureSettings(ctx context.Context) error {
	// Attribute order is significance order, not a set: Meilisearch weights an
	// earlier attribute above a later one. The previous order put a plot before
	// the title's own Arabic name and gave the synopsis the same weight as the
	// name, so a plot mentioning "inception" could outrank a work actually
	// titled "Inception". The title now comes first in both scripts.
	settings := map[string][]string{
		"searchableAttributes": {
			"title_en",
			"title_ar",
			// The normalized title is what makes "اسامة" match "أسامة" and
			// Arabic-Indic numerals match ASCII ones. It sits directly beside the
			// real titles so a folded match ranks as a title match.
			"title_normalized",
			"alternate_titles",
			"genres",
			"category_en",
			"category_ar",
			"plot_en",
			"plot_ar",
		},
		"filterableAttributes": {
			"type",
			"category_slug",
			"release_year",
			"genres",
			"status",
		},
		"sortableAttributes": {
			"rating",
			"release_year",
			"file_count",
		},
		// Ranking rules are set explicitly rather than left implicit. They are
		// Meilisearch's defaults today, and writing them down means a future
		// engine upgrade cannot silently change how results are ordered.
		// `attribute` is the rule that applies the searchable-attribute order
		// above, which is why the order matters.
		"rankingRules": {
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
	}

	path := "/indexes/" + url.PathEscape(c.index) + "/settings"
	response, err := c.doJSON(ctx, http.MethodPatch, path, settings)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("configure meilisearch settings: status %d: %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.host+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(request)
}
