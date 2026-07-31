# 1. Project Overview

* **Project Name:** NEXORA
* **Project Type:** Smart Media Library Management System & Local LAN Streaming Platform
* **Business Domain:** Entertainment, Media Management, Local Area Network (LAN) Gaming Lounge Operations
* **Primary Purpose:** Provide a high-performance, centralized media indexing, metadata scraping, instant searching, and HTTP Range streaming platform for 100+ Terabyte media collections without client-side software installations.
* **Target Users:** Internet lounge cashiers, system administrators, and lounge customers accessing media terminals over local Wi-Fi/LAN networks.
* **Main Business Value:** Eliminates manual disk directory navigation, automates metadata enrichment and offline caching, prevents playback failures on corrupted files, and enables zero-latency local video streaming.

---

# 2. Resume Summary

* Architected a LAN-first media management and streaming platform using Go and React to index and stream 100+ TB media libraries across local lounge networks with sub-10ms search latencies.
* Implemented a concurrent Go disk scanner utilizing worker pools, regex title parsing, and `fsnotify` file-system event watchers to ingest and normalize Arabic/English media releases.
* Developed an HTTP Range streaming server supporting `206 Partial Content` requests, dynamic WebVTT subtitle extraction via FFmpeg, and multi-threaded file migrations validated with SHA-256 checksums.
* Built a glassmorphic React 18 Single Page Application featuring instant typo-tolerant search via Meilisearch v1.11, Plyr video player integration, and automated offline metadata caching.

---

# 3. Core Features

* **Concurrent Hard Drive Scanner & Parser:** High-speed directory traversal using Go worker pools and regex title parsing for English and Arabic release patterns (e.g., `S04E05` and `الحلقة 1086`).
* **Instant Typo-Tolerant Search:** Full-text bilingual search indexing powered by Meilisearch v1.11 with sub-10ms response times.
* **HTTP Range Streaming Server:** Native Go video streaming supporting byte-range requests (`206 Partial Content`) for instant video seeking and scrubbing.
* **Offline Metadata Fetcher:** Automated scraping from TMDB and MyAnimeList APIs with local image asset caching for 100% offline network resilience.
* **FFmpeg Video Utilities:** Automated thumbnail generation, video integrity verification, codec extraction, and dynamic WebVTT subtitle injection.
* **Multi-Threaded Migration Engine:** Drives organization wizard and multi-threaded file copy engine featuring pause/resume, rate tracking, and SHA-256 checksum verification.
* **Glassmorphic React Client:** Netflix/Plex-styled dark theme featuring Plyr.js player integration, category filters, and disk management dashboards.

---

# 4. Technical Stack

* **Programming Languages:** Go (Golang 1.22), JavaScript (ESNext/JSX)
* **Frontend:** React 18.3, Vite 5.3, Tailwind CSS 3.4, Framer Motion 11.3, Plyr.js 3.7
* **Backend:** Go `net/http`, `github.com/jackc/pgx/v5`, `github.com/fsnotify/fsnotify` v1.8
* **Databases:** PostgreSQL 16 Alpine
* **Search Engine:** Meilisearch v1.11
* **Cache & Session Store:** Redis 7 Alpine
* **Media Utilities:** FFmpeg CLI, MediaInfo CLI
* **DevOps & Infrastructure:** Docker Compose, Local LAN Server Deployment
* **Build Tools:** Go Compiler, Vite Build Engine, npm

---

# 5. Architecture Analysis

* **Overall Architecture Style:** Centralized Server with Thin-Client Zero-Install Web Architecture.
* **Folder Organization:** 
  * Backend (`server/`): Go standard layout separating entry points (`cmd/api`) from core application logic (`internal/`).
  * Frontend (`client/`): Feature-based React structure separating components (`src/components`), contexts (`src/context`), pages (`src/pages`), and data layers (`src/data`).
* **Separation of Concerns:** 
  * `internal/scanner`: Handles directory traversal, regex parsing, and live filesystem watching.
  * `internal/api`: Manages HTTP routing, REST endpoints, and Range streaming handlers.
  * `internal/metadata`: Encapsulates external API fetching (TMDB/MAL) and disk image caching.
  * `internal/migration`: Controls drive reorganization previews and SHA-256 streaming file copies.
  * `internal/search`: Manages PostgreSQL-to-Meilisearch indexing synchronization.
* **Maintainability & Scalability:** Highly decoupled Go internal packages and client-side React components allow independent scalability of database queries, search indexing, and streaming throughput.

---

# 6. Software Engineering Practices

* **Worker Pools:** Go worker pools (`NEXORA_SCAN_WORKERS=8`) distribute filesystem traversal tasks across CPU cores.
* **Event-Driven File Watching:** Uses `fsnotify` for real-time filesystem monitoring without continuous disk polling.
* **Cryptographic Hashing:** Uses `crypto/sha256` for stream hashing to ensure data integrity during local migration operations.
* **Resilient Parsing:** Robust regular expression rules accommodating both Western and Arabic media naming patterns.
* **Offline Asset Caching:** Downloads and serves external posters/banners locally to ensure continuous operation without internet connectivity.
* **Automated SQL Migrations:** Versioned SQL migration scripts (`0001_init_schema.sql`) enforcing schema structure and initial data seeds.

---

# 7. Database Analysis

* **Database Engine:** PostgreSQL 16 Alpine
* **Schema Design:** Relational normalized schema separating media categories, media items, seasons, individual video files, and storage disk metadata.
* **Relationships & Integrity:**
  * Foreign key constraints linking `media_items` to `categories`, `seasons` to `media_items` (CASCADE delete), and `video_files` to `media_items`/`seasons`.
  * JSONB columns (`audio_tracks`, `subtitles`) for semi-structured media codec metadata.
* **Indexing Strategy:**
  * Unique index on `video_files(file_path)` to enforce single file entries.
  * Compound unique index on `media_items(type, LOWER(title_en), COALESCE(release_year, 0))` to prevent duplicate metadata creation during drive ingestion.
  * Single-column indexes on `category_id`, `title_en`, `type`, `season_id`, and `episode_number`.

---

# 8. Security Analysis

* **Network Isolation:** Designed for local area network (LAN) deployment behind firewall boundaries.
* **Path Traversal Protection:** Streaming handlers validate requested file paths against configured `NEXORA_MEDIA_ROOTS` to prevent directory traversal vulnerabilities.
* **Secret Isolation:** Database URLs, API keys, and service configurations are isolated in `.env` files and excluded from repository commits.
* **SQL Injection Prevention:** Uses `pgx/v5` parameterized queries across all database operations.
* **Input Sanitization:** Search query parameters and ingestion root inputs are sanitized prior to execution.

---

# 9. API Analysis

* **API Style:** RESTful HTTP JSON API with dedicated HTTP Range binary streaming endpoints.
* **Endpoint Structure:**
  * `GET /api/health`: System connectivity verification.
  * `GET /api/categories`: Category statistics for dashboard rendering.
  * `GET /api/search`: Meilisearch instant search proxy.
  * `GET /api/media/{id}/files`: Video file lookup by media ID.
  * `GET /api/scan`: Directory scan preview and regex metadata extraction.
  * `POST /api/ingest`: File scan and PostgreSQL persistence.
  * `POST /api/search/sync`: Batch index synchronization to Meilisearch.
  * `GET /api/stream`: Direct file streaming with HTTP `206 Partial Content` Range headers.
  * `POST /api/migration/preview` & `/copy`: Disk reorganization preview and checksum copy execution.
* **Error Handling:** Standardized HTTP status codes (200, 400, 404, 500) returning JSON error payloads.

---

# 10. Deployment & Infrastructure

* **Containerization:** Docker Compose orchestrating PostgreSQL 16, Meilisearch v1.11, and Redis 7 containers.
* **Hosting Target:** Local LAN Central Server (Windows/Linux PC).
* **Build Process:**
  * Backend: Compiled single static binary via `go build -o api.exe ./cmd/api`.
  * Frontend: Static HTML/JS bundle compiled via `vite build`.
* **Environment Configuration:** Managed via global `.env` file specifying HTTP port, database connection strings, media root paths, and external API tokens.

---

# 11. Development Quality

* **Code Organization:** Clean separation of concerns adhering to standard Go project layouts and modular React component design.
* **Maintainability:** Go backend logic is contained within private `internal/` packages, preventing internal API leakage.
* **Readability:** Clean, explicit Go function signatures and self-documenting React JSX structures.
* **Test Coverage:** Unit tests implemented for the regex title parser in `server/internal/scanner/parser_test.go`.

---

# 12. Engineering Competencies Demonstrated

* Full-Stack Web Development
* Backend Systems Engineering (Go)
* Database Modeling & Schema Design (PostgreSQL)
* Concurrent Programming & Worker Pools
* Media Streaming Protocols & HTTP Range Requests
* Full-Text Search Engine Integration (Meilisearch)
* File System Automation & Event Watching (`fsnotify`)
* Video Processing & CLI Integration (FFmpeg)
* RESTful API Design & Implementation
* Component-Based UI Development (React 18 & Tailwind CSS)
* Container Orchestration (Docker Compose)
* Data Integrity & Checksum Validation (SHA-256)

---

# 13. ATS Resume Keywords

* **Languages & Frameworks:** Go (Golang), React.js, JavaScript (ES6+), Vite, Tailwind CSS, HTML5, CSS3, SQL
* **Backend & Systems:** Concurrency, Goroutines, Worker Pools, REST API, HTTP Range Requests, Video Streaming, File System Watching (`fsnotify`), Regex Parsing
* **Databases & Search:** PostgreSQL, Meilisearch, Redis, Database Indexing, Relational Schema Design, Foreign Keys, JSONB, Data Ingestion
* **Media & Utilities:** FFmpeg, WebVTT Subtitles, Codec Extraction, Video Transcoding, SHA-256 Hashing, Data Integrity
* **DevOps & Tools:** Docker, Docker Compose, Git, Environment Configuration, npm, PowerShell, Multi-Threading

---

# 14. Suggested Resume Entry

### **NEXORA**
*Smart Media Library Management & Local LAN Streaming Platform*

* Built a centralized LAN media streaming platform in Go and React to manage and stream 100+ TB media collections across local gaming lounge networks with zero-install client browser access.
* Engineered a concurrent filesystem scanner using Go worker pools and regex algorithms to parse complex Arabic and English release filenames, integrating `fsnotify` for real-time disk change detection.
* Integrated Meilisearch v1.11 to deliver sub-10ms typo-tolerant full-text search across movie, series, and anime catalogs.
* Implemented an HTTP Range video streaming server supporting byte-range requests (`206 Partial Content`), FFmpeg thumbnail generation, dynamic WebVTT subtitle extraction, and SHA-256 verified multi-threaded file migrations.
* Developed a responsive dark glassmorphic React frontend featuring Plyr.js HTML5 video playback, category filtering, and disk space usage analytics.

**Technologies Used:** Go (Golang), React 18, Vite, Tailwind CSS, PostgreSQL 16, Meilisearch v1.11, Redis 7, FFmpeg, Docker Compose, Git.