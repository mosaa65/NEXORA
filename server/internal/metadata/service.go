package metadata

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

type Service struct {
	tmdb *TMDBClient
	mal  *MALClient
}

// LookupByExternalID fetches another locale for an already-confirmed TMDB ID.
// It must be used after matching so locale-specific search ranking cannot
// accidentally select a different title with the same name.
func (s *Service) LookupByExternalID(ctx context.Context, query Query, externalID string) (Result, error) {
	if s.tmdb == nil || !s.tmdb.Configured() {
		return Result{}, ErrNotConfigured
	}
	id, err := strconv.Atoi(externalID)
	if err != nil || id <= 0 {
		return Result{}, errors.New("invalid tmdb external id")
	}
	mediaKind := "movie"
	if query.Type == "series" || query.Type == "anime" || query.Type == "tv" {
		mediaKind = "tv"
	}
	if query.Language == "" {
		query.Language = "en-US"
	}
	return s.tmdb.fetchDetails(ctx, mediaKind, id, query.Language)
}

// LookupSeasonByExternalID fetches one known TV season in a requested locale.
func (s *Service) LookupSeasonByExternalID(ctx context.Context, externalID string, seasonNumber int, language string) (SeasonResult, error) {
	if s.tmdb == nil || !s.tmdb.Configured() {
		return SeasonResult{}, ErrNotConfigured
	}
	id, err := strconv.Atoi(externalID)
	if err != nil || id <= 0 || seasonNumber < 0 {
		return SeasonResult{}, errors.New("invalid tmdb season request")
	}
	return s.tmdb.fetchSeasonDetails(ctx, id, seasonNumber, language)
}

func NewService(tmdb *TMDBClient, mal *MALClient) *Service {
	return &Service{tmdb: tmdb, mal: mal}
}

func (s *Service) Lookup(ctx context.Context, query Query) (Result, error) {
	query.Title = strings.TrimSpace(query.Title)
	if query.Title == "" {
		return Result{}, errors.New("title is required")
	}
	if query.Language == "" {
		query.Language = "en-US"
	}

	// 1. If MAL is configured and type is anime
	if query.Type == "anime" && s.mal != nil && s.mal.Configured() {
		result, err := s.mal.Lookup(ctx, query)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrNotConfigured) {
			return Result{}, err
		}
	}

	// 2. If TMDB is configured
	if s.tmdb != nil && s.tmdb.Configured() {
		result, err := s.tmdb.Lookup(ctx, query)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrNotConfigured) {
			return Result{}, err
		}
	}

	// 3. Fallback to local catalog
	if local, found := findInLocalCatalog(query); found {
		return local, nil
	}

	if (s.tmdb == nil || !s.tmdb.Configured()) && (s.mal == nil || !s.mal.Configured()) {
		return Result{}, errors.New("لم يتم ضبط مفتاح TMDB في ملف .env (NEXORA_TMDB_API_KEY)")
	}

	return Result{}, ErrNotFound
}
