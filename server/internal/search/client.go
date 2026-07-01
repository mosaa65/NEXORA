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

type MediaDocument struct {
	ID           int64    `json:"id"`
	TitleAR      string   `json:"title_ar,omitempty"`
	TitleEN      string   `json:"title_en"`
	Type         string   `json:"type"`
	PlotAR       string   `json:"plot_ar,omitempty"`
	PlotEN       string   `json:"plot_en,omitempty"`
	ReleaseYear  int      `json:"release_year,omitempty"`
	Rating       float64  `json:"rating,omitempty"`
	PosterPath   string   `json:"poster_path,omitempty"`
	BannerPath   string   `json:"banner_path,omitempty"`
	Genres       []string `json:"genres,omitempty"`
	CategorySlug string   `json:"category_slug,omitempty"`
	CategoryAR   string   `json:"category_ar,omitempty"`
	CategoryEN   string   `json:"category_en,omitempty"`
	FileCount    int      `json:"file_count"`
}

type SyncResult struct {
	Indexed int    `json:"indexed"`
	TaskUID string `json:"taskUid,omitempty"`
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

func (c *Client) configureSettings(ctx context.Context) error {
	settings := map[string][]string{
		"searchableAttributes": {
			"title_ar",
			"title_en",
			"genres",
			"plot_ar",
			"plot_en",
			"category_ar",
			"category_en",
		},
		"filterableAttributes": {
			"type",
			"category_slug",
			"release_year",
			"genres",
		},
		"sortableAttributes": {
			"rating",
			"release_year",
			"file_count",
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
