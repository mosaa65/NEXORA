# NEXORA

NEXORA is a LAN-first smart media library manager for internet lounges and gaming cafes. The first implementation phase creates the Go backend foundation: PostgreSQL schema migrations, file tree scanning, Arabic/English filename parsing, filesystem event watching, and Meilisearch sync.

## Current Status

- `server/`: Go API backend skeleton with Phase 1 core.
- `client/`: React/Vite folder structure placeholder for later UI phases.
- PostgreSQL schema is in `server/migrations/0001_init_schema.sql`.
- Parser tests live in `server/internal/scanner/parser_test.go`.

## Backend Setup

Prerequisites:

- Go 1.22+
- PostgreSQL
- Meilisearch

Environment variables can be copied from `.env.example`.

```powershell
cd server
$env:NEXORA_DATABASE_URL="postgres://nexora:nexora@localhost:5432/nexora?sslmode=disable"
$env:NEXORA_MEILI_HOST="http://127.0.0.1:7700"
$env:NEXORA_MEDIA_ROOTS="D:\Media;E:\Media"
go mod tidy
go test ./...
go run ./cmd/api
```

## API Endpoints

- `GET /health`: checks API and PostgreSQL connectivity.
- `GET /api/scan?root=D:\Media`: scans video files and returns parsed metadata.
- `POST /api/ingest?root=D:\Media`: scans video files and stores basic media records in PostgreSQL.
- `POST /api/search/sync?limit=1000`: reads media items from PostgreSQL and indexes them into Meilisearch.

## Phase 1 Notes

- The parser supports English patterns like `S04E05`, `1x02`, `EP 12`, and Arabic patterns like `الحلقة 1086` and `الموسم 2 الحلقة 7`.
- The watcher uses filesystem events through `fsnotify` and can recursively watch media roots.
- Image caching, FFmpeg/MediaInfo processing, HTTP range streaming, migration wizard, and React UI are intentionally left for the next phases.
