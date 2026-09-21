package scanner

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ParseFileName extracts metadata from a filename alone, without folder context.
//
// The strategy is evidence-driven rather than "first regex wins":
//
//  1. strip release-group noise and bracket tags
//  2. collect a season/episode candidate from the strongest matching pattern
//  3. collect a part/disc candidate, which is never confused with an episode
//  4. derive a title from the text preceding the matched marker
//  5. score the result so weak parses are flagged instead of trusted
func ParseFileName(fileName string) ParsedName {
	original := filepath.Base(fileName)
	extension := strings.ToLower(filepath.Ext(original))
	base := strings.TrimSuffix(original, filepath.Ext(original))

	return parseName(base, original, extension, nil, nil)
}

// ParseFilePath extracts metadata from a full path, using parent folder names to
// enrich the title, season number, and classification. This is the primary
// entry point for the ingest pipeline.
func ParseFilePath(fullPath string) ParsedName {
	original := filepath.Base(fullPath)
	extension := strings.ToLower(filepath.Ext(original))
	base := strings.TrimSuffix(original, filepath.Ext(original))

	ancestors := pathSegments(filepath.Dir(fullPath))
	return parseName(base, original, extension, ancestors, nil)
}

// ParseWithSegments parses a file using the explicit list of path segments from
// the media root. Passing segments avoids re-deriving the hierarchy from a
// string and lets classification work on real segments rather than substrings.
func ParseWithSegments(fullPath string, rootSegments []string) ParsedName {
	parsed := ParseFilePath(fullPath)
	if len(rootSegments) == 0 {
		return parsed
	}
	if slug := DetectCategoryFromSegments(rootSegments); slug != "" {
		parsed.CategorySlug = slug
	}
	if origin := DetectOriginTagsFromFolders(rootSegments); origin != "" {
		parsed.OriginTag = origin
	}
	return parsed
}

// pathSegments returns up to five ancestor folder names from nearest to root,
// which is the context window the parser uses. The list is bounded so a
// pathological path cannot make parsing expensive.
func pathSegments(dir string) []string {
	segments := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		folder := filepath.Base(dir)
		if folder == "." || folder == string(filepath.Separator) || folder == dir || folder == "" {
			break
		}
		segments = append(segments, folder)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return segments
}

// parseName is the shared implementation for both filename and path parsing.
// `ancestors` are folder names from nearest to farthest; `rootSegments`
// (optional) are the path segments from the media root used for classification.
func parseName(base, original, extension string, ancestors []string, rootSegments []string) ParsedName {
	normalizedBase := normalizeWorkingName(base)

	parsed := ParsedName{
		Original:    original,
		Extension:   extension,
		Resolution:  canonicalResolution(normalizedBase),
		ReleaseYear: extractYear(normalizedBase),
	}

	// Strip release-group prefixes/suffixes before any title work. Doing this
	// first prevents "[SubGroup]" from ever reaching the title.
	stripped := releaseGroupRE.ReplaceAllString(base, " ")
	cleanedBrackets := bracketTagRE.ReplaceAllString(stripped, " ")
	// A franchise ordering prefix ("01 - Batman Begins") is detected BEFORE
	// normalization, because normalization collapses the separator run into a
	// single space, after which the ordering marker is indistinguishable from a
	// title word. Detecting it here also means "3 Idiots", "365 Days" and
	// "21 Jump Street" keep their leading number: a bare space is a name
	// boundary, and only an explicit separator run marks a franchise index.
	orderingPrefix, orderedName := splitOrderingPrefix(cleanedBrackets)

	working := normalizeWorkingName(cleanedBrackets)
	working = normalizeArabicNoise(working)

	evidence := Evidence{}

	// Part / disc detection runs before episodes, so "CD1" or "Part 2" is never
	// reported as an episode number. The marker itself is then removed from the
	// title, because "The Godfather Part 2" and "The Godfather" are one work.
	if part, source := detectPart(working); part > 0 {
		parsed.PartNumber = part
		parsed.PartSource = source
		working = stripPartMarker(working)
	}

	// A part marker is conviction that this file is a film sequel, not an
	// episode. Without this, "Part 3 - John Wick Chapter 3.mkv" inside a
	// "Franchises" folder let the trailing "3" become an episode number.
	partSeen := parsed.PartNumber > 0
	titleCandidate, matched := detectEpisode(working, &parsed, &evidence)

	// Named specials (OVA/SP/NCOP) when no numeric episode was found.
	if !parsed.IsEpisode {
		if special := detectSpecial(working); special != "" {
			parsed.SpecialKind = special
			parsed.IsEpisode = true
			evidence.ExplicitEpisode = true
			if parsed.SeasonNumber == 0 {
				parsed.SeasonNumber = 1
			}
		}
	}

	// When no episode marker exists, the whole working string is the title.
	if !matched {
		titleCandidate = working
	}

	// A franchise ordering prefix is removed from the title BEFORE folder
	// enrichment, because folder enrichment decides whether to prefer the folder
	// name by comparing lengths. Leaving "01 Batman Begins" in place made the
	// filename look longer than the folder and the stale prefix survived into
	// the final title.
	//
	// The name is normalized first: it was captured on the raw filename, so it
	// still carries the dots that cleanTitle cannot split on.
	if !parsed.IsEpisode {
		if orderingPrefix != "" && orderedName != "" {
			parsed.OrderingIndex = orderingPrefix
			titleCandidate = normalizeWorkingName(orderedName)
		}
	}

	parsed.Title = cleanTitle(titleCandidate)
	evidence.NumericOnlyTitle = isNumericOnly(parsed.Title)

	// The whole string is the fallback only when no episode marker was found at
	// all. When a marker was found at the very start ("Episode 12.mkv"), the
	// "title" would just be the marker text, which is not a title: leave it
	// empty so folder context can supply the real name.
	if matched && strings.TrimSpace(titleCandidate) == "" {
		parsed.Title = ""
	}

	// A title made entirely of noise means the filename carried no information.
	//
	// A numeric FILM title ("1917", "300") is the exception: it is a real title
	// whose release evidence proves it, so it is not treated as an episode marker.
	// A numeric film name needs BOTH a resolution marker and a release year in
	// the filename: "1917.2019.1080p" qualifies, a bare "01" does not.
	numericFilm := isNumericOnly(parsed.Title) &&
		numericFilmMarker.MatchString(normalizedBase) &&
		yearRE.MatchString(normalizedBase)
	if parsed.Title == "" || (isJunkOrWatermarkTitle(parsed.Title) && !numericFilm) {
		cleanedFull := cleanTitle(working)
		if !isJunkOrWatermarkTitle(cleanedFull) && !matched {
			parsed.Title = cleanedFull
		} else if numericFilm {
			// Keep the numeric film title rather than discarding it.
			titleCandidate = working
			parsed.Title = cleanTitle(titleCandidate)
		} else if parsed.Title == "" && !matched {
			// Do not invent a title. Leave it empty and record why.
			parsed.Title = ""
			parsed.Reasons = append(parsed.Reasons, "filename carries no usable title token")
		}
	}

	// Folder context enrichment is the highest-value evidence in a real library.
	if len(ancestors) > 0 {
		enrichFromFolders(&parsed, ancestors, &evidence)
	}

	// A path that ends in digits is weak evidence for an episode. Only accept it
	// when the surrounding context supports episodic content, which is the rule
	// that stops "Toy Story 2" and "Movie Part 2" from becoming episodes.
	if parsed.EpisodeSource == SourceTrailingNumber &&
		(partSeen ||
			!episodeContextSupportsTrailingNumber(parsed, ancestors) ||
			releaseYearVetoesEpisode(parsed, ancestors)) {
		parsed.EpisodeNumber = 0
		parsed.EpisodeEnd = 0
		parsed.EpisodeSource = SourceNone
		parsed.IsEpisode = false
		parsed.SeasonNumber = 0
		parsed.SeasonSource = SourceNone
		parsed.Reasons = append(parsed.Reasons, "trailing number treated as part of the title, not an episode")

		// The rejected number was treated as the episode marker, so everything
		// before it was taken as the title. Now that the number is part of the
		// name, the full working string is the title candidate again, otherwise a
		// release year silently truncates the name. This is the fix for
		// "01 - Batman Begins.2005.mkv" being titled "01 Batman Begins".
		if !parsed.IsEpisode {
			titleCandidate = working
		}
	}

	// A leading ordering number can also be rejected as an episode only now, so
	// the fallback strip runs on the final title. The pre-normalization prefix
	// (handled above) remains the primary mechanism because it sees the original
	// separators, where "01 - Batman Begins" is unambiguous and "3 Idiots" keeps
	// its number.
	if !parsed.IsEpisode && parsed.OrderingIndex == "" {
		parsed.Title = stripLeadingOrderNumber(parsed.Title)
	}

	// Category and origin are decided from path segments, never substrings.
	classificationSegments := segmentsForClassification(ancestors, rootSegments)
	if slug := DetectCategoryFromSegments(classificationSegments); slug != "" {
		parsed.CategorySlug = slug
		evidence.CategorySegment = true
	}
	if origin := DetectOriginTagsFromFolders(classificationSegments); origin != "" {
		parsed.OriginTag = origin
	}

	// The ordering-prefix strip runs last, on the FINAL title, so it sees the same
	// text the user will: "01 Batman Begins" -> "Batman Begins". It never runs for
	// an episode, and never touches a title that is only a number.
	if !parsed.IsEpisode {
		parsed.Title = stripLeadingOrderNumber(parsed.Title)
	}

	parsed.TitleAR, parsed.TitleEN = splitDualTitle(parsed.Title)
	parsed.TitleNormalized = NormalizeTitleForSearch(parsed.Title)
	parsed.DisplayTitle = buildDisplayTitle(parsed)

	parsed.Confidence, parsed.Reasons = scoreConfidence(parsed, evidence)
	parsed.ConfidenceBand = parsed.Confidence.Band()

	return parsed
}

// segmentsForClassification chooses which segments classification inspects.
func segmentsForClassification(ancestors, rootSegments []string) []string {
	if len(rootSegments) > 0 {
		return rootSegments
	}
	// Reverse so the outermost folders are inspected first, matching how an
	// owner organises "Media/Series/..." hierarchies.
	out := make([]string, len(ancestors))
	for i := range ancestors {
		out[i] = ancestors[len(ancestors)-1-i]
	}
	return out
}

// splitOrderingPrefix separates a franchise ordering index from the film name.
//
// It exists so the decision is made on the ORIGINAL separators, before
// normalization turns them into spaces. The distinction it encodes is the whole
// correctness of the rule:
//
//	"01 - Batman Begins"  -> index "01",     name "Batman Begins"
//	"04 - John Wick ..."  -> index "04",     name "John Wick ..."
//	"3 Idiots"            -> index "",       name unchanged  (the 3 is the title)
//	"21 Jump Street"      -> index "",       name unchanged
//	"365 Days"            -> index "",       name unchanged
//	"1917"                -> index "",       name unchanged
//
// A bare space is a word boundary inside a film name. Only an explicit
// separator character marks a franchise index. Getting this wrong produced
// learned aliases named "Idiots" and "Jump Street".
func splitOrderingPrefix(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", raw
	}

	index := 0
	for index < len(trimmed) && trimmed[index] >= '0' && trimmed[index] <= '9' {
		index++
	}
	if index == 0 || index >= len(trimmed) {
		return "", raw
	}

	// Count the separator characters that immediately follow the digits.
	separators := 0
	for index+separators < len(trimmed) {
		switch trimmed[index+separators] {
		case ' ', '-', '_', '.', ':', ')', ']':
			separators++
			continue
		}
		break
	}
	if separators == 0 {
		// Digits run straight into letters: "1917", "Se7en". Not an index.
		return "", raw
	}

	// The run must contain a real separator; a bare space is a name boundary.
	separatorRun := trimmed[index : index+separators]
	if !strings.ContainsAny(separatorRun, "-_.:)]") {
		return "", raw
	}

	tail := strings.TrimSpace(trimmed[index+separators:])
	if tail == "" {
		return "", raw
	}
	// The remainder must look like a title, not another marker or a quality tag.
	if isNumericOnly(tail) || resolutionRE.MatchString(tail) || isJunkOrWatermarkTitle(tail) {
		return "", raw
	}

	return trimmed[:index], tail
}

// stripLeadingOrderNumber removes a franchise ordering prefix such as
// "01 - Batman Begins" -> "Batman Begins".
//
// It only strips when a real title follows, so "01" on its own stays intact for
// the group and folder logic to interpret.
func stripLeadingOrderNumber(candidate string) string {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return candidate
	}

	// Locate the end of the leading numeric token.
	index := 0
	for index < len(trimmed) && trimmed[index] >= '0' && trimmed[index] <= '9' {
		index++
	}
	if index == 0 || index >= len(trimmed) {
		return candidate
	}

	// The number must be followed by an EXPLICIT separator run, not merely a
	// space. A space alone means the digits are part of the film's own name:
	//
	//	"3 Idiots"             → "3 Idiots"      (the 3 is the title)
	//	"21 Jump Street"       → "21 Jump Street"
	//	"12 Angry Men"         → "12 Angry Men"
	//	"365 Days"             → "365 Days"
	//	"01 - Batman Begins"   → "Batman Begins" (the 01 orders the franchise)
	//	"04 - John Wick ..."   → "John Wick ..."
	//
	// Stripping on a bare space is what produced aliases named "Idiots" and
	// "Jump Street", which then mis-resolved every later file.
	rest := trimmed[index:]
	if rest == "" {
		return candidate
	}

	// Count the separator characters that immediately follow the digits.
	separators := 0
	for separators < len(rest) {
		switch rest[separators] {
		case ' ', '-', '_', '.', ':', ')':
			separators++
		default:
			goto separated
		}
	}

separated:
	// A separator run that is only whitespace is a name boundary, not an ordering
	// prefix, so the digits stay.
	separatorRun := rest[:separators]
	if !strings.ContainsAny(separatorRun, "-_.:)") {
		return candidate
	}

	tail := strings.TrimSpace(rest[separators:])
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return candidate
	}

	// A remainder that is only a number or a quality tag is not a title.
	if isNumericOnly(tail) || resolutionRE.MatchString(tail) {
		return candidate
	}

	// The remainder must look like a title, not like another marker.
	if isJunkOrWatermarkTitle(tail) {
		return candidate
	}

	return tail
}

// detectEpisode fills SeasonNumber/EpisodeNumber and returns the title prefix.
func detectEpisode(working string, parsed *ParsedName, evidence *Evidence) (string, bool) {
	// Range forms first: they are the most specific, and a single-episode
	// pattern would otherwise consume only part of the range.
	if match := rangeSxxExxRE.FindStringSubmatchIndex(working); match != nil {
		parsed.SeasonNumber = atoiSubmatch(working, match, 1)
		parsed.EpisodeNumber = atoiSubmatch(working, match, 2)
		parsed.EpisodeEnd = atoiSubmatch(working, match, 3)
		parsed.IsEpisode = parsed.EpisodeNumber > 0
		parsed.SeasonSource = SourceFilenamePattern
		parsed.EpisodeSource = SourceFilenamePattern
		evidence.EpisodePattern = true
		evidence.ExplicitEpisode = true
		return strings.TrimSpace(working[:match[0]]), true
	}

	for _, pattern := range episodePatterns {
		match := pattern.re.FindStringSubmatchIndex(working)
		if match == nil {
			continue
		}

		if pattern.seasonGroup > 0 {
			raw := strings.TrimSpace(working[match[pattern.seasonGroup*2]:match[pattern.seasonGroup*2+1]])
			if num, ok := arabicWordToNum[raw]; ok {
				parsed.SeasonNumber = num
			} else {
				parsed.SeasonNumber = atoiSubmatch(working, match, pattern.seasonGroup)
			}
			parsed.SeasonSource = pattern.source
		} else if pattern.defaultSeason > 0 {
			parsed.SeasonNumber = pattern.defaultSeason
			parsed.SeasonSource = SourceFilenamePattern
		}

		parsed.EpisodeNumber = atoiSubmatch(working, match, pattern.episodeGroup)
		parsed.EpisodeSource = pattern.source
		parsed.IsEpisode = parsed.EpisodeNumber > 0
		evidence.EpisodePattern = true
		evidence.ExplicitEpisode = pattern.strong
		if pattern.source == SourceTrailingNumber {
			evidence.TrailingNumber = true
		}
		if parsed.SeasonNumber == 0 && parsed.IsEpisode {
			parsed.SeasonNumber = 1
		}
		return strings.TrimSpace(working[:match[0]]), true
	}
	return "", false
}

// stripPartMarker removes a part/CD/disc marker from a working name so it does
// not end up inside the title. Season and episode markers are untouched.
func stripPartMarker(working string) string {
	for _, pattern := range partPatterns {
		if loc := pattern.re.FindStringIndex(working); loc != nil {
			return strings.Join(strings.Fields(working[:loc[0]]+" "+working[loc[1]:]), " ")
		}
	}
	return working
}

// detectPart finds a part/CD/disc marker and returns its number.
func detectPart(working string) (int, NumberSource) {
	for _, pattern := range partPatterns {
		match := pattern.re.FindStringSubmatch(working)
		if match == nil {
			continue
		}
		raw := strings.TrimSpace(match[pattern.group])
		if num, ok := arabicWordToNum[raw]; ok {
			return num, SourceKeyword
		}
		if num, err := strconv.Atoi(raw); err == nil && num > 0 {
			return num, SourceFilenamePattern
		}
	}
	return 0, SourceNone
}

// detectSpecial recognises named non-episodic entries so anime libraries do not
// lose OVAs, specials, or clean openings.
func detectSpecial(working string) string {
	match := specialRE.FindStringSubmatch(strings.ToLower(working))
	if match == nil {
		return ""
	}
	switch match[1] {
	case "ova":
		return "OVA"
	case "ona":
		return "ONA"
	case "ncop", "op":
		return "NCOP"
	case "nced", "ed":
		return "NCED"
	default:
		return "Special"
	}
}

// episodeContextSupportsTrailingNumber decides whether a bare trailing number is
// an episode. It requires series-like context: a season folder, or an episodic
// category such as anime/series/kids.
func episodeContextSupportsTrailingNumber(parsed ParsedName, ancestors []string) bool {
	for _, folder := range ancestors {
		if parseSeasonFromFolder(folder) > 0 {
			return true
		}
	}
	switch parsed.CategorySlug {
	case "series", "anime", "kids", "documentaries":
		return true
	}
	return false
}

// releaseYearVetoesEpisode refuses to read a leading number as an episode when
// the filename is a film release name.
//
// The shape this targets is very common in ordered franchise folders:
//
//	Franchises/DC/01 - Batman Begins.2005.1080p.mkv
//	Franchises/John Wick/04 - John Wick Chapter 4.2023.1080p.mkv
//
// The leading "01" orders the film inside the franchise; it is not an episode.
// A release year beside a resolution marker is the signature of a film release
// name, and a real episode filename never carries a cinema release year. The
// veto is therefore keyed on the year plus the absence of any explicit season
// structure, so "Show.2020.S01E01.mkv" is still an episode.
func releaseYearVetoesEpisode(parsed ParsedName, ancestors []string) bool {
	if parsed.ReleaseYear == 0 {
		return false
	}
	// An explicit season folder is decisive: the file is episodic.
	if parsed.SeasonSource == SourceFolderPattern {
		return false
	}
	for _, folder := range ancestors {
		if parseSeasonFromFolder(folder) > 0 {
			return false
		}
	}
	// A series/season category makes the release year a coincidence.
	switch parsed.CategorySlug {
	case "series", "anime":
		return false
	}
	return true
}

// enrichFromFolders upgrades the parse using folder names, which are the
// strongest signal in a real library ("Series/Silo/الموسم الثالث/1.mkv").
func enrichFromFolders(parsed *ParsedName, ancestors []string, evidence *Evidence) {
	if len(ancestors) == 0 {
		return
	}

	seasonFolderIndex := -1
	for index, folder := range ancestors {
		if season := parseSeasonFromFolder(folder); season > 0 {
			// A folder explicitly declaring a season is authoritative over a
			// season inferred from the filename.
			parsed.SeasonNumber = season
			parsed.SeasonSource = SourceFolderPattern
			seasonFolderIndex = index
			evidence.SeasonFolder = true
			break
		}
	}

	titleFromFolder := ""
	primary := ancestors[0]

	if seasonFolderIndex == 0 {
		if prefix := titlePrefixBeforeSeason(primary); prefix != "" {
			titleFromFolder = prefix
		}
	}
	if titleFromFolder == "" && seasonFolderIndex >= 0 && len(ancestors) > seasonFolderIndex+1 {
		// Pure season folder ("الموسم الثالث"): the show name sits above it.
		titleFromFolder = ancestors[seasonFolderIndex+1]
	}
	if titleFromFolder == "" {
		titleFromFolder = primary
	}

	// A container folder is never a work title. The nearest ancestor that is a
	// real name wins, and if none exists the filename supplies the title.
	//
	// This is the fix for a real library layout:
	//
	//	.../مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Titanic.1997.mkv
	//
	// "أعمال" means "works" and is a browse grouping. The nearest real name
	// above it is "Leonardo DiCaprio" — an ACTOR, not a work — so using an
	// ancestor would be worse than using nothing. The filename is therefore
	// left as the only candidate, and "Titanic" is recovered from it.
	//
	// The previous behaviour accepted the container as a title whenever no
	// better candidate existed, which is how works literally named "أعمال" and
	// "Media" were created.
	if IsContainerFolderForTitle(titleFromFolder) {
		titleFromFolder = ""
	}
	if detectSpecial(normalizeWorkingName(titleFromFolder)) != "" && len(ancestors) > 1 {
		// A folder that is only a special marker ("OVA", "Specials") is not a
		// title either; the show name lives above it.
		titleFromFolder = ancestors[1]
	}
	// A folder that is only a special marker ("OVA", "Specials") is not a title;
	// the show name lives above it.
	if detectSpecial(normalizeWorkingName(titleFromFolder)) != "" {
		if len(ancestors) > 1 {
			titleFromFolder = ancestors[1]
		}
	}

	if applyFolderTitle(parsed, titleFromFolder) {
		evidence.FolderTitle = true
	}
	if parsed.Title != "" {
		evidence.FilenameTitle = true
	}
}

// containerFolderTitles are folder names that describe a container, a browse
// grouping or a category rather than a work.
//
// This list is deliberately duplicated from identity.containerFolderNames rather
// than imported: the scanner cannot depend on the identity package (identity
// depends on nothing, but the scanner is imported BY the code that wires them),
// and a shared list is the only way the two can agree. Both call sites are
// covered by tests that assert the same inputs classify the same way.
var containerFolderTitles = map[string]struct{}{
	"اعمال": {}, "أعمال": {}, "افلام": {}, "أفلام": {}, "مسلسلات": {}, "مسلسل": {},
	"مكتبه": {}, "مكتبة": {}, "القسم": {}, "قسم": {}, "الكل": {}, "متنوع": {},
	// "أخرى" folds through the alef-maqsura rule to "اخري", so the folded form
	// is stored rather than the written one.
	"افلام ومسلسلات": {}, "اخري": {}, "اخرى": {}, "جديد": {}, "قديم": {},
	"مكتبة حسب الممثلين": {}, "الممثلين": {}, "حسب الممثلين": {},
	// Actor and people folders. An actor's name is a real name, so without
	// these the nearest non-container ancestor would title every film in the
	// folder after the actor.
	"actors": {}, "actor": {}, "actresses": {}, "people": {}, "cast": {},
	"مثلين": {}, "مثلون": {}, "الممثلون": {}, "فنانون": {}, "نجوم": {},
	"franchises": {}, "collection": {}, "collections": {}, "boxset": {}, "box set": {},
	"movies": {}, "films": {}, "series": {}, "tv": {}, "shows": {},
	"library": {}, "media": {}, "video": {}, "videos": {}, "unsorted": {},
	"misc": {}, "other": {}, "others": {}, "extra": {}, "extras": {},
	"featurettes": {}, "bonus": {}, "sample": {}, "samples": {},
}

// IsContainerFolderForTitle reports whether a folder name is a container rather
// than a work title.
//
// Two checks run, because a container appears in two shapes:
//
//  1. the folder IS a container name ("أعمال", "Movies");
//  2. the folder is a container name with a QUALIFIER added ("مسلسلات تركية",
//     "أفلام أجنبية", "مكتبة حسب الممثلين"), which owners write constantly.
//
// Shape 2 is why a plain map lookup is not enough. The first container word in
// the folder decides, so "مسلسلات تركية" is a container because it begins with
// "مسلسلات", while "طائر الرفراف" is not because no word is a container word.
//
// The first word must match rather than any word: a work whose name happens to
// contain a container word (for example "The Library") must not be rejected.
func IsContainerFolderForTitle(folder string) bool {
	normalized := NormalizeTitleForSearch(folder)
	if normalized == "" {
		return false
	}
	// Shape 1: the whole name is a known container.
	if _, exists := containerFolderTitles[normalized]; exists {
		return true
	}
	// Shape 2: the name starts with a container word and the remaining words are
	// not a work title. This is checked by requiring the folder to be longer than
	// the container word and to still begin with it.
	for container := range containerFolderTitles {
		if container == normalized {
			return true
		}
		if strings.HasPrefix(normalized, container+" ") {
			return true
		}
	}
	return false
}

// titlePrefixBeforeSeason extracts the show name from a folder that also names
// a season, e.g. "Silo الموسم الثالث" -> "Silo".
func titlePrefixBeforeSeason(folder string) string {
	normalized := normalizeDigits(strings.TrimSpace(folder))
	loc := seasonWordPrefixRE.FindStringIndex(normalized)
	if loc == nil || loc[0] <= 0 {
		return ""
	}
	prefix := strings.TrimSpace(folder[:loc[0]])
	prefix = cleanTitle(normalizeWorkingName(prefix))
	if prefix == "" {
		return ""
	}
	if DetectCategoryFromSegments([]string{prefix}) != "" {
		return ""
	}
	if parseSeasonFromFolder(prefix) > 0 {
		return ""
	}
	return prefix
}

// applyFolderTitle decides whether the folder name should replace the filename
// title. Folder titles win when the filename title is missing, junk, numeric
// only, or strictly shorter (a common sign of a truncated episode name).
func applyFolderTitle(parsed *ParsedName, folder string) bool {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return false
	}
	if DetectCategoryFromSegments([]string{folder}) != "" {
		return false
	}
	if parseSeasonFromFolder(folder) > 0 {
		return false
	}

	cleaned := cleanTitle(normalizeWorkingName(folder))
	if cleaned == "" {
		return false
	}

	folderAR, folderEN := splitDualTitle(folder)
	if folderAR == "" && folderEN == "" {
		folderAR, folderEN = splitDualTitle(cleaned)
	}

	currentIsWeak := parsed.Title == "" || isJunkOrWatermarkTitle(parsed.Title) || isNumericOnly(parsed.Title)

	switch {
	case folderAR != "" && folderEN != "" && folderAR != folderEN:
		// Bilingual folder: keep both names, which is the best outcome.
		parsed.Title = folderEN + " - " + folderAR
		parsed.TitleAR = folderAR
		parsed.TitleEN = folderEN
		return true
	case currentIsWeak || len(cleaned) > len(parsed.Title):
		parsed.Title = cleaned
		if folderAR != "" {
			parsed.TitleAR = folderAR
		} else if containsArabic(cleaned) {
			parsed.TitleAR = cleaned
		}
		if folderEN != "" {
			parsed.TitleEN = folderEN
		} else if !containsArabic(cleaned) {
			parsed.TitleEN = cleaned
		}
		return true
	}

	if parsed.TitleAR == "" && folderAR != "" {
		parsed.TitleAR = folderAR
	}
	if parsed.TitleEN == "" && folderEN != "" {
		parsed.TitleEN = folderEN
	}
	return false
}

// buildDisplayTitle picks the best human-facing string without discarding the
// other variants, which search and admin screens still need.
func buildDisplayTitle(parsed ParsedName) string {
	if parsed.Title != "" {
		return parsed.Title
	}
	if parsed.TitleEN != "" {
		return parsed.TitleEN
	}
	return parsed.TitleAR
}

// scoreConfidence converts collected evidence into a score plus the reasons it
// was reduced. This is what makes "these files need review" possible.
func scoreConfidence(parsed ParsedName, evidence Evidence) (ParseConfidence, []string) {
	reasons := append([]string{}, parsed.Reasons...)
	score := 0.5

	switch {
	case evidence.FolderTitle && evidence.FilenameTitle:
		score += 0.3
	case evidence.FolderTitle || evidence.FilenameTitle:
		score += 0.15
	default:
		score -= 0.3
		reasons = append(reasons, "no reliable title in folder or filename")
	}

	if evidence.SeasonFolder {
		score += 0.1
	}
	if evidence.ExplicitEpisode {
		score += 0.1
	}
	if evidence.CategorySegment {
		score += 0.05
	}
	if parsed.ReleaseYear > 0 {
		score += 0.05
	}

	// Only penalise a trailing number when it actually became an episode. A
	// trailing number that was correctly rejected as part of the title is a
	// successful decision, not a weakness.
	if evidence.TrailingNumber && !evidence.ExplicitEpisode && parsed.EpisodeNumber > 0 {
		score -= 0.15
		reasons = append(reasons, "episode number inferred from a trailing number")
	}
	// A bare "N" title with no folder context is the weakest possible parse.
	if parsed.Title == "" && !evidence.FolderTitle {
		score -= 0.25
		reasons = append(reasons, "no title evidence in filename or folder")
	}
	if parsed.IsEpisode && parsed.SeasonSource == SourceNone {
		score -= 0.1
		reasons = append(reasons, "season number defaulted to 1")
	}
	if parsed.Title == "" {
		score -= 0.3
	}
	if isNumericOnly(parsed.Title) {
		score -= 0.25
		reasons = append(reasons, "filename contains only a numeric token")
	}

	if score > 0.98 {
		score = 0.98
	}
	if score < 0.05 {
		score = 0.05
	}
	return ParseConfidence(score), dedupeReasons(reasons)
}

func dedupeReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reasons))
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if reason == "" {
			continue
		}
		if _, exists := seen[reason]; exists {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}
