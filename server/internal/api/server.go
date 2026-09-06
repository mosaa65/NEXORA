package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	"nexora/server/internal/transfer"
)

type repository interface {
	Health(ctx context.Context) (db.Health, error)
	IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error)
	ClassifyOriginsFromPaths(ctx context.Context) (int, error)
	ListCategories(ctx context.Context) ([]db.CategorySummary, error)
	CreateCategory(ctx context.Context, nameAR, nameEN, slug string) (*db.CategorySummary, error)
	UpdateCategory(ctx context.Context, id int64, nameAR, nameEN, slug string) (*db.CategorySummary, error)
	DeleteCategory(ctx context.Context, id int64) error
	GetVideoFilePath(ctx context.Context, id int64) (string, error)
	GetSubtitlesForFile(ctx context.Context, videoFileID int64) ([]db.SubtitleStreamInfo, error)
	GetSubtitleByID(ctx context.Context, subID int64) (*db.SubtitleStreamInfo, error)
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

type transferService interface {
	ListDevices(ctx context.Context) ([]transfer.Device, error)
	ListDeviceApps(ctx context.Context, deviceID string) ([]transfer.DeviceApp, error)
	ListAppFolders(ctx context.Context, deviceID, bundleID string) ([]transfer.AppFolder, error)
	StartCopy(ctx context.Context, req transfer.CopyRequest) (*transfer.TransferJob, error)
	GetJob(jobID string) (*transfer.TransferJob, bool)
	ListJobs() []*transfer.TransferJob
	CancelJob(jobID string) bool
	ListDevicePath(ctx context.Context, deviceID, path string, deviceType transfer.DeviceType, appID string) ([]transfer.RemoteEntry, error)
	StatDevicePath(ctx context.Context, deviceID, path string, deviceType transfer.DeviceType, appID string) (transfer.RemoteEntry, error)
	CreateDeviceFolder(ctx context.Context, deviceID, path string, deviceType transfer.DeviceType, appID string) error
	EjectDevice(ctx context.Context, deviceID string) error
	SubscribeEvents(buffer int) (<-chan transfer.TransferEvent, func())
	SnapshotDevices() []transfer.Device
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
	transfer    transferService
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
	transferService transferService,
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
		transfer:    transferService,
		diskManager: disks.NewManager(),
		mux:         http.NewServeMux(),
		cache:       newResponseCache(config.RedisAddr, config.RedisPassword, config.RedisDB),
	}
	server.routes()
	go server.runTMDBQueue()
	return withGzip(server.withMiddleware(server.mux))
}

func (s *Server) runTMDBQueue() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		job, err := s.repository.ClaimTMDBQueueJob(ctx)
		if err != nil || job == nil {
			cancel()
			continue
		}

		err = s.processTMDBQueueJob(ctx, job)
		if err != nil {
			_ = s.repository.FinishTMDBQueueJob(ctx, job.ID, false, err.Error())
		} else {
			_ = s.repository.FinishTMDBQueueJob(ctx, job.ID, true, "تم التحديث بنجاح عبر الطابور الآلي")
		}
		cancel()
	}
}

func (s *Server) processTMDBQueueJob(ctx context.Context, job *db.TMDBQueueJob) error {
	item, err := s.repository.GetMediaItem(ctx, job.MediaID)
	if err != nil {
		return err
	}

	query := metadata.Query{
		Title: item.TitleAR,
		Type:  item.Type,
		Year:  item.ReleaseYear,
	}
	if query.Title == "" {
		query.Title = item.TitleEN
	}

	meta, err := s.metadata.Lookup(ctx, query)
	if err != nil {
		return err
	}

	doc, err := s.repository.UpdateMediaMetadata(ctx, item.ID, meta)
	if err != nil {
		return err
	}

	_, _ = s.search.IndexDocuments(ctx, []search.MediaDocument{*doc})
	return nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/categories", s.handleCategoriesList)
	s.mux.HandleFunc("POST /api/categories", s.requireAdminAuth(s.handleCategoryCreate))
	s.mux.HandleFunc("PUT /api/categories/{id}", s.requireAdminAuth(s.handleCategoryUpdate))
	s.mux.HandleFunc("DELETE /api/categories/{id}", s.requireAdminAuth(s.handleCategoryDelete))
	s.mux.HandleFunc("GET /api/catalog", s.handleCatalog)
	s.mux.HandleFunc("GET /api/collections", s.handleCollections)
	s.mux.HandleFunc("GET /api/collections/{slug}", s.handleCollectionDetail)
	s.mux.HandleFunc("PUT /api/admin/collections/{id}", s.requireAdminAuth(s.handleCollectionAdminUpdate))
	s.mux.HandleFunc("GET /api/people", s.handlePeople)
	s.mux.HandleFunc("GET /api/people/{slug}", s.handlePersonDetail)
	s.mux.HandleFunc("PUT /api/admin/people/{id}", s.requireAdminAuth(s.handlePersonAdminUpdate))
	s.mux.HandleFunc("POST /api/admin/catalog/sync-relations", s.requireAdminAuth(s.handleCatalogRelationSync))
	s.mux.HandleFunc("GET /api/showcases", s.handleShowcases)
	s.mux.HandleFunc("GET /api/hubs", s.handleSmartHubs)
	s.mux.HandleFunc("GET /api/hubs/{slug}", s.handleSmartHub)
	s.mux.HandleFunc("GET /api/hubs/{slug}/media", s.handleSmartHubMedia)
	s.mux.HandleFunc("GET /api/admin/hubs", s.requireAdminAuth(s.handleSmartHubsAdmin))
	s.mux.HandleFunc("POST /api/admin/hubs", s.requireAdminAuth(s.handleSmartHubCreate))
	s.mux.HandleFunc("PUT /api/admin/hubs/{slug}", s.requireAdminAuth(s.handleSmartHubSave))
	s.mux.HandleFunc("POST /api/media", s.requireAdminAuth(s.handleMediaCreate))
	s.mux.HandleFunc("GET /api/media/{id}", s.handleMediaDetail)
	s.mux.HandleFunc("PUT /api/media/{id}", s.requireAdminAuth(s.handleMediaUpdateFull))
	s.mux.HandleFunc("DELETE /api/media/{id}", s.requireAdminAuth(s.handleMediaDelete))
	s.mux.HandleFunc("GET /api/media/{id}/files", s.handleMediaFiles)
	s.mux.HandleFunc("GET /api/media/{id}/metadata/raw", s.handleMediaMetadataSnapshot)
	s.mux.HandleFunc("GET /api/media/{id}/metadata/seasons", s.handleMediaSeasonMetadataSnapshots)
	s.mux.HandleFunc("POST /api/media/{id}/enrich", s.requireAdminAuth(s.handleMediaEnrich))
	s.mux.HandleFunc("PUT /api/media/{id}/metadata", s.requireAdminAuth(s.handleMediaMetadataUpdate))
	s.mux.HandleFunc("GET /api/library/duplicates", s.handleDuplicates)
	s.mux.HandleFunc("GET /api/library/missing-episodes", s.handleMissingEpisodes)
	s.mux.HandleFunc("GET /api/library/corrupted", s.handleCorruptedFiles)
	s.mux.HandleFunc("GET /api/quality/report", s.handleQualityReport)
	s.mux.HandleFunc("GET /api/dashboard/stats", s.handleDashboardStats)
	s.mux.HandleFunc("GET /api/disks", s.handleDisksList)
	s.mux.HandleFunc("POST /api/disks/scan", s.requireAdminAuth(s.handleDisksScan))
	s.mux.HandleFunc("GET /api/scan", s.handleScan)
	s.mux.HandleFunc("POST /api/ingest", s.requireAdminAuth(s.handleIngest))
	s.mux.HandleFunc("POST /api/index", s.requireAdminAuth(s.handleIndex))
	s.mux.HandleFunc("POST /api/index/preview", s.requireAdminAuth(s.handleIndexPreview))
	s.mux.HandleFunc("POST /api/library/classify-origins", s.requireAdminAuth(s.handleClassifyOrigins))
	s.mux.HandleFunc("POST /api/search/sync", s.requireAdminAuth(s.handleSearchSync))
	s.mux.HandleFunc("POST /api/metadata/lookup", s.handleMetadataLookup)
	s.mux.HandleFunc("GET /api/tmdb/candidates", s.handleTMDBCandidates)
	s.mux.HandleFunc("POST /api/media/verify", s.handleMediaVerify)
	s.mux.HandleFunc("POST /api/media/inspect", s.handleMediaInspect)
	s.mux.HandleFunc("POST /api/media/thumbnail", s.handleThumbnail)
	s.mux.HandleFunc("POST /api/media/checksums", s.handleChecksums)
	s.mux.HandleFunc("POST /api/migration/preview", s.requireAdminAuth(s.handleMigrationPreview))
	s.mux.HandleFunc("POST /api/migration/copy", s.requireAdminAuth(s.handleMigrationCopy))

	// Transfer (USB, Android, iOS) Endpoints
	s.mux.HandleFunc("GET /api/transfer/devices", s.handleTransferDevices)
	s.mux.HandleFunc("GET /api/transfer/device-apps", s.handleTransferDeviceApps)
	s.mux.HandleFunc("GET /api/transfer/device-app-folders", s.handleTransferDeviceAppFolders)
	s.mux.HandleFunc("POST /api/transfer/copy", s.handleTransferCopy)
	s.mux.HandleFunc("GET /api/transfer/jobs", s.handleTransferJobsList)
	s.mux.HandleFunc("GET /api/transfer/job/{id}", s.handleTransferJobGet)
	s.mux.HandleFunc("POST /api/transfer/cancel/{id}", s.handleTransferJobCancel)
	s.mux.HandleFunc("GET /api/transfer/events", s.handleTransferEvents)

	// Phase 3 — File Browser & Eject
	s.mux.HandleFunc("GET /api/transfer/browse", s.handleTransferBrowse)
	s.mux.HandleFunc("POST /api/transfer/mkdir", s.handleTransferMkdir)
	s.mux.HandleFunc("POST /api/transfer/eject", s.handleTransferEject)

	s.mux.HandleFunc("GET /api/stream", s.handleStream)
	s.mux.HandleFunc("GET /api/stream/image", s.handleStreamImage)
	s.mux.HandleFunc("GET /api/stream/file/{id}", s.handleStreamByID)
	s.mux.HandleFunc("GET /api/stream/file/{id}/preview", s.handleFilePreview)
	s.mux.HandleFunc("GET /api/stream/file/{id}/subtitles", s.handleFileSubtitles)
	s.mux.HandleFunc("GET /api/stream/file/{id}/subtitles/{subId}", s.handleFileSubtitleStream)

	// TMDB Control Panel & Integration Endpoints
	s.mux.HandleFunc("GET /api/tmdb/settings", s.requireAdminAuth(s.handleTMDBSettingsGet))
	s.mux.HandleFunc("GET /api/tmdb/queue", s.requireAdminAuth(s.handleTMDBQueueGet))
	s.mux.HandleFunc("GET /api/tmdb/usage/history", s.requireAdminAuth(s.handleTMDBUsageHistory))
	s.mux.HandleFunc("POST /api/tmdb/queue", s.requireAdminAuth(s.handleTMDBQueueCreate))
	s.mux.HandleFunc("POST /api/tmdb/queue/{id}/cancel", s.requireAdminAuth(s.handleTMDBQueueCancel))
	s.mux.HandleFunc("PUT /api/tmdb/settings", s.requireAdminAuth(s.handleTMDBSettingsUpdate))
	s.mux.HandleFunc("GET /api/tmdb/stats", s.requireAdminAuth(s.handleTMDBStatsGet))
	s.mux.HandleFunc("GET /api/tmdb/modules", s.requireAdminAuth(s.handleTMDBModulesGet))
	s.mux.HandleFunc("PUT /api/tmdb/modules", s.requireAdminAuth(s.handleTMDBModulesUpdate))
	s.mux.HandleFunc("POST /api/tmdb/test", s.requireAdminAuth(s.handleTMDBTestConnection))
	s.mux.HandleFunc("GET /api/tmdb/configuration", s.handleTMDBConfigurationGet)
	s.mux.HandleFunc("GET /api/tmdb/preview/{id}", s.handleTMDBPreviewGet)

	// System Directory Tree Explorer & Admin Auth Endpoints
	s.mux.HandleFunc("GET /api/system/drives", s.requireAdminAuth(s.handleSystemDrives))
	s.mux.HandleFunc("GET /api/system/browse", s.requireAdminAuth(s.handleSystemBrowse))
	s.mux.HandleFunc("POST /api/system/open-file-location", s.handleOpenFileLocation)
	s.mux.HandleFunc("POST /api/admin/maintenance/clean-genres", s.requireAdminAuth(s.handleCleanGenres))
	s.mux.HandleFunc("POST /api/admin/login", s.handleAdminLogin)
	s.mux.HandleFunc("GET /api/admin/session", s.handleAdminSession)
	s.mux.HandleFunc("POST /api/admin/logout", s.handleAdminLogout)

	// Static assets serving for downloaded posters/banners/thumbnails
	if s.config.AssetImageDir != "" {
		_ = os.MkdirAll(s.config.AssetImageDir, 0o755)
		fileServer := http.StripPrefix("/assets/images/", http.FileServer(http.Dir(s.config.AssetImageDir)))
		s.mux.Handle("GET /assets/images/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
		}))
	}
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
