package scanner

import "testing"

func TestParseFileName(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		season     int
		episode    int
		resolution string
		year       int
		isEpisode  bool
	}{
		{
			name:       "Attack.on.Titan.S04E05.1080p.mkv",
			title:      "Attack on Titan",
			season:     4,
			episode:    5,
			resolution: "1080p",
			isEpisode:  true,
		},
		{
			name:       "ون بيس الحلقة 1086 4k.mp4",
			title:      "ون بيس",
			season:     1,
			episode:    1086,
			resolution: "4K",
			isEpisode:  true,
		},
		{
			name:       "ون بيس الحلقة ١٠٨٦ 4k.mp4",
			title:      "ون بيس",
			season:     1,
			episode:    1086,
			resolution: "4K",
			isEpisode:  true,
		},
		{
			name:       "The.Last.of.Us.1x02.720p.HDTV.mkv",
			title:      "The Last of Us",
			season:     1,
			episode:    2,
			resolution: "720p",
			isEpisode:  true,
		},
		{
			name:       "Naruto Shippuden EP 12 [1080p].mkv",
			title:      "Naruto Shippuden",
			season:     1,
			episode:    12,
			resolution: "1080p",
			isEpisode:  true,
		},
		{
			name:       "مسلسل البرنس الموسم 2 الحلقة 7 1080p.mkv",
			title:      "مسلسل البرنس",
			season:     2,
			episode:    7,
			resolution: "1080p",
			isEpisode:  true,
		},
		{
			name:       "Inception.2010.1080p.BluRay.mkv",
			title:      "Inception",
			resolution: "1080p",
			year:       2010,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ParseFileName(test.name)
			if got.Title != test.title {
				t.Fatalf("title = %q, want %q", got.Title, test.title)
			}
			if got.SeasonNumber != test.season {
				t.Fatalf("season = %d, want %d", got.SeasonNumber, test.season)
			}
			if got.EpisodeNumber != test.episode {
				t.Fatalf("episode = %d, want %d", got.EpisodeNumber, test.episode)
			}
			if got.Resolution != test.resolution {
				t.Fatalf("resolution = %q, want %q", got.Resolution, test.resolution)
			}
			if got.ReleaseYear != test.year {
				t.Fatalf("year = %d, want %d", got.ReleaseYear, test.year)
			}
			if got.IsEpisode != test.isEpisode {
				t.Fatalf("isEpisode = %v, want %v", got.IsEpisode, test.isEpisode)
			}
		})
	}
}
