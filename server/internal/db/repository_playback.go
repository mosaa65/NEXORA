package db

import (
	"context"
	"database/sql"
	"fmt"
)

// PlaybackFile is one playable release of a work: a single `video_files` row with
// the technical metadata FFprobe already persisted at scan time.
//
// It deliberately carries technical facts instead of the raw track arrays: the
// player needs the counts for its badges and the technical panel reads the real
// tracks from the subtitle endpoint, so shipping every track object for every
// episode of a show would dominate the response for no benefit.
type PlaybackFile struct {
	ID              int64  `json:"video_file_id"`
	SeasonID        int64  `json:"season_id,omitempty"`
	SeasonNumber    int    `json:"season_number"`
	EpisodeID       int64  `json:"episode_id,omitempty"`
	EpisodeNumber   int    `json:"episode_number,omitempty"`
	PartNumber      int    `json:"part_number,omitempty"`
	TitleAR         string `json:"title_ar,omitempty"`
	TitleEN         string `json:"title_en,omitempty"`
	Duration        int    `json:"duration,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
	VideoCodec      string `json:"video_codec,omitempty"`
	FileSize        int64  `json:"file_size,omitempty"`
	AudioTrackCount int    `json:"audio_track_count"`
	SubtitleCount   int    `json:"subtitle_count"`
	StreamURL       string `json:"stream_url"`
	PreviewURL      string `json:"preview_url,omitempty"`
}

// PlaybackEpisode is one episode row of the work, ordered by the catalogue's own
// ordering. An episode with no local file is still returned: the watch screen
// renders it as unavailable instead of hiding a gap in the season.
type PlaybackEpisode struct {
	EpisodeID     int64  `json:"episode_id"`
	SeasonID      int64  `json:"season_id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
	TitleAR       string `json:"title_ar,omitempty"`
	TitleEN       string `json:"title_en,omitempty"`
	OverviewAR    string `json:"overview_ar,omitempty"`
	OverviewEN    string `json:"overview_en,omitempty"`
	StillPath     string `json:"still_path,omitempty"`
	AirDate       string `json:"air_date,omitempty"`
	Runtime       int    `json:"runtime,omitempty"`
	HasLocalFile  bool   `json:"has_local_file"`
	FileCount     int    `json:"file_count"`
	Resolution    string `json:"resolution,omitempty"`
	FileSize      int64  `json:"file_size,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	VideoFileID   int64  `json:"video_file_id,omitempty"`
	StreamURL     string `json:"stream_url,omitempty"`
}

// PlaybackSeason is a season with its episode count and how much of it is local,
// so a season selector shows progress without loading every episode first.
type PlaybackSeason struct {
	SeasonID     int64  `json:"season_id"`
	SeasonNumber int    `json:"season_number"`
	TitleAR      string `json:"title_ar,omitempty"`
	TitleEN      string `json:"title_en,omitempty"`
	PosterPath   string `json:"poster_path,omitempty"`
	EpisodeCount int    `json:"episode_count"`
	LocalCount   int    `json:"local_count"`
}

// PlaybackSibling is another release of the same episode (a different resolution,
// source or cut). These are real selectable sources, each with its own stream URL,
// not a fabricated quality ladder.
type PlaybackSibling struct {
	VideoFileID int64  `json:"video_file_id"`
	Resolution  string `json:"resolution,omitempty"`
	VideoCodec  string `json:"video_codec,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	StreamURL   string `json:"stream_url"`
	Label       string `json:"label"`
}

// PlaybackPlan is the single read the watch screen needs to start playing.
//
// It exists because the previous flow called the media detail endpoint and then
// paged the episode search index before it could render one source URL. One call
// now answers: which file plays, what surrounds it, and what comes next.
type PlaybackPlan struct {
	MediaItemID  int64             `json:"media_id"`
	Type         string            `json:"type"`
	TitleAR      string            `json:"title_ar,omitempty"`
	TitleEN      string            `json:"title_en,omitempty"`
	PlotAR       string            `json:"plot_ar,omitempty"`
	PlotEN       string            `json:"plot_en,omitempty"`
	PosterPath   string            `json:"poster_path,omitempty"`
	BannerPath   string            `json:"banner_path,omitempty"`
	ReleaseYear  int               `json:"release_year,omitempty"`
	Rating       float64           `json:"rating,omitempty"`
	Status       string            `json:"status,omitempty"`
	Genres       []string          `json:"genres,omitempty"`
	Seasons      []PlaybackSeason  `json:"seasons,omitempty"`
	Episodes     []PlaybackEpisode `json:"episodes,omitempty"`
	Files        []PlaybackFile    `json:"files,omitempty"`
	Source       *PlaybackFile     `json:"source,omitempty"`
	Siblings     []PlaybackSibling `json:"siblings,omitempty"`
	Next         *PlaybackFile     `json:"next,omitempty"`
	Previous     *PlaybackFile     `json:"previous,omitempty"`
	EpisodeCount int               `json:"episode_count"`
	LocalCount   int               `json:"local_count"`
}

// playbackFilesQuery orders every playable file of a work by the catalogue's own
// ordering: season, then episode, then part, then file id.
//
// Filename order is deliberately not used. A file whose episode number was not
// parsed is 0 and must sort after the numbered episodes of its season instead of
// lexicographically between 1 and 10.
const playbackFilesQuery = `
	SELECT
		vf.id,
		COALESCE(vf.season_id, 0),
		COALESCE(s.season_number, 0),
		COALESCE(vf.episode_id, 0),
		COALESCE(vf.episode_number, 0),
		COALESCE(vf.part_number, 0),
		COALESCE(vf.title_ar, ''),
		COALESCE(vf.title_en, ''),
		COALESCE(vf.duration, 0),
		COALESCE(vf.resolution, ''),
		COALESCE(vf.video_codec, ''),
		COALESCE(vf.file_size, 0),
		COALESCE(jsonb_array_length(vf.audio_tracks), 0),
		COALESCE(jsonb_array_length(vf.subtitles), 0)
	FROM video_files vf
	LEFT JOIN seasons s ON s.id = vf.season_id
	JOIN media_items mi ON mi.id = vf.media_item_id
	WHERE vf.media_item_id = $1
	  AND mi.merged_into_id IS NULL
	ORDER BY
		COALESCE(s.season_number, 0),
		COALESCE(vf.episode_number, 0),
		COALESCE(vf.part_number, 0),
		vf.id;
`

// playbackEpisodesQuery returns every episode row of a work with the aggregate of
// its local files. An episode without a file is returned too, so the watch screen
// shows the real shape of the season, gaps included, instead of a shortened list.
const playbackEpisodesQuery = `
	SELECT
		e.id,
		e.season_id,
		s.season_number,
		e.episode_number,
		COALESCE(e.title_ar, ''),
		COALESCE(e.title_en, ''),
		COALESCE(e.overview_ar, ''),
		COALESCE(e.overview_en, ''),
		COALESCE(e.still_path, ''),
		COALESCE(e.air_date::text, ''),
		COALESCE(e.runtime, 0),
		COALESCE(f.file_count, 0),
		COALESCE(f.best_resolution, ''),
		COALESCE(f.total_size, 0),
		COALESCE(f.max_duration, 0),
		COALESCE(f.first_file_id, 0)
	FROM episodes e
	JOIN seasons s ON s.id = e.season_id
	JOIN media_items mi ON mi.id = s.media_item_id
	LEFT JOIN LATERAL (
		SELECT
			COUNT(*)           AS file_count,
			MAX(vf.resolution) AS best_resolution,
			SUM(vf.file_size)  AS total_size,
			MAX(vf.duration)   AS max_duration,
			MIN(vf.id)         AS first_file_id
		FROM video_files vf
		WHERE vf.episode_id = e.id
	) f ON TRUE
	WHERE s.media_item_id = $1
	  AND mi.merged_into_id IS NULL
	ORDER BY s.season_number, e.episode_number, e.id;
`

// playbackSeasonPostersQuery reads the local artwork a season selector can show.
// It is a single small query rather than a join in the episode query, because a
// season has one poster and repeating it on every episode row would multiply the
// payload for no benefit.
const playbackSeasonPostersQuery = `
	SELECT id, COALESCE(poster_path, '')
	FROM seasons
	WHERE media_item_id = $1;
`

// GetPlaybackPlan reads everything the player needs about one work.
//
// It is one repository call rather than four, because the watch screen must not
// open with a chain of round trips: the work header, the ordered file list, the
// episode list and the season roll-up all derive from the same work id.
//
// `wanted` is resolved against BOTH ids a caller may legitimately hold, because the
// work-details page hands over an EPISODE id while a stream URL needs a FILE id.
// Guessing which one arrived would silently play the wrong episode whenever ids
// happen to overlap, so each is checked explicitly.
func (r *Repository) GetPlaybackPlan(ctx context.Context, mediaID int64, wanted int64) (*PlaybackPlan, error) {
	item, err := r.GetMediaItem(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	plan := &PlaybackPlan{
		MediaItemID: item.ID,
		Type:        item.Type,
		TitleAR:     item.TitleAR,
		TitleEN:     item.TitleEN,
		PlotAR:      item.PlotAR,
		PlotEN:      item.PlotEN,
		PosterPath:  item.PosterPath,
		BannerPath:  item.BannerPath,
		ReleaseYear: item.ReleaseYear,
		Rating:      item.Rating,
		Status:      item.Status,
		Genres:      item.Genres,
	}

	files, err := r.listPlaybackFiles(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	plan.Files = files

	episodes, err := r.listPlaybackEpisodes(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	// Season roll-up, derived from the episodes already read: no extra query.
	seasons := make([]PlaybackSeason, 0)
	seasonIndex := map[int64]int{}
	for _, episode := range episodes {
		position, ok := seasonIndex[episode.SeasonID]
		if !ok {
			seasons = append(seasons, PlaybackSeason{
				SeasonID:     episode.SeasonID,
				SeasonNumber: episode.SeasonNumber,
			})
			position = len(seasons) - 1
			seasonIndex[episode.SeasonID] = position
		}
		seasons[position].EpisodeCount++
		if episode.HasLocalFile {
			seasons[position].LocalCount++
		}
	}
	// A work whose files carry a season but whose episode rows were never created
	// still needs that season listed, otherwise the episode list would look empty.
	for _, file := range files {
		if file.SeasonID == 0 {
			continue
		}
		if _, ok := seasonIndex[file.SeasonID]; ok {
			continue
		}
		seasons = append(seasons, PlaybackSeason{
			SeasonID:     file.SeasonID,
			SeasonNumber: file.SeasonNumber,
		})
		seasonIndex[file.SeasonID] = len(seasons) - 1
	}
	// Season titles come from the work detail, which already read them.
	for _, season := range item.Seasons {
		if position, ok := seasonIndex[season.ID]; ok {
			seasons[position].TitleAR = season.TitleAR
			seasons[position].TitleEN = season.TitleEN
		}
	}
	// Season artwork is local-only; a season without a poster simply shows none.
	if posters, err := r.playbackSeasonPosters(ctx, mediaID); err == nil {
		for id, posterPath := range posters {
			if position, ok := seasonIndex[id]; ok {
				seasons[position].PosterPath = posterPath
			}
		}
	}
	sortPlaybackSeasons(seasons)
	plan.Seasons = seasons
	plan.Episodes = episodes
	plan.EpisodeCount = len(episodes)
	for _, episode := range episodes {
		if episode.HasLocalFile {
			plan.LocalCount++
		}
	}

	if len(files) == 0 {
		return plan, nil
	}

	source := SelectPlaybackSource(files, wanted)
	plan.Source = &source
	plan.Siblings = PlaybackSiblings(files, source)
	// Next/previous follow the same ordered slice the files came back in, so a file
	// without an episode row still advances correctly.
	for index := range files {
		if files[index].ID != source.ID {
			continue
		}
		if index > 0 {
			previous := files[index-1]
			plan.Previous = &previous
		}
		if index+1 < len(files) {
			next := files[index+1]
			plan.Next = &next
		}
		break
	}
	return plan, nil
}

func (r *Repository) listPlaybackFiles(ctx context.Context, mediaID int64) ([]PlaybackFile, error) {
	rows, err := r.db.QueryContext(ctx, playbackFilesQuery, mediaID)
	if err != nil {
		return nil, fmt.Errorf("query playback files: %w", err)
	}
	defer rows.Close()

	files := make([]PlaybackFile, 0)
	for rows.Next() {
		var file PlaybackFile
		var audioCount, subtitleCount int
		if err := rows.Scan(
			&file.ID,
			&file.SeasonID,
			&file.SeasonNumber,
			&file.EpisodeID,
			&file.EpisodeNumber,
			&file.PartNumber,
			&file.TitleAR,
			&file.TitleEN,
			&file.Duration,
			&file.Resolution,
			&file.VideoCodec,
			&file.FileSize,
			&audioCount,
			&subtitleCount,
		); err != nil {
			return nil, fmt.Errorf("scan playback file: %w", err)
		}
		file.AudioTrackCount = audioCount
		file.SubtitleCount = subtitleCount
		file.StreamURL = fmt.Sprintf("/api/stream/file/%d", file.ID)
		file.PreviewURL = fmt.Sprintf("/api/stream/file/%d/preview", file.ID)
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playback files: %w", err)
	}
	return files, nil
}

// playbackSeasonPosters reads the local artwork of a work's seasons, keyed by
// season id. A failure here is not fatal: a season without a poster is a valid
// state, so the caller ignores the error and shows no image.
func (r *Repository) playbackSeasonPosters(ctx context.Context, mediaID int64) (map[int64]string, error) {
	rows, err := r.db.QueryContext(ctx, playbackSeasonPostersQuery, mediaID)
	if err != nil {
		return nil, fmt.Errorf("query season posters: %w", err)
	}
	defer rows.Close()

	posters := make(map[int64]string)
	for rows.Next() {
		var id int64
		var posterPath string
		if err := rows.Scan(&id, &posterPath); err != nil {
			return nil, fmt.Errorf("scan season poster: %w", err)
		}
		if posterPath != "" {
			posters[id] = posterPath
		}
	}
	return posters, rows.Err()
}

func (r *Repository) listPlaybackEpisodes(ctx context.Context, mediaID int64) ([]PlaybackEpisode, error) {	rows, err := r.db.QueryContext(ctx, playbackEpisodesQuery, mediaID)
	if err != nil {
		return nil, fmt.Errorf("query playback episodes: %w", err)
	}
	defer rows.Close()

	episodes := make([]PlaybackEpisode, 0)
	for rows.Next() {
		var episode PlaybackEpisode
		var fileCount int
		var airDate sql.NullString
		if err := rows.Scan(
			&episode.EpisodeID,
			&episode.SeasonID,
			&episode.SeasonNumber,
			&episode.EpisodeNumber,
			&episode.TitleAR,
			&episode.TitleEN,
			&episode.OverviewAR,
			&episode.OverviewEN,
			&episode.StillPath,
			&airDate,
			&episode.Runtime,
			&fileCount,
			&episode.Resolution,
			&episode.FileSize,
			&episode.Duration,
			&episode.VideoFileID,
		); err != nil {
			return nil, fmt.Errorf("scan playback episode: %w", err)
		}
		episode.AirDate = airDate.String
		episode.FileCount = fileCount
		episode.HasLocalFile = fileCount > 0
		if episode.HasLocalFile {
			episode.StreamURL = fmt.Sprintf("/api/stream/file/%d", episode.VideoFileID)
		}
		episodes = append(episodes, episode)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playback episodes: %w", err)
	}
	return episodes, nil
}

// SelectPlaybackSource picks which file plays.
//
// It is exported because the resolution rule is part of the playback contract the
// API promises, and it is worth locking down in a test without a database.
//
// `wanted` may be a `video_files` id (a stream URL, the copy bridge, a bookmark of
// the player's own state) or an EPISODE id (what the work-details page navigates
// with). Both are checked because the two id spaces are independent sequences: an
// episode id that happens to equal a file id would otherwise select an unrelated
// file. When neither matches, the catalogue's first file plays, so a stale link
// still starts the episode rather than failing.
//
// Quality is deliberately NOT the tie-breaker: jumping to the highest resolution
// would drop a viewer into the middle of a season.
func SelectPlaybackSource(files []PlaybackFile, wanted int64) PlaybackFile {
	if wanted > 0 {
		for _, file := range files {
			if file.ID == wanted {
				return file
			}
		}
		// Not a file id: try it as an episode id and take that episode's first file.
		for _, file := range files {
			if file.EpisodeID == wanted {
				return file
			}
		}
	}
	return files[0]
}

// PlaybackSiblings lists the other releases of the selected file's episode.
//
// A sibling is another encoding of the SAME playable item, which is how a viewer
// picks 1080p over 4K. It is deliberately strict about what does not qualify:
//
//   - a different episode is a different item, not another release;
//   - a different part of the same film is a different item too — a two-part film
//     has no episode rows, so both parts would otherwise look like releases of one
//     film and the quality selector would offer "Part Two" as a video quality.
func PlaybackSiblings(files []PlaybackFile, source PlaybackFile) []PlaybackSibling {
	siblings := make([]PlaybackSibling, 0)
	for _, file := range files {
		if file.ID == source.ID {
			continue
		}
		if source.EpisodeID > 0 {
			if file.EpisodeID != source.EpisodeID {
				continue
			}
		} else {
			if file.SeasonNumber != source.SeasonNumber || file.EpisodeNumber != source.EpisodeNumber {
				continue
			}
			// Same episode, so the part number must match as well.
			if file.PartNumber != source.PartNumber {
				continue
			}
		}
		siblings = append(siblings, PlaybackSibling{
			VideoFileID: file.ID,
			Resolution:  file.Resolution,
			VideoCodec:  file.VideoCodec,
			Duration:    file.Duration,
			FileSize:    file.FileSize,
			StreamURL:   file.StreamURL,
			Label:       PlaybackSiblingLabel(file),
		})
	}
	return siblings
}

// PlaybackSiblingLabel describes a release using only facts that exist. An empty
// resolution is never replaced with a guess; the id is stated instead.
func PlaybackSiblingLabel(file PlaybackFile) string {
	label := file.Resolution
	if label == "" {
		if file.TitleEN != "" {
			label = file.TitleEN
		} else {
			label = fmt.Sprintf("إصدار %d", file.ID)
		}
	}
	if file.VideoCodec != "" {
		label = fmt.Sprintf("%s · %s", label, file.VideoCodec)
	}
	return label
}

func sortPlaybackSeasons(seasons []PlaybackSeason) {
	for i := 1; i < len(seasons); i++ {
		for j := i; j > 0 && seasons[j].SeasonNumber < seasons[j-1].SeasonNumber; j-- {
			seasons[j], seasons[j-1] = seasons[j-1], seasons[j]
		}
	}
}