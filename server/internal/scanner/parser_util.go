package scanner

import (
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// -----------------------------------------------------------------------------
// Classification, origin and normalization utilities.
//
// The parsing engine lives in parser_engine.go; the pattern tables live in
// parser_patterns.go. This file holds the helpers they share.
// -----------------------------------------------------------------------------

// DetectCategoryFromSegments classifies content from normalized path segments
// rather than a substring of the whole path. This removes the classic false
// positive where "/NotMovies/" matches the "movies" keyword.
//
// Segments are expected outermost-first (e.g. ["media","series","turkish"]).
// The deepest matching segment wins, because "Movies/Series/..." is a series.
func DetectCategoryFromSegments(segments []string) string {
	best := ""
	bestIndex := -1
	for index, segment := range segments {
		normalized := normalizeSegment(segment)
		if normalized == "" {
			continue
		}
		if slug, exists := categorySegments[normalized]; exists {
			if index > bestIndex {
				best = slug
				bestIndex = index
			}
			continue
		}
		// Multi-word folder names ("مسلسلات تركية", "tv series") contain a
		// category keyword as a whole word; match on tokens, not substrings.
		for token := range tokensOf(normalized) {
			if slug, exists := categorySegments[token]; exists && index > bestIndex {
				best = slug
				bestIndex = index
			}
		}
	}
	return best
}

// DetectCategoryFromPath keeps the original public helper working by deriving
// segments from the path and delegating to the segment classifier.
func DetectCategoryFromPath(fullPath string) string {
	return DetectCategoryFromSegments(pathSegmentsFromPath(fullPath))
}

// pathSegmentsFromPath splits a full path into normalized segments, dropping the
// filename itself so only folders influence classification.
func pathSegmentsFromPath(fullPath string) []string {
	dir := filepath.Dir(fullPath)
	raw := strings.Split(filepath.ToSlash(dir), "/")
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." || segment == "/" {
			continue
		}
		// Skip a Windows drive root such as "D:".
		if len(segment) == 2 && segment[1] == ':' {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

// normalizeSegment lowercases, strips Arabic diacritics, folds separators and
// trims decorative symbols so "مسلسلات_تركية" and a decorated "⬛وثائقي" classify
// the same as their plain forms.
func normalizeSegment(segment string) string {
	segment = normalizeDigits(segment)
	segment = strings.ToLower(segment)
	segment = stripArabicDiacritics(segment)
	segment = strings.Map(func(r rune) rune {
		switch r {
		case '_', '.', '-', '\u2013', '\u2014', ',', '\u060C', '/', '\\':
			return ' '
		default:
			return r
		}
	}, segment)
	segment = strings.Join(strings.Fields(segment), " ")
	segment = strings.TrimFunc(segment, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' '
	})
	return strings.TrimSpace(segment)
}

// tokensOf returns the token set of an already normalized segment.
func tokensOf(normalized string) map[string]struct{} {
	fields := strings.Fields(normalized)
	out := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		out[field] = struct{}{}
	}
	return out
}

// originSegmentKeywords maps folder keywords to a production-origin hint. The
// keys are normalized tokens, so lookup is a map hit rather than a substring
// scan over the whole path.
var originSegmentKeywords = []struct {
	tokens []string
	tag    string
}{
	{[]string{"تركي", "تركية", "تركيا", "turkish", "turkey"}, "تركي"},
	{[]string{"كوري", "كورية", "كوريا", "korean", "korea"}, "كوري"},
	{[]string{"هندي", "هندية", "هند", "indian", "india", "bollywood"}, "هندي"},
	{[]string{"إسباني", "اسباني", "إسبانية", "اسبانية", "spanish", "spain"}, "إسباني"},
	{[]string{"ياباني", "يابانية", "japanese", "japan"}, "ياباني"},
	{[]string{"صيني", "صينية", "chinese", "china"}, "صيني"},
	{[]string{"خليجي", "خليجية", "gulf"}, "خليجي"},
	{[]string{"مصري", "مصرية", "egyptian", "egypt"}, "مصري"},
	{[]string{"سوري", "سورية", "syrian"}, "سوري"},
	{[]string{"لبناني", "لبنانية", "lebanese"}, "لبناني"},
	{[]string{"سعودي", "سعودية", "saudi"}, "سعودي"},
	{[]string{"كويتي", "كويتية", "kuwaiti"}, "كويتي"},
	{[]string{"عربي", "عربية", "عرب", "arabic", "arab"}, "عربي"},
	{[]string{"أجنبي", "اجنبي", "أجنبية", "اجنبية", "هوليوود", "hollywood",
		"أمريكي", "امريكي", "بريطاني", "english", "american", "british"}, "أجنبي"},
}

// DetectOriginTagsFromFolders inspects folder segments and returns a single
// origin hint. Folder names are deliberately preferred over filenames, because
// a filename mentioning "Arabic" usually describes subtitles, not production.
func DetectOriginTagsFromFolders(segments []string) string {
	tag := ""
	bestIndex := -1
	for index, segment := range segments {
		normalized := normalizeSegment(segment)
		if normalized == "" {
			continue
		}
		// Subtitle folders describe language, not origin.
		if strings.Contains(normalized, "subtitle") || strings.Contains(normalized, "ترجم") {
			continue
		}
		tokens := tokensOf(normalized)
		for _, entry := range originSegmentKeywords {
			matched := false
			for _, candidate := range entry.tokens {
				if _, ok := tokens[candidate]; ok {
					matched = true
					break
				}
			}
			if matched && index > bestIndex {
				tag = entry.tag
				bestIndex = index
			}
		}
	}
	return tag
}

// DetectOriginTagsFromPath keeps the original slice-returning API working.
func DetectOriginTagsFromPath(fullPath string) []string {
	tag := DetectOriginTagsFromFolders(pathSegmentsFromPath(fullPath))
	if tag == "" {
		return nil
	}
	return []string{tag}
}

// parseSeasonFromFolder extracts a season number from a folder name.
// Returns 0 when the folder is not a season folder.
func parseSeasonFromFolder(folder string) int {
	normalized := normalizeDigits(strings.TrimSpace(folder))

	if match := seasonFolderRE.FindStringSubmatch(normalized); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return n
		}
	}

	if match := seasonFolderArabicNumRE.FindStringSubmatch(normalized); match != nil {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return n
		}
	}

	if match := seasonFolderArabicWordRE.FindStringSubmatch(normalized); match != nil {
		word := strings.TrimSpace(match[1])
		if n, ok := arabicWordToNum[word]; ok {
			return n
		}
	}

	if match := seasonFolderArabicWordsRE.FindStringSubmatch(normalized); match != nil {
		word := strings.TrimSpace(match[1])
		if n, ok := arabicWordToNum[word]; ok {
			return n
		}
	}

	// An episode-range folder implies season 1: "الحلقات 1-50", "Episodes 1-24".
	if episodeRangeFolderRE.MatchString(normalized) {
		return 1
	}

	return 0
}

// -----------------------------------------------------------------------------
// Normalization
// -----------------------------------------------------------------------------

// normalizeWorkingName turns a filename into spaced tokens: separators become
// spaces and Arabic/Persian digits become ASCII so every pattern matches one
// representation. The original string is always preserved by the caller.
func normalizeWorkingName(input string) string {
	input = normalizeDigits(input)
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '.', '_', '[', ']', '(', ')', '{', '}', '+', '|', '\u2013', '\u2014':
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

// normalizeArabicNoise collapses the formatting quirks that make otherwise
// identical Arabic titles compare unequal: diacritics, tatweel and repeated
// whitespace. It never removes letters or changes hamza forms, so the displayed
// name keeps its original spelling.
func normalizeArabicNoise(input string) string {
	input = stripArabicDiacritics(input)
	input = strings.ReplaceAll(input, "\u0640", "") // tatweel
	return strings.Join(strings.Fields(input), " ")
}

// stripArabicDiacritics removes harakat, Quranic marks and tatweel.
func stripArabicDiacritics(input string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 0x064B && r <= 0x065F: // harakat
			return -1
		case r == 0x0670: // superscript alef
			return -1
		case r >= 0x06D6 && r <= 0x06ED: // Quranic marks
			return -1
		case r == 0x0640: // tatweel
			return -1
		default:
			return r
		}
	}, input)
}

// foldArabicAlef normalises alef variants and hamza forms for *search only*.
// The original spelling is kept in Title/TitleAR; this feeds search tokens.
func foldArabicAlef(input string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ':
			return 'ا'
		case 'ى':
			return 'ي'
		case 'ة':
			return 'ه'
		case 'ؤ':
			return 'و'
		case 'ئ':
			return 'ي'
		default:
			return r
		}
	}, input)
}

// NormalizeTitleForSearch produces the canonical key used to match
// "ون بيس", "One Piece", "one_piece" and "One.Piece" to the same logical title.
// It is search data only; it never replaces the stored display title.
func NormalizeTitleForSearch(title string) string {
	normalized := normalizeWorkingName(title)
	normalized = stripArabicDiacritics(normalized)
	normalized = foldArabicAlef(normalized)
	normalized = strings.ToLower(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	return strings.TrimSpace(normalized)
}

// SearchTokens splits a title into the tokens a search engine should index,
// combining the original spelling with the folded one so either matches.
func SearchTokens(title string) []string {
	normalized := NormalizeTitleForSearch(title)
	if normalized == "" {
		return nil
	}
	fields := strings.Fields(normalized)
	seen := make(map[string]struct{}, len(fields)*2)
	tokens := make([]string, 0, len(fields)*2)
	add := func(token string) {
		if token == "" {
			return
		}
		if _, exists := seen[token]; exists {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	for _, word := range fields {
		add(word)
	}
	// A single concatenated form lets "ون بيس" and "ونبيس" both resolve.
	add(strings.Join(fields, ""))
	// A multi-word phrase supports phrase-level matching.
	if len(fields) > 1 {
		add(strings.Join(fields, " "))
	}
	return tokens
}

// normalizeDigits converts Arabic-Indic and Persian digits to ASCII.
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

// -----------------------------------------------------------------------------
// Title cleanup
// -----------------------------------------------------------------------------

// cleanTitle strips release noise, resolutions and years from a title candidate.
//
// It normalizes separators first because its callers may hand it either an
// already-normalized string or a raw filename fragment, and a dotted release
// name ("Batman Begins.2005.1080p") must clean identically to its spaced form.
func cleanTitle(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	input = normalizeWorkingName(input)

	// Truncate at a release year, but ONLY when real title text precedes it.
	//
	// containsLetter alone is not enough: for a numeric film title such as
	// "1917.2019.1080p" the before-year text is "1917" with no letters, so the
	// play is to keep the whole thing and let the token filter drop the year.
	// Requiring a letter keeps "Batman Begins.2005" truncating correctly while
	// "1917" and "300" survive as titles.
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
		cleaned = append(cleaned, token)
	}

	// Drop a release year, but never the title itself.
	//
	// "Batman Begins 2005 1080p" -> "Batman Begins" (2005 is a release year).
	// "1917 2019 1080p"        -> "1917"          (the first number IS the title,
	//                                             the second is the release year).
	//
	// A year token is kept only when nothing precedes it, which is exactly the
	// A numeric token is dropped when a NON-YEAR token already precedes it, which
	// is what distinguishes a release year from a numeric film title.
	filtered := make([]string, 0, len(cleaned))
	titleSeen := false
	for _, token := range cleaned {
		if yearRE.MatchString(token) && titleSeen {
			continue
		}
		filtered = append(filtered, token)
		if !yearRE.MatchString(token) {
			titleSeen = true
		}
	}
	return strings.Join(filtered, " ")
}

// isJunkOrWatermarkTitle reports whether a candidate title carries no meaning.
func isJunkOrWatermarkTitle(title string) bool {
	t := strings.TrimSpace(title)
	if t == "" {
		return true
	}
	if _, err := strconv.Atoi(t); err == nil || len(t) <= 1 {
		return true
	}
	words := strings.Fields(strings.ToLower(t))
	meaningful := 0
	for _, w := range words {
		w = strings.Trim(w, " ._-[](){}")
		if _, isNoise := noiseTokens[w]; isNoise {
			continue
		}
		if _, err := strconv.Atoi(w); err == nil {
			continue
		}
		meaningful++
	}
	return meaningful == 0
}

// isNumericOnly reports whether a title is nothing but digits, which is a strong
// signal that the real title lives in a parent folder.
func isNumericOnly(title string) bool {
	t := strings.TrimSpace(title)
	if t == "" {
		return false
	}
	for _, r := range t {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// splitDualTitle separates an "English - Arabic" combined title.
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

// -----------------------------------------------------------------------------
// Primitive predicates
// -----------------------------------------------------------------------------

func containsEnglish(input string) bool {
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
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

// canonicalResolution maps the many quality spellings to one canonical value.
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
