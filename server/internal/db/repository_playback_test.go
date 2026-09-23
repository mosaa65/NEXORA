package db

import "testing"

// TestSelectPlaybackSourceResolvesBothIDSpaces pins the mapping the watch screen
// depends on: `?file=` arrives as a `video_files` id from the player's own saved
// state and as an EPISODE id from the work-details page. The two id spaces are
// independent sequences, so each is resolved explicitly instead of assumed — an
// episode id that happens to equal an unrelated file id must not win.
func TestSelectPlaybackSourceResolvesBothIDSpaces(t *testing.T) {
	files := []PlaybackFile{
		{ID: 10, SeasonNumber: 1, EpisodeNumber: 1, EpisodeID: 500},
		{ID: 11, SeasonNumber: 1, EpisodeNumber: 1, EpisodeID: 500}, // a second release
		{ID: 12, SeasonNumber: 1, EpisodeNumber: 2, EpisodeID: 501},
	}

	// A file id selects exactly that file, including a non-first release.
	if got := SelectPlaybackSource(files, 11); got.ID != 11 {
		t.Fatalf("file id 11 must select file 11, got %d", got.ID)
	}

	// An episode id selects that episode's first release.
	if got := SelectPlaybackSource(files, 501); got.ID != 12 {
		t.Fatalf("episode id 501 must select its first file (12), got %d", got.ID)
	}

	// An id in neither space falls back to the catalogue's first file, so a stale
	// bookmark still starts playing instead of erroring.
	if got := SelectPlaybackSource(files, 999); got.ID != 10 {
		t.Fatalf("an unknown id must fall back to the first file, got %d", got.ID)
	}

	// No request at all also starts at the beginning.
	if got := SelectPlaybackSource(files, 0); got.ID != 10 {
		t.Fatalf("an empty request must fall back to the first file, got %d", got.ID)
	}

	// An empty catalogue must not panic; the caller decides what to do with it.
	if got := SelectPlaybackSource(nil, 5); got.ID != 0 {
		t.Fatalf("an empty file list must return the zero value, got %#v", got)
	}
}

// TestPlaybackSiblingsExcludeOtherPartsAndEpisodes guards the quality selector: it
// must only ever offer another encoding of the SAME playable item. A two-part film
// has no episode rows, so both parts share season 0 / episode 0 and would otherwise
// be offered as "video quality 2".
func TestPlaybackSiblingsExcludeOtherPartsAndEpisodes(t *testing.T) {
	movieParts := []PlaybackFile{
		{ID: 1, PartNumber: 1, Resolution: "1920x1080"},
		{ID: 2, PartNumber: 2, Resolution: "1920x1080"},
		{ID: 3, PartNumber: 1, Resolution: "1280x720"},
	}
	siblings := PlaybackSiblings(movieParts, movieParts[0])
	if len(siblings) != 1 || siblings[0].VideoFileID != 3 {
		t.Fatalf("part 1 must only see its own other release, got %#v", siblings)
	}

	episodes := []PlaybackFile{
		{ID: 10, EpisodeID: 500, SeasonNumber: 1, EpisodeNumber: 1},
		{ID: 11, EpisodeID: 500, SeasonNumber: 1, EpisodeNumber: 1},
		{ID: 12, EpisodeID: 501, SeasonNumber: 1, EpisodeNumber: 2},
	}
	siblingsByEpisode := PlaybackSiblings(episodes, episodes[0])
	if len(siblingsByEpisode) != 1 || siblingsByEpisode[0].VideoFileID != 11 {
		t.Fatalf("an episode must only see its own other release, got %#v", siblingsByEpisode)
	}

	// A single-release item offers no selector at all.
	if got := PlaybackSiblings([]PlaybackFile{{ID: 1, EpisodeID: 500}}, PlaybackFile{ID: 1, EpisodeID: 500}); len(got) != 0 {
		t.Fatalf("a lone release must have no siblings, got %#v", got)
	}
}

// TestPlaybackSiblingLabelUsesOnlyRealFacts locks the "no invented data" rule: an
// empty resolution must never become a fabricated quality name.
func TestPlaybackSiblingLabelUsesOnlyRealFacts(t *testing.T) {
	withResolution := PlaybackSiblingLabel(PlaybackFile{ID: 1, Resolution: "3840x2160", VideoCodec: "hevc"})
	if withResolution != "3840x2160 · hevc" {
		t.Fatalf("unexpected label: %q", withResolution)
	}

	// No resolution, but a title: the title is used rather than a guessed quality.
	withTitle := PlaybackSiblingLabel(PlaybackFile{ID: 2, TitleEN: "Director's Cut"})
	if withTitle != "Director's Cut" {
		t.Fatalf("a titleless resolution must fall back to the title, got %q", withTitle)
	}

	// Nothing at all: the id is stated, so it is honest about being an id.
	bare := PlaybackSiblingLabel(PlaybackFile{ID: 7})
	if bare != "إصدار 7" {
		t.Fatalf("unexpected bare label: %q", bare)
	}
}
