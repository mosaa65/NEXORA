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

type MALConfig struct {
	ClientID    string
	AccessToken string
	BaseURL     string
	ImageDir    string
}

type MALClient struct {
	config MALConfig
	client *http.Client
}

type malSearchResponse struct {
	Data []struct {
		Node malAnime `json:"node"`
	} `json:"data"`
}

type malAnime struct {
	ID                int     `json:"id"`
	Title             string  `json:"title"`
	Mean              float64 `json:"mean"`
	StartDate         string  `json:"start_date"`
	Synopsis          string  `json:"synopsis"`
	MainPicture       malImage `json:"main_picture"`
	AlternativeTitles struct {
		English string `json:"en"`
		Japanese string `json:"ja"`
	} `json:"alternative_titles"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

type malImage struct {
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

func NewMALClient(config MALConfig) *MALClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.myanimelist.net/v2"
	}
	return &MALClient{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *MALClient) Configured() bool {
	return c.config.ClientID != "" || c.config.AccessToken != ""
}

func (c *MALClient) Lookup(ctx context.Context, query Query) (Result, error) {
	if !c.Configured() {
		return Result{}, ErrNotConfigured
	}

	values := url.Values{}
	values.Set("q", query.Title)
	values.Set("limit", "1")
	values.Set("fields", "id,title,main_picture,alternative_titles,start_date,synopsis,mean,genres")

	endpoint := strings.TrimRight(c.config.BaseURL, "/") + "/anime?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	if c.config.AccessToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	} else {
		request.Header.Set("X-MAL-CLIENT-ID", c.config.ClientID)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return Result{}, fmt.Errorf("mal lookup failed: status %d", response.StatusCode)
	}

	var payload malSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Result{}, err
	}
	if len(payload.Data) == 0 {
		return Result{}, ErrNotFound
	}

	anime := payload.Data[0].Node
	result := Result{
		Provider:      "mal",
		ExternalID:    strconv.Itoa(anime.ID),
		Title:         firstNonEmpty(anime.AlternativeTitles.English, anime.Title),
		OriginalTitle: firstNonEmpty(anime.Title, anime.AlternativeTitles.Japanese),
		Overview:      anime.Synopsis,
		ReleaseYear:   yearFromDate(anime.StartDate),
		Rating:        anime.Mean,
		PosterPath:    firstNonEmpty(anime.MainPicture.Large, anime.MainPicture.Medium),
	}
	for _, genre := range anime.Genres {
		if genre.Name != "" {
			result.Genres = append(result.Genres, genre.Name)
		}
	}

	if cached, err := cacheRemoteImage(ctx, c.client, result.PosterPath, c.config.ImageDir, "mal", "poster_"+result.ExternalID); err == nil {
		result.CachedPosterPath = cached
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}

	return result, nil
}
