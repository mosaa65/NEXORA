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

type tmdbDetails struct {
	tmdbResult
	OriginalLanguage      string `json:"original_language"`
	OriginCountry        []string `json:"origin_country"`
	ProductionCountries []struct {
		ISO31661 string `json:"iso_3166_1"`
		Name     string `json:"name"`
	} `json:"production_countries"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
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
	return c.fetchDetails(ctx, mediaKind, item.ID, query.Language)
}

// fetchDetails hydrates the selected search candidate in one request. The raw
// response is retained in PostgreSQL for the local, offline-facing API.
func (c *TMDBClient) fetchDetails(ctx context.Context, mediaKind string, id int, language string) (Result, error) {
	if strings.TrimSpace(language) == "" {
		language = "en-US"
	}
	values := url.Values{}
	values.Set("language", language)
	values.Set("include_image_language", "ar,en,null")
	if mediaKind == "tv" {
		values.Set("append_to_response", "aggregate_credits,images,videos,keywords,external_ids,recommendations,similar,content_ratings,translations")
	} else {
		// These are all public, title-specific movie resources. Keeping them in
		// the details response makes the local snapshot useful while the LAN is
		// offline, without calling TMDB every time the details page is opened.
		values.Set("append_to_response", "alternative_titles,credits,external_ids,images,keywords,lists,recommendations,release_dates,reviews,similar,translations,videos")
	}
	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}
	endpoint := fmt.Sprintf("%s/3/%s/%d?%s", strings.TrimRight(c.config.BaseURL, "/"), mediaKind, id, values.Encode())
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
		return Result{}, fmt.Errorf("tmdb details failed: status %d", response.StatusCode)
	}
	var rawPayload json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&rawPayload); err != nil {
		return Result{}, err
	}
	var details tmdbDetails
	if err := json.Unmarshal(rawPayload, &details); err != nil {
		return Result{}, fmt.Errorf("decode tmdb details: %w", err)
	}
	var supplementalWarning string
	if mediaKind == "movie" {
		// Watch-provider availability is a separate regional resource. Store it
		// under a stable JSON key; the UI must show JustWatch attribution before
		// it presents this data to users.
		if providers, err := c.fetchResource(ctx, fmt.Sprintf("/3/movie/%d/watch/providers", id)); err == nil {
			var document map[string]json.RawMessage
			if json.Unmarshal(rawPayload, &document) == nil {
				document["watch_providers"] = providers
				if expanded, marshalErr := json.Marshal(document); marshalErr == nil {
					rawPayload = expanded
				}
			}
		} else {
			// Provider availability is supplementary; a failure must not discard
			// the core metadata snapshot.
			supplementalWarning = "watch providers: " + err.Error()
		}
	}
	// Profile and related-title artwork are needed by the local details page.
	// Cache a bounded selection only for the canonical English snapshot so a
	// refresh does not multiply image downloads for ar-SA and en-US.
	if strings.HasPrefix(strings.ToLower(language), "en") {
		var imageWarnings []string
		rawPayload, imageWarnings = c.cacheDetailImages(ctx, rawPayload)
		if len(imageWarnings) > 0 {
			supplementalWarning = strings.TrimSpace(supplementalWarning + "; " + strings.Join(imageWarnings, "; "))
		}
	}
	// Re-decode in case a supplementary resource expanded the local snapshot.
	if err := json.Unmarshal(rawPayload, &details); err != nil {
		return Result{}, fmt.Errorf("decode expanded tmdb details: %w", err)
	}
	item := details.tmdbResult
	result := Result{
		Provider:      "tmdb",
		ExternalID:    strconv.Itoa(item.ID),
		Locale:        language,
		Title:         firstNonEmpty(item.Title, item.Name),
		OriginalTitle: firstNonEmpty(item.OriginalTitle, item.OriginalName),
		Overview:      item.Overview,
		ReleaseYear:   yearFromDate(firstNonEmpty(item.ReleaseDate, item.FirstAirDate)),
		Rating:        item.VoteAverage,
		PosterPath:    c.imageURL("w500", item.PosterPath),
		BannerPath:    c.imageURL("original", item.BackdropPath),
		RawPayload:    rawPayload,
		Genres:        mergeTags(normalizeGenres(details.Genres), normalizeOriginTags(details)),
	}
	if supplementalWarning != "" {
		result.Warnings = append(result.Warnings, supplementalWarning)
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

// normalizeOriginTags turns TMDB's country fields into the short Arabic tags
// used consistently by the library filters. We deliberately keep these with
// genres in the existing schema: they are search facets, not a second source
// of truth, and this avoids breaking already-indexed libraries with a migration.
func normalizeOriginTags(details tmdbDetails) []string {
	codes := make([]string, 0, len(details.OriginCountry)+len(details.ProductionCountries))
	codes = append(codes, details.OriginCountry...)
	for _, country := range details.ProductionCountries {
		codes = append(codes, country.ISO31661)
	}

	tags := make([]string, 0, 2)
	for _, code := range codes {
		switch strings.ToUpper(strings.TrimSpace(code)) {
		case "TR":
			tags = append(tags, "تركي")
		case "KR", "KP":
			tags = append(tags, "كوري")
		case "SA", "AE", "EG", "SY", "LB", "JO", "IQ", "KW", "QA", "BH", "OM", "MA", "DZ", "TN":
			tags = append(tags, "عربي")
		case "US", "GB", "CA", "AU", "NZ":
			tags = append(tags, "أجنبي")
		case "IN":
			tags = append(tags, "هندي")
		case "ES":
			tags = append(tags, "إسباني")
		case "JP":
			tags = append(tags, "ياباني")
		}
	}

	// TV details always provide origin_country, while films may only expose an
	// original language. Use it as a conservative fallback when countries are
	// absent, so newly enriched items remain browsable.
	if len(tags) == 0 {
		switch strings.ToLower(strings.TrimSpace(details.OriginalLanguage)) {
		case "tr":
			tags = append(tags, "تركي")
		case "ko":
			tags = append(tags, "كوري")
		case "ar":
			tags = append(tags, "عربي")
		case "hi", "ta", "te", "ml", "bn":
			tags = append(tags, "هندي")
		case "es":
			tags = append(tags, "إسباني")
		case "en":
			tags = append(tags, "أجنبي")
		case "ja":
			tags = append(tags, "ياباني")
		}
	}
	return mergeTags(tags)
}

func mergeTags(groups ...[]string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, group := range groups {
		for _, tag := range group {
			tag = strings.TrimSpace(tag)
			if tag != "" && !seen[tag] {
				seen[tag] = true
				result = append(result, tag)
			}
		}
	}
	return result
}

// cacheDetailImages adds local_*_path fields to the raw local snapshot. The
// original TMDB paths are kept too, so a later richer asset sync can reuse them.
func (c *TMDBClient) cacheDetailImages(ctx context.Context, raw json.RawMessage) (json.RawMessage, []string) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return raw, []string{"decode embedded images: " + err.Error()}
	}

	warnings := make([]string, 0)
	cacheEntries := func(containerKey, listKey, sourceField, localField, size, keyPrefix string, limit int) {
		container, _ := document[containerKey].(map[string]any)
		entries, _ := container[listKey].([]any)
		for index, item := range entries {
			if index >= limit {
				break
			}
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			path, _ := entry[sourceField].(string)
			if path == "" {
				continue
			}
			id := fmt.Sprint(entry["id"])
			if id == "<nil>" || id == "" {
				id = strings.Trim(path, "/")
			}
			cached, err := cacheRemoteImage(ctx, c.client, c.imageURL(size, path), c.config.ImageDir, "tmdb", keyPrefix+"_"+id)
			if err != nil {
				warnings = append(warnings, keyPrefix+": "+err.Error())
				continue
			}
			if cached != "" {
				entry[localField] = cached
			}
		}
	}

	// Film and TV credits use different envelope keys but the member shape is
	// the same for profile paths.
	cacheEntries("credits", "cast", "profile_path", "local_profile_path", "w185", "profile", 20)
	cacheEntries("aggregate_credits", "cast", "profile_path", "local_profile_path", "w185", "profile", 20)
	cacheEntries("recommendations", "results", "poster_path", "local_poster_path", "w342", "related_poster", 12)
	cacheEntries("similar", "results", "poster_path", "local_poster_path", "w342", "related_poster", 12)
	cacheEntries("images", "posters", "file_path", "local_poster_path", "w500", "gallery_poster", 12)
	cacheEntries("images", "backdrops", "file_path", "local_backdrop_path", "w780", "gallery_backdrop", 12)
	cacheEntries("images", "logos", "file_path", "local_logo_path", "w500", "gallery_logo", 8)

	expanded, err := json.Marshal(document)
	if err != nil {
		return raw, append(warnings, "encode embedded images: "+err.Error())
	}
	return expanded, warnings
}

func (c *TMDBClient) fetchResource(ctx context.Context, path string) (json.RawMessage, error) {
	values := url.Values{}
	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}
	endpoint := strings.TrimRight(c.config.BaseURL, "/") + path
	if encoded := values.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.config.BearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb resource failed: status %d", response.StatusCode)
	}
	var payload json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// fetchSeasonDetails retains the complete season document, whose episodes
// array contains episode names, overviews, air dates, runtimes and stills.
func (c *TMDBClient) fetchSeasonDetails(ctx context.Context, seriesID, seasonNumber int, language string) (SeasonResult, error) {
	if strings.TrimSpace(language) == "" {
		language = "en-US"
	}
	values := url.Values{}
	values.Set("language", language)
	values.Set("include_image_language", "ar,en,null")
	values.Set("append_to_response", "aggregate_credits,credits,external_ids,images,translations,videos")
	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}
	endpoint := fmt.Sprintf("%s/3/tv/%d/season/%d?%s", strings.TrimRight(c.config.BaseURL, "/"), seriesID, seasonNumber, values.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return SeasonResult{}, err
	}
	if c.config.BearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return SeasonResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return SeasonResult{}, fmt.Errorf("tmdb season details failed: status %d", response.StatusCode)
	}
	var payload json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return SeasonResult{}, err
	}
	return SeasonResult{Provider: "tmdb", ExternalID: strconv.Itoa(seriesID), Locale: language, SeasonNumber: seasonNumber, RawPayload: payload}, nil
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

func normalizeGenres(genres []struct {
	Name string `json:"name"`
}) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(g string) {
		g = strings.TrimSpace(g)
		if g != "" && !seen[g] {
			seen[g] = true
			result = append(result, g)
		}
	}

	for _, item := range genres {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		add(name)
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(lower, "action") || strings.Contains(lower, "adventure") || strings.Contains(lower, "حركة") || strings.Contains(lower, "مغامرة"):
			add("أكشن")
			add("مغامرة")
		case strings.Contains(lower, "anim") || strings.Contains(lower, "رسوم"):
			add("أنمي")
			add("كرتون")
		case strings.Contains(lower, "drama") || strings.Contains(lower, "دراما"):
			add("دراما")
		case strings.Contains(lower, "comedy") || strings.Contains(lower, "كوميد"):
			add("كوميديا")
		case strings.Contains(lower, "sci-fi") || strings.Contains(lower, "science") || strings.Contains(lower, "خيال علمي"):
			add("خيال علمي")
		case strings.Contains(lower, "fantasy") || strings.Contains(lower, "فانتاز"):
			add("فانتازيا")
		case strings.Contains(lower, "mystery") || strings.Contains(lower, "غموض"):
			add("غموض")
		case strings.Contains(lower, "crime") || strings.Contains(lower, "جريم"):
			add("جريمة")
		case strings.Contains(lower, "horror") || strings.Contains(lower, "رعب"):
			add("رعب")
		case strings.Contains(lower, "romance") || strings.Contains(lower, "رومان"):
			add("رومانسي")
		case strings.Contains(lower, "document") || strings.Contains(lower, "وثائق"):
			add("وثائقي")
		case strings.Contains(lower, "family") || strings.Contains(lower, "kids") || strings.Contains(lower, "عائل") || strings.Contains(lower, "أطفال"):
			add("عائلي")
		case strings.Contains(lower, "war") || strings.Contains(lower, "حرب"):
			add("حرب")
		case strings.Contains(lower, "history") || strings.Contains(lower, "تاريخ"):
			add("تاريخي")
		}
	}
	return result
}
