package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ParsedName holds all metadata extracted from a media file name and path.
type ParsedName struct {
	Original      string `json:"original"`
	Title         string `json:"title"`
	TitleAR       string `json:"titleAr,omitempty"`
	TitleEN       string `json:"titleEn,omitempty"`
	SeasonNumber  int    `json:"seasonNumber,omitempty"`
	EpisodeNumber int    `json:"episodeNumber,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	ReleaseYear   int    `json:"releaseYear,omitempty"`
	Extension     string `json:"extension,omitempty"`
	IsEpisode     bool   `json:"isEpisode"`
}

type episodePattern struct {
	re            *regexp.Regexp
	seasonGroup   int
	episodeGroup  int
	defaultSeason int
}

var (
	bracketTagRE = regexp.MustCompile(`\[[^\]]*\]|\([^\)]*(?:1080p|720p|4k|bluray|hevc|x264|x265)[^\)]*\)`)
	resolutionRE = regexp.MustCompile(`(?i)(?:^|\s)(4320p|2160p|1080p|1080i|720p|576p|480p|360p|8k|4k|uhd|fhd|hd)(?:\s|$)`)
	yearRE       = regexp.MustCompile(`(?:^|\s)((?:19|20)\d{2})(?:\s|$)`)

	// seasonFolderRE matches folder names like "Season 01", "Season 1", "الموسم 01",
	// "الموسم الأول", "الحلقات 1-50", "الحلقات ٥١-١٠٠"
	seasonFolderRE = regexp.MustCompile(`(?i)^(?:season|s)\s*(\d{1,3})$`)
	seasonFolderArabicNumRE = regexp.MustCompile(`^(?:الموسم|موسم)\s*(\d{1,3})$`)
	seasonFolderArabicWordRE = regexp.MustCompile(`^(?:الموسم|موسم)\s*(الأول|الاول|الأولى|الاولى|الثاني|الثانية|الثالث|الثالثة|الرابع|الرابعة|الخامس|الخامسة|السادس|السادسة|السابع|السابعة|الثامن|الثامنة|التاسع|التاسعة|العاشر|العاشرة)$`)
	episodeRangeFolderRE = regexp.MustCompile(`^(?:الحلقات|حلقات|Episodes?)\s*\d+`)

	// categoryFolderKeywords maps folder name substrings to category slugs.
	categoryFolderKeywords = []struct {
		keywords []string
		slug     string
	}{
		{[]string{"anime", "أنمي", "انمي"}, "anime"},
		{[]string{"cartoon", "كرتون", "رسوم", "kids", "أطفال", "اطفال"}, "kids"},
		{[]string{"documentary", "document", "وثائقي", "وثائقية", "وثائقيات"}, "documentaries"},
		{[]string{"play", "مسرح", "مسرحيات", "مسرحية"}, "plays"},
		{[]string{"series", "مسلسل", "مسلسلات"}, "series"},
		{[]string{"movie", "movies", "أفلام", "افلام", "فيلم", "cinema"}, "movies"},
	}

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
			re:           regexp.MustCompile(`(?:الموسم|موسم)\s*(الأول|الاول|الثاني|الثالث|الرابع|الخامس|السادس|السابع|الثامن|التاسع|العاشر)\s*(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`),
			seasonGroup:  1,
			episodeGroup: 2,
		},
		{
			re:            regexp.MustCompile(`(?i)(?:^|\s)(?:episode|ep|e)\s*(\d{1,4})(?:\s|$)`),
			episodeGroup:  1,
			defaultSeason: 1,
		},
		{
			re:            regexp.MustCompile(`(?:^|\s)(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`),
			episodeGroup:  1,
			defaultSeason: 1,
		},
		{
			re:            regexp.MustCompile(`(?:^|\s)-\s*(\d{1,4})(?:\s|$)`),
			episodeGroup:  1,
			defaultSeason: 1,
		},
	}

	arabicWordToNum = map[string]int{
		"الأول": 1, "الاول": 1, "الأولى": 1, "الاولى": 1,
		"الثاني": 2, "الثانية": 2,
		"الثالث": 3, "الثالثة": 3,
		"الرابع": 4, "الرابعة": 4,
		"الخامس": 5, "الخامسة": 5,
		"السادس": 6, "السادسة": 6,
		"السابع": 7, "السابعة": 7,
		"الثامن": 8, "الثامنة": 8,
		"التاسع": 9, "التاسعة": 9,
		"العاشر": 10, "العاشرة": 10,
	}

	noiseTokens = map[string]struct{}{
		"aac": {}, "ac3": {}, "bdrip": {}, "bluray": {}, "brrip": {}, "cam": {},
		"ddp": {}, "dl": {}, "dual": {}, "dvdrip": {}, "dvd": {}, "h264": {}, "h265": {},
		"hdcam": {}, "hdtv": {}, "hevc": {}, "proper": {}, "repack": {}, "rip": {},
		"web": {}, "webrip": {}, "webdl": {}, "web-dl": {}, "x264": {}, "x265": {}, "yts": {},
		"hdr": {}, "hdr10": {}, "dv": {}, "atmos": {}, "remux": {}, "dts": {}, "dts-hd": {},
		"10bit": {}, "extended": {}, "unrated": {}, "flac": {}, "sub": {}, "dub": {},
		"مترجم": {}, "مدبلج": {}, "كامل": {}, "نسخة": {}, "جودة": {}, "عالية": {},
	}
)

// ParseFileName extracts metadata from a filename only (no folder context).
func ParseFileName(fileName string) ParsedName {
	original := filepath.Base(fileName)
	extension := strings.ToLower(filepath.Ext(original))
	base := strings.TrimSuffix(original, filepath.Ext(original))

	normalizedBase := normalizeWorkingName(base)
	resolution := canonicalResolution(normalizedBase)
	year := extractYear(normalizedBase)

	cleanedBrackets := bracketTagRE.ReplaceAllString(base, " ")
	working := normalizeWorkingName(cleanedBrackets)

	parsed := ParsedName{
		Original:    original,
		Extension:   extension,
		Resolution:  resolution,
		ReleaseYear: year,
	}

	titleCandidate := working
	for _, pattern := range episodePatterns {
		match := pattern.re.FindStringSubmatchIndex(working)
		if match == nil {
			continue
		}

		if pattern.seasonGroup > 0 {
			rawSeason := working[match[pattern.seasonGroup*2]:match[pattern.seasonGroup*2+1]]
			if num, ok := arabicWordToNum[rawSeason]; ok {
				parsed.SeasonNumber = num
			} else {
				parsed.SeasonNumber = atoiSubmatch(working, match, pattern.seasonGroup)
			}
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

	parsed.TitleAR, parsed.TitleEN = splitDualTitle(parsed.Title)

	return parsed
}

// ParseFilePath extracts metadata from a full file path, using parent folder
// names to enrich the title, season number, and category classification.
// This is the primary parser for the ingest pipeline.
//
// Folder structure examples it handles:
//   Anime - أنمي/Attack on Titan - هجوم العمالقة/Season 01/file.mkv
//   مسلسلات/صراع العروش/الموسم الأول/file.mkv
//   Movies/Inception (2010)/Inception.2010.1080p.mkv
//   كرتون/Tom and Jerry - توم وجيري/Season 01/file.mp4
func ParseFilePath(fullPath string) ParsedName {
	// Start with filename-only parse
	parsed := ParseFileName(filepath.Base(fullPath))

	// Walk up the directory tree to extract context from parent folders.
	// We examine up to 4 ancestors (file -> season? -> title? -> category? -> root?)
	dir := filepath.Dir(fullPath)
	ancestors := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		folder := filepath.Base(dir)
		if folder == "." || folder == string(filepath.Separator) || folder == dir {
			break
		}
		ancestors = append(ancestors, folder)
		dir = filepath.Dir(dir)
	}

	// Try to extract season from immediate parent folder
	// e.g., "Season 01", "الموسم الأول", "الحلقات 1-50"
	if len(ancestors) > 0 {
		folderSeason := parseSeasonFromFolder(ancestors[0])
		if folderSeason > 0 && parsed.SeasonNumber <= 0 {
			parsed.SeasonNumber = folderSeason
		}
		// If the parent folder is a season folder and we have episode info,
		// the grandparent is likely the show title
		if folderSeason > 0 && len(ancestors) > 1 {
			enrichTitleFromFolder(&parsed, ancestors[1])
		} else if folderSeason == 0 {
			// Parent folder might be the show/movie title itself
			enrichTitleFromFolder(&parsed, ancestors[0])
		}
	}

	// If we still don't have season info but have episode info, default to season 1
	if parsed.IsEpisode && parsed.SeasonNumber == 0 {
		parsed.SeasonNumber = 1
	}

	return parsed
}

// DetectCategoryFromPath determines the content category by examining the full path.
func DetectCategoryFromPath(fullPath string) string {
	lowerPath := strings.ToLower(filepath.ToSlash(fullPath))
	for _, entry := range categoryFolderKeywords {
		for _, keyword := range entry.keywords {
			lowerKeyword := strings.ToLower(keyword)
			if strings.Contains(lowerPath, lowerKeyword) || strings.Contains(fullPath, keyword) {
				return entry.slug
			}
		}
	}
	return ""
}

// DetectOriginTagsFromPath extracts a production-origin hint from the library
// folder structure. Folder names are intentionally preferred over filenames:
// a filename can say "Arabic subtitles" while the enclosing directory
// "مسلسلات تركية" is the owner's deliberate classification.
func DetectOriginTagsFromPath(fullPath string) []string {
	segments := strings.Split(strings.ToLower(filepath.ToSlash(filepath.Dir(fullPath))), "/")
	meaningfulSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		// A "Arabic subtitles" directory describes a sidecar asset, not the
		// production country of the programme stored above it.
		if strings.Contains(segment, "subtitle") || strings.Contains(segment, "ترجم") {
			continue
		}
		meaningfulSegments = append(meaningfulSegments, segment)
	}
	path := strings.Join(meaningfulSegments, "/")
	containsAny := func(terms ...string) bool {
		for _, term := range terms {
			if strings.Contains(path, strings.ToLower(term)) {
				return true
			}
		}
		return false
	}

	switch {
	case containsAny("تركي", "تركية", "تركيا", "turkish", "turkey"):
		return []string{"تركي"}
	case containsAny("كوري", "كورية", "كوريا", "korean", "korea"):
		return []string{"كوري"}
	case containsAny("عربي", "عربية", "عرب", "خليجي", "مصري", "مصرية", "سوري", "سورية", "لبناني", "سعودي", "كويتي", "arabic", "egyptian", "gulf"):
		return []string{"عربي"}
	case containsAny("هندي", "هندية", "هند", "indian", "india", "bollywood"):
		return []string{"هندي"}
	case containsAny("إسباني", "اسباني", "إسبانية", "اسبانية", "spanish", "spain"):
		return []string{"إسباني"}
	case containsAny("ياباني", "يابانية", "japanese", "japan"):
		return []string{"ياباني"}
	case containsAny("أجنبي", "اجنبي", "أجنبية", "اجنبية", "هوليوود", "أمريكي", "امريكي", "بريطاني", "english", "american", "british", "hollywood"):
		return []string{"أجنبي"}
	default:
		return nil
	}
}

// parseSeasonFromFolder extracts a season number from a folder name.
// Returns 0 if the folder is not a season folder.
func parseSeasonFromFolder(folder string) int {
	normalized := normalizeDigits(strings.TrimSpace(folder))

	// English: "Season 01", "Season 1", "S01", "s1"
	if match := seasonFolderRE.FindStringSubmatch(normalized); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return n
		}
	}

	// Arabic with digits: "الموسم 01", "موسم 3"
	if match := seasonFolderArabicNumRE.FindStringSubmatch(normalized); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return n
		}
	}

	// Arabic with words: "الموسم الأول", "الموسم الثاني"
	if match := seasonFolderArabicWordRE.FindStringSubmatch(normalized); match != nil {
		if n, ok := arabicWordToNum[match[1]]; ok {
			return n
		}
	}

	// Episode range folder implies season 1 (e.g., "الحلقات 1-50", "Episodes 1-24")
	if episodeRangeFolderRE.MatchString(normalized) {
		return 1
	}

	return 0
}

// enrichTitleFromFolder upgrades the parsed title using a parent folder name
// when the folder name is richer than the filename-derived title.
func enrichTitleFromFolder(parsed *ParsedName, folder string) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return
	}

	// Don't use category-level folders as titles
	if DetectCategoryFromPath(folder) != "" {
		return
	}

	// Don't use season folders as titles
	if parseSeasonFromFolder(folder) > 0 {
		return
	}

	cleaned := cleanTitle(normalizeWorkingName(folder))
	if cleaned == "" {
		return
	}

	folderAR, folderEN := splitDualTitle(folder)
	if folderAR == "" && folderEN == "" {
		folderAR, folderEN = splitDualTitle(cleaned)
	}

	// Use folder title if it's richer than the filename title
	if folderAR != "" && folderEN != "" && folderAR != folderEN {
		parsed.Title = folderEN + " - " + folderAR
		parsed.TitleAR = folderAR
		parsed.TitleEN = folderEN
	} else if len(cleaned) > len(parsed.Title) || (folderAR != "" && parsed.TitleAR == "") {
		parsed.Title = cleaned
		parsed.TitleAR = folderAR
		parsed.TitleEN = folderEN
	}

	// Always prefer folder Arabic title if filename lacks one
	if parsed.TitleAR == "" && folderAR != "" {
		parsed.TitleAR = folderAR
	}
	if parsed.TitleEN == "" && folderEN != "" {
		parsed.TitleEN = folderEN
	}
}

// =========================================================================
// Utility functions
// =========================================================================

func normalizeWorkingName(input string) string {
	input = normalizeDigits(input)
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '.', '_', '[', ']', '(', ')', '{', '}', '+', '|', '–', '—':
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

func splitDualTitle(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, " - ") {
		parts := strings.SplitN(raw, " - ", 2)
		if len(parts) == 2 {
			p1 := strings.TrimSpace(parts[0])
			p2 := strings.TrimSpace(parts[1])
			if containsArabic(p1) && !containsArabic(p2) {
				return p1, p2
			} else if !containsArabic(p1) && containsArabic(p2) {
				return p2, p1
			}
		}
	}
	if containsArabic(raw) && containsEnglish(raw) {
		words := strings.Fields(raw)
		var arWords, enWords []string
		for _, w := range words {
			cleanW := strings.Trim(w, "-–—")
			if cleanW == "" {
				continue
			}
			if containsArabic(cleanW) {
				arWords = append(arWords, cleanW)
			} else if containsLetter(cleanW) {
				enWords = append(enWords, cleanW)
			}
		}
		if len(arWords) > 0 && len(enWords) > 0 {
			return strings.Join(arWords, " "), strings.Join(enWords, " ")
		}
	}
	if containsArabic(raw) {
		return raw, raw
	}
	return "", raw
}

func containsEnglish(input string) bool {
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func canonicalResolution(input string) string {
	match := resolutionRE.FindStringSubmatch(input)
	if len(match) < 2 {
		return ""
	}
	value := strings.ToLower(match[1])
	switch value {
	case "4k", "uhd", "2160p":
		return "4K"
	case "8k", "4320p":
		return "8K"
	case "fhd", "1080p", "1080i":
		return "1080p"
	case "hd", "720p":
		return "720p"
	case "480p", "576p", "360p":
		return value
	default:
		return strings.ToUpper(value)
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

func containsArabic(input string) bool {
	for _, r := range input {
		if unicode.In(r, unicode.Arabic) {
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
