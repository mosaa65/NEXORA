package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/migration"
	"nexora/server/internal/quality"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type mockRepo struct {
	categories []db.CategorySummary
	mediaItem  *db.MediaItemDetail
	stats      *db.DashboardStats
	disks      []db.StorageDisk
	verifiedID int64
	verified   media.VerifyResult
}

func (m *mockRepo) Health(ctx context.Context) (db.Health, error) {
	return db.Health{DatabaseOK: true, CheckedAt: time.Now().UTC()}, nil
}
func (m *mockRepo) IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error) {
	return db.IngestResult{Scanned: len(files), Imported: len(files)}, nil
}
func (m *mockRepo) ClassifyOriginsFromPaths(ctx context.Context) (int, error) { return 0, nil }
func (m *mockRepo) ListCategories(ctx context.Context) ([]db.CategorySummary, error) {
	return m.categories, nil
}
func (m *mockRepo) CreateCategory(ctx context.Context, nameAR, nameEN, slug string) (*db.CategorySummary, error) {
	return &db.CategorySummary{ID: 10, NameAR: nameAR, NameEN: nameEN, Slug: slug}, nil
}
func (m *mockRepo) UpdateCategory(ctx context.Context, id int64, nameAR, nameEN, slug string) error {
	return nil
}
func (m *mockRepo) DeleteCategory(ctx context.Context, id int64) error {
	return nil
}
func (m *mockRepo) ListSearchDocuments(ctx context.Context, limit int) ([]search.MediaDocument, error) {
	return []search.MediaDocument{}, nil
}
func (m *mockRepo) ListVideoFiles(ctx context.Context, mediaItemID int64) ([]db.VideoFile, error) {
	return []db.VideoFile{{ID: 1, MediaItemID: mediaItemID, TitleEN: "Test File", FilePath: "test.mp4", FileSize: 1024}}, nil
}
func (m *mockRepo) GetVideoFilePath(ctx context.Context, id int64) (string, error) {
	return "test.mp4", nil
}
func (m *mockRepo) GetVideoFileIDByPath(ctx context.Context, path string) (int64, error) {
	return 1, nil
}
func (m *mockRepo) UpdateVideoTechnicalDetails(ctx context.Context, id int64, details media.InspectResult) error {
	return nil
}
func (m *mockRepo) UpdateVideoVerification(ctx context.Context, id int64, result media.VerifyResult) error {
	m.verifiedID = id
	m.verified = result
	return nil
}
func (m *mockRepo) ListDuplicateGroups(ctx context.Context) ([]db.DuplicateGroup, error) {
	return []db.DuplicateGroup{}, nil
}
func (m *mockRepo) ListMissingEpisodes(ctx context.Context) ([]db.MissingEpisode, error) {
	return []db.MissingEpisode{}, nil
}
func (m *mockRepo) ListCorruptedFiles(ctx context.Context) ([]db.CorruptedFile, error) {
	return []db.CorruptedFile{}, nil
}
func (m *mockRepo) CalculateChecksums(ctx context.Context, mediaItemID int64) (db.ChecksumResult, error) {
	return db.ChecksumResult{Scanned: 1, Updated: 1}, nil
}
func (m *mockRepo) GetMediaItem(ctx context.Context, id int64) (*db.MediaItemDetail, error) {
	if m.mediaItem != nil {
		return m.mediaItem, nil
	}
	return &db.MediaItemDetail{ID: id, TitleEN: "Inception", TitleAR: "إنسبشن", Type: "movie"}, nil
}
func (m *mockRepo) ListMediaItems(ctx context.Context, opts db.ListMediaOptions) (*db.MediaListResult, error) {
	return &db.MediaListResult{Total: 1, Limit: opts.Limit, Offset: opts.Offset, Items: []search.MediaDocument{{ID: 1, TitleEN: "Inception"}}}, nil
}
func (m *mockRepo) ListProviderCollections(ctx context.Context, limit int) ([]db.ProviderCollection, error) {
	return []db.ProviderCollection{{ID: 1, Slug: "tmdb-collection-1", Provider: "tmdb", ExternalID: "1", TitleAR: "سلسلة اختبار", TitleEN: "Test Collection", LocalItemCount: 2}}, nil
}
func (m *mockRepo) GetProviderCollection(ctx context.Context, slug string) (*db.ProviderCollection, error) {
	return &db.ProviderCollection{ID: 1, Slug: slug, Provider: "tmdb", ExternalID: "1", TitleEN: "Test Collection", LocalItemCount: 2}, nil
}
func (m *mockRepo) GetProviderCollectionByID(ctx context.Context, id int64) (*db.ProviderCollection, error) {
	return &db.ProviderCollection{ID: id, Slug: "tmdb-collection-1", Provider: "tmdb", ExternalID: "1", TitleEN: "Test Collection", LocalItemCount: 2}, nil
}
func (m *mockRepo) ProviderCollectionNeedsRefresh(ctx context.Context, externalID string) (bool, error) {
	return false, nil
}
func (m *mockRepo) ListProviderCollectionRefreshCandidates(ctx context.Context, limit int) ([]db.ProviderCollection, error) {
	return []db.ProviderCollection{}, nil
}
func (m *mockRepo) ListProviderCollectionMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.ProviderCollection, *db.MediaListResult, error) {
	return &db.ProviderCollection{ID: 1, Slug: slug, TitleEN: "Test Collection", LocalItemCount: 2}, &db.MediaListResult{Total: 2, Limit: opts.Limit, Items: []search.MediaDocument{{ID: 1, TitleEN: "Part One"}, {ID: 2, TitleEN: "Part Two"}}}, nil
}
func (m *mockRepo) ListProviderCollectionParts(ctx context.Context, slug string) (*db.ProviderCollection, []db.ProviderCollectionPart, error) {
	return &db.ProviderCollection{ID: 1, Slug: slug, TitleEN: "Test Collection"}, []db.ProviderCollectionPart{{ExternalID: "1", Title: "Part One", Local: true, MediaID: 1}, {ExternalID: "2", Title: "Part Two", Local: false}}, nil
}
func (m *mockRepo) ListPeople(ctx context.Context, limit int) ([]db.Person, error) {
	return []db.Person{{ID: 1, Slug: "tmdb-person-1", Provider: "tmdb", ExternalID: "1", NameAR: "شخص اختبار", NameEN: "Test Person", LocalMediaCount: 2}}, nil
}
func (m *mockRepo) GetPerson(ctx context.Context, slug string) (*db.Person, error) {
	return &db.Person{ID: 1, Slug: slug, Provider: "tmdb", ExternalID: "1", NameEN: "Test Person", LocalMediaCount: 2}, nil
}
func (m *mockRepo) ListPersonMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.Person, *db.MediaListResult, error) {
	return &db.Person{ID: 1, Slug: slug, NameEN: "Test Person", LocalMediaCount: 2}, &db.MediaListResult{Total: 2, Limit: opts.Limit, Items: []search.MediaDocument{{ID: 1, TitleEN: "Part One"}, {ID: 2, TitleEN: "Part Two"}}}, nil
}
func (m *mockRepo) SyncCatalogRelationsFromSnapshots(ctx context.Context) (*db.CatalogRelationSyncResult, error) {
	return &db.CatalogRelationSyncResult{SnapshotsProcessed: 2, CollectionsLinked: 1, CreditsLinked: 4}, nil
}
func (m *mockRepo) SaveProviderCollectionMetadata(ctx context.Context, meta metadata.CollectionResult) error {
	return nil
}
func (m *mockRepo) UpdateProviderCollectionAdmin(ctx context.Context, id int64, update db.CatalogEntityAdminUpdate) error {
	return nil
}
func (m *mockRepo) UpdatePersonAdmin(ctx context.Context, id int64, update db.CatalogEntityAdminUpdate) error {
	return nil
}
func (m *mockRepo) ListShowcases(ctx context.Context, opts db.ShowcaseOptions) (*db.ShowcaseResult, error) {
	return &db.ShowcaseResult{Context: opts.Context, Slides: []db.ShowcaseSlide{{ID: "media-1", Kind: "featured", MediaID: 1, TitleEN: "Inception"}}}, nil
}
func (m *mockRepo) ListSmartHubs(ctx context.Context, scope string) ([]db.SmartHub, error) {
	return []db.SmartHub{{Slug: "movies-animation", TitleAR: "أفلام كرتون", ItemCount: 4}}, nil
}
func (m *mockRepo) GetSmartHub(ctx context.Context, slug string) (*db.SmartHub, error) {
	return &db.SmartHub{Slug: slug, TitleAR: "أفلام كرتون", ItemCount: 4}, nil
}
func (m *mockRepo) ListSmartHubMedia(ctx context.Context, slug string, opts db.ListMediaOptions) (*db.MediaListResult, *db.SmartHub, error) {
	return &db.MediaListResult{Total: 1, Limit: opts.Limit, Offset: opts.Offset, Items: []search.MediaDocument{{ID: 1, TitleEN: "Inception"}}}, &db.SmartHub{Slug: slug, TitleAR: "أفلام كرتون", ItemCount: 1}, nil
}
func (m *mockRepo) ListSmartHubsAdmin(ctx context.Context) ([]db.SmartHub, error) {
	return []db.SmartHub{}, nil
}
func (m *mockRepo) SaveSmartHub(ctx context.Context, slug string, req db.SmartHubRequest) (*db.SmartHub, error) {
	return &db.SmartHub{Slug: slug, TitleAR: req.TitleAR}, nil
}
func (m *mockRepo) ListCollections(ctx context.Context) ([]db.Collection, error) {
	return []db.Collection{}, nil
}
func (m *mockRepo) SaveCollection(ctx context.Context, id int64, req db.CollectionRequest) (*db.Collection, error) {
	return &db.Collection{ID: id, Slug: req.Slug, TitleEN: req.TitleEN}, nil
}
func (m *mockRepo) DeleteCollection(ctx context.Context, id int64) error { return nil }
func (m *mockRepo) UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error) {
	return &search.MediaDocument{ID: id, TitleEN: meta.Title, ReleaseYear: meta.ReleaseYear}, nil
}
func (m *mockRepo) GetMetadataSnapshot(ctx context.Context, mediaItemID int64, locale string) (*db.MetadataSnapshot, error) {
	return &db.MetadataSnapshot{Provider: "tmdb", ExternalID: "1", Locale: locale, Payload: json.RawMessage(`{"id":1}`)}, nil
}
func (m *mockRepo) SaveSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, snapshots []metadata.SeasonResult) error {
	return nil
}
func (m *mockRepo) GetSeasonMetadataSnapshots(ctx context.Context, mediaItemID int64, locale string) ([]db.SeasonMetadataSnapshot, error) {
	return []db.SeasonMetadataSnapshot{}, nil
}
func (m *mockRepo) GetDashboardStats(ctx context.Context) (*db.DashboardStats, error) {
	if m.stats != nil {
		return m.stats, nil
	}
	return &db.DashboardStats{TotalMedia: 10, TotalFiles: 25, TotalStorageBytes: 1048576}, nil
}
func (m *mockRepo) ListDisks(ctx context.Context) ([]db.StorageDisk, error) {
	return m.disks, nil
}
func (m *mockRepo) SaveDisks(ctx context.Context, disks []db.StorageDisk) error {
	m.disks = disks
	return nil
}
func (m *mockRepo) CreateMediaItem(ctx context.Context, req db.CreateMediaRequest) (*search.MediaDocument, error) {
	return &search.MediaDocument{ID: 999, TitleEN: req.TitleEN, TitleAR: req.TitleAR, Type: req.Type}, nil
}
func (m *mockRepo) UpdateMediaFull(ctx context.Context, id int64, req db.UpdateMediaRequest) (*search.MediaDocument, error) {
	return &search.MediaDocument{ID: id, TitleEN: req.TitleEN, TitleAR: req.TitleAR, Type: req.Type}, nil
}
func (m *mockRepo) DeleteMediaItem(ctx context.Context, id int64) error {
	return nil
}
func (m *mockRepo) CacheLocalArtwork(sourcePath string) string { return sourcePath }
func (m *mockRepo) GetTMDBSettings(ctx context.Context) (*metadata.TMDBSettings, error) {
	settings := metadata.DefaultSettings()
	return &settings, nil
}
func (m *mockRepo) SaveTMDBSettings(ctx context.Context, settings metadata.TMDBSettings) error {
	return nil
}
func (m *mockRepo) GetTMDBUsageSummary(ctx context.Context) (*metadata.TMDBUsageSummary, error) {
	return &metadata.TMDBUsageSummary{}, nil
}
func (m *mockRepo) LogTMDBUsage(ctx context.Context, entry db.TMDBLogEntry) error { return nil }

type mockSearch struct{}

func (m *mockSearch) IndexDocuments(ctx context.Context, documents []search.MediaDocument) (search.SyncResult, error) {
	return search.SyncResult{Indexed: len(documents)}, nil
}
func (m *mockSearch) SearchDocuments(ctx context.Context, query string, limit int, filter string) (search.SearchResult, error) {
	return search.SearchResult{Query: query, Hits: []search.MediaDocument{{ID: 1, TitleEN: "Inception"}}}, nil
}

type mockMetadata struct{}

func (m *mockMetadata) Lookup(ctx context.Context, query metadata.Query) (metadata.Result, error) {
	return metadata.Result{Title: query.Title, ReleaseYear: 2010, Rating: 8.8, Provider: "tmdb"}, nil
}
func (m *mockMetadata) LookupByExternalID(ctx context.Context, query metadata.Query, externalID string) (metadata.Result, error) {
	return metadata.Result{Title: "Inception", ExternalID: externalID, Locale: query.Language, Provider: "tmdb"}, nil
}
func (m *mockMetadata) LookupSeasonByExternalID(ctx context.Context, externalID string, seasonNumber int, language string) (metadata.SeasonResult, error) {
	return metadata.SeasonResult{Provider: "tmdb", ExternalID: externalID, Locale: language, SeasonNumber: seasonNumber, RawPayload: json.RawMessage(`{"episodes":[]}`)}, nil
}
func (m *mockMetadata) LookupCollectionByExternalID(ctx context.Context, externalID, language string) (metadata.CollectionResult, error) {
	return metadata.CollectionResult{Provider: "tmdb", ExternalID: externalID, Locale: language, Title: "Test Collection", PartsCount: 2, RawPayload: json.RawMessage(`{"id":1,"parts":[]}`)}, nil
}
func (m *mockMetadata) GetTMDBSettings() metadata.TMDBSettings         { return metadata.DefaultSettings() }
func (m *mockMetadata) SetTMDBSettings(settings metadata.TMDBSettings) {}
func (m *mockMetadata) FetchTMDBConfiguration(ctx context.Context) (*metadata.TMDBRemoteConfig, error) {
	return &metadata.TMDBRemoteConfig{}, nil
}

type mockProcessor struct{}

func (m *mockProcessor) Verify(ctx context.Context, path string) (media.VerifyResult, error) {
	return media.VerifyResult{Path: path, Healthy: true}, nil
}
func (m *mockProcessor) Inspect(ctx context.Context, path string) (media.InspectResult, error) {
	return media.InspectResult{Path: path, Duration: 120, Resolution: "1080p"}, nil
}
func (m *mockProcessor) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, at time.Duration) (string, error) {
	return outputPath, nil
}

type mockMigration struct{}

func (m *mockMigration) Preview(ctx context.Context, root string) (migration.PreviewResult, error) {
	return migration.PreviewResult{Root: root}, nil
}
func (m *mockMigration) Copy(ctx context.Context, request migration.CopyRequest) (migration.CopyResult, error) {
	return migration.CopyResult{}, nil
}

type mockQuality struct{}

func (m *mockQuality) GenerateReport(ctx context.Context) (*quality.QualityReport, error) {
	return &quality.QualityReport{
		GeneratedAt:          time.Now().UTC(),
		TotalMedia:           10,
		TotalFiles:           50,
		TotalSizeBytes:       1000000000,
		TotalWastedBytes:     50000000,
		DuplicateGroupsCount: 1,
		MissingEpisodesCount: 2,
		CorruptedFilesCount:  0,
	}, nil
}
func (m *mockQuality) FindDuplicates(ctx context.Context) ([]db.DuplicateGroup, int64, error) {
	return []db.DuplicateGroup{}, 0, nil
}
func (m *mockQuality) FindMissingEpisodes(ctx context.Context) ([]quality.MissingEpisodeDetail, error) {
	return []quality.MissingEpisodeDetail{}, nil
}
func (m *mockQuality) VerifyFile(ctx context.Context, filePath string) (bool, string, error) {
	return true, "", nil
}
func (m *mockQuality) ListCorruptedFiles(ctx context.Context) ([]quality.CorruptedFileDetail, error) {
	return []quality.CorruptedFileDetail{}, nil
}

func setupTestServer() http.Handler {
	cfg := config.Config{}
	repo := &mockRepo{
		categories: []db.CategorySummary{
			{ID: 1, NameAR: "أفلام", NameEN: "Movies", Slug: "movies", MediaCount: 5, FileCount: 5},
		},
		disks: []db.StorageDisk{
			{DiskLetter: "D", DiskLabel: "Media HDD", TotalSpace: 2000000000000, FreeSpace: 500000000000, IsActive: true},
		},
	}
	sc := scanner.New(scanner.Options{})
	searchSvc := &mockSearch{}
	metaSvc := &mockMetadata{}
	proc := &mockProcessor{}
	mig := &mockMigration{}
	qual := &mockQuality{}

	return NewServer(cfg, repo, sc, searchSvc, metaSvc, proc, mig, qual)
}

func TestHealthEndpoint(t *testing.T) {
	handler := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if res["ok"] != true {
		t.Errorf("expected ok=true, got: %v", res["ok"])
	}
}

func TestMediaListAndDetailEndpoints(t *testing.T) {
	handler := setupTestServer()

	// 1. Test GET /api/media
	req := httptest.NewRequest(http.MethodGet, "/api/media?category=movies", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/media, got: %d", rec.Code)
	}

	var listRes db.MediaListResult
	if err := json.NewDecoder(rec.Body).Decode(&listRes); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listRes.Total != 1 || len(listRes.Items) != 1 {
		t.Errorf("expected 1 item, got: %d", listRes.Total)
	}

	// 2. Test GET /api/media/1
	reqDetail := httptest.NewRequest(http.MethodGet, "/api/media/1", nil)
	recDetail := httptest.NewRecorder()
	handler.ServeHTTP(recDetail, reqDetail)

	if recDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/media/1, got: %d", recDetail.Code)
	}

	var detailRes db.MediaItemDetail
	if err := json.NewDecoder(recDetail.Body).Decode(&detailRes); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailRes.TitleEN != "Inception" {
		t.Errorf("expected TitleEN=Inception, got: %s", detailRes.TitleEN)
	}
}

func TestDashboardStatsEndpoint(t *testing.T) {
	handler := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var stats db.DashboardStats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}

	if stats.TotalMedia != 10 {
		t.Errorf("expected TotalMedia=10, got: %d", stats.TotalMedia)
	}
}

func TestOfflineCatalogGraphEndpoints(t *testing.T) {
	handler := setupTestServer()
	for _, testCase := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/franchises"},
		{http.MethodGet, "/api/franchises/tmdb-collection-1/media"},
		{http.MethodGet, "/api/people"},
		{http.MethodGet, "/api/people/tmdb-person-1/media"},
		{http.MethodPost, "/api/admin/catalog/sync-relations"},
		{http.MethodPut, "/api/admin/franchises/1"},
		{http.MethodPut, "/api/admin/people/1"},
	} {
		body := bytes.NewBufferString("")
		if testCase.method == http.MethodPut {
			body = bytes.NewBufferString(`{"is_featured":true}`)
		}
		req := httptest.NewRequest(testCase.method, testCase.path, body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: expected 200, got %d: %s", testCase.method, testCase.path, rec.Code, rec.Body.String())
		}
	}
}

func TestDisksEndpoint(t *testing.T) {
	handler := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/disks", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var res struct {
		Count int              `json:"count"`
		Disks []db.StorageDisk `json:"disks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode disks: %v", err)
	}

	if res.Count != 1 || len(res.Disks) != 1 {
		t.Errorf("expected 1 disk, got: %d", res.Count)
	}
}

func TestMediaVerifyPersistsIndexedFileResult(t *testing.T) {
	cfg := config.Config{}
	repo := &mockRepo{}
	handler := NewServer(cfg, repo, scanner.New(scanner.Options{}), &mockSearch{}, &mockMetadata{}, &mockProcessor{}, &mockMigration{}, &mockQuality{})

	req := httptest.NewRequest(http.MethodPost, "/api/media/verify", bytes.NewBufferString(`{"fileId":1}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.verifiedID != 1 || !repo.verified.Healthy {
		t.Fatalf("verification was not persisted: id=%d result=%#v", repo.verifiedID, repo.verified)
	}
}

func TestIndexStreamsFilesInBoundedBatches(t *testing.T) {
	root := t.TempDir()
	for _, relativePath := range []string{"one.mp4", filepath.Join("nested", "two.mkv")} {
		path := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("media"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	repo := &mockRepo{}
	handler := NewServer(
		config.Config{MediaRoots: []string{root}},
		repo,
		scanner.New(scanner.Options{Workers: 2}),
		&mockSearch{},
		&mockMetadata{},
		&mockProcessor{},
		&mockMigration{},
		&mockQuality{},
	)
	payload, err := json.Marshal(map[string][]string{"roots": []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/index", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var result indexResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 2 || result.Imported != 2 || result.Inspected != 2 {
		t.Fatalf("unexpected streaming index result: %#v", result)
	}
}

func TestQualityReportEndpoint(t *testing.T) {
	handler := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/quality/report", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var report quality.QualityReport
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("decode quality report: %v", err)
	}

	if report.TotalMedia != 10 || report.TotalFiles != 50 {
		t.Errorf("expected 10 media, 50 files, got: %d, %d", report.TotalMedia, report.TotalFiles)
	}
	if report.DuplicateGroupsCount != 1 {
		t.Errorf("expected 1 duplicate group, got: %d", report.DuplicateGroupsCount)
	}
}
