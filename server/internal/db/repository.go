package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexora/server/internal/search"
)

type Repository struct {
	db            *sql.DB
	assetImageDir string
}

func NewRepository(database *sql.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) SetAssetImageDir(dir string) {
	r.assetImageDir = dir
}

func (r *Repository) CacheLocalArtwork(sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	if r.assetImageDir == "" {
		return "/api/stream/image?path=" + url.QueryEscape(sourcePath)
	}

	targetDir := filepath.Join(r.assetImageDir, "local")
	_ = os.MkdirAll(targetDir, 0o755)

	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".jpg"
	}
	hash := sha256.Sum256([]byte(filepath.Clean(sourcePath)))
	hashHex := hex.EncodeToString(hash[:])[:16]
	destFileName := "local_" + hashHex + ext
	destPath := filepath.Join(targetDir, destFileName)

	srcStat, err := os.Stat(sourcePath)
	if err != nil {
		return "/api/stream/image?path=" + url.QueryEscape(sourcePath)
	}
	destStat, err := os.Stat(destPath)
	if err != nil || destStat.Size() != srcStat.Size() {
		srcFile, err := os.Open(sourcePath)
		if err == nil {
			defer srcFile.Close()
			destFile, err := os.Create(destPath)
			if err == nil {
				defer destFile.Close()
				_, _ = io.Copy(destFile, srcFile)
			}
		}
	}
	return "/assets/images/local/" + destFileName
}

type Health struct {
	DatabaseOK bool      `json:"databaseOk"`
	CheckedAt  time.Time `json:"checkedAt"`
}

type IngestResult struct {
	Scanned  int `json:"scanned"`
	Imported int `json:"imported"`
}

type CategorySummary struct {
	ID         int64  `json:"id"`
	NameAR     string `json:"name_ar"`
	NameEN     string `json:"name_en"`
	Slug       string `json:"slug"`
	MediaCount int    `json:"media_count"`
	FileCount  int    `json:"file_count"`
}

type VideoFile struct {
	ID                    int64           `json:"id"`
	MediaItemID           int64           `json:"media_item_id"`
	SeasonID              int64           `json:"season_id,omitempty"`
	EpisodeNumber         int             `json:"episode_number,omitempty"`
	TitleAR               string          `json:"title_ar,omitempty"`
	TitleEN               string          `json:"title_en,omitempty"`
	FilePath              string          `json:"-"`
	FileSize              int64           `json:"file_size"`
	Duration              int             `json:"duration,omitempty"`
	Resolution            string          `json:"resolution,omitempty"`
	VideoCodec            string          `json:"video_codec,omitempty"`
	AudioTracks           json.RawMessage `json:"audio_tracks,omitempty"`
	Subtitles             json.RawMessage `json:"subtitles,omitempty"`
	VerificationStatus    string          `json:"verification_status,omitempty"`
	VerificationError     string          `json:"verification_error,omitempty"`
	VerificationCheckedAt *time.Time      `json:"verification_checked_at,omitempty"`
	StreamURL             string          `json:"stream_url,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

// DuplicateGroup contains files with the same verified SHA-256 checksum.
// Checksums are intentionally calculated on demand so routine ingest stays fast
// even for very large libraries.
type DuplicateGroup struct {
	Checksum string      `json:"checksum"`
	FileSize int64       `json:"file_size"`
	Files    []VideoFile `json:"files"`
}

type MissingEpisode struct {
	MediaItemID  int64 `json:"media_item_id"`
	SeasonID     int64 `json:"season_id"`
	SeasonNumber int   `json:"season_number"`
	Episode      int   `json:"episode_number"`
}

// CorruptedFile is a persisted FFmpeg verification failure for an indexed file.
type CorruptedFile struct {
	ID          int64     `json:"id"`
	MediaItemID int64     `json:"media_item_id"`
	Title       string    `json:"title"`
	FilePath    string    `json:"file_path"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

type ChecksumResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

// MetadataSnapshot is the locally cached, provider-owned detail document.
// The API exposes this only through the NEXORA server, never directly from TMDB.
type MetadataSnapshot struct {
	Provider   string          `json:"provider"`
	ExternalID string          `json:"externalId"`
	Locale     string          `json:"locale"`
	Payload    json.RawMessage `json:"payload"`
	FetchedAt  time.Time       `json:"fetchedAt"`
	ExpiresAt  time.Time       `json:"expiresAt"`
}

// SeasonMetadataSnapshot retains a whole TMDB season response locally. Its
// payload includes the provider's episode list and season-level extras.
type SeasonMetadataSnapshot struct {
	Provider     string          `json:"provider"`
	ExternalID   string          `json:"externalId"`
	Locale       string          `json:"locale"`
	SeasonNumber int             `json:"seasonNumber"`
	Payload      json.RawMessage `json:"payload"`
	FetchedAt    time.Time       `json:"fetchedAt"`
	ExpiresAt    time.Time       `json:"expiresAt"`
}

type SeasonDetail struct {
	ID           int64       `json:"id"`
	SeasonNumber int         `json:"season_number"`
	TitleAR      string      `json:"title_ar,omitempty"`
	TitleEN      string      `json:"title_en,omitempty"`
	Episodes     []VideoFile `json:"episodes"`
}

type MediaItemDetail struct {
	ID           int64          `json:"id"`
	CategoryID   int64          `json:"category_id"`
	CategorySlug string         `json:"category_slug,omitempty"`
	CategoryAR   string         `json:"category_ar,omitempty"`
	CategoryEN   string         `json:"category_en,omitempty"`
	TitleAR      string         `json:"title_ar,omitempty"`
	TitleEN      string         `json:"title_en"`
	Type         string         `json:"type"`
	PlotAR        string         `json:"plot_ar,omitempty"`
	PlotEN        string         `json:"plot_en,omitempty"`
	ReleaseYear   int            `json:"release_year,omitempty"`
	Rating        float64        `json:"rating,omitempty"`
	PosterPath    string         `json:"poster_path,omitempty"`
	BannerPath    string         `json:"banner_path,omitempty"`
	Genres        []string       `json:"genres,omitempty"`
	GenreIDs      []int          `json:"genre_ids,omitempty"`
	ContentRating string         `json:"content_rating,omitempty"`
	Status        string         `json:"status,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	FileCount     int            `json:"file_count"`
	Seasons       []SeasonDetail `json:"seasons,omitempty"`
	Files         []VideoFile    `json:"files,omitempty"`
}

// mediaCardSummary is deliberately returned with every catalogue/search item.
// It is calculated from persisted database data only; browser card rendering
// never needs to contact TMDB or inspect individual files.
type mediaCardSummary struct {
	Status             string
	SeasonCount        int
	TMDBSeasonCount    int
	TMDBEpisodeCount   int
	TotalSize          int64
	BestResolution     string
	RuntimeMinutes     int
	HasArabicAudio     bool
	HasArabicSubtitles bool
}

type ListMediaOptions struct {
	IDs          []int64
	CategorySlug string
	Type         string
	Types        []string
	Categories   []string
	TagsAny      []string
	YearFrom     int
	YearTo       int
	RatingGTE    float64
	Search       string
	Sort         string
	Limit        int
	Offset       int
}

type MediaListResult struct {
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Items  []search.MediaDocument `json:"items"`
}

// ProviderCollection is a locally persisted TMDB-style film franchise. It is
// deliberately not the editorial Collection type below: provider identity is
// stable while an editorial collection is controlled by the library owner.
type ProviderCollection struct {
	ID             int64  `json:"id"`
	Slug           string `json:"slug"`
	Provider       string `json:"provider"`
	ExternalID     string `json:"external_id"`
	Kind           string `json:"kind"`
	TitleAR        string `json:"title_ar,omitempty"`
	TitleEN        string `json:"title_en"`
	OverviewAR     string `json:"overview_ar,omitempty"`
	OverviewEN     string `json:"overview_en,omitempty"`
	PosterPath     string `json:"poster_path,omitempty"`
	BackdropPath   string `json:"backdrop_path,omitempty"`
	PartsCount     int    `json:"parts_count"`
	Rating         float64 `json:"rating,omitempty"`
	LocalItemCount int    `json:"local_item_count"`
	IsFeatured     bool   `json:"is_featured"`
	IsHidden       bool   `json:"is_hidden"`
}

// ProviderCollectionPart is an official TMDB collection member. Local media
// fields are populated only when this part exists in the NEXORA library.
type ProviderCollectionPart struct {
	ExternalID string  `json:"external_id"`
	Title      string  `json:"title"`
	TitleAR    string  `json:"title_ar,omitempty"`
	TitleEN    string  `json:"title_en,omitempty"`
	Year       int     `json:"year,omitempty"`
	Rating     float64 `json:"rating,omitempty"`
	Overview   string  `json:"overview,omitempty"`
	OverviewAR string  `json:"overview_ar,omitempty"`
	OverviewEN string  `json:"overview_en,omitempty"`
	PosterPath string  `json:"poster_path,omitempty"`
	Local      bool    `json:"local"`
	MediaID    int64   `json:"media_id,omitempty"`
}

type Person struct {
	ID                 int64   `json:"id"`
	Slug               string  `json:"slug"`
	Provider           string  `json:"provider"`
	ExternalID         string  `json:"external_id"`
	NameAR             string  `json:"name_ar,omitempty"`
	NameEN             string  `json:"name_en"`
	KnownForDepartment string  `json:"known_for_department,omitempty"`
	ProfilePath        string  `json:"profile_path,omitempty"`
	Popularity         float64 `json:"popularity,omitempty"`
	LocalMediaCount    int     `json:"local_media_count"`
	IsFeatured         bool    `json:"is_featured"`
	IsHidden           bool    `json:"is_hidden"`
}

// CatalogRelationSyncResult describes an offline rebuild of the derived
// franchise and people graph. It reads metadata_snapshots only; it never calls
// TMDB and is therefore safe for a disconnected library server.
type CatalogRelationSyncResult struct {
	SnapshotsProcessed int `json:"snapshots_processed"`
	CollectionsLinked  int `json:"collections_linked"`
	CreditsLinked      int `json:"credits_linked"`
}

// CatalogEntityAdminUpdate is shared by provider collections and people. The
// pointer fields make partial administrative updates explicit and safe.
type CatalogEntityAdminUpdate struct {
	IsFeatured   *bool `json:"is_featured"`
	IsHidden     *bool `json:"is_hidden"`
	SortPriority *int  `json:"sort_priority"`
}

// ShowcaseOptions selects locally persisted editorial collections and media
// summaries for the reusable hero. It never consults a remote metadata source.
type ShowcaseOptions struct {
	Context      string
	CategorySlug string
	Limit        int
}

type ShowcaseTarget struct {
	Category string          `json:"category,omitempty"`
	Filters  json.RawMessage `json:"filters,omitempty"`
}

type ShowcaseSlide struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	MediaID         int64           `json:"media_id,omitempty"`
	TitleAR         string          `json:"title_ar,omitempty"`
	TitleEN         string          `json:"title_en,omitempty"`
	DescriptionAR   string          `json:"description_ar,omitempty"`
	DescriptionEN   string          `json:"description_en,omitempty"`
	ArtworkPath     string          `json:"artwork_path,omitempty"`
	ArtworkPosition string          `json:"artwork_position,omitempty"`
	Accent          string          `json:"accent,omitempty"`
	ItemCount       int             `json:"item_count,omitempty"`
	Type            string          `json:"type,omitempty"`
	Status          string          `json:"status,omitempty"`
	ReleaseYear     int             `json:"release_year,omitempty"`
	Rating          float64         `json:"rating,omitempty"`
	BestResolution  string          `json:"best_resolution,omitempty"`
	Genres          []string        `json:"genres,omitempty"`
	Target          *ShowcaseTarget `json:"target,omitempty"`
}

type ShowcaseResult struct {
	Context string          `json:"context"`
	Slides  []ShowcaseSlide `json:"slides"`
}

type Collection struct {
	ID                 int64           `json:"id"`
	Slug               string          `json:"slug"`
	TitleAR            string          `json:"title_ar"`
	TitleEN            string          `json:"title_en"`
	DescriptionAR      string          `json:"description_ar"`
	DescriptionEN      string          `json:"description_en"`
	ArtworkPath        string          `json:"artwork_path"`
	ArtworkPosition    string          `json:"artwork_position"`
	Accent             string          `json:"accent"`
	TargetCategorySlug string          `json:"target_category_slug"`
	TargetFilters      json.RawMessage `json:"target_filters"`
	Priority           int             `json:"priority"`
	IsActive           bool            `json:"is_active"`
	ItemCount          int             `json:"item_count"`
	ItemIDs            []int64         `json:"item_ids,omitempty"`
}

type CollectionRequest struct {
	Slug               string          `json:"slug"`
	TitleAR            string          `json:"title_ar"`
	TitleEN            string          `json:"title_en"`
	DescriptionAR      string          `json:"description_ar"`
	DescriptionEN      string          `json:"description_en"`
	ArtworkPath        string          `json:"artwork_path"`
	ArtworkPosition    string          `json:"artwork_position"`
	Accent             string          `json:"accent"`
	TargetCategorySlug string          `json:"target_category_slug"`
	TargetFilters      json.RawMessage `json:"target_filters"`
	Priority           int             `json:"priority"`
	IsActive           bool            `json:"is_active"`
	ItemIDs            []int64         `json:"item_ids,omitempty"`
}

type HubRule struct {
	Types      []string `json:"types,omitempty"`
	Categories []string `json:"categories,omitempty"`
	TagsAny    []string `json:"tags_any,omitempty"`
	YearFrom   int      `json:"year_from,omitempty"`
	YearTo     int      `json:"year_to,omitempty"`
	RatingGTE  float64  `json:"rating_gte,omitempty"`
}

type SmartHub struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	Source          string   `json:"source"`
	Scope           string   `json:"scope"`
	TitleAR         string   `json:"title_ar"`
	TitleEN         string   `json:"title_en"`
	DescriptionAR   string   `json:"description_ar"`
	DescriptionEN   string   `json:"description_en"`
	ArtworkPath     string   `json:"artwork_path"`
	ArtworkPosition string   `json:"artwork_position"`
	Accent          string   `json:"accent"`
	Icon            string   `json:"icon"`
	Rule            HubRule  `json:"rule"`
	Priority        int      `json:"priority"`
	ItemCount       int      `json:"item_count"`
	PreviewArtwork  []string `json:"preview_artwork,omitempty"`
	IsActive        bool     `json:"is_active"`
	MinItemCount    int      `json:"min_item_count"`
}

type SmartHubRequest struct {
	Slug            string  `json:"slug"`
	Scope           string  `json:"scope"`
	TitleAR         string  `json:"title_ar"`
	TitleEN         string  `json:"title_en"`
	DescriptionAR   string  `json:"description_ar"`
	DescriptionEN   string  `json:"description_en"`
	ArtworkPath     string  `json:"artwork_path"`
	ArtworkPosition string  `json:"artwork_position"`
	Accent          string  `json:"accent"`
	Icon            string  `json:"icon"`
	Rule            HubRule `json:"rule"`
	Priority        int     `json:"priority"`
	IsActive        bool    `json:"is_active"`
	MinItemCount    int     `json:"min_item_count"`
}

type StorageDisk struct {
	ID          int64      `json:"id"`
	DiskLetter  string     `json:"disk_letter"`
	DiskLabel   string     `json:"disk_label"`
	TotalSpace  int64      `json:"total_space"`
	FreeSpace   int64      `json:"free_space"`
	UsedSpace   int64      `json:"used_space"`
	UsedPercent float64    `json:"used_percent"`
	IsActive    bool       `json:"is_active"`
	LastScanned *time.Time `json:"last_scanned,omitempty"`
}

type DashboardStats struct {
	TotalMedia           int64             `json:"total_media"`
	TotalFiles           int64             `json:"total_files"`
	TotalStorageBytes    int64             `json:"total_storage_bytes"`
	MissingEpisodesCount int               `json:"missing_episodes_count"`
	DuplicatesCount      int               `json:"duplicates_count"`
	CorruptedFilesCount  int64             `json:"corrupted_files_count"`
	Categories           []CategorySummary `json:"categories"`
	Disks                []StorageDisk     `json:"disks"`
}

func (r *Repository) Health(ctx context.Context) (Health, error) {
	if err := r.db.PingContext(ctx); err != nil {
		return Health{CheckedAt: time.Now().UTC()}, err
	}
	return Health{DatabaseOK: true, CheckedAt: time.Now().UTC()}, nil
}
