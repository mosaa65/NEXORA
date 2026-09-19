package identity

import (
	"path/filepath"
	"sort"
	"strings"
)

// -----------------------------------------------------------------------------
// Group context
// -----------------------------------------------------------------------------

// GroupInput is the minimal description of one sibling file, used to build the
// neighbourhood context. Only fields the scanner already has are required, so
// collecting a group costs no extra filesystem work.
type GroupInput struct {
	Path          string
	ParsedTitle   string
	Season        int
	Episode       int
	IsEpisode     bool
	EpisodeStrong bool
	Year          int
}

// BuildGroupContext summarises the directory a file lives in.
//
// Resolving a file purely in isolation is the primary cause of invented works:
// "01.mkv" tells you nothing, but "01.mkv, 02.mkv, 03.mkv, 04.mkv" inside a
// folder called "Silo" tells you almost everything. Group resolution is what
// makes a bare number safe to interpret.
func BuildGroupContext(directory string, siblings []GroupInput, seasonFolder bool) GroupContext {
	context := GroupContext{Size: len(siblings), SeasonFolder: seasonFolder}

	episodeNumbers := make(map[int]struct{}, len(siblings))
	titleVotes := make(map[string]int, len(siblings))

	for _, sibling := range siblings {
		if sibling.IsEpisode {
			context.EpisodicSiblings++
			if sibling.Episode > 0 {
				episodeNumbers[sibling.Episode] = struct{}{}
			}
		}
		if sibling.Year > 0 {
			context.YearSiblings++
		}
		// Only count titles that are actually usable, so a watermark folder does
		// not become the consensus.
		if title := strings.TrimSpace(sibling.ParsedTitle); title != "" {
			if quality := AssessTitle(title, filepath.Base(sibling.Path)); quality.Usable {
				titleVotes[Normalize(title)]++
			}
		}
	}

	for number := range episodeNumbers {
		context.EpisodeNumbers = append(context.EpisodeNumbers, number)
	}
	sort.Ints(context.EpisodeNumbers)

	bestVotes := 0
	for normalized, votes := range titleVotes {
		// The consensus title is stored normalized because that is how it is
		// compared; provenance of the raw form lives on the file row.
		if votes > bestVotes {
			context.ConsensusTitle = normalized
			context.ConsensusVotes = votes
			bestVotes = votes
		}
	}
	if bestVotes < 2 {
		// A single vote is not a consensus and must not be treated as one.
		context.ConsensusTitle = ""
		context.ConsensusVotes = 0
	}

	return context
}

// -----------------------------------------------------------------------------
// Media type detection
// -----------------------------------------------------------------------------

// MediaType is the structural kind of a work. It is separate from Category:
// "anime" is a category that is usually a series, but an anime film is a movie.
type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeSeries  MediaType = "series"
	MediaTypeUnknown MediaType = "unknown"
)

// TypeEvidence is the itemised reasoning for a movie/series decision, so the
// admin UI can explain a classification instead of asserting it.
type TypeEvidence struct {
	MediaType MediaType
	Score     float64
	Reasons   []string
}

// DetectMediaType decides whether evidence describes a movie or a series.
//
// The design forbids deciding from the filename alone, because
// "Toy Story 2.mkv" ends in a number and "Silo S03E04.mkv" is unambiguous. The
// decision therefore weighs structure (episode markers, season folders,
// neighbouring episodes) far above naming.
func DetectMediaType(evidence Evidence) TypeEvidence {
	seriesScore := 0.0
	movieScore := 0.0
	reasons := make([]string, 0, 4)

	// --- Series evidence ---
	if evidence.ParserEpisodeStrong {
		seriesScore += 45
		reasons = append(reasons, "filename carries an explicit episode marker")
	}
	if evidence.SeasonFolderNumber > 0 {
		seriesScore += 25
		reasons = append(reasons, "a season folder declares the season")
	}
	if evidence.Group.SeasonFolder {
		seriesScore += 15
		reasons = append(reasons, "the directory is shaped like a season")
	}
	if episodeRange := evidence.Group.ContiguousEpisodeRange(); episodeRange.Contiguous {
		seriesScore += 30
		reasons = append(reasons, "sibling episodes form the run "+itoa(episodeRange.Low)+"-"+itoa(episodeRange.High))
	} else if evidence.Group.EpisodicSiblings >= 3 {
		seriesScore += 18
		reasons = append(reasons, itoa(evidence.Group.EpisodicSiblings)+" sibling files are episodes")
	}
	switch evidence.CategoryHint {
	case "series", "anime":
		seriesScore += 12
		if evidence.CategoryHint == "anime" {
			reasons = append(reasons, "category is anime, which is normally episodic")
		}
	case "kids", "documentaries":
		// Ambiguous: both contain films and series, so no points either way.
	}

	// --- Movie evidence ---
	if evidence.ParsedYear > 0 {
		movieScore += 20
		reasons = append(reasons, "filename carries a release year, typical of a film release name")
	}
	switch evidence.CategoryHint {
	case "movies":
		movieScore += 25
		reasons = append(reasons, "category folder is movies")
	case "plays":
		movieScore += 30
		reasons = append(reasons, "category folder is plays, which are standalone")
	}
	// A standalone file with strong title and no episode structure.
	if !evidence.ParserIsEpisode && evidence.ParsedPart > 0 {
		movieScore += 15
		reasons = append(reasons, "a part marker indicates a multi-part film")
	}
	// Deliberately NO points for "it is the only file here". A single file is not
	// evidence of a film: an episode folder holding one episode looks identical,
	// and scoring it as a movie is how series ended up filed as films.
	// A group dominated by release years rather than episode numbers is a folder
	// of films.
	if evidence.Group.YearSiblings >= 2 && evidence.Group.EpisodicSiblings == 0 {
		movieScore += 15
		reasons = append(reasons, "siblings carry release years rather than episode numbers")
	}

	// Contradiction handling: a movie folder holding an episode run is a series.
	if evidence.Group.ContiguousEpisodeRange().Contiguous {
		seriesScore += 10
	}

	switch {
	case seriesScore == 0 && movieScore == 0:
		return TypeEvidence{MediaType: MediaTypeUnknown, Score: 0,
			Reasons: []string{"no structural evidence for either a movie or a series"}}
	case seriesScore >= movieScore+10:
		return TypeEvidence{MediaType: MediaTypeSeries, Score: seriesScore - movieScore, Reasons: reasons}
	case movieScore >= seriesScore+10:
		return TypeEvidence{MediaType: MediaTypeMovie, Score: movieScore - seriesScore, Reasons: reasons}
	default:
		// Too close to call. Returning an explicit unknown is the requirement:
		// guessing here is what created series-with-one-movie-files.
		return TypeEvidence{MediaType: MediaTypeUnknown, Score: 0,
			Reasons: append(reasons, "movie and series evidence are too close to decide safely")}
	}
}

// -----------------------------------------------------------------------------
// Season and episode resolution
// -----------------------------------------------------------------------------

// SeasonResolution is the outcome of deciding which season a file belongs to.
type SeasonResolution struct {
	Number     int
	Source     string
	Confidence float64
	Reason     string
}

// ResolveSeason decides the season number, preferring folder evidence over
// filename evidence because owners name folders deliberately while release
// filenames are inconsistent.
func ResolveSeason(evidence Evidence) SeasonResolution {
	// 1. An explicit season folder is authoritative.
	if evidence.SeasonFolderNumber > 0 {
		return SeasonResolution{
			Number:     evidence.SeasonFolderNumber,
			Source:     "folder",
			Confidence: 0.95,
			Reason:     "a folder explicitly declares the season",
		}
	}

	// 2. An episode-range folder ("الحلقات 1-50") implies season 1.
	if evidence.Group.SeasonFolder && evidence.ParsedSeason > 0 {
		return SeasonResolution{
			Number:     evidence.ParsedSeason,
			Source:     "folder_pattern",
			Confidence: 0.8,
			Reason:     "the folder pattern supplies the season",
		}
	}

	// 3. A strong filename season marker (S01E02, 1x02).
	if evidence.ParsedSeason > 0 && evidence.ParserEpisodeStrong {
		return SeasonResolution{
			Number:     evidence.ParsedSeason,
			Source:     "filename",
			Confidence: 0.85,
			Reason:     "the filename declares both season and episode",
		}
	}

	// 4. A bare episode number inside a season-shaped folder is season 1.
	if evidence.ParserEpisodeStrong && evidence.Group.SeasonFolder {
		return SeasonResolution{
			Number:     1,
			Source:     "inferred_from_folder",
			Confidence: 0.6,
			Reason:     "an episode folder with no season number defaults to 1",
		}
	}

	// 5. A file with an episode but no season anywhere: still season 1 for a
	//    series, but recorded with low confidence so it surfaces in review.
	if evidence.ParserEpisodeStrong {
		return SeasonResolution{
			Number:     1,
			Source:     "default",
			Confidence: 0.45,
			Reason:     "no season information was found; defaulted to 1",
		}
	}

	return SeasonResolution{Number: 0, Source: "none", Confidence: 0,
		Reason: "the file shows no season structure"}
}

// EpisodeResolution is the outcome of deciding which episode a file is.
type EpisodeResolution struct {
	Number     int
	End        int
	Valid      bool
	Confidence float64
	Reason     string
}

// ResolveEpisode decides the episode number, refusing to interpret a bare number
// without structural support. This is the rule that stops "Toy Story 2" from
// becoming episode 2.
func ResolveEpisode(evidence Evidence) EpisodeResolution {
	// A strong marker (S01E04, 1x04, الحلقة 4, "Title - 042") is trusted.
	if evidence.ParserEpisodeStrong && evidence.ParsedEpisode > 0 {
		return EpisodeResolution{
			Number:     evidence.ParsedEpisode,
			Valid:      true,
			Confidence: 0.9,
			Reason:     "an explicit episode pattern matched",
		}
	}

	// A weak marker (a bare trailing number) needs neighbourhood support: a
	// season folder, a contiguous sibling run, or a category that is episodic.
	if evidence.ParsedEpisode > 0 && !evidence.ParserEpisodeStrong {
		supported := evidence.SeasonFolderNumber > 0 ||
			evidence.Group.SeasonFolder ||
			evidence.Group.EpisodicSiblings >= 2 ||
			evidence.Group.Size >= 3
		if evidence.Group.ContiguousEpisodeRange().Contiguous {
			supported = true
		}
		switch evidence.CategoryHint {
		case "series", "anime", "kids":
			supported = true
		}
		if supported {
			return EpisodeResolution{
				Number:     evidence.ParsedEpisode,
				Valid:      true,
				Confidence: 0.65,
				Reason:     "a bare number is supported by the surrounding structure",
			}
		}
		return EpisodeResolution{
			Valid:      false,
			Confidence: 0.2,
			Reason:     "a bare number with no structural support is treated as part of the title",
		}
	}

	return EpisodeResolution{Valid: false, Confidence: 0,
		Reason: "no episode information was found"}
}
