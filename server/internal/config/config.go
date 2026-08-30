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
	AssetImageDir  string
	MeiliHost      string
	MeiliAPIKey    string
	MeiliIndex     string
	TMDBAPIKey     string
	TMDBBearer     string
	TMDBBaseURL    string
	TMDBImageURL   string
	MALClientID    string
	MALAccessToken string
	MALBaseURL     string
	FFmpegPath     string
	FFprobePath    string
	ScanWorkers    int
	WatchRecursive bool
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPAddr:       envString("NEXORA_HTTP_ADDR", ":8080"),
		DatabaseURL:    envString("NEXORA_DATABASE_URL", "postgres://nexora:nexora@localhost:15432/nexora?sslmode=disable"),
		MigrationsDir:  envString("NEXORA_MIGRATIONS_DIR", "migrations"),
		MediaRoots:     envPathList("NEXORA_MEDIA_ROOTS"),
		AssetImageDir:  envString("NEXORA_ASSET_IMAGE_DIR", "assets/images"),
		MeiliHost:      strings.TrimRight(envString("NEXORA_MEILI_HOST", "http://127.0.0.1:7700"), "/"),
		MeiliAPIKey:    os.Getenv("NEXORA_MEILI_API_KEY"),
		MeiliIndex:     envString("NEXORA_MEILI_INDEX", "media_items"),
		TMDBAPIKey:     os.Getenv("NEXORA_TMDB_API_KEY"),
		TMDBBearer:     os.Getenv("NEXORA_TMDB_BEARER_TOKEN"),
		TMDBBaseURL:    strings.TrimRight(envString("NEXORA_TMDB_BASE_URL", "https://api.themoviedb.org"), "/"),
		TMDBImageURL:   strings.TrimRight(envString("NEXORA_TMDB_IMAGE_BASE_URL", "https://image.tmdb.org/t/p"), "/"),
		MALClientID:    os.Getenv("NEXORA_MAL_CLIENT_ID"),
		MALAccessToken: os.Getenv("NEXORA_MAL_ACCESS_TOKEN"),
		MALBaseURL:     strings.TrimRight(envString("NEXORA_MAL_BASE_URL", "https://api.myanimelist.net/v2"), "/"),
		FFmpegPath:     envString("NEXORA_FFMPEG_PATH", "ffmpeg"),
		FFprobePath:    envString("NEXORA_FFPROBE_PATH", "ffprobe"),
		ScanWorkers:    envInt("NEXORA_SCAN_WORKERS", 8),
		WatchRecursive: envBool("NEXORA_WATCH_RECURSIVE", true),
		RedisAddr:      envString("NEXORA_REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:  os.Getenv("NEXORA_REDIS_PASSWORD"),
		RedisDB:        envInt("NEXORA_REDIS_DB", 0),
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

func loadDotEnv() {
	candidates := []string{".env", "server/.env", "../.env", "../../.env"}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				v = strings.Trim(v, `"'`+"\r")
				if _, exists := os.LookupEnv(k); !exists || os.Getenv(k) == "" {
					_ = os.Setenv(k, v)
				}
			}
		}
		break
	}
}
