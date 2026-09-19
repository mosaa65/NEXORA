package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nexora/server/internal/copybridge"
)

func main() {
	serviceFlag := flag.Bool("service", false, "Run as a Windows service (managed by Windows SCM)")
	installFlag := flag.Bool("install", false, "Install NEXORA Copy Bridge as a Windows service")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall NEXORA Copy Bridge Windows service")
	startFlag := flag.Bool("start", false, "Start the NEXORA Copy Bridge Windows service")
	stopFlag := flag.Bool("stop", false, "Stop the NEXORA Copy Bridge Windows service")
	debugFlag := flag.Bool("debug", false, "Run interactively in debug/console mode")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// 1. Handle service management CLI actions
	if *installFlag {
		if err := installService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("NEXORA Copy Bridge service installed successfully.")
		return
	}

	if *uninstallFlag {
		if err := uninstallService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error uninstalling service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("NEXORA Copy Bridge service uninstalled successfully.")
		return
	}

	if *startFlag {
		if err := startService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("NEXORA Copy Bridge service started successfully.")
		return
	}

	if *stopFlag {
		if err := stopService(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("NEXORA Copy Bridge service stopped successfully.")
		return
	}

	// 2. Check if running under Windows SCM (Service Control Manager)
	if *serviceFlag || isWindowsService() {
		if err := runWindowsService(); err != nil {
			slog.Error("Windows service execution failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	// 3. Interactive console mode
	if *debugFlag {
		slog.Info("Starting NEXORA copy bridge in DEBUG mode")
	}

	cfg := copybridge.LoadConfig()
	if cfg.CommandToken == "" {
		// One warning, not one per request: an operator needs to know the mutating
		// commands are unauthenticated, but the log must not become noise.
		slog.Warn("NEXORA_COPY_BRIDGE_TOKEN is not set: copy/mkdir/eject commands are " +
			"protected only by the loopback bind and the CORS policy. Set it to require " +
			"an Authorization header on mutating commands.")
	}
	service, err := copybridge.NewService(cfg)
	if err != nil {
		slog.Error("copy bridge init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer service.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           copybridge.NewServer(cfg, service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("NEXORA copy bridge listening (interactive mode)", slog.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("copy bridge http server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("copy bridge graceful shutdown failed", slog.Any("error", err))
	} else {
		slog.Info("copy bridge shut down gracefully")
	}
}
