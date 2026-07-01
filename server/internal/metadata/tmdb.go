package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TMDBConfig struct {
	APIKey       string
	BearerToken  string
	BaseURL      string
	ImageBaseURL string
	ImageDir     string
}

type TMDBClient struct {
	config TMDBConfig
	client *http.Client
}

type tmdbSearchResponse struct {
	Results []tmdbResult `json:"results"`
}

type tmdbResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
}

func NewTMDBClient(config TMDBConfig) *TMDBClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.themoviedb.org"
	}
	if config.ImageBaseURL == "" {
		config.ImageBaseURL = "https://image.tmdb.org/t/p"
	}
	return &TMDBClient{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *TMDBClient) Configured() bool {
	return c.config.APIKey != "" || c.config.BearerToken != ""
}

func (c *TMDBClient) Lookup(ctx context.Context, query Query) (Result, error) {
	if !c.Configured() {
		return Result{}, ErrNotConfigured
	}

	mediaKind := "movie"
	if query.Type == "series" || query.Type == "anime" || query.Type == "tv" {
		mediaKind = "tv"
	}

	values := url.Values{}
	values.Set("query", query.Title)
	values.Set("include_adult", "false")
	values.Set("page", "1")
	if query.Language != "" {
		values.Set("language", query.Language)
	}
	if query.Year > 0 {
		if mediaKind == "movie" {
			values.Set("year", strconv.Itoa(query.Year))
		} else {
			values.Set("first_air_date_year", strconv.Itoa(query.Year))
		}
	}
	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}

	endpoint := fmt.Sprintf("%s/3/search/%s?%s", strings.TrimRight(c.config.BaseURL, "/"), mediaKind, values.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	if c.config.BearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return Result{}, fmt.Errorf("tmdb lookup failed: status %d", response.StatusCode)
	}

	var payload tmdbSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Result{}, err
	}
	if len(payload.Results) == 0 {
		return Result{}, ErrNotFound
	}

	item := payload.Results[0]
	result := Result{
		Provider:      "tmdb",
		ExternalID:    strconv.Itoa(item.ID),
		Title:         firstNonEmpty(item.Title, item.Name),
		OriginalTitle: firstNonEmpty(item.OriginalTitle, item.OriginalName),
		Overview:      item.Overview,
		ReleaseYear:   yearFromDate(firstNonEmpty(item.ReleaseDate, item.FirstAirDate)),
		Rating:        item.VoteAverage,
		PosterPath:    c.imageURL("w500", item.PosterPath),
		BannerPath:    c.imageURL("original", item.BackdropPath),
	}

	if cached, err := cacheRemoteImage(ctx, c.client, result.PosterPath, c.config.ImageDir, "tmdb", "poster_"+result.ExternalID); err == nil {
		result.CachedPosterPath = cached
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}
	if cached, err := cacheRemoteImage(ctx, c.client, result.BannerPath, c.config.ImageDir, "tmdb", "banner_"+result.ExternalID); err == nil {
		result.CachedBannerPath = cached
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}

	return result, nil
}

func (c *TMDBClient) imageURL(size, path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimRight(c.config.ImageBaseURL, "/") + "/" + strings.Trim(size, "/") + "/" + strings.TrimLeft(path, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
