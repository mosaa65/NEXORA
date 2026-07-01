package app

import (
	"context"
	"errors"
	"log"
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
	"nexora/server/internal/scanner"
	"nexora/server/internal/search"
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	sqlDB, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer sqlDB.Close()

	migrationsDir := resolveMigrationsDir(cfg.MigrationsDir)
	if err := db.RunMigrations(ctx, sqlDB, migrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	repository := db.NewRepository(sqlDB)
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

	if len(cfg.MediaRoots) > 0 {
		eventWatcher := scanner.NewEventWatcher(scannerService, cfg.WatchRecursive)
		go func() {
			err := eventWatcher.Watch(ctx, cfg.MediaRoots, func(event scanner.Event) error {
				if event.File != nil {
					if _, err := repository.IngestScannedFiles(ctx, []scanner.FileInfo{*event.File}); err != nil {
						log.Printf("media %s ingest failed: %s: %v", event.Kind, event.Path, err)
						return nil
					}
					log.Printf("media %s indexed: %s -> %s", event.Kind, event.Path, event.File.Parsed.Title)
				} else {
					log.Printf("media %s: %s", event.Kind, event.Path)
				}
				return nil
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("media watcher stopped: %v", err)
			}
		}()
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(cfg, repository, scannerService, searchClient, metadataService, mediaProcessor),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("NEXORA API listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
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
