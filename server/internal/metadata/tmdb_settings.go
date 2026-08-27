package metadata

import (
	"encoding/json"
	"time"
)

// ImageMode defines how image assets are served and stored
type ImageMode string

const (
	ImageModeHybrid ImageMode = "hybrid" // Posters & Backdrops downloaded locally; Cast/Gallery/Stills served via remote CDN URL
	ImageModeLocal  ImageMode = "local"  // All enabled images downloaded to local disk (100% offline)
	ImageModeRemote ImageMode = "remote" // Zero local downloads; all images served via TMDB CDN URLs
)

// FetchMode defines pre-configured data ingestion profiles
type FetchMode string

const (
	FetchModeEssential FetchMode = "essential" // Title, Plot, Rating, Genres, Year, Poster, Backdrop, Keywords, Season count (Ultra Fast, Lowest Bandwidth)
	FetchModeStandard  FetchMode = "standard"  // Essential + Cast names (text), Trailers, External IDs, Translations, Content Ratings (Recommended)
	FetchModeFull      FetchMode = "full"      // Standard + Cast images, Gallery backdrops/posters, Recommendations, Similar, Watch Providers, Reviews
	FetchModeCustom    FetchMode = "custom"    // User-customized module toggles
)

// ModuleConfig defines individual toggle switches for every TMDB metadata field
type ModuleConfig struct {
	// Core text fields
	FetchTitle            bool `json:"fetch_title"`
	FetchOverview         bool `json:"fetch_overview"`
	FetchGenres           bool `json:"fetch_genres"`
	FetchKeywords         bool `json:"fetch_keywords"`
	FetchReleaseDates     bool `json:"fetch_release_dates"`
	FetchContentRatings   bool `json:"fetch_content_ratings"`
	FetchExternalIDs      bool `json:"fetch_external_ids"`
	FetchTranslations     bool `json:"fetch_translations"`
	FetchAlternativeTitles bool `json:"fetch_alternative_titles"`
	FetchTrailers         bool `json:"fetch_trailers"`
	FetchCreditsText      bool `json:"fetch_credits_text"`      // Cast & Crew names + roles
	FetchRecommendations  bool `json:"fetch_recommendations"`   // Recommended titles text
	FetchSimilar          bool `json:"fetch_similar"`           // Similar titles text
	FetchReviews          bool `json:"fetch_reviews"`           // User reviews
	FetchWatchProviders   bool `json:"fetch_watch_providers"`   // JustWatch streaming providers

	// Images control
	FetchPoster           bool `json:"fetch_poster"`
	FetchBackdrop         bool `json:"fetch_backdrop"`
	MaxCastImages         int  `json:"max_cast_images"`         // 0 = none, >0 = limit
	MaxGalleryPosters     int  `json:"max_gallery_posters"`     // 0 = none, >0 = limit
	MaxGalleryBackdrops   int  `json:"max_gallery_backdrops"`   // 0 = none, >0 = limit
	MaxGalleryLogos       int  `json:"max_gallery_logos"`       // 0 = none, >0 = limit
	MaxRelatedPosters     int  `json:"max_related_posters"`     // 0 = none, >0 = limit

	// Seasons & TV control
	FetchSeasonOverview   bool `json:"fetch_season_overview"`
	FetchSeasonPosters    bool `json:"fetch_season_posters"`
	FetchEpisodeOverview  bool `json:"fetch_episode_overview"`
	FetchEpisodeStills    bool `json:"fetch_episode_stills"`
}

// TMDBSettings is the complete configurable state of TMDB metadata operations
type TMDBSettings struct {
	FetchMode            FetchMode    `json:"fetch_mode"`
	ImageMode            ImageMode    `json:"image_mode"`
	PreferredLanguage    string       `json:"preferred_language"`     // Default: ar-SA
	FallbackLanguage     string       `json:"fallback_language"`      // Default: en-US
	IncludeImageLanguage string       `json:"include_image_language"` // Default: ar,en,null
	DailyBandwidthMB     int64        `json:"daily_bandwidth_mb"`     // Daily bandwidth quota in MB (e.g. 500), 0 = unlimited
	EnableRateLimitDelay bool         `json:"enable_rate_limit_delay"`
	RateLimitRequestsPerSec int       `json:"rate_limit_requests_per_sec"` // Default 35
	PosterSize           string       `json:"poster_size"`            // w500, w342, original
	BackdropSize         string       `json:"backdrop_size"`          // original, w1280, w780
	ProfileSize          string       `json:"profile_size"`           // w185, h632, original
	StillSize            string       `json:"still_size"`             // w300, original
	AutoRefreshEnabled   bool         `json:"auto_refresh_enabled"`
	RefreshIntervalDays  int          `json:"refresh_interval_days"`
	RefreshOnOpen        bool         `json:"refresh_on_open"`
	RefreshStaleDays     int          `json:"refresh_stale_days"`
	QueueMaxConcurrent   int          `json:"queue_max_concurrent"`
	Modules              ModuleConfig `json:"modules"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// TMDBRemoteConfig represents the payload from TMDB GET /3/configuration
type TMDBRemoteConfig struct {
	Images struct {
		BaseURL       string   `json:"base_url"`
		SecureBaseURL string   `json:"secure_base_url"`
		BackdropSizes []string `json:"backdrop_sizes"`
		LogoSizes     []string `json:"logo_sizes"`
		PosterSizes   []string `json:"poster_sizes"`
		ProfileSizes  []string `json:"profile_sizes"`
		StillSizes    []string `json:"still_sizes"`
	} `json:"images"`
	ChangeKeys []string `json:"change_keys"`
}

// DefaultSettings returns the recommended production defaults (Hybrid Mode)
func DefaultSettings() TMDBSettings {
	return TMDBSettings{
		FetchMode:            FetchModeStandard,
		ImageMode:            ImageModeHybrid,
		PreferredLanguage:    "ar-SA",
		FallbackLanguage:     "en-US",
		IncludeImageLanguage: "ar,en,null",
		DailyBandwidthMB:     500, // 500 MB daily quota protection
		EnableRateLimitDelay: true,
		RateLimitRequestsPerSec: 35,
		PosterSize:           "w500",
		BackdropSize:         "original",
		ProfileSize:          "w185",
		StillSize:            "w300",
		AutoRefreshEnabled:   false,
		RefreshIntervalDays:  30,
		RefreshOnOpen:        false,
		RefreshStaleDays:     7,
		QueueMaxConcurrent:   1,
		Modules: ModuleConfig{
			FetchTitle:            true,
			FetchOverview:         true,
			FetchGenres:           true,
			FetchKeywords:         true,
			FetchReleaseDates:     true,
			FetchContentRatings:   true,
			FetchExternalIDs:      true,
			FetchTranslations:     true,
			FetchAlternativeTitles: true,
			FetchTrailers:         true,
			FetchCreditsText:      true,
			FetchRecommendations:  true,
			FetchSimilar:          true,
			FetchReviews:          false,
			FetchWatchProviders:   false,
			FetchPoster:           true,
			FetchBackdrop:         true,
			MaxCastImages:         0, // Text names only by default in standard
			MaxGalleryPosters:     0,
			MaxGalleryBackdrops:   0,
			MaxGalleryLogos:       0,
			MaxRelatedPosters:     0,
			FetchSeasonOverview:   true,
			FetchSeasonPosters:    false,
			FetchEpisodeOverview:  true,
			FetchEpisodeStills:    false,
		},
		UpdatedAt: time.Now().UTC(),
	}
}

// ApplyProfile overrides the modules with a preset profile
func (s *TMDBSettings) ApplyProfile(profile FetchMode) {
	s.FetchMode = profile
	switch profile {
	case FetchModeEssential:
		s.Modules = ModuleConfig{
			FetchTitle:            true,
			FetchOverview:         true,
			FetchGenres:           true,
			FetchKeywords:         true,
			FetchReleaseDates:     true,
			FetchContentRatings:   true,
			FetchExternalIDs:      false,
			FetchTranslations:     false,
			FetchAlternativeTitles: false,
			FetchTrailers:         false,
			FetchCreditsText:      false,
			FetchRecommendations:  false,
			FetchSimilar:          false,
			FetchReviews:          false,
			FetchWatchProviders:   false,
			FetchPoster:           true,
			FetchBackdrop:         true,
			MaxCastImages:         0,
			MaxGalleryPosters:     0,
			MaxGalleryBackdrops:   0,
			MaxGalleryLogos:       0,
			MaxRelatedPosters:     0,
			FetchSeasonOverview:   false,
			FetchSeasonPosters:    false,
			FetchEpisodeOverview:  false,
			FetchEpisodeStills:    false,
		}
	case FetchModeStandard:
		s.Modules = ModuleConfig{
			FetchTitle:            true,
			FetchOverview:         true,
			FetchGenres:           true,
			FetchKeywords:         true,
			FetchReleaseDates:     true,
			FetchContentRatings:   true,
			FetchExternalIDs:      true,
			FetchTranslations:     true,
			FetchAlternativeTitles: true,
			FetchTrailers:         true,
			FetchCreditsText:      true,
			FetchRecommendations:  true,
			FetchSimilar:          true,
			FetchReviews:          false,
			FetchWatchProviders:   false,
			FetchPoster:           true,
			FetchBackdrop:         true,
			MaxCastImages:         0,
			MaxGalleryPosters:     0,
			MaxGalleryBackdrops:   0,
			MaxGalleryLogos:       0,
			MaxRelatedPosters:     0,
			FetchSeasonOverview:   true,
			FetchSeasonPosters:    false,
			FetchEpisodeOverview:  true,
			FetchEpisodeStills:    false,
		}
	case FetchModeFull:
		s.Modules = ModuleConfig{
			FetchTitle:            true,
			FetchOverview:         true,
			FetchGenres:           true,
			FetchKeywords:         true,
			FetchReleaseDates:     true,
			FetchContentRatings:   true,
			FetchExternalIDs:      true,
			FetchTranslations:     true,
			FetchAlternativeTitles: true,
			FetchTrailers:         true,
			FetchCreditsText:      true,
			FetchRecommendations:  true,
			FetchSimilar:          true,
			FetchReviews:          true,
			FetchWatchProviders:   true,
			FetchPoster:           true,
			FetchBackdrop:         true,
			MaxCastImages:         20,
			MaxGalleryPosters:     12,
			MaxGalleryBackdrops:   12,
			MaxGalleryLogos:       8,
			MaxRelatedPosters:     12,
			FetchSeasonOverview:   true,
			FetchSeasonPosters:    true,
			FetchEpisodeOverview:  true,
			FetchEpisodeStills:    true,
		}
	}
}

// TMDBUsageSummary encapsulates network & quota monitoring
type TMDBUsageSummary struct {
	TotalRequests          int64     `json:"total_requests"`
	RequestsToday          int64     `json:"requests_today"`
	RequestsThisMonth      int64     `json:"requests_this_month"`
	TotalBytesDownloaded   int64     `json:"total_bytes_downloaded"`
	BytesToday             int64     `json:"bytes_today"`
	MBToday                float64   `json:"mb_today"`
	DailyQuotaMB           int64     `json:"daily_quota_mb"`
	DailyQuotaUsedPercent  float64   `json:"daily_quota_used_percent"`
	TotalImagesDownloaded  int64     `json:"total_images_downloaded"`
	ImagesToday            int64     `json:"images_today"`
	EnrichedMediaCount     int64     `json:"enriched_media_count"`
	PendingMediaCount      int64     `json:"pending_media_count"`
	LastRequestAt          *time.Time `json:"last_request_at,omitempty"`
}

type TMDBUsageDay struct {
	Day              string  `json:"day"`
	Requests         int64   `json:"requests"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	MBDownloaded     float64 `json:"mb_downloaded"`
	ImagesDownloaded int64   `json:"images_downloaded"`
	Successful       int64   `json:"successful"`
	Failed           int64   `json:"failed"`
}

// ModuleItemInfo represents a descriptor for UI render
type ModuleItemInfo struct {
	ID                  string `json:"id"`
	Category            string `json:"category"` // "text", "media", "tv", "extra"
	NameAR              string `json:"name_ar"`
	NameEN              string `json:"name_en"`
	DescriptionAR       string `json:"description_ar"`
	EstimatedBandwidth  string `json:"estimated_bandwidth"`
	Enabled             bool   `json:"enabled"`
	IsEssential         bool   `json:"is_essential"`
}

// GetModuleList returns a friendly UI manifest of all available toggles
func GetModuleList(cfg ModuleConfig) []ModuleItemInfo {
	return []ModuleItemInfo{
		{
			ID: "fetch_title", Category: "text", NameAR: "العناوين (عربي / إنجليزي)", NameEN: "Titles (AR/EN)",
			DescriptionAR: "جلب الاسم الرسمي الإنجليزي والاسم المعرب المترجم", EstimatedBandwidth: "~0 KB (نص)",
			Enabled: cfg.FetchTitle, IsEssential: true,
		},
		{
			ID: "fetch_overview", Category: "text", NameAR: "قصة العمل والملخص", NameEN: "Overview & Synopsis",
			DescriptionAR: "ملخص أحداث الفيلم أو المسلسل بالعربية والإنجليزية", EstimatedBandwidth: "~1 KB (نص)",
			Enabled: cfg.FetchOverview, IsEssential: true,
		},
		{
			ID: "fetch_genres", Category: "text", NameAR: "التصنيفات والأنواع", NameEN: "Genres & Origin Tags",
			DescriptionAR: "أكشن، دراما، كوميديا، أنمي، ونطاق الدولة (كوري، تركي، أجنبي...)", EstimatedBandwidth: "~0 KB (نص)",
			Enabled: cfg.FetchGenres, IsEssential: true,
		},
		{
			ID: "fetch_keywords", Category: "text", NameAR: "الكلمات المفتاحية للبحث", NameEN: "Search Keywords",
			DescriptionAR: "كلمات دلالية تفصيلية لتحسين البحث باللغتين", EstimatedBandwidth: "~1 KB (نص)",
			Enabled: cfg.FetchKeywords, IsEssential: false,
		},
		{
			ID: "fetch_poster", Category: "media", NameAR: "البوستر الرئيسي (Poster)", NameEN: "Primary Poster",
			DescriptionAR: "صورة الغلاف الأساسية المعتمدة للواجهة", EstimatedBandwidth: "~80-150 KB",
			Enabled: cfg.FetchPoster, IsEssential: true,
		},
		{
			ID: "fetch_backdrop", Category: "media", NameAR: "الخلفية العريضة (Backdrop)", NameEN: "Primary Backdrop",
			DescriptionAR: "صورة البانر العريضة في هيدر التفاصيل", EstimatedBandwidth: "~200-400 KB",
			Enabled: cfg.FetchBackdrop, IsEssential: true,
		},
		{
			ID: "fetch_credits_text", Category: "text", NameAR: "طاقم العمل والممثلين (نص فقط)", NameEN: "Cast & Crew Names",
			DescriptionAR: "أسماء الممثلين والمخرج وأدوارهم بدون تحميل صورهم", EstimatedBandwidth: "~2 KB (نص)",
			Enabled: cfg.FetchCreditsText, IsEssential: false,
		},
		{
			ID: "fetch_trailers", Category: "media", NameAR: "مقاطع الفيديو الدعائية (Trailers)", NameEN: "YouTube Trailers",
			DescriptionAR: "معرفات وروابط الإعلانات الترويجية لتشغيلها مباشرة", EstimatedBandwidth: "~0 KB (روابط)",
			Enabled: cfg.FetchTrailers, IsEssential: false,
		},
		{
			ID: "fetch_external_ids", Category: "text", NameAR: "المعرفات الخارجية (IMDb / TVDB)", NameEN: "External IDs",
			DescriptionAR: "روابط ومعرفات قواعد البيانات العالمية", EstimatedBandwidth: "~0 KB (نص)",
			Enabled: cfg.FetchExternalIDs, IsEssential: false,
		},
		{
			ID: "fetch_content_ratings", Category: "text", NameAR: "التصنيف العمري (Content Rating)", NameEN: "Age Certification",
			DescriptionAR: "الرقابة العائلية وتصنيف الأعمار (PG-13, R, TV-MA)", EstimatedBandwidth: "~0 KB (نص)",
			Enabled: cfg.FetchContentRatings, IsEssential: false,
		},
		{
			ID: "fetch_translations", Category: "text", NameAR: "قائمة الترجمات النصية", NameEN: "Metadata Translations",
			DescriptionAR: "فهرس بجميع اللغات المتوفرة للعمل على TMDB", EstimatedBandwidth: "~2 KB (نص)",
			Enabled: cfg.FetchTranslations, IsEssential: false,
		},
		{
			ID: "fetch_season_overview", Category: "tv", NameAR: "بيانات المواسم والحلقات (نص)", NameEN: "Season & Episode Info",
			DescriptionAR: "أرقام المواسم، أسماء الحلقات، تواريخ العرض والمدد", EstimatedBandwidth: "~3-10 KB",
			Enabled: cfg.FetchSeasonOverview, IsEssential: false,
		},
		{
			ID: "fetch_recommendations", Category: "extra", NameAR: "الأعمال الموصى بها والمشابهة", NameEN: "Recommendations & Similar",
			DescriptionAR: "قوائم بالأفلام والمسلسلات المقترحة بناءً على العمل", EstimatedBandwidth: "~2 KB (نص)",
			Enabled: cfg.FetchRecommendations, IsEssential: false,
		},
		{
			ID: "fetch_reviews", Category: "extra", NameAR: "مراجعات ونقاد TMDB", NameEN: "User Reviews",
			DescriptionAR: "آراء وتقييمات المشاهدين المكتوبة", EstimatedBandwidth: "~5 KB (نص)",
			Enabled: cfg.FetchReviews, IsEssential: false,
		},
		{
			ID: "fetch_watch_providers", Category: "extra", NameAR: "مزودو البث الخارجي (JustWatch)", NameEN: "Watch Providers",
			DescriptionAR: "منصات البث مثل Netflix / Shahid مع نسب JustWatch", EstimatedBandwidth: "~1 KB (نص)",
			Enabled: cfg.FetchWatchProviders, IsEssential: false,
		},
	}
}

func (s *TMDBSettings) JSON() []byte {
	b, _ := json.Marshal(s)
	return b
}
