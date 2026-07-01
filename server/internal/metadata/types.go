package metadata

import "errors"

var (
	ErrNotConfigured = errors.New("metadata provider is not configured")
	ErrNotFound      = errors.New("metadata not found")
)

type Query struct {
	Title    string `json:"title"`
	Type     string `json:"type"`
	Year     int    `json:"year,omitempty"`
	Language string `json:"language,omitempty"`
}

type Result struct {
	Provider         string   `json:"provider"`
	ExternalID       string   `json:"externalId"`
	Title            string   `json:"title"`
	OriginalTitle    string   `json:"originalTitle,omitempty"`
	Overview         string   `json:"overview,omitempty"`
	ReleaseYear      int      `json:"releaseYear,omitempty"`
	Rating           float64  `json:"rating,omitempty"`
	PosterPath       string   `json:"posterPath,omitempty"`
	BannerPath       string   `json:"bannerPath,omitempty"`
	CachedPosterPath string   `json:"cachedPosterPath,omitempty"`
	CachedBannerPath string   `json:"cachedBannerPath,omitempty"`
	Genres           []string `json:"genres,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}
