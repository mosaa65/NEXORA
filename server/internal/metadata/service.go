package metadata

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	tmdb *TMDBClient
	mal  *MALClient
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
