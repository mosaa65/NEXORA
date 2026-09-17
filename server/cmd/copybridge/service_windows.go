//go:build windows

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"nexora/server/internal/copybridge"
)

const (
	serviceName        = "NEXORACopyBridge"
	serviceDisplayName = "NEXORA USB Copy Bridge Service"
	serviceDescription = "NEXORA Local USB Transfer Bridge for client workstations"
)

type nexoraBridgeService struct{}

func (m *nexoraBridgeService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	cfg := copybridge.LoadConfig()
	service, err := copybridge.NewService(cfg)
	if err != nil {
		slog.Error("copy bridge init failed", slog.Any("error", err))
		return false, 1
	}
	defer service.Close()

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           copybridge.NewServer(cfg, service),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("NEXORA copy bridge Windows service running", slog.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("copy bridge http server failed", slog.Any("error", err))
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = httpServer.Shutdown(shutdownCtx)
			cancel()
			return false, 0
		default:
			slog.Warn("unexpected control request", slog.Uint64("cmd", uint64(c.Cmd)))
		}
	}
	return false, 0
}

func isWindowsService() bool {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isSvc
}

func runWindowsService() error {
	return svc.Run(serviceName, &nexoraBridgeService{})
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s is already installed", serviceName)
	}

	s, err = m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		StartType:   mgr.StartAutomatic,
	}, "-service")
	if err != nil {
		return fmt.Errorf("create service %s: %w", serviceName, err)
	}
	defer s.Close()

	slog.Info("Service installed successfully", slog.String("name", serviceName), slog.String("path", exePath))
	return nil
}

func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err == nil {
		slog.Info("Stopping service before deletion", slog.Any("status", status))
		time.Sleep(1 * time.Second)
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service %s: %w", serviceName, err)
	}

	slog.Info("Service uninstalled successfully", slog.String("name", serviceName))
	return nil
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service %s: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("start service %s: %w", serviceName, err)
	}

	slog.Info("Service started successfully", slog.String("name", serviceName))
	return nil
}

func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service %s: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("stop service %s: %w", serviceName, err)
	}

	slog.Info("Service stop signal sent", slog.Any("status", status))
	return nil
}
