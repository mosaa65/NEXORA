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

	if query.Type == "anime" && s.mal != nil && s.mal.Configured() {
		result, err := s.mal.Lookup(ctx, query)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrNotConfigured) {
			return Result{}, err
		}
	}

	if s.tmdb != nil && s.tmdb.Configured() {
		return s.tmdb.Lookup(ctx, query)
	}

	if query.Type == "anime" && s.mal != nil {
		return s.mal.Lookup(ctx, query)
	}
	if s.tmdb != nil {
		return s.tmdb.Lookup(ctx, query)
	}
	return Result{}, ErrNotConfigured
}
