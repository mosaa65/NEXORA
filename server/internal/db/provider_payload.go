package db

import (
	"encoding/json"
	"fmt"
)

// providerSeasonPayload mirrors the TMDB /tv/{id}/season/{n} response shape as
// it is stored in season_metadata_snapshots.raw_payload.
//
// Only the fields the projection needs are declared. TMDB returns many more
// (credits, images, videos, external ids, episode groups) and decoding them
// would waste memory on data this system does not project, so unknown fields are
// ignored by the decoder rather than modelled.
type providerSeasonPayload struct {
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	AirDate      string `json:"air_date"`
	Episodes     []struct {
		EpisodeNumber int    `json:"episode_number"`
		SeasonNumber  int    `json:"season_number"`
		Name          string `json:"name"`
		Overview      string `json:"overview"`
		StillPath     string `json:"still_path"`
		AirDate       string `json:"air_date"`
		Runtime       int    `json:"runtime"`
	} `json:"episodes"`
}

// decodeProviderSeason turns one stored season payload into the value the
// enrichment pass walks.
//
// The caller-supplied season number is the authority when the payload omits one
// or reports a different value, because the row it came from was filed under
// that number. A provider payload is trusted for everything else.
//
// Episodes with a missing or non-positive episode number are skipped rather than
// defaulted: an episode that cannot be identified by number cannot be matched to
// a local file, and inventing a number would corrupt the season's ordering.
func decodeProviderSeason(seasonNumber int, raw []byte) (providerSeason, error) {
	var payload providerSeasonPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return providerSeason{}, fmt.Errorf("decode season %d payload: %w", seasonNumber, err)
	}

	number := payload.SeasonNumber
	if number == 0 && seasonNumber != 0 {
		number = seasonNumber
	}

	season := providerSeason{
		SeasonNumber: number,
		Name:         payload.Name,
		Overview:     payload.Overview,
		PosterPath:   payload.PosterPath,
		AirDate:      payload.AirDate,
		Episodes:     make([]providerEpisode, 0, len(payload.Episodes)),
	}

	for _, episode := range payload.Episodes {
		if episode.EpisodeNumber <= 0 {
			continue
		}
		season.Episodes = append(season.Episodes, providerEpisode{
			EpisodeNumber: episode.EpisodeNumber,
			Title:         episode.Name,
			Overview:      episode.Overview,
			StillPath:     episode.StillPath,
			AirDate:       episode.AirDate,
			Runtime:       episode.Runtime,
		})
	}

	return season, nil
}
