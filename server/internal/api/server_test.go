package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/migration"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

type mockRepo struct {
	categories []db.CategorySummary
	mediaItem  *db.MediaItemDetail
	stats      *db.DashboardStats
	disks      []db.StorageDisk
}

func (m *mockRepo) Health(ctx context.Context) (db.Health, error) {
	return db.Health{DatabaseOK: true, CheckedAt: time.Now().UTC()}, nil
}
func (m *mockRepo) IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (db.IngestResult, error) {
	return db.IngestResult{Scanned: len(files), Imported: len(files)}, nil
}
func (m *mockRepo) ListCategories(ctx context.Context) ([]db.CategorySummary, error) {
	return m.categories, nil
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
func (m *mockRepo) ListDuplicateGroups(ctx context.Context) ([]db.DuplicateGroup, error) {
	return []db.DuplicateGroup{}, nil
}
func (m *mockRepo) ListMissingEpisodes(ctx context.Context) ([]db.MissingEpisode, error) {
	return []db.MissingEpisode{}, nil
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
func (m *mockRepo) UpdateMediaMetadata(ctx context.Context, id int64, meta metadata.Result) (*search.MediaDocument, error) {
	return &search.MediaDocument{ID: id, TitleEN: meta.Title, ReleaseYear: meta.ReleaseYear}, nil
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

	return NewServer(cfg, repo, sc, searchSvc, metaSvc, proc, mig)
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
