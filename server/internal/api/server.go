package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/disks"
	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/migration"
	"nexora/server/internal/quality"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type repository interface {
	Health(ctx context.Context) (db.Health, error)
	IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error)
	ClassifyOriginsFromPaths(ctx context.Context) (int, error)
	ListCategories(ctx context.Context) ([]db.CategorySummary, error)
	CreateCategory(ctx context.Context, nameAR, nameEN, slug string) (*db.CategorySummary, error)
	UpdateCategory(ctx context.Context, id int64, nameAR, nameEN, slug string) error
	DeleteCategory(ctx context.Context, id int64) error
	ListCollections(ctx context.Context) ([]db.Collection, error)
	SaveCollection(ctx context.Context, id int64, req db.CollectionRequest) (*db.Collection, error)
	DeleteCollection(ctx context.Context, id int64) error
	ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error)
	ListVideoFiles(ctx context.Context, mediaItemID int64) ([]db.VideoFile, error)
	GetVideoFilePath(ctx context.Context, id int64) (string, error)
	GetVideoFileIDByPath(ctx context.Context, path string) (int64, error)
	UpdateVideoTechnicalDetails(ctx context.Context, id int64, details media.InspectResult) error
	UpdateVideoVerification(ctx context.Context, id int64, result media.VerifyResult) error
	ListDuplicateGroups(ctx context.Context) ([]db.DuplicateGroup, error)
	ListMissingEpisodes(ctx context.Context) ([]db.MissingEpisode, error)
	ListCorruptedFiles(ctx context.Context) ([]db.CorruptedFile, error)
	CalculateChecksums(ctx context.Context, mediaItemID int64) (db.ChecksumResult, error)
	GetMediaItem(ctx context.Context, id int64) (*db.MediaItemDetail, error)
	ListMediaItems(ctx context.Context, opts db.ListMediaOptions) (*db.MediaListResult, error)
	ListProviderCollections(ctx context.Context, limit int) ([]db.ProviderCollection, error)
	GetProviderCollection(ctx context.Context, slug string) (*db.ProviderCollection, error)
	GetProviderCollectionByID(ctx context.Context, id int64) (*db.ProviderCollection, error)
	ProviderCollectionNeedsRefresh(ctx context.Context, externalID string) (bool, error)
	ListProviderCollectionRefreshCandidates(ctx context.Context, limit int) ([]db.ProviderCollection, error)
	ListProviderCollectionMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.ProviderCollection, *db.MediaListResult, error)
	ListProviderCollectionParts(ctx context.Context, slug string) (*db.ProviderCollection, []db.ProviderCollectionPart, error)
	UpdateProviderCollectionAdmin(ctx context.Context, id int64, update db.CatalogEntityAdminUpdate) error
	ListPeople(ctx context.Context, limit int) ([]db.Person, error)
	GetPerson(ctx context.Context, slug string) (*db.Person, error)
	ListPersonMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.Person, *db.MediaListResult, error)
	UpdatePersonAdmin(ctx context.Context, id int64, update db.CatalogEntityAdminUpdate) error
	SyncCatalogRelationsFromSnapshots(ctx context.Context) (*db.CatalogRelationSyncResult, error)
	SaveProviderCollectionMetadata(ctx context.Context, meta metadata.CollectionResult) error
	ListShowcases(ctx context.Context, opts db.ShowcaseOptions) (*db.ShowcaseResult, error)
	ListSmartHubs(ctx context.Context, scope string) ([]db.SmartHub, error)
	GetSmartHub(ctx context.Context, slug string) (*db.SmartHub, error)
	ListSmartHubMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.MediaListResult, *db.SmartHub, error)
	ListSmartHubsAdmin(ctx context.Context) ([]db.SmartHub, error)
	SaveSmartHub(ctx context.Context, slug string, req db.SmartHubRequest) (*db.SmartHub, error)
	UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error)
	GetMetadataSnapshot(ctx context.Context, mediaItemID int64, locale string) (*db.MetadataSnapshot, error)
	SaveSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, snapshots []metadata.SeasonResult) error
	GetSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, locale string) ([]db.SeasonMetadataSnapshot, error)
	GetDashboardStats(ctx context.Context) (*db.DashboardStats, error)
	ListDisks(ctx context.Context) ([]db.StorageDisk, error)
	SaveDisks(ctx context.Context, disks []db.StorageDisk) error
	CreateMediaItem(ctx context.Context, req db.CreateMediaRequest) (*search.MediaDocument, error)
	UpdateMediaFull(ctx context.Context, id int64, req db.UpdateMediaRequest) (*search.MediaDocument, error)
	DeleteMediaItem(ctx context.Context, id int64) error
	GetTMDBSettings(ctx context.Context) (*metadata.TMDBSettings, error)
	SaveTMDBSettings(ctx context.Context, s metadata.TMDBSettings) error
	GetTMDBUsageSummary(ctx context.Context) (*metadata.TMDBUsageSummary, error)
	GetTMDBUsageHistory(ctx context.Context, days int) ([]metadata.TMDBUsageDay, error)
	LogTMDBUsage(ctx context.Context, entry db.TMDBLogEntry) error
	EnqueueTMDBRefresh(ctx context.Context, mediaID int64, priority int) error
	EnqueueStaleTMDBRefreshes(ctx context.Context, staleDays, limit int) error
	EnqueueTMDBRefreshIfStale(ctx context.Context, mediaID int64, staleDays int) error
	ListTMDBQueue(ctx context.Context, limit int) ([]db.TMDBQueueJob, error)
	CancelTMDBQueueJob(ctx context.Context, id int64) error
	ClaimTMDBQueueJob(ctx context.Context) (*db.TMDBQueueJob, error)
	FinishTMDBQueueJob(ctx context.Context, id int64, succeeded bool, message string) error
	CacheLocalArtwork(sourcePath string) string
	CleanAndSyncAllGenres(ctx context.Context) (int, error)
}

type searchClient interface {
	IndexDocuments(ctx context.Context, documents []search.MediaDocument) (search.SyncResult, error)
	SearchDocuments(ctx context.Context, query string, limit int, filter string) (search.SearchResult, error)
}

type metadataService interface {
	Lookup(ctx context.Context, query metadata.Query) (metadata.Result, error)
	SearchCandidates(ctx context.Context, query metadata.Query) ([]metadata.Candidate, error)
	LookupByExternalID(ctx context.Context, query metadata.Query, externalID string) (metadata.Result, error)
	LookupSeasonByExternalID(ctx context.Context, externalID string, seasonNumber int, language string) (metadata.SeasonResult, error)
	LookupCollectionByExternalID(ctx context.Context, externalID, language string) (metadata.CollectionResult, error)
	GetTMDBSettings() metadata.TMDBSettings
	SetTMDBSettings(settings metadata.TMDBSettings)
	FetchTMDBConfiguration(ctx context.Context) (*metadata.TMDBRemoteConfig, error)
}

type mediaProcessor interface {
	Verify(ctx context.Context, path string) (media.VerifyResult, error)
	Inspect(ctx context.Context, path string) (media.InspectResult, error)
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string, at time.Duration) (string, error)
}

type migrationService interface {
	Preview(ctx context.Context, root string) (migration.PreviewResult, error)
	Copy(ctx context.Context, request migration.CopyRequest) (migration.CopyResult, error)
}

type qualityService interface {
	GenerateReport(ctx context.Context) (*quality.QualityReport, error)
	FindDuplicates(ctx context.Context) ([]db.DuplicateGroup, int64, error)
	FindMissingEpisodes(ctx context.Context) ([]quality.MissingEpisodeDetail, error)
	VerifyFile(ctx context.Context, filePath string) (bool, string, error)
	ListCorruptedFiles(ctx context.Context) ([]quality.CorruptedFileDetail, error)
}

type Server struct {
	config      config.Config
	repository  repository
	scanner     *scanner.Scanner
	search      searchClient
	metadata    metadataService
	processor   mediaProcessor
	migration   migrationService
	quality     qualityService
	diskManager *disks.Manager
	mux         *http.ServeMux
	cache       *responseCache
}

func NewServer(
	config config.Config,
	repository repository,
	scannerService *scanner.Scanner,
	searchClient searchClient,
	metadataService metadataService,
	processor mediaProcessor,
	migrationService migrationService,
	qualityService qualityService,
) http.Handler {
	server := &Server{
		config:      config,
		repository:  repository,
		scanner:     scannerService,
		search:      searchClient,
		metadata:    metadataService,
		processor:   processor,
		migration:   migrationService,
		quality:     qualityService,
		diskManager: disks.NewManager(),
		mux:         http.NewServeMux(),
		cache:       newResponseCache(config.RedisAddr, config.RedisPassword, config.RedisDB),
	}
	server.routes()
	go server.runTMDBQueue()
	return server.withMiddleware(server.mux)
}

func (s *Server) runTMDBQueue() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		settings := s.metadata.GetTMDBSettings()
		if settings.AutoRefreshEnabled {
			_ = s.repository.EnqueueStaleTMDBRefreshes(context.Background(), settings.RefreshIntervalDays, 100)
		}
		workers := settings.QueueMaxConcurrent
		if workers < 1 { workers = 1 }
		if workers > 4 { workers = 4 }
		var group sync.WaitGroup
		for index := 0; index < workers; index++ {
			job, err := s.repository.ClaimTMDBQueueJob(context.Background())
			if err != nil || job == nil { break }
			group.Add(1)
			go func(job *db.TMDBQueueJob) {
				defer group.Done()
				s.processTMDBQueueJob(job)
			}(job)
		}
		group.Wait()
	}
}

func (s *Server) processTMDBQueueJob(job *db.TMDBQueueJob) {
		request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media/%d/enrich", job.MediaItemID), nil)
		request.SetPathValue("id", strconv.FormatInt(job.MediaItemID, 10))
		response := httptest.NewRecorder()
		s.handleMediaEnrich(response, request)
		succeeded := response.Code >= 200 && response.Code < 300
		message := ""
		if !succeeded { message = strings.TrimSpace(response.Body.String()) }
		_ = s.repository.FinishTMDBQueueJob(context.Background(), job.ID, succeeded, message)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/categories", s.handleCategories)
	s.mux.HandleFunc("POST /api/categories", s.handleCategoryCreate)
	s.mux.HandleFunc("PUT /api/categories/{id}", s.handleCategoryUpdate)
	s.mux.HandleFunc("DELETE /api/categories/{id}", s.handleCategoryDelete)
	s.mux.HandleFunc("GET /api/admin/collections", s.handleCollections)
	s.mux.HandleFunc("POST /api/admin/collections", s.handleCollectionSave)
	s.mux.HandleFunc("PUT /api/admin/collections/{id}", s.handleCollectionSave)
	s.mux.HandleFunc("DELETE /api/admin/collections/{id}", s.handleCollectionDelete)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/media", s.handleMediaList)
	s.mux.HandleFunc("GET /api/franchises", s.handleFranchises)
	s.mux.HandleFunc("GET /api/franchises/{slug}", s.handleFranchise)
	s.mux.HandleFunc("GET /api/franchises/{slug}/media", s.handleFranchiseMedia)
	s.mux.HandleFunc("PUT /api/admin/franchises/{id}", s.handleFranchiseAdminUpdate)
	s.mux.HandleFunc("POST /api/admin/franchises/{id}/refresh", s.handleFranchiseRefresh)
	s.mux.HandleFunc("POST /api/admin/franchises/refresh-missing", s.handleFranchiseRefreshMissing)
	s.mux.HandleFunc("GET /api/people", s.handlePeople)
	s.mux.HandleFunc("GET /api/people/{slug}", s.handlePerson)
	s.mux.HandleFunc("GET /api/people/{slug}/media", s.handlePersonMedia)
	s.mux.HandleFunc("PUT /api/admin/people/{id}", s.handlePersonAdminUpdate)
	s.mux.HandleFunc("POST /api/admin/catalog/sync-relations", s.handleCatalogRelationSync)
	s.mux.HandleFunc("GET /api/showcases", s.handleShowcases)
	s.mux.HandleFunc("GET /api/hubs", s.handleSmartHubs)
	s.mux.HandleFunc("GET /api/hubs/{slug}", s.handleSmartHub)
	s.mux.HandleFunc("GET /api/hubs/{slug}/media", s.handleSmartHubMedia)
	s.mux.HandleFunc("GET /api/admin/hubs", s.handleSmartHubsAdmin)
	s.mux.HandleFunc("POST /api/admin/hubs", s.handleSmartHubCreate)
	s.mux.HandleFunc("PUT /api/admin/hubs/{slug}", s.handleSmartHubSave)
	s.mux.HandleFunc("POST /api/media", s.handleMediaCreate)
	s.mux.HandleFunc("GET /api/media/{id}", s.handleMediaDetail)
	s.mux.HandleFunc("PUT /api/media/{id}", s.handleMediaUpdateFull)
	s.mux.HandleFunc("DELETE /api/media/{id}", s.handleMediaDelete)
	s.mux.HandleFunc("GET /api/media/{id}/files", s.handleMediaFiles)
	s.mux.HandleFunc("GET /api/media/{id}/metadata/raw", s.handleMediaMetadataSnapshot)
	s.mux.HandleFunc("GET /api/media/{id}/metadata/seasons", s.handleMediaSeasonMetadataSnapshots)
	s.mux.HandleFunc("POST /api/media/{id}/enrich", s.handleMediaEnrich)
	s.mux.HandleFunc("PUT /api/media/{id}/metadata", s.handleMediaMetadataUpdate)
	s.mux.HandleFunc("GET /api/library/duplicates", s.handleDuplicates)
	s.mux.HandleFunc("GET /api/library/missing-episodes", s.handleMissingEpisodes)
	s.mux.HandleFunc("GET /api/library/corrupted", s.handleCorruptedFiles)
	s.mux.HandleFunc("GET /api/quality/report", s.handleQualityReport)
	s.mux.HandleFunc("GET /api/dashboard/stats", s.handleDashboardStats)
	s.mux.HandleFunc("GET /api/disks", s.handleDisksList)
	s.mux.HandleFunc("POST /api/disks/scan", s.handleDisksScan)
	s.mux.HandleFunc("GET /api/scan", s.handleScan)
	s.mux.HandleFunc("POST /api/ingest", s.handleIngest)
	s.mux.HandleFunc("POST /api/index", s.handleIndex)
	s.mux.HandleFunc("POST /api/index/preview", s.handleIndexPreview)
	s.mux.HandleFunc("POST /api/library/classify-origins", s.handleClassifyOrigins)
	s.mux.HandleFunc("POST /api/search/sync", s.handleSearchSync)
	s.mux.HandleFunc("POST /api/metadata/lookup", s.handleMetadataLookup)
	s.mux.HandleFunc("GET /api/tmdb/candidates", s.handleTMDBCandidates)
	s.mux.HandleFunc("POST /api/media/verify", s.handleMediaVerify)
	s.mux.HandleFunc("POST /api/media/inspect", s.handleMediaInspect)
	s.mux.HandleFunc("POST /api/media/thumbnail", s.handleThumbnail)
	s.mux.HandleFunc("POST /api/media/checksums", s.handleChecksums)
	s.mux.HandleFunc("POST /api/migration/preview", s.handleMigrationPreview)
	s.mux.HandleFunc("POST /api/migration/copy", s.handleMigrationCopy)
	s.mux.HandleFunc("GET /api/stream", s.handleStream)
	s.mux.HandleFunc("GET /api/stream/image", s.handleStreamImage)
	s.mux.HandleFunc("GET /api/stream/file/{id}", s.handleStreamByID)
	s.mux.HandleFunc("GET /api/stream/file/{id}/subtitles", s.handleFileSubtitles)
	s.mux.HandleFunc("GET /api/stream/file/{id}/subtitles/{subId}", s.handleFileSubtitleStream)

	// TMDB Control Panel & Integration Endpoints
	s.mux.HandleFunc("GET /api/tmdb/settings", s.handleTMDBSettingsGet)
	s.mux.HandleFunc("GET /api/tmdb/queue", s.handleTMDBQueueGet)
	s.mux.HandleFunc("GET /api/tmdb/usage/history", s.handleTMDBUsageHistory)
	s.mux.HandleFunc("POST /api/tmdb/queue", s.handleTMDBQueueCreate)
	s.mux.HandleFunc("POST /api/tmdb/queue/{id}/cancel", s.handleTMDBQueueCancel)
	s.mux.HandleFunc("PUT /api/tmdb/settings", s.handleTMDBSettingsUpdate)
	s.mux.HandleFunc("GET /api/tmdb/stats", s.handleTMDBStatsGet)
	s.mux.HandleFunc("GET /api/tmdb/modules", s.handleTMDBModulesGet)
	s.mux.HandleFunc("PUT /api/tmdb/modules", s.handleTMDBModulesUpdate)
	s.mux.HandleFunc("POST /api/tmdb/test", s.handleTMDBTestConnection)
	s.mux.HandleFunc("GET /api/tmdb/configuration", s.handleTMDBConfigurationGet)
	s.mux.HandleFunc("GET /api/tmdb/preview/{id}", s.handleTMDBPreviewGet)

	// System Directory Tree Explorer & Admin Auth Endpoints
	s.mux.HandleFunc("GET /api/system/drives", s.handleSystemDrives)
	s.mux.HandleFunc("GET /api/system/browse", s.handleSystemBrowse)
	s.mux.HandleFunc("POST /api/admin/maintenance/clean-genres", s.handleCleanGenres)
	s.mux.HandleFunc("POST /api/admin/login", s.handleAdminLogin)
	s.mux.HandleFunc("GET /api/admin/session", s.handleAdminSession)
	s.mux.HandleFunc("POST /api/admin/logout", s.handleAdminLogout)

	// Static assets serving for downloaded posters/banners/thumbnails
	if s.config.AssetImageDir != "" {
		_ = os.MkdirAll(s.config.AssetImageDir, 0o755)
		fileServer := http.StripPrefix("/assets/images/", http.FileServer(http.Dir(s.config.AssetImageDir)))
		s.mux.Handle("GET /assets/images/", fileServer)
	}
}

type indexResult struct {
	Roots         []string          `json:"roots"`
	Scanned       int               `json:"scanned"`
	Imported      int               `json:"imported"`
	Inspected     int               `json:"inspected"`
	InspectFailed int               `json:"inspectFailed"`
	SearchSync    search.SyncResult `json:"searchSync"`
	Warnings      []string          `json:"warnings,omitempty"`
}

const (
	ingestBatchSize  = 128
	maxIndexWarnings = 100
)

func (r *indexResult) addWarning(message string) {
	if len(r.Warnings) < maxIndexWarnings {
		r.Warnings = append(r.Warnings, message)
		return
	}
	if len(r.Warnings) == maxIndexWarnings {
		r.Warnings = append(r.Warnings, "additional indexing warnings were omitted")
	}
}

// handleIndex is the unified first-pass library workflow. Metadata artwork and
// subtitle extraction are deliberately separate jobs because they can require
// external providers or long-running FFmpeg work.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}
	result := indexResult{Roots: request.Roots, Warnings: []string{}}
	batch := make([]scanner.FileInfo, 0, ingestBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ingested, err := s.repository.IngestScannedFiles(r.Context(), batch)
		result.Imported += ingested.Imported
		if err != nil {
			return err
		}
		for _, file := range batch {
			id, err := s.repository.GetVideoFileIDByPath(r.Context(), file.Path)
			if err != nil {
				result.InspectFailed++
				result.addWarning("could not locate indexed file: " + file.Path)
				continue
			}
			details, err := s.processor.Inspect(r.Context(), file.Path)
			if err != nil {
				result.InspectFailed++
				result.addWarning("could not inspect: " + file.Path)
				continue
			}
			if err := s.repository.UpdateVideoTechnicalDetails(r.Context(), id, details); err != nil {
				result.InspectFailed++
				result.addWarning("could not save inspection: " + file.Path)
				continue
			}
			result.Inspected++
		}
		batch = batch[:0]
		return nil
	}
	err := s.scanner.Walk(r.Context(), request.Roots, func(file scanner.FileInfo) error {
		result.Scanned++
		batch = append(batch, file)
		if len(batch) == cap(batch) {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}
	documents, err := s.repository.ListSearchDocuments(r.Context(), 10000)
	if err != nil {
		result.addWarning("search sync skipped: " + err.Error())
	} else if syncResult, err := s.search.IndexDocuments(r.Context(), documents); err != nil {
		result.addWarning("search sync skipped: " + err.Error())
	} else {
		result.SearchSync = syncResult
	}
	writeJSON(w, http.StatusOK, result)
}

type indexPreviewItem struct {
	Title         string               `json:"title"`
	TitleAR       string               `json:"title_ar"`
	TitleEN       string               `json:"title_en"`
	Type          string               `json:"type"`
	CategorySlug  string               `json:"category_slug"`
	OriginTags    []string             `json:"origin_tags"`
	ReleaseYear   int                  `json:"release_year"`
	PosterPath    string               `json:"poster_path,omitempty"`
	RawPosterPath string               `json:"raw_poster_path,omitempty"`
	TotalFiles    int                  `json:"total_files"`
	TotalSize     int64                `json:"total_size"`
	Seasons       []indexPreviewSeason `json:"seasons"`
}

type indexPreviewSeason struct {
	SeasonNumber int                `json:"season_number"`
	Title        string             `json:"title"`
	EpisodeCount int                `json:"episode_count"`
	Episodes     []indexPreviewFile `json:"episodes"`
}

type indexPreviewFile struct {
	Path          string `json:"path"`
	EpisodeNumber int    `json:"episode_number"`
	Size          int64  `json:"size"`
	Resolution    string `json:"resolution"`
}

// handleIndexPreview generates a non-destructive dry-run preview of what would be indexed.
func (s *Server) handleIndexPreview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(request.Roots) == 0 {
		request.Roots = s.config.MediaRoots
	}
	if len(request.Roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provide at least one media root"})
		return
	}
	for _, root := range request.Roots {
		if !s.mediaPathAllowed(root) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "media root is outside configured roots"})
			return
		}
	}

	type mediaGroupKey struct {
		titleKey string
		mType    string
	}

	groups := make(map[mediaGroupKey]*indexPreviewItem)
	var groupOrder []mediaGroupKey
	totalScanned := 0

	err := s.scanner.Walk(r.Context(), request.Roots, func(file scanner.FileInfo) error {
		totalScanned++
		categorySlug, mediaType := db.ClassifyMedia(file.Path, file.Parsed)
		title := file.Parsed.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))
		}
		titleEN := file.Parsed.TitleEN
		if titleEN == "" {
			titleEN = title
		}
		titleAR := file.Parsed.TitleAR
		if titleAR == "" {
			titleAR = title
		}

		key := mediaGroupKey{
			titleKey: strings.ToLower(titleEN),
			mType:    mediaType,
		}

		group, exists := groups[key]
		if !exists {
			localPosterURL := ""
			if file.ArtworkPath != "" {
				localPosterURL = s.repository.CacheLocalArtwork(file.ArtworkPath)
			}
			originTags := scanner.DetectOriginTagsFromPath(file.Path)
			if originTags == nil {
				originTags = []string{}
			}

			group = &indexPreviewItem{
				Title:         title,
				TitleAR:       titleAR,
				TitleEN:       titleEN,
				Type:          mediaType,
				CategorySlug:  categorySlug,
				OriginTags:    originTags,
				ReleaseYear:   file.Parsed.ReleaseYear,
				PosterPath:    localPosterURL,
				RawPosterPath: file.ArtworkPath,
				Seasons:       []indexPreviewSeason{},
			}
			groups[key] = group
			groupOrder = append(groupOrder, key)
		}

		group.TotalFiles++
		group.TotalSize += file.Size
		if group.PosterPath == "" && file.ArtworkPath != "" {
			group.PosterPath = s.repository.CacheLocalArtwork(file.ArtworkPath)
			group.RawPosterPath = file.ArtworkPath
		}

		seasonNum := file.Parsed.SeasonNumber
		if seasonNum <= 0 {
			seasonNum = 1
		}

		// Find or add season
		var targetSeason *indexPreviewSeason
		for idx := range group.Seasons {
			if group.Seasons[idx].SeasonNumber == seasonNum {
				targetSeason = &group.Seasons[idx]
				break
			}
		}
		if targetSeason == nil {
			group.Seasons = append(group.Seasons, indexPreviewSeason{
				SeasonNumber: seasonNum,
				Title:        fmt.Sprintf("Season %02d", seasonNum),
				Episodes:     []indexPreviewFile{},
			})
			targetSeason = &group.Seasons[len(group.Seasons)-1]
		}

		targetSeason.EpisodeCount++
		targetSeason.Episodes = append(targetSeason.Episodes, indexPreviewFile{
			Path:          file.Path,
			EpisodeNumber: file.Parsed.EpisodeNumber,
			Size:          file.Size,
			Resolution:    file.Parsed.Resolution,
		})

		return nil
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	resultItems := make([]*indexPreviewItem, 0, len(groupOrder))
	for _, k := range groupOrder {
		resultItems = append(resultItems, groups[k])
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roots":      request.Roots,
		"totalFiles": totalScanned,
		"totalMedia": len(resultItems),
		"mediaItems": resultItems,
	})
}

// handleStreamImage streams a local image file safely.
func (s *Server) handleStreamImage(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if filePath == "" {
		http.Error(w, "missing image path", http.StatusBadRequest)
		return
	}
	if !s.mediaPathAllowed(filePath) {
		http.Error(w, "forbidden path", http.StatusForbidden)
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".jfif", ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".bmp":
		contentType = "image/bmp"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, file)
}

// handleClassifyOrigins repairs country tags from the user's existing folder
// hierarchy. It is deliberately local-only: no title matching or third-party
// request can overwrite a folder such as "مسلسلات/عربي".
func (s *Server) handleClassifyOrigins(w http.ResponseWriter, r *http.Request) {
	updated, err := s.repository.ClassifyOriginsFromPaths(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Country tags are also searchable, so refresh the local search index.
	if documents, err := s.repository.ListSearchDocuments(r.Context(), 10000); err == nil {
		if _, err := s.search.IndexDocuments(r.Context(), documents); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true, "updated": updated,
				"warning": "origin tags updated but search index refresh failed: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}

func (s *Server) handleMediaInspect(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FileID int64 `json:"fileId"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if request.FileID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId must be a positive integer"})
		return
	}
	path, err := s.repository.GetVideoFilePath(r.Context(), request.FileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}
	details, err := s.processor.Inspect(r.Context(), path)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error()})
		return
	}
	if err := s.repository.UpdateVideoTechnicalDetails(r.Context(), request.FileID, details); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) handleDuplicates(w http.ResponseWriter, r *http.Request) {
	groups, err := s.repository.ListDuplicateGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(groups), "groups": groups})
}

func (s *Server) handleMissingEpisodes(w http.ResponseWriter, r *http.Request) {
	missing, err := s.repository.ListMissingEpisodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(missing), "episodes": missing})
}

func (s *Server) handleCorruptedFiles(w http.ResponseWriter, r *http.Request) {
	files, err := s.repository.ListCorruptedFiles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(files), "files": files})
}

func (s *Server) handleChecksums(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MediaItemID int64 `json:"mediaItemId,omitempty"`
	}
	if r.Body != http.NoBody {
		if err := decodeJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
	}
	if request.MediaItemID < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "mediaItemId must be positive"})
		return
	}
	result, err := s.repository.CalculateChecksums(r.Context(), request.MediaItemID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health, err := s.repository.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":    false,
			"error": err.Error(),
			"time":  time.Now().UTC(),
		})
		return
	}
	redisOK, redisStatus := s.cache.RedisStatus(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"database": health,
		"cache": map[string]any{
			"redisOk":     redisOK,
			"redisStatus": redisStatus,
		},
	})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	roots := r.URL.Query()["root"]
	if len(roots) == 0 {
		roots = s.config.MediaRoots
	}
	if len(roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "provide at least one root query parameter or set NEXORA_MEDIA_ROOTS",
		})
		return
	}

	files, err := s.scanner.Scan(r.Context(), roots)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roots": roots,
		"count": len(files),
		"files": files,
	})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.repository.ListCategories(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":      len(categories),
		"categories": categories,
	})
}

func (s *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		NameAR string `json:"name_ar"`
		NameEN string `json:"name_en"`
		Slug   string `json:"slug"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	cat, err := s.repository.CreateCategory(r.Context(), request.NameAR, request.NameEN, request.Slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

func (s *Server) handleCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "category id must be a positive integer"})
		return
	}

	var request struct {
		NameAR string `json:"name_ar"`
		NameEN string `json:"name_en"`
		Slug   string `json:"slug"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.UpdateCategory(r.Context(), id, request.NameAR, request.NameEN, request.Slug); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "category id must be a positive integer"})
		return
	}

	if err := s.repository.DeleteCategory(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_id": id})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	items, err := s.repository.ListCollections(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"collections": items})
}
func (s *Server) handleCollectionSave(w http.ResponseWriter, r *http.Request) {
	var req db.CollectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	var id int64
	if raw := r.PathValue("id"); raw != "" {
		var ok bool
		id, ok = parsePositiveID(raw)
		if !ok {
			writeJSON(w, 400, map[string]any{"error": "invalid collection id"})
			return
		}
	}
	item, err := s.repository.SaveCollection(r.Context(), id, req)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) handleCollectionDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, 400, map[string]any{"error": "invalid collection id"})
		return
	}
	if err := s.repository.DeleteCollection(r.Context(), id); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 24
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	filterParts := make([]string, 0, 2)
	if rawType := strings.TrimSpace(r.URL.Query().Get("type")); rawType != "" {
		filterParts = append(filterParts, `type = "`+escapeFilterValue(rawType)+`"`)
	}
	if rawCategory := strings.TrimSpace(r.URL.Query().Get("category")); rawCategory != "" {
		filterParts = append(filterParts, `category_slug = "`+escapeFilterValue(rawCategory)+`"`)
	}

	result, err := s.search.SearchDocuments(r.Context(), query, limit, strings.Join(filterParts, " AND "))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMediaFiles(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	files, err := s.repository.ListVideoFiles(r.Context(), mediaID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	for index := range files {
		files[index].StreamURL = "/api/stream/file/" + strconv.FormatInt(files[index].ID, 10)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"media_id": mediaID,
		"count":    len(files),
		"files":    files,
	})
}

func (s *Server) handleMediaMetadataSnapshot(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}
	locale := strings.TrimSpace(r.URL.Query().Get("locale"))
	if locale == "" {
		locale = "en-US"
	}
	snapshot, err := s.repository.GetMetadataSnapshot(r.Context(), mediaID, locale)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleMediaSeasonMetadataSnapshots(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}
	locale := strings.TrimSpace(r.URL.Query().Get("locale"))
	if locale == "" {
		locale = "en-US"
	}
	snapshots, err := s.repository.GetSeasonMetadataSnapshots(r.Context(), mediaID, locale)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mediaId": mediaID, "locale": locale, "items": snapshots})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	roots := r.URL.Query()["root"]
	if len(roots) == 0 {
		roots = s.config.MediaRoots
	}
	if len(roots) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "provide at least one root query parameter or set NEXORA_MEDIA_ROOTS",
		})
		return
	}

	result := db.IngestResult{}
	batch := make([]scanner.FileInfo, 0, ingestBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ingested, err := s.repository.IngestScannedFiles(r.Context(), batch)
		result.Imported += ingested.Imported
		if err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	err := s.scanner.Walk(r.Context(), roots, func(file scanner.FileInfo) error {
		result.Scanned++
		batch = append(batch, file)
		if len(batch) == cap(batch) {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "partial": result})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleSearchSync(w http.ResponseWriter, r *http.Request) {
	limit := 1000
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	documents, err := s.repository.ListSearchDocuments(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.search.IndexDocuments(r.Context(), documents)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleMetadataLookup(w http.ResponseWriter, r *http.Request) {
	var request metadata.Query
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	result, err := s.metadata.Lookup(r.Context(), request)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, metadata.ErrNotConfigured) {
			status = http.StatusFailedDependency
		}
		if errors.Is(err, metadata.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMediaVerify(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FileID int64  `json:"fileId,omitempty"`
		Path   string `json:"path,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if request.FileID < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId must be a positive integer"})
		return
	}
	if request.FileID > 0 {
		path, err := s.repository.GetVideoFilePath(r.Context(), request.FileID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		request.Path = path
	}
	if strings.TrimSpace(request.Path) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "fileId or path is required"})
		return
	}
	if !s.mediaPathAllowed(request.Path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}

	result, err := s.processor.Verify(r.Context(), request.Path)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error(), "result": result})
		return
	}
	if request.FileID == 0 {
		if id, lookupErr := s.repository.GetVideoFileIDByPath(r.Context(), request.Path); lookupErr == nil {
			request.FileID = id
		}
	}
	if request.FileID > 0 {
		if err := s.repository.UpdateVideoVerification(r.Context(), request.FileID, result); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "result": result})
			return
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Path       string `json:"path"`
		OutputPath string `json:"outputPath,omitempty"`
		Second     int    `json:"second,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(request.Path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}
	if request.OutputPath == "" {
		base := strings.TrimSuffix(filepath.Base(request.Path), filepath.Ext(request.Path))
		request.OutputPath = filepath.Join(s.config.AssetImageDir, "thumbnails", safeFileName(base)+".jpg")
	}

	outputPath, err := s.processor.GenerateThumbnail(r.Context(), request.Path, request.OutputPath, time.Duration(request.Second)*time.Second)
	if err != nil {
		writeJSON(w, http.StatusFailedDependency, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"thumbnailPath": outputPath})
}

func (s *Server) handleMigrationPreview(w http.ResponseWriter, r *http.Request) {
	var request migration.PreviewRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(request.Root) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "root is required"})
		return
	}
	if !s.mediaPathAllowed(request.Root) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration root is outside configured media roots"})
		return
	}
	result, err := s.migration.Preview(r.Context(), request.Root)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMigrationCopy(w http.ResponseWriter, r *http.Request) {
	var request migration.CopyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !s.mediaPathAllowed(request.Target) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration target is outside configured media roots"})
		return
	}
	for _, source := range request.Sources {
		if !s.mediaPathAllowed(source) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "migration source is outside configured media roots"})
			return
		}
	}
	result, err := s.migration.Copy(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	s.serveMediaPath(w, r, path)
}

func (s *Server) handleStreamByID(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	s.serveMediaPath(w, r, path)
}

func (s *Server) serveMediaPath(w http.ResponseWriter, r *http.Request, path string) {
	if !s.mediaPathAllowed(path) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "media path is outside configured roots"})
		return
	}

	file, err := os.Open(path)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "path must point to a file"})
		return
	}

	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			// All catalogue writes invalidate the server cache across L1 and Redis.
			s.cache.clear(r.Context())
			next.ServeHTTP(w, r)
			return
		}

		ttl := catalogueCacheTTL(r.URL.Path)
		if ttl <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		key := cacheKey(r)
		if entry, source, ok := s.cache.get(r.Context(), key); ok {
			w.Header().Set("Content-Type", entry.ContentType)
			w.Header().Set("Cache-Control", "public, max-age=30, must-revalidate")
			w.Header().Set("X-NEXORA-Cache", "HIT ("+source+")")
			w.WriteHeader(entry.Status)
			_, _ = w.Write(entry.Body)
			return
		}

		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, r)
		result := recorder.Result()
		body := recorder.Body.Bytes()
		for header, values := range result.Header {
			w.Header()[header] = append([]string(nil), values...)
		}
		w.Header().Set("Cache-Control", "public, max-age=30, must-revalidate")
		w.Header().Set("X-NEXORA-Cache", "MISS")
		w.WriteHeader(result.StatusCode)
		_, _ = w.Write(body)
		if result.StatusCode >= http.StatusOK && result.StatusCode < http.StatusMultipleChoices {
			s.cache.set(r.Context(), key, cachedResponse{
				Status:      result.StatusCode,
				Body:        append([]byte(nil), body...),
				ContentType: result.Header.Get("Content-Type"),
				ExpiresAt:   time.Now().Add(ttl),
			})
		}
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		if parsed, err := strconv.Atoi(rawOffset); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	opts := db.ListMediaOptions{
		CategorySlug: strings.TrimSpace(r.URL.Query().Get("category")),
		Type:         strings.TrimSpace(r.URL.Query().Get("type")),
		Search:       strings.TrimSpace(r.URL.Query().Get("q")),
		Sort:         strings.TrimSpace(r.URL.Query().Get("sort")),
		Limit:        limit,
		Offset:       offset,
	}

	result, err := s.repository.ListMediaItems(r.Context(), opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleFranchises is intentionally database-only. TMDB is contacted only by
// the enrich workflow; browsing a franchise rail is safe on an offline LAN.
func (s *Server) handleFranchises(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	items, err := s.repository.ListProviderCollections(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"franchises": items})
}

func (s *Server) handleFranchise(w http.ResponseWriter, r *http.Request) {
	item, err := s.repository.GetProviderCollection(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleFranchiseMedia(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	collection, result, err := s.repository.ListProviderCollectionMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	_, parts, partsErr := s.repository.ListProviderCollectionParts(r.Context(), r.PathValue("slug"))
	if partsErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": partsErr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"franchise": collection, "total": result.Total, "items": result.Items, "parts": parts})
}

func (s *Server) handleFranchiseAdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "franchise id must be positive"})
		return
	}
	var update db.CatalogEntityAdminUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid update payload"})
		return
	}
	if err := s.repository.UpdateProviderCollectionAdmin(r.Context(), id, update); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

// handleFranchiseRefresh is the explicit admin control for an existing
// collection. It fetches en-US and ar-SA once, caches fields and images, then
// leaves all visitor-facing routes fully local.
func (s *Server) handleFranchiseRefresh(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "franchise id must be positive"})
		return
	}
	collection, err := s.repository.GetProviderCollectionByID(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	warnings := s.refreshProviderCollectionMetadata(r.Context(), collection.ExternalID)
	if len(warnings) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "some collection fields could not be refreshed", "warnings": warnings})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "external_id": collection.ExternalID})
}

// handleFranchiseRefreshMissing upgrades pre-existing lightweight collection
// links in a deliberate batch. It is never reached by customer browsing.
func (s *Server) handleFranchiseRefreshMissing(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	candidates, err := s.repository.ListProviderCollectionRefreshCandidates(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	refreshed, warnings := make([]string, 0, len(candidates)), make([]string, 0)
	for _, candidate := range candidates {
		issues := s.refreshProviderCollectionMetadata(r.Context(), candidate.ExternalID)
		if len(issues) > 0 {
			warnings = append(warnings, candidate.ExternalID+": "+strings.Join(issues, "; "))
			continue
		}
		refreshed = append(refreshed, candidate.ExternalID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "refreshed": refreshed, "warnings": warnings})
}

// handlePeople and its detail endpoints are backed exclusively by media_credits
// and people tables populated during enrichment or the local rebuild action.
func (s *Server) handlePeople(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	people, err := s.repository.ListPeople(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"people": people})
}

func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	person, err := s.repository.GetPerson(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, person)
}

func (s *Server) handlePersonMedia(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	person, result, err := s.repository.ListPersonMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"person": person, "total": result.Total, "items": result.Items})
}

func (s *Server) handlePersonAdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "person id must be positive"})
		return
	}
	var update db.CatalogEntityAdminUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid update payload"})
		return
	}
	if err := s.repository.UpdatePersonAdmin(r.Context(), id, update); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

// handleCatalogRelationSync deliberately uses only cached TMDB snapshots. It
// backfills imported libraries without another network request or an API key.
func (s *Server) handleCatalogRelationSync(w http.ResponseWriter, r *http.Request) {
	result, err := s.repository.SyncCatalogRelationsFromSnapshots(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (s *Server) handleShowcases(w http.ResponseWriter, r *http.Request) {
	limit := 6
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	result, err := s.repository.ListShowcases(r.Context(), db.ShowcaseOptions{
		Context:      strings.TrimSpace(r.URL.Query().Get("context")),
		CategorySlug: strings.TrimSpace(r.URL.Query().Get("category")),
		Limit:        limit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSmartHubs(w http.ResponseWriter, r *http.Request) {
	hubs, err := s.repository.ListSmartHubs(r.Context(), strings.TrimSpace(r.URL.Query().Get("scope")))
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hubs": hubs})
}
func (s *Server) handleSmartHub(w http.ResponseWriter, r *http.Request) {
	hub, err := s.repository.GetSmartHub(r.Context(), r.PathValue("slug"))
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, hub)
}
func (s *Server) handleSmartHubMedia(w http.ResponseWriter, r *http.Request) {
	limit := 24
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	result, hub, err := s.repository.ListSmartHubMedia(r.Context(), r.PathValue("slug"), db.ListMediaOptions{Limit: limit, Offset: offset, Sort: strings.TrimSpace(r.URL.Query().Get("sort"))})
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hub": hub, "total": result.Total, "limit": result.Limit, "offset": result.Offset, "items": result.Items})
}
func (s *Server) handleSmartHubsAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := s.repository.ListSmartHubsAdmin(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"hubs": items})
}
func (s *Server) handleSmartHubSave(w http.ResponseWriter, r *http.Request) {
	var req db.SmartHubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	item, err := s.repository.SaveSmartHub(r.Context(), r.PathValue("slug"), req)
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, item)
}
func (s *Server) handleSmartHubCreate(w http.ResponseWriter, r *http.Request) {
	var req db.SmartHubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	item, err := s.repository.SaveSmartHub(r.Context(), "", req)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleMediaDetail(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, item)
	settings := s.metadata.GetTMDBSettings()
	if settings.AutoRefreshEnabled && settings.RefreshOnOpen {
		_ = s.repository.EnqueueTMDBRefreshIfStale(r.Context(), mediaID, settings.RefreshStaleDays)
	}
}

func (s *Server) handleMediaCreate(w http.ResponseWriter, r *http.Request) {
	var request db.CreateMediaRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.CreateMediaItem(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if s.search != nil && doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) handleMediaUpdateFull(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	var request db.UpdateMediaRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.UpdateMediaFull(r.Context(), mediaID, request)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	if s.search != nil && doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	if err := s.repository.DeleteMediaItem(r.Context(), mediaID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_id": mediaID})
}

func (s *Server) handleMediaEnrich(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}

	// Sync current settings from DB into metadata service if available
	if dbSettings, err := s.repository.GetTMDBSettings(r.Context()); err == nil && dbSettings != nil {
		s.metadata.SetTMDBSettings(*dbSettings)
	}

	lookupTitle := item.TitleEN
	if lookupTitle == "" {
		lookupTitle = item.TitleAR
	}

	var canonical metadata.Result
	var lookupErr error
	if selectedID := strings.TrimSpace(r.URL.Query().Get("tmdb_id")); selectedID != "" {
		canonical, lookupErr = s.metadata.LookupByExternalID(r.Context(), metadata.Query{Type: item.Type, Language: "en-US"}, selectedID)
	} else {
		canonical, lookupErr = s.metadata.Lookup(r.Context(), metadata.Query{
			Title: lookupTitle, Type: item.Type, Year: item.ReleaseYear, Language: "en-US",
		})
	}
	if lookupErr != nil {
		_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
			MediaItemID:  &mediaID,
			RequestKind:  "lookup",
			Endpoint:     "/search",
			StatusCode:   502,
			DurationMS:   int(time.Since(startTime).Milliseconds()),
			ErrorMessage: lookupErr.Error(),
		})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "metadata lookup failed: " + lookupErr.Error(), "title": lookupTitle, "mediaId": mediaID})
		return
	}

	// The confirmed English search result defines the TMDB ID. Arabic is then
	// fetched by that same ID, never via a second ambiguous title search.
	results := []metadata.Result{canonical}
	var lookupWarnings []string
	if canonical.Provider == "tmdb" {
		arabic, arabicErr := s.metadata.LookupByExternalID(r.Context(), metadata.Query{Type: item.Type, Language: "ar-SA"}, canonical.ExternalID)
		if arabicErr != nil {
			lookupWarnings = append(lookupWarnings, "ar-SA: "+arabicErr.Error())
		} else {
			results = []metadata.Result{arabic, canonical}
		}
	}

	var doc *search.MediaDocument
	for _, metaResult := range results {
		updated, updateErr := s.repository.UpdateMediaMetadata(r.Context(), mediaID, metaResult)
		if updateErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": updateErr.Error()})
			return
		}
		if updated != nil {
			doc = updated
		}
	}

	// Fetch complete franchise details only during this explicit enrich action.
	// Normal franchise GET requests remain strictly database-only.
	if canonical.Provider == "tmdb" && item.Type == "movie" {
		if collectionID := tmdbCollectionID(canonical.RawPayload); collectionID != "" {
			needsRefresh, needsErr := s.repository.ProviderCollectionNeedsRefresh(r.Context(), collectionID)
			if needsErr != nil {
				lookupWarnings = append(lookupWarnings, "collection cache state: "+needsErr.Error())
			} else if needsRefresh {
				lookupWarnings = append(lookupWarnings, s.refreshProviderCollectionMetadata(r.Context(), collectionID)...)
			}
		}
	}

	if doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	seasonCount := 0
	if canonical.Provider == "tmdb" && isSeriesType(item.Type) {
		for _, seasonNumber := range tmdbSeasonNumbers(canonical.RawPayload) {
			englishSeason, seasonErr := s.metadata.LookupSeasonByExternalID(r.Context(), canonical.ExternalID, seasonNumber, "en-US")
			if seasonErr != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d en-US: %v", seasonNumber, seasonErr))
				continue
			}
			seasonSnapshots := []metadata.SeasonResult{englishSeason}
			arabicSeason, arabicSeasonErr := s.metadata.LookupSeasonByExternalID(r.Context(), canonical.ExternalID, seasonNumber, "ar-SA")
			if arabicSeasonErr != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d ar-SA: %v", seasonNumber, arabicSeasonErr))
			} else {
				seasonSnapshots = append(seasonSnapshots, arabicSeason)
			}
			if err := s.repository.SaveSeasonMetadataSnapshots(r.Context(), mediaID, seasonSnapshots); err != nil {
				lookupWarnings = append(lookupWarnings, fmt.Sprintf("season %d cache: %v", seasonNumber, err))
				continue
			}
			seasonCount++
		}
	}

	// Log successful enrich usage
	_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
		MediaItemID:      &mediaID,
		RequestKind:      "enrich",
		Endpoint:         "/details",
		StatusCode:       200,
		BytesDownloaded:  int64(len(canonical.RawPayload)),
		ImagesDownloaded: 2, // Poster + Backdrop
		DurationMS:       int(time.Since(startTime).Milliseconds()),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"metadata":      results,
		"document":      doc,
		"warnings":      lookupWarnings,
		"seasonsCached": seasonCount,
	})
}

func isSeriesType(mediaType string) bool {
	return mediaType == "series" || mediaType == "anime" || mediaType == "tv"
}

func tmdbCollectionID(raw json.RawMessage) string {
	var payload struct {
		Collection *struct {
			ID json.Number `json:"id"`
		} `json:"belongs_to_collection"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil || payload.Collection == nil {
		return ""
	}
	return payload.Collection.ID.String()
}

func (s *Server) refreshProviderCollectionMetadata(ctx context.Context, externalID string) []string {
	warnings := make([]string, 0)
	for _, language := range []string{"en-US", "ar-SA"} {
		collection, err := s.metadata.LookupCollectionByExternalID(ctx, externalID, language)
		if err != nil {
			warnings = append(warnings, "collection "+language+": "+err.Error())
			continue
		}
		if err := s.repository.SaveProviderCollectionMetadata(ctx, collection); err != nil {
			warnings = append(warnings, "collection cache "+language+": "+err.Error())
		}
	}
	return warnings
}

func tmdbSeasonNumbers(raw json.RawMessage) []int {
	var payload struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
		} `json:"seasons"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	numbers := make([]int, 0, len(payload.Seasons))
	for _, season := range payload.Seasons {
		if season.SeasonNumber >= 0 {
			numbers = append(numbers, season.SeasonNumber)
		}
	}
	return numbers
}

func (s *Server) handleMediaMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	var request struct {
		Title       string   `json:"title"`
		Overview    string   `json:"overview"`
		ReleaseYear int      `json:"releaseYear"`
		Rating      float64  `json:"rating"`
		PosterPath  string   `json:"posterPath"`
		BannerPath  string   `json:"bannerPath"`
		Genres      []string `json:"genres"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	doc, err := s.repository.UpdateMediaMetadata(r.Context(), mediaID, metadata.Result{
		Title:       request.Title,
		Overview:    request.Overview,
		ReleaseYear: request.ReleaseYear,
		Rating:      request.Rating,
		PosterPath:  request.PosterPath,
		BannerPath:  request.BannerPath,
		Genres:      request.Genres,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if doc != nil {
		_, _ = s.search.IndexDocuments(r.Context(), []search.MediaDocument{*doc})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "document": doc})
}

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repository.GetDashboardStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleQualityReport(w http.ResponseWriter, r *http.Request) {
	if s.quality == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "quality service not available"})
		return
	}

	report, err := s.quality.GenerateReport(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleDisksList(w http.ResponseWriter, r *http.Request) {
	disks, err := s.repository.ListDisks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// If no disks recorded yet, auto-scan
	if len(disks) == 0 && s.diskManager != nil {
		if scanned, scanErr := s.diskManager.ScanDisks(r.Context()); scanErr == nil && len(scanned) > 0 {
			_ = s.repository.SaveDisks(r.Context(), scanned)
			disks = scanned
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(disks),
		"disks": disks,
	})
}

func (s *Server) handleDisksScan(w http.ResponseWriter, r *http.Request) {
	if s.diskManager == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "disk manager not available"})
		return
	}

	disks, err := s.diskManager.ScanDisks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.SaveDisks(r.Context(), disks); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "disks": disks})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(disks),
		"disks": disks,
	})
}

func (s *Server) handleFileSubtitles(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	subs := media.FindExternalSubtitles(path)
	for i := range subs {
		subs[i].Path = fmt.Sprintf("/api/stream/file/%d/subtitles/%d", fileID, subs[i].Index)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"file_id":   fileID,
		"count":     len(subs),
		"subtitles": subs,
	})
}

func (s *Server) handleFileSubtitleStream(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file id must be a positive integer"})
		return
	}
	subIndex, err := strconv.Atoi(r.PathValue("subId"))
	if err != nil || subIndex <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subId must be a positive integer"})
		return
	}

	path, err := s.repository.GetVideoFilePath(r.Context(), fileID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	subs := media.FindExternalSubtitles(path)
	var targetSub *media.SubtitleInfo
	for _, sub := range subs {
		if sub.Index == subIndex {
			targetSub = &sub
			break
		}
	}
	if targetSub == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "subtitle not found"})
		return
	}

	file, err := os.Open(targetSub.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if strings.ToLower(filepath.Ext(targetSub.Path)) == ".srt" {
		if err := media.ConvertSRTToWebVTT(file, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, _ = io.Copy(w, file)
	}
}

func (s *Server) mediaPathAllowed(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if len(s.config.MediaRoots) == 0 {
		return true
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absolutePath = strings.ToLower(filepath.Clean(absolutePath))

	for _, root := range s.config.MediaRoots {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		absoluteRoot = strings.ToLower(filepath.Clean(absoluteRoot))
		if absolutePath == absoluteRoot || strings.HasPrefix(absolutePath, absoluteRoot+string(os.PathSeparator)) {
			return true
		}
	}

	// Also allow any existing local path on disk
	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

var unsafeFileName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeFileName(input string) string {
	input = strings.Trim(unsafeFileName.ReplaceAllString(input, "_"), "._-")
	if input == "" {
		return "thumbnail"
	}
	return input
}

func escapeFilterValue(input string) string {
	return strings.ReplaceAll(input, `"`, `\"`)
}

func parsePositiveID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

func (s *Server) handleTMDBSettingsGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleTMDBQueueGet(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.repository.ListTMDBQueue(r.Context(), 100)
	if err != nil { writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()}); return }
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleTMDBUsageHistory(w http.ResponseWriter, r *http.Request) {
	days := 90
	if raw := r.URL.Query().Get("days"); raw != "" { if parsed, err := strconv.Atoi(raw); err == nil { days = parsed } }
	history, err := s.repository.GetTMDBUsageHistory(r.Context(), days)
	if err != nil { writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()}); return }
	writeJSON(w, http.StatusOK, map[string]any{"history": history})
}

func (s *Server) handleTMDBQueueCreate(w http.ResponseWriter, r *http.Request) {
	var request struct { MediaItemID int64 `json:"media_item_id"`; Priority int `json:"priority"` }
	if err := decodeJSON(r, &request); err != nil || request.MediaItemID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media_item_id must be positive"}); return
	}
	if err := s.repository.EnqueueTMDBRefresh(r.Context(), request.MediaItemID, request.Priority); err != nil { writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()}); return }
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "media_item_id": request.MediaItemID})
}

func (s *Server) handleTMDBQueueCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(r.PathValue("id"))
	if !ok { writeJSON(w, http.StatusBadRequest, map[string]any{"error": "queue id must be positive"}); return }
	if err := s.repository.CancelTMDBQueueJob(r.Context(), id); err != nil { writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()}); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (s *Server) handleTMDBCandidates(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	if title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "title is required"})
		return
	}
	year := 0
	if raw := r.URL.Query().Get("year"); raw != "" {
		year, _ = strconv.Atoi(raw)
	}
	candidates, err := s.metadata.SearchCandidates(r.Context(), metadata.Query{Title: title, Type: strings.TrimSpace(r.URL.Query().Get("type")), Year: year, Language: "en-US"})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}

func (s *Server) handleTMDBSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var settings metadata.TMDBSettings
	if err := decodeJSON(r, &settings); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := s.repository.SaveTMDBSettings(r.Context(), settings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Sync into in-memory service
	s.metadata.SetTMDBSettings(settings)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "settings": settings})
}

func (s *Server) handleTMDBStatsGet(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repository.GetTMDBUsageSummary(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleTMDBModulesGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	modules := metadata.GetModuleList(settings.Modules)
	writeJSON(w, http.StatusOK, map[string]any{
		"fetch_mode": settings.FetchMode,
		"image_mode": settings.ImageMode,
		"modules":    modules,
	})
}

func (s *Server) handleTMDBModulesUpdate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FetchMode *metadata.FetchMode    `json:"fetch_mode,omitempty"`
		ImageMode *metadata.ImageMode    `json:"image_mode,omitempty"`
		Modules   *metadata.ModuleConfig `json:"modules,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	current, err := s.repository.GetTMDBSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if request.FetchMode != nil {
		current.ApplyProfile(*request.FetchMode)
	}
	if request.ImageMode != nil {
		current.ImageMode = *request.ImageMode
	}
	if request.Modules != nil {
		current.Modules = *request.Modules
		current.FetchMode = metadata.FetchModeCustom
	}

	if err := s.repository.SaveTMDBSettings(r.Context(), *current); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.metadata.SetTMDBSettings(*current)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"settings": current,
		"modules":  metadata.GetModuleList(current.Modules),
	})
}

func (s *Server) handleTMDBTestConnection(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	remoteConfig, err := s.metadata.FetchTMDBConfiguration(r.Context())
	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
			RequestKind:  "test_connection",
			Endpoint:     "/configuration",
			StatusCode:   502,
			DurationMS:   int(latency),
			ErrorMessage: err.Error(),
		})
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"connected": false,
			"error":     err.Error(),
			"latencyMs": latency,
		})
		return
	}

	_ = s.repository.LogTMDBUsage(r.Context(), db.TMDBLogEntry{
		RequestKind: "test_connection",
		Endpoint:    "/configuration",
		StatusCode:  200,
		DurationMS:  int(latency),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"connected":     true,
		"latencyMs":     latency,
		"configuration": remoteConfig,
		"checkedAt":     time.Now().UTC(),
	})
}

func (s *Server) handleTMDBConfigurationGet(w http.ResponseWriter, r *http.Request) {
	remoteConfig, err := s.metadata.FetchTMDBConfiguration(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, remoteConfig)
}

func (s *Server) handleTMDBPreviewGet(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parsePositiveID(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "media id must be a positive integer"})
		return
	}

	item, err := s.repository.GetMediaItem(r.Context(), mediaID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}

	settings, _ := s.repository.GetTMDBSettings(r.Context())
	if settings == nil {
		def := metadata.DefaultSettings()
		settings = &def
	}

	// Calculate estimated requests & bytes
	estimatedRequests := 2 // Search + Details EN
	estimatedBytes := int64(350 * 1024)
	if settings.ImageMode == metadata.ImageModeLocal || settings.ImageMode == metadata.ImageModeHybrid {
		estimatedBytes += 250 * 1024 // Poster + Backdrop download
	}
	if settings.Modules.MaxCastImages > 0 && settings.ImageMode == metadata.ImageModeLocal {
		estimatedBytes += int64(settings.Modules.MaxCastImages * 30 * 1024)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"media_id":               mediaID,
		"title_en":               item.TitleEN,
		"title_ar":               item.TitleAR,
		"type":                   item.Type,
		"fetch_mode":             settings.FetchMode,
		"image_mode":             settings.ImageMode,
		"estimatedRequests":      estimatedRequests,
		"estimatedBytes":         estimatedBytes,
		"estimatedSizeFormatted": fmt.Sprintf("%.2f MB", float64(estimatedBytes)/(1024.0*1024.0)),
	})
}

// System Drives & Directory Tree Explorer
type SystemDirectoryItem struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	IsDir      bool      `json:"is_dir"`
	ModifiedAt time.Time `json:"modified_at"`
}

func (s *Server) handleSystemDrives(w http.ResponseWriter, r *http.Request) {
	if s.diskManager == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "disk manager unavailable"})
		return
	}
	disksList, err := s.diskManager.ScanDisks(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"drives": disksList})
}

func (s *Server) handleSystemBrowse(w http.ResponseWriter, r *http.Request) {
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath == "" {
		// If empty, return drives
		s.handleSystemDrives(w, r)
		return
	}

	cleanPath := filepath.Clean(targetPath)
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "cannot read directory: " + err.Error()})
		return
	}

	dirs := make([]SystemDirectoryItem, 0)
	for _, entry := range entries {
		// Skip hidden and system directories
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "$") {
			continue
		}
		if entry.IsDir() {
			info, _ := entry.Info()
			modTime := time.Now()
			if info != nil {
				modTime = info.ModTime()
			}
			dirs = append(dirs, SystemDirectoryItem{
				Name:       entry.Name(),
				Path:       filepath.Join(cleanPath, entry.Name()),
				IsDir:      true,
				ModifiedAt: modTime,
			})
		}
	}

	parent := filepath.Dir(cleanPath)
	if parent == cleanPath {
		parent = ""
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current_path": cleanPath,
		"parent_path":  parent,
		"directories":  dirs,
		"count":        len(dirs),
	})
}

// Admin Authentication Handlers
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid payload"})
		return
	}

	// Default admin credentials (or configurable via ENV / DB)
	adminUser := os.Getenv("NEXORA_ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("NEXORA_ADMIN_PASS")
	if adminPass == "" {
		adminPass = "admin123"
	}

	if req.Username == adminUser && req.Password == adminPass {
		// Return token and user profile
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    true,
			"token": "nexora_admin_auth_token_active",
			"user": map[string]any{
				"username": adminUser,
				"name":     "مدير النظام الرئيسي",
				"role":     "superadmin",
			},
		})
		return
	}

	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"ok":    false,
		"error": "اسم المستخدم أو كلمة المرور غير صحيحة",
	})
}

func (s *Server) handleAdminSession(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if strings.Contains(token, "nexora_admin_auth_token_active") {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"user": map[string]any{
				"username": "admin",
				"name":     "مدير النظام الرئيسي",
				"role":     "superadmin",
			},
		})
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]any{"authenticated": false})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "تم تسجيل الخروج بنجاح"})
}

func (s *Server) handleCleanGenres(w http.ResponseWriter, r *http.Request) {
	updated, err := s.repository.CleanAndSyncAllGenres(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"updated": updated,
		"message": fmt.Sprintf("تم تنظيف وتوحيد التصنيفات واستخراج التصنيف العمري لـ %d عملاً بنجاح", updated),
	})
}
