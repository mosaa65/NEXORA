package scanner

import (
	"regexp"
)

// ParseConfidence expresses how much the parser trusts its own output. The
// parser never invents metadata: when the evidence is thin it reports LOW and
// records the reason, so an admin screen can list what needs review instead of
// silently storing a wrong title.
type ParseConfidence float64

const (
	// ConfidenceLow means the parse is a guess; treat the fields as advisory.
	ConfidenceLow ParseConfidence = 0.35
	// ConfidenceMedium means the title is credible but some fields are inferred.
	ConfidenceMedium ParseConfidence = 0.65
	// ConfidenceHigh means folder and filename evidence agree.
	ConfidenceHigh ParseConfidence = 0.9
)

// Band renders the confidence as the HIGH/MEDIUM/LOW label the report uses.
func (c ParseConfidence) Band() string {
	switch {
	case c >= 0.8:
		return "HIGH"
	case c >= 0.5:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// NumberSource records where a season/episode number came from. This is the
// mechanism that stops "Toy Story 2" from becoming episode 2: a number only
// counts as an episode when the *evidence* says so.
type NumberSource string

const (
	SourceNone            NumberSource = ""
	SourceFilenamePattern NumberSource = "filename_pattern" // S01E01, 1x01, الحلقة 1
	SourceFolderPattern   NumberSource = "folder_pattern"   // "Season 1", "الموسم الأول"
	SourceTrailingNumber  NumberSource = "trailing_number"  // bare "01" - weakest
	SourceKeyword         NumberSource = "keyword"          // OVA/SP/Movie specials
)

// Evidence is the collected proof behind a parse decision. It is what turns the
// parser from "first regex wins" into a weighted judgement.
type Evidence struct {
	FolderTitle      bool
	FilenameTitle    bool
	SeasonFolder     bool
	EpisodePattern   bool
	ExplicitEpisode  bool
	CategorySegment  bool
	TrailingNumber   bool
	NoisePenalty     int
	NumericOnlyTitle bool
}

// ParsedName holds all metadata extracted from a media file name and path.
// Every field is additive: the original text is never destroyed, and the
// normalized/display/AR/EN variants are kept side by side for search.
type ParsedName struct {
	Original        string `json:"original"`
	Title           string `json:"title"`
	TitleAR         string `json:"titleAr,omitempty"`
	TitleEN         string `json:"titleEn,omitempty"`
	TitleNormalized string `json:"titleNormalized,omitempty"`
	DisplayTitle    string `json:"displayTitle,omitempty"`
	SeasonNumber    int    `json:"seasonNumber,omitempty"`
	EpisodeNumber   int    `json:"episodeNumber,omitempty"`
	EpisodeEnd      int    `json:"episodeEnd,omitempty"`
	PartNumber      int    `json:"partNumber,omitempty"`
	// OrderingIndex is a franchise sequence marker such as the "01" in
	// "01 - Batman Begins". It is kept rather than discarded so the ordering is
	// still available for display and for detecting a franchise collection; it is
	// never part of the title and never an episode number.
	OrderingIndex string `json:"orderingIndex,omitempty"`
	Resolution    string `json:"resolution,omitempty"`
	ReleaseYear   int    `json:"releaseYear,omitempty"`
	Extension     string `json:"extension,omitempty"`
	IsEpisode     bool   `json:"isEpisode"`
	// SpecialKind marks non-numbered entries such as OVA, Special, SP, NCOP.
	SpecialKind string `json:"specialKind,omitempty"`
	// CategorySlug is the classification decided from path segments.
	CategorySlug string `json:"categorySlug,omitempty"`
	// OriginTag is the production-origin hint derived from folder segments.
	OriginTag string `json:"originTag,omitempty"`

	Confidence     ParseConfidence `json:"confidence"`
	ConfidenceBand string          `json:"confidenceBand,omitempty"`
	SeasonSource   NumberSource    `json:"seasonSource,omitempty"`
	EpisodeSource  NumberSource    `json:"episodeSource,omitempty"`
	PartSource     NumberSource    `json:"partSource,omitempty"`
	// Reasons lists why confidence was reduced, in owner-readable English.
	Reasons []string `json:"reasons,omitempty"`
}

// -----------------------------------------------------------------------------
// Patterns
// -----------------------------------------------------------------------------

type episodePattern struct {
	re            *regexp.Regexp
	seasonGroup   int
	episodeGroup  int
	endGroup      int
	defaultSeason int
	source        NumberSource
	// strong marks patterns that unambiguously mean "episode", which is what
	// lets a number override an otherwise movie-like path.
	strong bool
}

type partPattern struct {
	re    *regexp.Regexp
	group int
}

var (
	bracketTagRE = regexp.MustCompile(`\[[^\]]*\]|\([^\)]*(?:1080p|720p|4k|bluray|hevc|x264|x265)[^\)]*\)`)
	resolutionRE = regexp.MustCompile(`(?i)(?:^|\s)(4320p|2160p|1080p|1080i|720p|576p|480p|360p|8k|4k|uhd|fhd|hd)(?:\s|$)`)
	yearRE       = regexp.MustCompile(`(?:^|\s)((?:19|20)\d{2})(?:\s|$)`)

	// Release group tags: "[SubGroup] Title", "[SubGroup] Title [tags]", or a
	// trailing "-GROUP" suffix.
	//
	// The suffix form is tightly constrained because a greedy version destroys
	// real titles. It must be the LAST token, must look like a release brand
	// (letters and digits only, no spaces), and must not be a plausible word that
	// follows a separator in an ordered franchise name:
	//
	//	"...01 - Batman Begins.2005.1080p"  the tail after "-" is not a brand
	//	"...Movie-REPACK"                   a brand, correctly removed
	releaseGroupRE = regexp.MustCompile(`(?i)^\s*\[[^\]]+\]\s*|(?:^|\s)\[[^\]]*(?:sub|raw|group|team|fansub)[^\]]*\]`)

	// Part / disc markers. Kept separate from seasons and episodes on purpose.
	partPatterns = []partPattern{
		{regexp.MustCompile(`(?i)(?:^|[\s._\-])(?:part|pt)\s*[._\-]?\s*(\d{1,2})(?:[\s._\-]|$)`), 1},
		{regexp.MustCompile(`(?i)(?:^|[\s._\-])(?:cd|disc|disk)\s*[._\-]?\s*(\d{1,2})(?:[\s._\-]|$)`), 1},
		{regexp.MustCompile(`(?:^|[\s._\-])(?:الجزء|المقطع|القسم)\s*(?:رقم\s*)?(\d{1,2})(?:[\s._\-]|$)`), 1},
		{regexp.MustCompile(`(?:^|[\s._\-])(?:الجزء|المقطع|القسم)\s*(الأول|الاول|الأولى|الاولى|الثاني|الثانية|الثالث|الثالثة|الرابع|الرابعة|الخامس|الخامسة|السادس|السادسة|السابع|السابعة|الثامن|الثامنة|التاسع|التاسعة|العاشر|العاشرة)(?:[\s._\-]|$)`), 1},
	}

	// numericFilmMarker recognises a filename that is a numeric FILM release name
	// rather than an episode index: a resolution marker AND a release year beside
	// a number. It is the signature that separates "1917.2019.1080p" (a film)
	// from "01.mkv" (an episode marker).
	numericFilmMarker = regexp.MustCompile(`(?i)(?:2160p|1080p|1080i|720p|480p|4k|uhd|bluray|bdrip|webrip|web-dl|webdl|hdtv|remux|x26[45]|h26[45]|hevc)`)

	// Anime/season-agnostic special markers.
	specialRE = regexp.MustCompile(`(?i)(?:^|[\s._\-(\[])(ova|ona|special|specials|sp|ncop|nced|op|ed)(?:[\s._\-\])\d]|$)`)

	// Episode range forms: S01E01-E02, S01E01E02, 01-02.
	rangeSxxExxRE = regexp.MustCompile(`(?i)(?:^|\s)s\s*(\d{1,2})\s*e\s*(\d{1,4})\s*[-~]\s*e?\s*(\d{1,4})(?:\s|$)`)
	rangeBareRE   = regexp.MustCompile(`(?:^|\s)(\d{1,4})\s*[-~]\s*(\d{1,4})(?:\s|$)`)

	categorySegments = map[string]string{
		"anime": "anime", "أنمي": "anime", "انمي": "anime", "انيمي": "anime",
		"cartoon": "kids", "cartoons": "kids", "كرتون": "kids", "رسوم": "kids",
		"kids": "kids", "أطفال": "kids", "اطفال": "kids", "اطفال وكرتون": "kids",
		"documentary": "documentaries", "documentaries": "documentaries",
		"وثائقي": "documentaries", "وثائقية": "documentaries", "وثائقيات": "documentaries",
		"docu": "documentaries", "documentarys": "documentaries",
		"play": "plays", "plays": "plays", "مسرح": "plays", "مسرحيات": "plays", "مسرحية": "plays",
		"series": "series", "serie": "series", "مسلسل": "series", "مسلسلات": "series",
		"movie": "movies", "movies": "movies", "أفلام": "movies", "افلام": "movies",
		"فيلم": "movies", "cinema": "movies", "أفلام أجنبية": "movies", "افلام اجنبية": "movies",
	}
)

// noiseTokens are release/quality/watermark tokens that must never appear in a
// title. The list is deliberately generous: false removals are recoverable, a
// polluted title is not.
var noiseTokens = map[string]struct{}{
	"aac": {}, "ac3": {}, "bdrip": {}, "bluray": {}, "brrip": {}, "cam": {},
	"ddp": {}, "dl": {}, "dual": {}, "dvdrip": {}, "dvd": {}, "h264": {}, "h265": {},
	"hdcam": {}, "hdtv": {}, "hevc": {}, "proper": {}, "repack": {}, "rip": {},
	"web": {}, "webrip": {}, "webdl": {}, "web-dl": {}, "x264": {}, "x265": {}, "yts": {},
	"hdr": {}, "hdr10": {}, "dv": {}, "atmos": {}, "remux": {}, "dts": {}, "dts-hd": {},
	"10bit": {}, "extended": {}, "unrated": {}, "flac": {}, "sub": {}, "dub": {},
	"xvid": {}, "divx": {}, "hdrip": {},
	"amzn": {}, "nf": {}, "dsnp": {}, "atvp": {}, "hmax": {}, "pcok": {},
	"multi": {}, "vostfr": {}, "truefrench": {}, "eng": {}, "ara": {},
	"مترجم": {}, "مدبلج": {}, "كامل": {}, "نسخة": {}, "جودة": {}, "عالية": {},
	"الانطلاقه": {}, "الانطلاقة": {}, "انطلاقه": {}, "انطلاقة": {}, "نت": {},
	"المترجم": {}, "اكوام": {}, "akoam": {}, "ماي": {}, "سيما": {}, "mycima": {},
	"عرب": {}, "سيد": {}, "arabseed": {}, "فاصل": {}, "اعلاني": {}, "إعلاني": {},
	"faselhd": {}, "fasel": {}, "ايجي": {}, "بست": {}, "egybest": {}, "شاهد": {},
	"فور": {}, "يو": {}, "shahid4u": {}, "shahid": {}, "سينما": {}, "للجميع": {},
	"cimalek": {}, "arabp2p": {}, "dardarkom": {}, "دار": {}, "داركم": {},
	"topcinema": {}, "توب": {}, "elcinema": {}, "cima4u": {}, "myegy": {},
	"عربليونز": {}, "arablionz": {}, "wecima": {}, "اكسترا": {}, "extra": {},
}

// arabicWordToNum maps Arabic ordinal words to numbers for season parsing.
var arabicWordToNum = map[string]int{
	"الأول": 1, "الاول": 1, "الأولى": 1, "الاولى": 1, "الاولاني": 1,
	"الثاني": 2, "الثانية": 2, "تاني": 2,
	"الثالث": 3, "الثالثة": 3, "تالت": 3,
	"الرابع": 4, "الرابعة": 4, "رابع": 4,
	"الخامس": 5, "الخامسة": 5, "خامس": 5,
	"السادس": 6, "السادسة": 6, "سادس": 6,
	"السابع": 7, "السابعة": 7, "سابع": 7,
	"الثامن": 8, "الثامنة": 8, "تامن": 8,
	"التاسع": 9, "التاسعة": 9, "تاسع": 9,
	"العاشر": 10, "العاشرة": 10, "عاشر": 10,
	"الحادي عشر": 11, "الحادية عشر": 11, "الحادية عشرة": 11,
	"الثاني عشر": 12, "الثانية عشر": 12, "الثانية عشرة": 12,
	"الثالث عشر": 13, "الثالثة عشر": 13, "الثالثة عشرة": 13,
	"الرابع عشر": 14, "الرابعة عشر": 14, "الرابعة عشرة": 14,
	"الخامس عشر": 15, "الخامسة عشر": 15, "الخامسة عشرة": 15,
	"السادس عشر": 16, "السادسة عشر": 16, "السادسة عشرة": 16,
	"السابع عشر": 17, "السابعة عشر": 17, "السابعة عشرة": 17,
	"الثامن عشر": 18, "الثامنة عشر": 18, "الثامنة عشرة": 18,
	"التاسع عشر": 19, "التاسعة عشر": 19, "التاسعة عشرة": 19,
	"العشرون": 20, "العشرين": 20,
}

var (
	seasonFolderRE            = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:season|s)\s*(\d{1,3})$`)
	seasonFolderArabicNumRE   = regexp.MustCompile(`(?:^|[\s._-])(?:الموسم|موسم|الجزء|جزء)\s*(?:رقم\s*)?(\d{1,3})$`)
	seasonFolderArabicWordRE  = regexp.MustCompile(`(?:^|[\s._-])(?:الموسم|موسم|الجزء|جزء)\s*(الأول|الاول|الأولى|الاولى|الاولاني|الثاني|الثانية|تاني|الثالث|الثالثة|تالت|الرابع|الرابعة|رابع|الخامس|الخامسة|خامس|السادس|السادسة|سادس|السابع|السابعة|سابع|الثامن|الثامنة|تامن|التاسع|التاسعة|تاسع|العاشر|العاشرة|عاشر|الحادي\s*عشر|الحادية\s*عشرة?|الثاني\s*عشر|الثانية\s*عشرة?|الثالث\s*عشر|الثالثة\s*عشرة?|الرابع\s*عشر|الرابعة\s*عشرة?|الخامس\s*عشر|الخامسة\s*عشرة?|السادس\s*عشر|السادسة\s*عشرة?|السابع\s*عشر|السابعة\s*عشرة?|الثامن\s*عشر|الثامنة\s*عشرة?|التاسع\s*عشر|التاسعة\s*عشرة?|العشرون|العشرين)$`)
	seasonFolderArabicWordsRE = regexp.MustCompile(`(?:^|[\s._-])(?:الموسم|موسم)\s*(\S+)$`)
	episodeRangeFolderRE      = regexp.MustCompile(`^(?:الحلقات|حلقات|Episodes?|حلقات)\s*\d+`)

	// seasonWordPrefixRE locates where a season token starts inside a folder
	// name that combines a show title with a season, e.g. "Silo الموسم الثالث".
	seasonWordPrefixRE = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:season\s*\d|s\s*\d{1,3}$|الموسم|موسم|الجزء|جزء)`)
)

// episodePatterns are evaluated in order, but a match no longer decides the
// outcome on its own: strong patterns set IsEpisode directly, weak ones only
// supply a candidate number that later evidence must justify.
var episodePatterns = []episodePattern{
	{re: regexp.MustCompile(`(?i)(?:^|\s)s\s*(\d{1,2})\s*e\s*(\d{1,4})(?:\s|$)`), seasonGroup: 1, episodeGroup: 2, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?i)(?:^|\s)(\d{1,2})x(\d{1,4})(?:\s|$)`), seasonGroup: 1, episodeGroup: 2, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?i)(?:season|s)\s*(\d{1,2})\s*(?:episode|ep|e)\s*(\d{1,4})(?:\s|$)`), seasonGroup: 1, episodeGroup: 2, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?:الموسم|موسم)\s*(\d{1,2})\s*(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`), seasonGroup: 1, episodeGroup: 2, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?:الموسم|موسم)\s*(الأول|الاول|الثاني|الث|الرابع|الخامس|السادس|السابع|الثامن|التاسع|العاشر)\s*(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`), seasonGroup: 1, episodeGroup: 2, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?i)(?:^|\s)(?:episode|ep|e)\s*(\d{1,4})(?:\s|$)`), episodeGroup: 1, defaultSeason: 1, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?:^|\s)(?:الحلقة|حلقة|ح)\s*(\d{1,4})(?:\s|$)`), episodeGroup: 1, defaultSeason: 1, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?i)(?:^|\s)ep\.?\s*(\d{1,4})(?:\s|$)`), episodeGroup: 1, defaultSeason: 1, source: SourceFilenamePattern, strong: true},
	// "Title - 042" is the standard anime release form, so a dash-number after a
	// real title is strong episode evidence, not a weak trailing number. A bare
	// trailing number stays weak and is rejected without episodic context.
	{re: regexp.MustCompile(`(?:^|\s)-\s*(\d{1,4})(?:\s|$)`), episodeGroup: 1, defaultSeason: 1, source: SourceFilenamePattern, strong: true},
	{re: regexp.MustCompile(`(?:\s|^)(\d{1,4})$`), episodeGroup: 1, defaultSeason: 1, source: SourceTrailingNumber},
}
