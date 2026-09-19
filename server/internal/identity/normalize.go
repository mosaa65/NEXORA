// Package identity implements Entity Resolution: deciding which logical work,
// season and episode a physical file belongs to.
//
// It exists because a filesystem file is not a media entity. The previous ingest
// path turned every parsed filename straight into a `media_items` row, which
// produced degenerate "works" from real data such as:
//
//	"الكنز ج1 الحلقه"        a season+episode marker became a work title
//	"Fate_Apocrypha"         separators never normalized
//	"الكنزنت"                a site watermark concatenated into the title
//	"منور فديوهات زابيا"      a channel description became a work
//	"Fate Stay Night -"      a trailing dash left behind by the parser
//
// Entity Resolution replaces that with an evidence-weighted search over existing
// entities, and refuses to create a work when the evidence is weak.
package identity

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// -----------------------------------------------------------------------------
// Normalization
// -----------------------------------------------------------------------------

// Normalize produces the canonical comparison key for a title.
//
// This is deliberately more aggressive than the display normalizer: it exists to
// make "Breaking Bad", "Breaking.Bad", "Breaking_Bad", "breaking-bad" and
// "breaking  bad" compare equal, which is the property that stops one show from
// becoming five works.
//
// It never replaces the original text: callers always store both.
func Normalize(title string) string {
	if title == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(title))

	for _, r := range title {
		switch {
		// Digits are checked before letters so Arabic-Indic and Persian numerals
		// fold to ASCII. unicode.IsDigit is true for both, and unicode.IsLetter is
		// false, so this order matters.
		case unicode.IsDigit(r):
			builder.WriteRune(foldRune(r))
		case unicode.IsLetter(r):
			builder.WriteRune(foldRune(r))
		case unicode.IsSpace(r):
			builder.WriteRune(' ')
		default:
			// Punctuation, separators, brackets and symbols all collapse to a
			// single space: they carry no identity information.
			builder.WriteRune(' ')
		}
	}

	collapsed := strings.Join(strings.Fields(builder.String()), " ")
	return strings.TrimSpace(collapsed)
}

// foldRune lowercases and folds the Arabic letter variants that are written
// differently but read identically. Without this "أسامة" and "اسامة" are two
// different works.
func foldRune(r rune) rune {
	switch r {
	case 'أ', 'إ', 'آ', 'ٱ', 'ٲ', 'ٳ':
		return 'ا'
	case 'ى':
		return 'ي'
	case 'ئ':
		return 'ي'
	case 'ؤ':
		return 'و'
	case 'ة':
		return 'ه'
	case 'ک':
		return 'ك'
	case 'ی':
		return 'ي'
	}
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	// Arabic-Indic and Persian digits fold to ASCII so "الحلقة ١" and "الحلقة 1"
	// produce the same key.
	switch {
	case r >= '٠' && r <= '٩':
		return '0' + (r - '٠')
	case r >= '۰' && r <= '۹':
		return '0' + (r - '۰')
	}
	return unicode.ToLower(r)
}

// NormalizeCompact removes all spaces, producing the key used to match
// "one piece" with "onepiece" and "OnePiece".
func NormalizeCompact(title string) string {
	return strings.ReplaceAll(Normalize(title), " ", "")
}

// Tokens returns the meaningful word set of a title, used for set-similarity
// scoring and for the search projection.
func Tokens(title string) []string {
	normalized := Normalize(title)
	if normalized == "" {
		return nil
	}
	seen := make(map[string]struct{}, 8)
	out := make([]string, 0, 8)
	for _, token := range strings.Fields(normalized) {
		if len(token) <= 1 && !isDigit(token) {
			// Single letters are noise in either script.
			continue
		}
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func isDigit(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(value) > 0
}

// -----------------------------------------------------------------------------
// Similarity
// -----------------------------------------------------------------------------

// Similarity returns a 0..1 measure of how likely two titles describe the same
// work. It combines several cheap signals because no single one is reliable on
// real library names:
//
//   - exact normalized equality is definitive
//   - one containing the other handles "Silo" vs "Silo Arabic"
//   - token overlap handles word reordering and extra qualifiers
//   - edit distance handles typos and short abbreviations
//
// The maximum is taken rather than an average, because one strong signal should
// not be diluted by the absence of the others.
func Similarity(a, b string) float64 {
	normA, normB := Normalize(a), Normalize(b)
	if normA == "" || normB == "" {
		return 0
	}
	if normA == normB {
		return 1.0
	}

	best := 0.0

	// 1. Compact equality: "one piece" vs "onepiece".
	if NormalizeCompact(a) == NormalizeCompact(b) {
		return 0.97
	}

	// 2. Containment on whole-word boundaries. Substring containment is not
	//    enough: "silo" is a substring of "silosomethingelse" but not the same
	//    work, so the shorter string must align to word boundaries.
	if containsWords(normA, normB) || containsWords(normB, normA) {
		longer, shorter := normA, normB
		if len([]rune(normA)) < len([]rune(normB)) {
			longer, shorter = normB, normA
		}
		// Reward containment proportionally to how much of the longer title the
		// shorter one covers, so "silo" vs "silo arabic" scores high and
		// "silo" vs "silo the complete collection boxset" scores lower.
		ratio := float64(len([]rune(shorter))) / float64(len([]rune(longer)))
		best = maxFloat(best, 0.75+0.2*ratio)
	}

	// 3. Token set similarity (Jaccard) over meaningful tokens.
	tokensA, tokensB := Tokens(a), Tokens(b)
	if len(tokensA) > 0 && len(tokensB) > 0 {
		best = maxFloat(best, 0.55*jaccard(tokensA, tokensB))
	}

	// 4. Edit distance for typos and truncation.
	best = maxFloat(best, 0.6*jaroWinkler(normA, normB))

	return best
}

// containsWordSequence reports whether the whole word sequence of `needle`
// appears in `haystack`, so containment is word-aligned rather than a raw
// substring test. Both arguments are expected to be already normalized.
func containsWordSequence(haystack, needle string) bool { return containsWords(haystack, needle) }

// containsWords reports whether the whole word sequence of `needle` appears in
// `haystack`, so containment is word-aligned rather than a raw substring test.
func containsWords(haystack, needle string) bool {
	if needle == "" || len(needle) >= len(haystack) {
		return false
	}
	// Pad both so the match must start and end on a word boundary.
	return strings.Contains(" "+haystack+" ", " "+needle+" ")
}

func jaccard(a, b []string) float64 {
	setA := make(map[string]struct{}, len(a))
	for _, token := range a {
		setA[token] = struct{}{}
	}
	intersection := 0
	for _, token := range b {
		if _, ok := setA[token]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union <= 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// jaroWinkler computes string similarity with a prefix bonus, which suits media
// titles: "interstellar" and "intersteller" differ by one character, while
// "the office" and "the offce" differ by one but share a long prefix.
func jaroWinkler(a, b string) float64 {
	runesA, runesB := []rune(a), []rune(b)
	lenA, lenB := len(runesA), len(runesB)
	if lenA == 0 || lenB == 0 {
		return 0
	}
	if lenA == 1 && lenB == 1 {
		if runesA[0] == runesB[0] {
			return 1
		}
		return 0
	}

	window := maxInt(lenA, lenB)/2 - 1
	if window < 0 {
		window = 0
	}

	matchedA := make([]bool, lenA)
	matchedB := make([]bool, lenB)
	matches := 0

	for i := 0; i < lenA; i++ {
		start := maxInt(0, i-window)
		end := minInt(lenB-1, i+window)
		for j := start; j <= end; j++ {
			if matchedB[j] || runesA[i] != runesB[j] {
				continue
			}
			matchedA[i] = true
			matchedB[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}

	// Transpositions: matched characters that are out of order.
	transpositions := 0
	k := 0
	for i := 0; i < lenA; i++ {
		if !matchedA[i] {
			continue
		}
		for !matchedB[k] {
			k++
		}
		if runesA[i] != runesB[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	jaro := (m/float64(lenA) + m/float64(lenB) + (m-float64(transpositions)/2)/m) / 3

	// Winkler prefix bonus, capped at four characters.
	prefix := 0
	for i := 0; i < minInt(4, minInt(lenA, lenB)); i++ {
		if runesA[i] != runesB[i] {
			break
		}
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}

// -----------------------------------------------------------------------------
// Title quality
// -----------------------------------------------------------------------------

// TitleQuality reports whether a parsed title is usable as a work name.
//
// This is the guard that prevents a watermark, a channel description or a bare
// episode marker from becoming a work. It is intentionally conservative: a
// false negative sends the file to human review, which is recoverable, whereas a
// false positive pollutes the library permanently.
type TitleQuality struct {
	Usable   bool
	Reason   string
	Severity int // 0 = fine, 1 = suspicious, 2 = unusable
}

// AssessTitle judges a candidate title against the failure modes seen in real
// libraries.
func AssessTitle(title, filename string) TitleQuality {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return TitleQuality{Usable: false, Reason: "empty title", Severity: 2}
	}

	// A title that is only a number is normally an episode marker, not a work.
	//
	// The exception is a numeric FILM title ("1917", "300", "2012") which is
	// distinguishable by the release evidence beside it in the filename: a
	// release year plus a resolution marker means this is a film release name,
	// whereas a bare "01.mkv" beside nothing carries no such signal.
	if isDigitsOnly(trimmed) {
		if numericFilmTitle(filename) {
			return TitleQuality{Usable: true, Severity: 0}
		}
		return TitleQuality{Usable: false, Reason: "title is only a number", Severity: 2}
	}

	// A single character or a single very short token carries no identity.
	if len([]rune(trimmed)) < 2 {
		return TitleQuality{Usable: false, Reason: "title is too short", Severity: 2}
	}

	// A title must contain at least one letter in some script.
	if !containsLetter(trimmed) {
		return TitleQuality{Usable: false, Reason: "title has no letters", Severity: 2}
	}

	// Trailing separators are a parse artifact ("Fate Stay Night -").
	if strings.HasSuffix(trimmed, "-") || strings.HasSuffix(trimmed, "_") ||
		strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "_") {
		return TitleQuality{Usable: false, Reason: "title has a dangling separator", Severity: 1}
	}

	// A high ratio of digits to letters means the "title" is really a file tag
	// such as "1080p 264" or "01 x2 10".
	letters, digits := 0, 0
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		}
	}
	if digits > 0 && letters > 0 && float64(digits)/float64(letters) > 0.5 {
		return TitleQuality{Usable: false, Reason: "title is mostly numeric tokens", Severity: 1}
	}

	// A title that is a single token identical to a known structural word means
	// the parser consumed the marker instead of the name.
	if isStructuralWord(trimmed) {
		return TitleQuality{Usable: false, Reason: "title is a structural keyword", Severity: 2}
	}

	// Watermark detection compares the title against the whole filename, because
	// site branding appears as a prefix ("akoam_ep05") or a concatenation
	// ("الكنزنت 01") rather than as a standalone word.
	if filename != "" {
		if watermark, found := dominantWatermark(filename); found {
			titleNorm := NormalizeCompact(trimmed)
			// The title is watermark branding when it contains the brand and the
			// brand accounts for essentially all of it. The slack is small on
			// purpose: a real title that merely mentions a brand must survive.
			if watermark != "" && strings.Contains(titleNorm, watermark) {
				slack := len([]rune(titleNorm)) - len([]rune(watermark))
				if slack <= 1 {
					return TitleQuality{Usable: false, Reason: "title is site watermark branding", Severity: 2}
				}
			}
		}
	}

	// A single unbroken token that contains no separator and matches a known
	// brand is branding, not a title ("الكنزنت").
	if len(strings.Fields(trimmed)) == 1 {
		if _, isBrand := watermarkTokens[Normalize(trimmed)]; isBrand {
			return TitleQuality{Usable: false, Reason: "title is a known site brand", Severity: 2}
		}
	}

	// Long runs with no separator often mean a watermark was concatenated onto
	// the real title. Arabic and Latin words average well under 18 characters,
	// so a much longer unbroken run is suspicious.
	for _, token := range strings.Fields(trimmed) {
		runes := []rune(token)
		if len(runes) >= 18 {
			return TitleQuality{Usable: false, Reason: "title contains an implausibly long token", Severity: 1}
		}
	}

	return TitleQuality{Usable: true, Severity: 0}
}

// watermarkTokens are site and channel names seen in real Arabic and English
// media libraries. A title made of these is not a work.
//
// Only tokens that are unambiguously site branding belong here. Ordinary Arabic
// words are deliberately excluded: "الكنز" means "The Treasure" and is a real
// title, so treating it as a watermark would destroy legitimate works. A
// watermark is recognised instead when it dominates the whole filename.
var watermarkTokens = map[string]struct{}{
	"akoam": {}, "mycima": {}, "arabseed": {}, "faselhd": {}, "egybest": {},
	"shahid4u": {}, "cima4u": {}, "myegy": {}, "wecima": {}, "arablionz": {},
	"cimalek": {}, "arabp2p": {}, "dardarkom": {}, "topcinema": {}, "elcinema": {},
	"اكوام": {}, "اكوام نت": {}, "سيما": {}, "فاصل": {}, "فاصل اعلاني": {},
	"اعلاني": {}, "شاهد": {}, "زابيا": {}, "الكنزنت": {},
	"مترجم": {}, "مدبلج": {},
}

// structuralWords are words that describe structure rather than a name. Keys are
// stored in normalized form (Arabic letter variants folded, no diacritics), so
// the map lookup matches what Normalize produces.
var structuralWords = map[string]struct{}{
	"season": {}, "seasons": {}, "episode": {}, "episodes": {}, "ep": {},
	"movie": {}, "movies": {}, "series": {}, "part": {}, "disc": {}, "cd": {},
	"ova": {}, "ona": {}, "special": {}, "specials": {}, "sp": {},
	"ncop": {}, "nced": {},
	// Arabic, already normalized: ة->ه, أ->ا.
	"الموسم": {}, "موسم": {}, "الحلقه": {}, "حلقه": {}, "الحلقات": {}, "حلقات": {},
	"الجزء": {}, "جزء": {}, "المقطع": {}, "القسم": {},
	"مسلسل": {}, "مسلسلات": {}, "فيلم": {}, "افلام": {}, "مسرحيه": {},
	"كامل": {}, "ج1": {}, "ج2": {}, "ج3": {},
}

// isStructuralWord reports whether a title is only a structural keyword.
// The stored keys are in normalized form, so the lookup must normalize too:
// "الحلقة" folds to "الحلقه" and would never match the raw key otherwise.
func isStructuralWord(value string) bool {
	normalized := Normalize(value)
	if normalized == "" {
		return false
	}
	if _, exists := structuralWords[normalized]; exists {
		return true
	}
	// A title made only of structural words is still structural, e.g.
	// "الموسم الحلقة".
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return false
	}
	for _, field := range fields {
		if _, exists := structuralWords[field]; !exists {
			return false
		}
	}
	return true
}

// dominantWatermark reports the strongest watermark token present in a filename,
// if any. Landscape HD and similar are included because they are release brands
// that must never become a title.
func dominantWatermark(filename string) (string, bool) {
	if filename == "" {
		return "", false
	}
	normalized := NormalizeCompact(filename)
	best := ""
	for token := range watermarkTokens {
		folded := NormalizeCompact(token)
		if folded == "" || !strings.Contains(normalized, folded) {
			continue
		}
		if len([]rune(folded)) > len([]rune(best)) {
			best = folded
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

// releaseMarkerRE matches a resolution or quality marker, which is the signature
// of a film release name rather than an episode index.
var releaseMarkerRE = regexp.MustCompile(`(?i)(?:^|\s)(?:4320p|2160p|1080p|1080i|720p|576p|480p|360p|8k|4k|uhd|fhd|bluray|bdrip|webrip|web-dl|webdl|hdtv|remux|x26[45]|h26[45]|hevc)(?:\s|$)`)

// releaseYearRE matches a four-digit year in a release name.
var releaseYearRE = regexp.MustCompile(`(?:^|\s)((?:19|20)\d{2})(?:\s|$)`)

// numericFilmTitle reports whether a numeric-only title is a film name rather
// than an episode marker.
//
// The discriminator is release evidence in the same filename: a resolution
// marker beside the number means the name describes a film release
// ("1917.2019.1080p"), and a bare number with no such context is an episode
// marker ("01").
//
// This is deliberately self-contained: the identity package must not depend on
// the scanner's pattern tables to judge a title's quality.
func numericFilmTitle(filename string) bool {
	if filename == "" {
		return false
	}
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	normalized := normalizeForPatternMatch(base)

	// A resolution marker and a release year together are the signature of a
	// film release name; a bare "01" carries neither.
	if !releaseMarkerRE.MatchString(normalized) {
		return false
	}
	if !releaseYearRE.MatchString(normalized) {
		return false
	}

	// The numeric token must itself be film-like. Four digits is a year-scale
	// title ("1917"); three digits is plausible as a film number ("300") but
	// also as an episode index, so it is only accepted when a separate numeric
	// token follows it.
	fields := strings.Fields(normalized)
	for index, token := range fields {
		if !isDigitsOnly(token) {
			continue
		}
		switch {
		case len(token) >= 4:
			return true
		case len(token) == 3 && index+1 < len(fields) && isDigitsOnly(fields[index+1]):
			return true
		}
	}
	return false
}

// normalizeForPatternMatch converts separators to spaces so the release patterns
// can match a dotted or underscored filename.
func normalizeForPatternMatch(raw string) string {
	mapped := strings.Map(func(r rune) rune {
		switch r {
		case '.', '_', '-', '[', ']', '(', ')':
			return ' '
		default:
			return r
		}
	}, normalizeForDigits(raw))
	return strings.Join(strings.Fields(mapped), " ")
}

// normalizeForDigits folds Arabic-Indic and Persian digits to ASCII.
func normalizeForDigits(raw string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		default:
			return r
		}
	}, raw)
}

func isDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsLetter(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// maxInt/minInt/maxFloat exist because the generic builtins and comparable
// helpers are not available for both int and float64 in one name without
// shadowing the language builtins, which is a correctness hazard.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
