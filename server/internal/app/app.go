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
	"nexora/server/internal/transfer"
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
	scannerService := scanner.New(scanner.Options{
		Workers:   cfg.ScanWorkers,
		QueueSize: cfg.ScanQueueSize,
		Discovery: scanner.DiscoveryConfig{
			FollowSymlinks: cfg.FollowSymlinks,
			IgnoreHidden:   cfg.IgnoreHidden,
			IgnoreDirs:     cfg.IgnoreDirs,
			MaxDepth:       cfg.ScanMaxDepth,
		},
		ProgressInterval: cfg.ScanProgressInterval,
	})
	scannerService.SetConfiguredRoots(cfg.MediaRoots)
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

	// Central server does NOT expose its local USB devices to the network.
	// USB transfers are handled locally and securely on each workstation via NEXORA Copy Bridge.
	var transferService *transfer.Service
	if os.Getenv("NEXORA_SERVER_USB_TRANSFER") == "true" {
		transferService = transfer.NewService(transfer.Options{
			AndroidTargetFolder: cfg.AndroidTargetFolder,
			IOSBundleID:         cfg.IOSBundleID,
		})
	}

	// The resolution session is the single write path for every file that reaches
	// the catalogue: the on-demand scan, the debounced watcher, and the scheduled
	// reconciliation sweep all share it, so no path can bypass entity resolution.
	resolutionSession := db.NewResolutionSession()

	if len(cfg.MediaRoots) > 0 {
		logInterruptedScans(ctx, repository)

		// The watcher is the FAST PATH. It is debounced and stability-checked so
		// an in-progress download is ingested once, after it stops growing.
		eventWatcher := scanner.NewEventWatcherWithOptions(scannerService, scanner.WatcherOptions{
			Recursive:          cfg.WatchRecursive,
			Debounce:           cfg.WatchDebounce,
			Stability:          cfg.WatchStability,
			RootRetryInterval:  cfg.WatchRetryInterval,
			ErrorRetryInterval: cfg.WatchErrorRetry,
			Logger:             slog.Default(),
		})
		go func() {
			err := eventWatcher.Watch(ctx, cfg.MediaRoots, func(event scanner.Event) error {
				handleWatchEvent(ctx, repository, resolutionSession, cfg, event)
				return nil
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("media watcher stopped", slog.Any("error", err))
			}
		}()

		// Reconciliation is the SOURCE OF TRUTH. fsnotify can overflow its buffer,
		// lose events on network shares, or miss everything that happened while the
		// server was down; a periodic incremental sweep settles that drift.
		if cfg.ReconcileInterval > 0 {
			scheduler := scanner.NewWatchScheduler(scannerService, scanner.WatchSchedulerOptions{
				Interval: cfg.ReconcileInterval,
				Roots:    cfg.MediaRoots,
				Mode:     scanner.ModeReconcile,
				Logger:   slog.Default(),
			}, func(roots []string) ([]scanner.KnownFile, error) {
				return repository.ListKnownFiles(ctx, roots)
			}, func(result scanner.ReconcileResult, decisions []scanner.ReconcileDecision) error {
				return nil
			})
			emit := buildReconcileEmitter(ctx, repository, resolutionSession)
			go func() {
				if err := scheduler.Run(ctx, emit); err != nil && !errors.Is(err, context.Canceled) {
					slog.Error("reconciliation scheduler stopped", slog.Any("error", err))
				}
			}()
		}
	}

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: api.NewServer(cfg, repository, scannerService, searchClient, metadataService,
			mediaProcessor, migrationService, qualityService, transferService, resolutionSession),
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
