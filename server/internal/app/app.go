package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"nexora/server/internal/api"
	"nexora/server/internal/config"
	"nexora/server/internal/db"
	"nexora/server/internal/media"
	"nexora/server/internal/metadata"
	"nexora/server/internal/migration"
	"nexora/server/internal/quality"
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

func Run() {
	// Initialize default structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	sqlDB, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("open database failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer sqlDB.Close()

	migrationsDir := resolveMigrationsDir(cfg.MigrationsDir)
	if err := db.RunMigrations(ctx, sqlDB, migrationsDir); err != nil {
		slog.Error("run migrations failed", slog.Any("error", err))
		os.Exit(1)
	}

	repository := db.NewRepository(sqlDB)
	repository.SetAssetImageDir(cfg.AssetImageDir)
	scannerService := scanner.New(scanner.Options{Workers: cfg.ScanWorkers})
	searchClient := search.NewClient(search.Config{
		Host:   cfg.MeiliHost,
		APIKey: cfg.MeiliAPIKey,
		Index:  cfg.MeiliIndex,
	})
	metadataService := metadata.NewService(
		metadata.NewTMDBClient(metadata.TMDBConfig{
			APIKey:       cfg.TMDBAPIKey,
			BearerToken:  cfg.TMDBBearer,
			BaseURL:      cfg.TMDBBaseURL,
			ImageBaseURL: cfg.TMDBImageURL,
			ImageDir:     cfg.AssetImageDir,
		}),
		metadata.NewMALClient(metadata.MALConfig{
			ClientID:    cfg.MALClientID,
			AccessToken: cfg.MALAccessToken,
			BaseURL:     cfg.MALBaseURL,
			ImageDir:    cfg.AssetImageDir,
		}),
	)
	mediaProcessor := media.NewProcessor(cfg.FFmpegPath, cfg.FFprobePath)
	migrationService := migration.New(migration.Options{})
	qualityService := quality.NewService(sqlDB, cfg.FFmpegPath, cfg.FFprobePath)

	if len(cfg.MediaRoots) > 0 {
		eventWatcher := scanner.NewEventWatcher(scannerService, cfg.WatchRecursive)
		go func() {
			err := eventWatcher.Watch(ctx, cfg.MediaRoots, func(event scanner.Event) error {
				if event.File != nil {
					if _, err := repository.IngestScannedFiles(ctx, []scanner.FileInfo{*event.File}); err != nil {
						slog.Warn("media ingest failed", slog.String("kind", string(event.Kind)), slog.String("path", event.Path), slog.Any("error", err))
						return nil
					}
					slog.Info("media indexed", slog.String("kind", string(event.Kind)), slog.String("path", event.Path), slog.String("title", event.File.Parsed.Title))
				} else {
					slog.Info("media event", slog.String("kind", string(event.Kind)), slog.String("path", event.Path))
				}
				return nil
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("media watcher stopped", slog.Any("error", err))
			}
		}()
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(cfg, repository, scannerService, searchClient, metadataService, mediaProcessor, migrationService, qualityService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("NEXORA API listening", slog.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", slog.Any("error", err))
	} else {
		slog.Info("server shut down gracefully")
	}
}

func resolveMigrationsDir(dir string) string {
	if dir == "" {
		dir = "migrations"
	}
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	serverRelative := filepath.Join("server", dir)
	if _, err := os.Stat(serverRelative); err == nil {
		return serverRelative
	}
	return dir
}
