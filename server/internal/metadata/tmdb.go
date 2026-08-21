package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	config       TMDBConfig
	client       *http.Client
	settingsLock sync.RWMutex
	settings     TMDBSettings
	remoteConfig *TMDBRemoteConfig
}

type tmdbSearchResponse struct {
	Page         int          `json:"page"`
	TotalResults int          `json:"total_results"`
	TotalPages   int          `json:"total_pages"`
	Results      []tmdbResult `json:"results"`
}

type tmdbResult struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	Name             string   `json:"name"`
	OriginalTitle    string   `json:"original_title"`
	OriginalName     string   `json:"original_name"`
	Overview         string   `json:"overview"`
	ReleaseDate      string   `json:"release_date"`
	FirstAirDate     string   `json:"first_air_date"`
	PosterPath       string   `json:"poster_path"`
	BackdropPath     string   `json:"backdrop_path"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Popularity       float64  `json:"popularity"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginCountry    []string `json:"origin_country"`
	OriginalLanguage string   `json:"original_language"`
	MediaType        string   `json:"media_type"`
}

type tmdbDetails struct {
	tmdbResult
	OriginalLanguage    string   `json:"original_language"`
	OriginCountry       []string `json:"origin_country"`
	ProductionCountries []struct {
		ISO31661 string `json:"iso_3166_1"`
		Name     string `json:"name"`
	} `json:"production_countries"`
	Genres []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
}

// FetchExecutionStats tracks network cost per lookup
type FetchExecutionStats struct {
	BytesDownloaded  int64
	ImagesDownloaded int
	APIRequests      int
	Duration         time.Duration
}

func NewTMDBClient(config TMDBConfig) *TMDBClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.themoviedb.org"
	}
	if config.ImageBaseURL == "" {
		config.ImageBaseURL = "https://image.tmdb.org/t/p"
	}
	return &TMDBClient{
		config:   config,
		client:   &http.Client{Timeout: 30 * time.Second},
		settings: DefaultSettings(),
	}
}

func (c *TMDBClient) SetSettings(s TMDBSettings) {
	c.settingsLock.Lock()
	defer c.settingsLock.Unlock()
	c.settings = s
}

func (c *TMDBClient) GetSettings() TMDBSettings {
	c.settingsLock.RLock()
	defer c.settingsLock.RUnlock()
	return c.settings
}

func (c *TMDBClient) Configured() bool {
	return c.config.APIKey != "" || c.config.BearerToken != ""
}

// FetchRemoteConfiguration requests TMDB /3/configuration to inspect available image sizes
func (c *TMDBClient) FetchRemoteConfiguration(ctx context.Context) (*TMDBRemoteConfig, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}

	payload, _, err := c.fetchResource(ctx, "/3/configuration")
	if err != nil {
		return nil, fmt.Errorf("fetch tmdb configuration: %w", err)
	}

	var remoteCfg TMDBRemoteConfig
	if err := json.Unmarshal(payload, &remoteCfg); err != nil {
		return nil, fmt.Errorf("decode tmdb configuration: %w", err)
	}

	c.settingsLock.Lock()
	c.remoteConfig = &remoteCfg
	c.settingsLock.Unlock()

	return &remoteCfg, nil
}

// SmartLookup tries the best match, with intelligent fallback across movie/tv/anime
func (c *TMDBClient) Lookup(ctx context.Context, query Query) (Result, error) {
	if !c.Configured() {
		return Result{}, ErrNotConfigured
	}

	settings := c.GetSettings()
	query.Title = strings.TrimSpace(query.Title)
	if query.Title == "" {
		return Result{}, errors.New("title is required")
	}

	// Determine primary media kinds to test
	primaryKind := "movie"
	isSeriesCandidate := query.Type == "series" || query.Type == "anime" || query.Type == "tv"
	if isSeriesCandidate {
		primaryKind = "tv"
	}

	// 1. Primary Search
	candidates, err := c.searchCandidates(ctx, primaryKind, query)
	if err == nil && len(candidates) > 0 {
		best, confidence := c.evaluateBestMatch(candidates, query)
		if confidence >= 0.35 {
			return c.fetchDetailsWithSettings(ctx, primaryKind, best.ID, query.Language, settings)
		}
	}

	// 2. Cross-Category Fallback: If anime or series wasn't found in TV, check Movies (e.g. One Piece Movies or Anime Films)
	if query.Type == "anime" || query.Type == "series" {
		if movieCandidates, err := c.searchCandidates(ctx, "movie", query); err == nil && len(movieCandidates) > 0 {
			best, conf := c.evaluateBestMatch(movieCandidates, query)
			if conf >= 0.40 {
				res, fetchErr := c.fetchDetailsWithSettings(ctx, "movie", best.ID, query.Language, settings)
				if fetchErr == nil {
					res.Warnings = append(res.Warnings, "تمت المطابقة كفيلم/عمل مستقل بناءً على مطابقة أدق في TMDB")
					return res, nil
				}
			}
		}
	} else if query.Type == "movie" {
		// If movie search failed, check TV shows (sometimes miniseries are cataloged as TV)
		if tvCandidates, err := c.searchCandidates(ctx, "tv", query); err == nil && len(tvCandidates) > 0 {
			best, conf := c.evaluateBestMatch(tvCandidates, query)
			if conf >= 0.50 {
				res, fetchErr := c.fetchDetailsWithSettings(ctx, "tv", best.ID, query.Language, settings)
				if fetchErr == nil {
					res.Warnings = append(res.Warnings, "تمت المطابقة كمسلسل/برنامج تلفزيوني بناءً على مطابقة أدق في TMDB")
					return res, nil
				}
			}
		}
	}

	if len(candidates) > 0 {
		// Fallback to top candidate if confidence was marginal
		best := candidates[0]
		return c.fetchDetailsWithSettings(ctx, primaryKind, best.ID, query.Language, settings)
	}

	return Result{}, ErrNotFound
}

func (c *TMDBClient) searchCandidates(ctx context.Context, mediaKind string, query Query) ([]tmdbResult, error) {
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
	responseBytes, statusCode, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	if statusCode >= 300 {
		return nil, fmt.Errorf("tmdb search failed with status %d", statusCode)
	}

	var payload tmdbSearchResponse
	if err := json.Unmarshal(responseBytes, &payload); err != nil {
		return nil, err
	}
	return payload.Results, nil
}

// evaluateBestMatch scores search candidates based on title distance, year proximity, and popularity
func (c *TMDBClient) evaluateBestMatch(candidates []tmdbResult, query Query) (tmdbResult, float64) {
	if len(candidates) == 0 {
		return tmdbResult{}, 0
	}

	normQuery := normalizeTitleForMatch(query.Title)
	bestScore := -1.0
	bestIdx := 0

	for idx, cand := range candidates {
		titleEN := normalizeTitleForMatch(cand.Title)
		if titleEN == "" {
			titleEN = normalizeTitleForMatch(cand.Name)
		}
		origTitle := normalizeTitleForMatch(cand.OriginalTitle)
		if origTitle == "" {
			origTitle = normalizeTitleForMatch(cand.OriginalName)
		}

		score := 0.0

		// Exact match bonus
		if normQuery == titleEN || normQuery == origTitle {
			score += 0.60
		} else if strings.Contains(titleEN, normQuery) || strings.Contains(normQuery, titleEN) {
			score += 0.40
		} else if origTitle != "" && (strings.Contains(origTitle, normQuery) || strings.Contains(normQuery, origTitle)) {
			score += 0.35
		}

		// Year proximity
		candYear := yearFromDate(cand.ReleaseDate)
		if candYear == 0 {
			candYear = yearFromDate(cand.FirstAirDate)
		}
		if query.Year > 0 && candYear > 0 {
			diff := math.Abs(float64(query.Year - candYear))
			if diff == 0 {
				score += 0.30
			} else if diff <= 1 {
				score += 0.20
			} else if diff <= 2 {
				score += 0.10
			}
		} else if query.Year == 0 {
			score += 0.10
		}

		// Popularity weight (up to 0.10)
		popScore := math.Min(cand.Popularity/200.0, 0.10)
		score += popScore

		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}

	return candidates[bestIdx], bestScore
}

func normalizeTitleForMatch(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	t = strings.ReplaceAll(t, ".", " ")
	t = strings.ReplaceAll(t, "-", " ")
	t = strings.ReplaceAll(t, "_", " ")
	t = strings.ReplaceAll(t, ":", " ")
	fields := strings.Fields(t)
	return strings.Join(fields, " ")
}

// buildAppendList constructs append_to_response dynamically matching the active ModuleConfig
func (c *TMDBClient) buildAppendList(mediaKind string, modules ModuleConfig) string {
	var parts []string

	if modules.FetchAlternativeTitles {
		parts = append(parts, "alternative_titles")
	}

	if mediaKind == "tv" {
		if modules.FetchCreditsText || modules.MaxCastImages > 0 {
			parts = append(parts, "aggregate_credits")
		}
		if modules.FetchContentRatings {
			parts = append(parts, "content_ratings")
		}
	} else {
		if modules.FetchCreditsText || modules.MaxCastImages > 0 {
			parts = append(parts, "credits")
		}
		if modules.FetchReleaseDates {
			parts = append(parts, "release_dates")
		}
	}

	if modules.FetchExternalIDs {
		parts = append(parts, "external_ids")
	}
	if modules.FetchKeywords {
		parts = append(parts, "keywords")
	}
	if modules.FetchTrailers {
		parts = append(parts, "videos")
	}
	if modules.FetchTranslations {
		parts = append(parts, "translations")
	}
	if modules.FetchRecommendations {
		parts = append(parts, "recommendations")
	}
	if modules.FetchSimilar {
		parts = append(parts, "similar")
	}
	if modules.FetchReviews {
		parts = append(parts, "reviews")
	}

	// Images append if any image downloading/display is requested
	if modules.FetchPoster || modules.FetchBackdrop || modules.MaxGalleryPosters > 0 || modules.MaxGalleryBackdrops > 0 || modules.MaxGalleryLogos > 0 {
		parts = append(parts, "images")
	}

	return strings.Join(parts, ",")
}

// fetchDetailsWithSettings hydrates metadata respecting user switches & image mode
func (c *TMDBClient) fetchDetailsWithSettings(ctx context.Context, mediaKind string, id int, language string, settings TMDBSettings) (Result, error) {
	if strings.TrimSpace(language) == "" {
		language = settings.FallbackLanguage
		if language == "" {
			language = "en-US"
		}
	}

	values := url.Values{}
	values.Set("language", language)
	if settings.IncludeImageLanguage != "" {
		values.Set("include_image_language", settings.IncludeImageLanguage)
	}

	appendList := c.buildAppendList(mediaKind, settings.Modules)
	if appendList != "" {
		values.Set("append_to_response", appendList)
	}

	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}

	endpoint := fmt.Sprintf("%s/3/%s/%d?%s", strings.TrimRight(c.config.BaseURL, "/"), mediaKind, id, values.Encode())
	rawPayload, statusCode, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Result{}, err
	}
	if statusCode >= 300 {
		return Result{}, fmt.Errorf("tmdb details failed with status %d", statusCode)
	}

	var details tmdbDetails
	if err := json.Unmarshal(rawPayload, &details); err != nil {
		return Result{}, fmt.Errorf("decode tmdb details: %w", err)
	}

	var supplementalWarnings []string

	// Watch providers if enabled
	if settings.Modules.FetchWatchProviders && mediaKind == "movie" {
		if providers, _, err := c.fetchResource(ctx, fmt.Sprintf("/3/movie/%d/watch/providers", id)); err == nil {
			var doc map[string]json.RawMessage
			if json.Unmarshal(rawPayload, &doc) == nil {
				doc["watch_providers"] = providers
				if expanded, marshalErr := json.Marshal(doc); marshalErr == nil {
					rawPayload = expanded
				}
			}
		} else {
			supplementalWarnings = append(supplementalWarnings, "watch providers: "+err.Error())
		}
	}

	// Process secondary image caching if en-US and profile allows it
	if strings.HasPrefix(strings.ToLower(language), "en") {
		var imgWarnings []string
		rawPayload, imgWarnings = c.cacheDetailImagesWithConfig(ctx, rawPayload, settings)
		if len(imgWarnings) > 0 {
			supplementalWarnings = append(supplementalWarnings, imgWarnings...)
		}
	}

	item := details.tmdbResult
	posterSize := settings.PosterSize
	if posterSize == "" {
		posterSize = "w500"
	}
	backdropSize := settings.BackdropSize
	if backdropSize == "" {
		backdropSize = "original"
	}

	rawPosterURL := c.imageURL(posterSize, item.PosterPath)
	rawBackdropURL := c.imageURL(backdropSize, item.BackdropPath)

	result := Result{
		Provider:      "tmdb",
		ExternalID:    strconv.Itoa(item.ID),
		Locale:        language,
		Title:         firstNonEmpty(item.Title, item.Name),
		OriginalTitle: firstNonEmpty(item.OriginalTitle, item.OriginalName),
		Overview:      item.Overview,
		ReleaseYear:   yearFromDate(firstNonEmpty(item.ReleaseDate, item.FirstAirDate)),
		Rating:        item.VoteAverage,
		PosterPath:    rawPosterURL,
		BannerPath:    rawBackdropURL,
		RawPayload:    rawPayload,
		Genres:        mergeTags(normalizeGenres(details.Genres), normalizeOriginTags(details)),
		Warnings:      supplementalWarnings,
	}

	// 1. Poster Handling
	if settings.Modules.FetchPoster && rawPosterURL != "" {
		// In hybrid or local mode, poster is cached locally. In remote mode, CDN url is used.
		cacheRes, err := cacheRemoteImage(ctx, c.client, rawPosterURL, c.config.ImageDir, "tmdb", "poster_"+result.ExternalID, settings.ImageMode)
		if err == nil {
			result.CachedPosterPath = cacheRes.URL
		} else {
			result.Warnings = append(result.Warnings, "poster cache: "+err.Error())
			result.CachedPosterPath = rawPosterURL
		}
	} else if !settings.Modules.FetchPoster {
		result.PosterPath = ""
		result.CachedPosterPath = ""
	}

	// 2. Backdrop Handling
	if settings.Modules.FetchBackdrop && rawBackdropURL != "" {
		cacheRes, err := cacheRemoteImage(ctx, c.client, rawBackdropURL, c.config.ImageDir, "tmdb", "banner_"+result.ExternalID, settings.ImageMode)
		if err == nil {
			result.CachedBannerPath = cacheRes.URL
		} else {
			result.Warnings = append(result.Warnings, "backdrop cache: "+err.Error())
			result.CachedBannerPath = rawBackdropURL
		}
	} else if !settings.Modules.FetchBackdrop {
		result.BannerPath = ""
		result.CachedBannerPath = ""
	}

	return result, nil
}

// cacheDetailImagesWithConfig downloads extra images (cast, gallery, related) strictly according to active limits
func (c *TMDBClient) cacheDetailImagesWithConfig(ctx context.Context, raw json.RawMessage, settings TMDBSettings) (json.RawMessage, []string) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return raw, []string{"decode embedded images: " + err.Error()}
	}

	cfg := settings.Modules
	warnings := make([]string, 0)

	cacheEntries := func(containerKey, listKey, sourceField, localField, size, keyPrefix string, limit int) {
		if limit <= 0 {
			return
		}
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
			fullURL := c.imageURL(size, path)
			cached, err := cacheRemoteImage(ctx, c.client, fullURL, c.config.ImageDir, "tmdb", keyPrefix+"_"+id, settings.ImageMode)
			if err != nil {
				warnings = append(warnings, keyPrefix+": "+err.Error())
				continue
			}
			if cached.URL != "" {
				entry[localField] = cached.URL
			}
		}
	}

	// Apply configured limits (default 0 in Standard/Essential mode, saving tons of bandwidth)
	profileSize := settings.ProfileSize
	if profileSize == "" {
		profileSize = "w185"
	}
	cacheEntries("credits", "cast", "profile_path", "local_profile_path", profileSize, "profile", cfg.MaxCastImages)
	cacheEntries("aggregate_credits", "cast", "profile_path", "local_profile_path", profileSize, "profile", cfg.MaxCastImages)
	cacheEntries("recommendations", "results", "poster_path", "local_poster_path", "w342", "related_poster", cfg.MaxRelatedPosters)
	cacheEntries("similar", "results", "poster_path", "local_poster_path", "w342", "related_poster", cfg.MaxRelatedPosters)
	cacheEntries("images", "posters", "file_path", "local_poster_path", "w500", "gallery_poster", cfg.MaxGalleryPosters)
	cacheEntries("images", "backdrops", "file_path", "local_backdrop_path", "w780", "gallery_backdrop", cfg.MaxGalleryBackdrops)
	cacheEntries("images", "logos", "file_path", "local_logo_path", "w500", "gallery_logo", cfg.MaxGalleryLogos)

	expanded, err := json.Marshal(document)
	if err != nil {
		return raw, append(warnings, "encode embedded images: "+err.Error())
	}
	return expanded, warnings
}

// fetchSeasonDetailsWithSettings fetches TV season data respecting season switches
func (c *TMDBClient) fetchSeasonDetailsWithSettings(ctx context.Context, seriesID, seasonNumber int, language string, settings TMDBSettings) (SeasonResult, error) {
	if strings.TrimSpace(language) == "" {
		language = settings.FallbackLanguage
		if language == "" {
			language = "en-US"
		}
	}
	values := url.Values{}
	values.Set("language", language)
	if settings.IncludeImageLanguage != "" {
		values.Set("include_image_language", settings.IncludeImageLanguage)
	}

	var appendParts []string
	if settings.Modules.FetchCreditsText || settings.Modules.MaxCastImages > 0 {
		appendParts = append(appendParts, "aggregate_credits", "credits")
	}
	if settings.Modules.FetchExternalIDs {
		appendParts = append(appendParts, "external_ids")
	}
	if settings.Modules.FetchTrailers {
		appendParts = append(appendParts, "videos")
	}
	if settings.Modules.FetchTranslations {
		appendParts = append(appendParts, "translations")
	}
	if settings.Modules.FetchSeasonPosters || settings.Modules.FetchEpisodeStills {
		appendParts = append(appendParts, "images")
	}

	if len(appendParts) > 0 {
		values.Set("append_to_response", strings.Join(appendParts, ","))
	}

	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}

	endpoint := fmt.Sprintf("%s/3/tv/%d/season/%d?%s", strings.TrimRight(c.config.BaseURL, "/"), seriesID, seasonNumber, values.Encode())
	payload, statusCode, err := c.doGet(ctx, endpoint)
	if err != nil {
		return SeasonResult{}, err
	}
	if statusCode >= 300 {
		return SeasonResult{}, fmt.Errorf("tmdb season details failed: status %d", statusCode)
	}

	return SeasonResult{
		Provider:     "tmdb",
		ExternalID:   strconv.Itoa(seriesID),
		Locale:       language,
		SeasonNumber: seasonNumber,
		RawPayload:   payload,
	}, nil
}

func (c *TMDBClient) fetchResource(ctx context.Context, path string) (json.RawMessage, int, error) {
	values := url.Values{}
	if c.config.APIKey != "" && c.config.BearerToken == "" {
		values.Set("api_key", c.config.APIKey)
	}
	endpoint := strings.TrimRight(c.config.BaseURL, "/") + path
	if encoded := values.Encode(); encoded != "" {
		if strings.Contains(endpoint, "?") {
			endpoint += "&" + encoded
		} else {
			endpoint += "?" + encoded
		}
	}
	return c.doGet(ctx, endpoint)
}

// doGet handles HTTP execution with Exponential Backoff on 429 rate limits
func (c *TMDBClient) doGet(ctx context.Context, endpoint string) ([]byte, int, error) {
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, 0, err
		}
		if c.config.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests { // 429 Rate limit
			resp.Body.Close()
			backoff := time.Duration(300*(1<<attempt))*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
			select {
			case <-ctx.Done():
				return nil, 429, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, resp.StatusCode, readErr
		}
		return body, resp.StatusCode, nil
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}
	return nil, 429, errors.New("tmdb request failed after rate limit retries")
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
	ID   int    `json:"id"`
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
