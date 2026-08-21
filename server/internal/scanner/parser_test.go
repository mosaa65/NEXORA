package scanner

import (
	"testing"
)

func TestParseFileName(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		wantTitle  string
		wantSeason int
		wantEp     int
		wantRes    string
		wantYear   int
		wantIsEp   bool
	}{
		{
			name:       "Standard S04E05",
			fileName:   "Attack.on.Titan.S04E05.1080p.mkv",
			wantTitle:  "Attack on Titan",
			wantSeason: 4, wantEp: 5, wantRes: "1080p", wantIsEp: true,
		},
		{
			name:       "Arabic episode with 4K",
			fileName:   "ون_بيس_الحلقة_1086_4k.mp4",
			wantTitle:  "ون بيس",
			wantSeason: 1, wantEp: 1086, wantRes: "4K", wantIsEp: true,
		},
		{
			name:       "Arabic episode with Persian digits",
			fileName:   "ون_بيس_الحلقة_١٠٨٦_4k.mp4",
			wantTitle:  "ون بيس",
			wantSeason: 1, wantEp: 1086, wantRes: "4K", wantIsEp: true,
		},
		{
			name:       "1x02 format",
			fileName:   "The.Last.of.Us.1x02.720p.HDTV.mkv",
			wantTitle:  "The Last of Us",
			wantSeason: 1, wantEp: 2, wantRes: "720p", wantIsEp: true,
		},
		{
			name:       "Bracket tags with EP",
			fileName:   "Naruto_Shippuden_EP_12_[1080p].mkv",
			wantTitle:  "Naruto Shippuden",
			wantSeason: 1, wantEp: 12, wantRes: "1080p", wantIsEp: true,
		},
		{
			name:       "Arabic season and episode",
			fileName:   "مسلسل_البرنس_الموسم_2_الحلقة_7_1080p.mkv",
			wantTitle:  "مسلسل البرنس",
			wantSeason: 2, wantEp: 7, wantRes: "1080p", wantIsEp: true,
		},
		{
			name:      "Movie with year and resolution",
			fileName:  "Inception.2010.1080p.BluRay.mkv",
			wantTitle: "Inception",
			wantYear:  2010, wantRes: "1080p", wantIsEp: false,
		},
		{
			name:      "Movie without episode info",
			fileName:  "Interstellar.4K.HDR.mkv",
			wantTitle: "Interstellar",
			wantRes:   "4K", wantIsEp: false,
		},
		{
			name:       "Release group brackets stripped with anime episode number",
			fileName:   "[SubGroup] Dragon Ball Z - 042 [BDRip 1080p] [FLAC].mkv",
			wantTitle:  "Dragon Ball Z",
			wantSeason: 1,
			wantEp:     42,
			wantRes:    "1080p",
			wantIsEp:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFileName(tt.fileName)
			if result.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.SeasonNumber != tt.wantSeason {
				t.Errorf("season = %d, want %d", result.SeasonNumber, tt.wantSeason)
			}
			if result.EpisodeNumber != tt.wantEp {
				t.Errorf("episode = %d, want %d", result.EpisodeNumber, tt.wantEp)
			}
			if result.Resolution != tt.wantRes {
				t.Errorf("resolution = %q, want %q", result.Resolution, tt.wantRes)
			}
			if tt.wantYear > 0 && result.ReleaseYear != tt.wantYear {
				t.Errorf("year = %d, want %d", result.ReleaseYear, tt.wantYear)
			}
			if result.IsEpisode != tt.wantIsEp {
				t.Errorf("isEpisode = %v, want %v", result.IsEpisode, tt.wantIsEp)
			}
		})
	}
}

func TestParseFilePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantTitle  string
		wantAR     string
		wantEN     string
		wantSeason int
		wantEp     int
		wantIsEp   bool
	}{
		{
			name:       "Anime folder structure with dual title",
			path:       "D:/Media/Anime - أنمي/Attack on Titan - هجوم العمالقة/Season 01/Attack.on.Titan.S01E01.1080p.mkv",
			wantTitle:  "Attack on Titan - هجوم العمالقة",
			wantAR:     "هجوم العمالقة",
			wantEN:     "Attack on Titan",
			wantSeason: 1,
			wantEp:     1,
			wantIsEp:   true,
		},
		{
			name:       "Arabic season folder الموسم الأول",
			path:       "E:/مسلسلات/صراع العروش/الموسم الأول/GOT.S01E03.mkv",
			wantTitle:  "صراع العروش",
			wantAR:     "صراع العروش",
			wantSeason: 1,
			wantEp:     3,
			wantIsEp:   true,
		},
		{
			name:       "Movie in titled folder",
			path:       "F:/Movies/Inception (2010)/Inception.2010.1080p.BluRay.mkv",
			wantTitle:  "Inception",
			wantSeason: 0,
			wantIsEp:   false,
		},
		{
			name:       "Cartoon folder with Arabic",
			path:       "D:/كرتون/Tom and Jerry - توم وجيري/Season 01/Tom.and.Jerry.S01E05.mp4",
			wantTitle:  "Tom and Jerry - توم وجيري",
			wantAR:     "توم وجيري",
			wantEN:     "Tom and Jerry",
			wantSeason: 1,
			wantEp:     5,
			wantIsEp:   true,
		},
		{
			name:       "Episode without season folder defaults to 1",
			path:       "D:/Anime/One Piece/One.Piece.EP.500.mkv",
			wantSeason: 1,
			wantEp:     500,
			wantIsEp:   true,
		},
		{
			name:       "Arabic season number folder",
			path:       "E:/مسلسلات/الاختيار/الموسم 2/الاختيار.S02E10.mkv",
			wantSeason: 2,
			wantEp:     10,
			wantIsEp:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilePath(tt.path)
			if tt.wantTitle != "" && result.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", result.Title, tt.wantTitle)
			}
			if tt.wantAR != "" && result.TitleAR != tt.wantAR {
				t.Errorf("titleAR = %q, want %q", result.TitleAR, tt.wantAR)
			}
			if tt.wantEN != "" && result.TitleEN != tt.wantEN {
				t.Errorf("titleEN = %q, want %q", result.TitleEN, tt.wantEN)
			}
			if result.SeasonNumber != tt.wantSeason {
				t.Errorf("season = %d, want %d", result.SeasonNumber, tt.wantSeason)
			}
			if tt.wantEp > 0 && result.EpisodeNumber != tt.wantEp {
				t.Errorf("episode = %d, want %d", result.EpisodeNumber, tt.wantEp)
			}
			if result.IsEpisode != tt.wantIsEp {
				t.Errorf("isEpisode = %v, want %v", result.IsEpisode, tt.wantIsEp)
			}
		})
	}
}

func TestDetectCategoryFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"D:/Media/Anime - أنمي/One Piece/Season 01/ep01.mkv", "anime"},
		{"D:/Media/انمي/Naruto/ep01.mkv", "anime"},
		{"D:/كرتون/SpongeBob/S01E01.mkv", "kids"},
		{"D:/Cartoons/Tom and Jerry/S01E01.mkv", "kids"},
		{"D:/أطفال/Dora/S01E01.mkv", "kids"},
		{"D:/وثائقي/Planet Earth/S01E01.mkv", "documentaries"},
		{"D:/Documentaries/Nature/ep01.mkv", "documentaries"},
		{"D:/مسرحيات/Play01.mkv", "plays"},
		{"D:/مسلسلات/Breaking Bad/S01E01.mkv", "series"},
		{"D:/Series/The Office/S01E01.mkv", "series"},
		{"D:/Movies/Inception.mkv", "movies"},
		{"D:/أفلام/TheMessage.mkv", "movies"},
		{"D:/RandomFolder/file.mkv", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectCategoryFromPath(tt.path)
			if got != tt.want {
				t.Errorf("DetectCategoryFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectOriginTagsFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"D:/Media/مسلسلات/عربي/مسلسل الاختيار/الموسم 1/الحلقة 1.mp4", "عربي"},
		{"D:/Media/مسلسلات تركية/الطائر الرفراف/Season 01/E01.mkv", "تركي"},
		{"D:/Media/Series/Korean/Moving/Season 01/E01.mkv", "كوري"},
		{"D:/Media/Series/English/Breaking Bad/Season 01/E01.mkv", "أجنبي"},
		// The file name is intentionally ignored: subtitle language is not origin.
		{"D:/Media/Series/Breaking Bad/Arabic subtitles/E01.mkv", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			tags := DetectOriginTagsFromPath(tt.path)
			got := ""
			if len(tags) > 0 {
				got = tags[0]
			}
			if got != tt.want {
				t.Errorf("DetectOriginTagsFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseSeasonFromFolder(t *testing.T) {
	tests := []struct {
		folder string
		want   int
	}{
		{"Season 01", 1},
		{"Season 1", 1},
		{"Season 12", 12},
		{"s03", 3},
		{"S1", 1},
		{"الموسم 01", 1},
		{"الموسم 3", 3},
		{"الموسم الأول", 1},
		{"الموسم الثاني", 2},
		{"الموسم العاشر", 10},
		{"موسم الاول", 1},
		{"الحلقات 1-50", 1},
		{"Episodes 1-24", 1},
		{"Random Folder", 0},
		{"Attack on Titan", 0},
	}

	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			got := parseSeasonFromFolder(tt.folder)
			if got != tt.want {
				t.Errorf("parseSeasonFromFolder(%q) = %d, want %d", tt.folder, got, tt.want)
			}
		})
	}
}

func TestSplitDualTitle(t *testing.T) {
	tests := []struct {
		raw    string
		wantAR string
		wantEN string
	}{
		{"Attack on Titan - هجوم العمالقة", "هجوم العمالقة", "Attack on Titan"},
		{"هجوم العمالقة - Attack on Titan", "هجوم العمالقة", "Attack on Titan"},
		{"Inception", "", "Inception"},
		{"صراع العروش", "صراع العروش", "صراع العروش"},
		{"Tom and Jerry - توم وجيري", "توم وجيري", "Tom and Jerry"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			gotAR, gotEN := splitDualTitle(tt.raw)
			if gotAR != tt.wantAR {
				t.Errorf("splitDualTitle(%q) ar = %q, want %q", tt.raw, gotAR, tt.wantAR)
			}
			if gotEN != tt.wantEN {
				t.Errorf("splitDualTitle(%q) en = %q, want %q", tt.raw, gotEN, tt.wantEN)
			}
		})
	}
}
