package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type ParsedName struct {
	Original      string `json:"original"`
	Title         string `json:"title"`
	SeasonNumber  int    `json:"seasonNumber,omitempty"`
	EpisodeNumber int    `json:"episodeNumber,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	ReleaseYear   int    `json:"releaseYear,omitempty"`
	Extension     string `json:"extension,omitempty"`
	IsEpisode     bool   `json:"isEpisode"`
}

type episodePattern struct {
	re           *regexp.Regexp
	seasonGroup  int
	episodeGroup int
	defaultSeason int
}

var (
	resolutionRE = regexp.MustCompile(`(?i)(?:^|\s)(4320p|2160p|1080p|720p|576p|480p|360p|8k|4k|uhd|fhd|hd)(?:\s|$)`)
	yearRE       = regexp.MustCompile(`(?:^|\s)((?:19|20)\d{2})(?:\s|$)`)

	episodePatterns = []episodePattern{
		{
			re:           regexp.MustCompile(`(?i)(?:^|\s)s\s*(\d{1,2})\s*e\s*(\d{1,4})(?:\s|$)`),
			seasonGroup:  1,
			episodeGroup: 2,
		},
		{
			re:           regexp.MustCompile(`(?i)(?:^|\s)(\d{1,2})x(\d{1,4})(?:\s|$)`),
			seasonGroup:  1,
			episodeGroup: 2,
		},
		{
			re:           regexp.MustCompile(`(?i)(?:season|s)\s*(\d{1,2})\s*(?:episode|ep|e)\s*(\d{1,4})(?:\s|$)`),
			seasonGroup:  1,
			episodeGroup: 2,
		},
		{
			re:           regexp.MustCompile(`(?:الموسم|موسم)\s*(\d{1,2})\s*(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`),
			seasonGroup:  1,
			episodeGroup: 2,
		},
		{
			re:             regexp.MustCompile(`(?i)(?:^|\s)(?:episode|ep|e)\s*(\d{1,4})(?:\s|$)`),
			episodeGroup:   1,
			defaultSeason:  1,
		},
		{
			re:             regexp.MustCompile(`(?:^|\s)(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`),
			episodeGroup:   1,
			defaultSeason:  1,
		},
	}

	noiseTokens = map[string]struct{}{
		"aac": {}, "ac3": {}, "bdrip": {}, "bluray": {}, "brrip": {}, "cam": {},
		"ddp": {}, "dl": {}, "dual": {}, "dvdrip": {}, "h264": {}, "h265": {},
		"hdcam": {}, "hdtv": {}, "hevc": {}, "proper": {}, "repack": {}, "rip": {},
		"web": {}, "webrip": {}, "x264": {}, "x265": {}, "yts": {},
	}
)

func ParseFileName(fileName string) ParsedName {
	original := filepath.Base(fileName)
	extension := strings.ToLower(filepath.Ext(original))
	base := strings.TrimSuffix(original, filepath.Ext(original))
	working := normalizeWorkingName(base)

	parsed := ParsedName{
		Original:  original,
		Extension: extension,
		Resolution: canonicalResolution(working),
		ReleaseYear: extractYear(working),
	}

	titleCandidate := working
	for _, pattern := range episodePatterns {
		match := pattern.re.FindStringSubmatchIndex(working)
		if match == nil {
			continue
		}

		if pattern.seasonGroup > 0 {
			parsed.SeasonNumber = atoiSubmatch(working, match, pattern.seasonGroup)
		} else {
			parsed.SeasonNumber = pattern.defaultSeason
		}
		parsed.EpisodeNumber = atoiSubmatch(working, match, pattern.episodeGroup)
		parsed.IsEpisode = parsed.EpisodeNumber > 0
		if parsed.SeasonNumber == 0 && parsed.IsEpisode {
			parsed.SeasonNumber = 1
		}
		titleCandidate = strings.TrimSpace(working[:match[0]])
		break
	}

	parsed.Title = cleanTitle(titleCandidate)
	if parsed.Title == "" {
		parsed.Title = cleanTitle(working)
	}

	return parsed
}

func normalizeWorkingName(input string) string {
	input = normalizeDigits(input)
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '.', '_', '-', '[', ']', '(', ')', '{', '}', '+', '|':
			return ' '
		default:
			if unicode.IsControl(r) {
				return ' '
			}
			return r
		}
	}, input)
	return strings.Join(strings.Fields(replaced), " ")
}

func cleanTitle(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if match := yearRE.FindStringSubmatchIndex(input); match != nil && match[0] > 0 {
		beforeYear := strings.TrimSpace(input[:match[0]])
		if containsLetter(beforeYear) {
			input = beforeYear
		}
	}

	tokens := strings.Fields(input)
	cleaned := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.Trim(token, " ._-[](){}")
		if token == "" {
			continue
		}
		lower := strings.ToLower(token)
		if _, ok := noiseTokens[lower]; ok {
			continue
		}
		if resolutionRE.MatchString(token) {
			continue
		}
		if yearRE.MatchString(token) && len(cleaned) > 0 {
			continue
		}
		cleaned = append(cleaned, token)
	}

	return strings.Join(cleaned, " ")
}

func canonicalResolution(input string) string {
	match := resolutionRE.FindStringSubmatch(input)
	if len(match) < 2 {
		return ""
	}
	value := strings.ToLower(match[1])
	switch value {
	case "4k", "uhd":
		return "4K"
	case "8k", "4320p":
		if value == "8k" {
			return "8K"
		}
		return "4320p"
	case "fhd":
		return "1080p"
	case "hd":
		return "720p"
	default:
		return value
	}
}

func extractYear(input string) int {
	match := yearRE.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0
	}
	year, _ := strconv.Atoi(match[1])
	return year
}

func atoiSubmatch(input string, match []int, group int) int {
	start := group * 2
	if start+1 >= len(match) || match[start] < 0 || match[start+1] < 0 {
		return 0
	}
	value, err := strconv.Atoi(input[match[start]:match[start+1]])
	if err != nil {
		return 0
	}
	return value
}

func containsLetter(input string) bool {
	for _, r := range input {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func normalizeDigits(input string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		default:
			return r
		}
	}, input)
}
