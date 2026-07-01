package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MigrationsDir  string
	MediaRoots     []string
	MeiliHost      string
	MeiliAPIKey    string
	MeiliIndex     string
	ScanWorkers    int
	WatchRecursive bool
}

func Load() Config {
	return Config{
		HTTPAddr:       envString("NEXORA_HTTP_ADDR", ":8080"),
		DatabaseURL:    envString("NEXORA_DATABASE_URL", "postgres://nexora:nexora@localhost:5432/nexora?sslmode=disable"),
		MigrationsDir:  envString("NEXORA_MIGRATIONS_DIR", "migrations"),
		MediaRoots:     envPathList("NEXORA_MEDIA_ROOTS"),
		MeiliHost:      strings.TrimRight(envString("NEXORA_MEILI_HOST", "http://127.0.0.1:7700"), "/"),
		MeiliAPIKey:    os.Getenv("NEXORA_MEILI_API_KEY"),
		MeiliIndex:     envString("NEXORA_MEILI_INDEX", "media_items"),
		ScanWorkers:    envInt("NEXORA_SCAN_WORKERS", 8),
		WatchRecursive: envBool("NEXORA_WATCH_RECURSIVE", true),
	}
}

func WatchInterval() time.Duration {
	return time.Duration(envInt("NEXORA_WATCH_INTERVAL_SECONDS", 3)) * time.Second
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envPathList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, string(os.PathListSeparator))
	roots := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			roots = append(roots, part)
		}
	}
	return roots
}
