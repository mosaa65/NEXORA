# NEXORA - Developer/Agent Coding Prompt & Specifications
## Objective: Build a Smart Media Library Management System for LAN Lounge Networks

You are an expert full-stack developer. Your task is to build **NEXORA**, a premium, high-performance media library management system designed for LAN-based internet lounges and gaming cafes managing 100+ Terabytes of storage (Movies, Series, Anime, Kids, Plays, Documentaries).

---

### 1. Architectural Concept: Centralized Server & Thin Clients
* **Central Server (Go Backend):** Hosts the media files, scans hard drives, extracts metadata, manages PostgreSQL, index data into Meilisearch, and streams video files via HTTP.
* **Clients (React Frontend):** Lightweight Single Page Application running in client browsers. Clients search, browse, and play media files by streaming them directly from the server over the local network (Zero-installation on client PCs).

---

### 2. Technology Stack & Dependencies

#### Server / Backend
* **Language:** Go (Golang) - chosen for low memory footprint, extreme concurrency (goroutines), and blazing-fast I/O.
* **Database:** PostgreSQL (Core schema and relationships).
* **Search Engine:** Meilisearch (Typo-tolerant, Arabic/English instant search).
* **Caching:** Redis (For API response caching and active sessions).
* **Media Processing:** 
  * **FFmpeg:** For generating thumbnails and video verification.
  * **MediaInfo:** CLI/Library to read codecs, audio tracks, and embedded subtitles.

#### Client / Frontend
* **Framework:** React.js (built via Vite).
* **Styling:** Tailwind CSS + Framer Motion (for smooth micro-animations).
* **Design Aesthetic:** Deep premium look (Royal Purple, Electric Blue, Charcoal Black) with Glassmorphism and subtle neon glow cards (matching Netflix/Plex UI styling).
* **Media Player:** Plyr.js or Video.js with full support for local streaming, WebVTT subtitles, and audio track switching.

---

### 3. Core Engine Specifications to Implement

#### A. Smart Scanner & Regex Parser (Go Backend)
Build a parser that extracts title, season, episode, and quality from file names. It must support both Arabic and English names, including mixed styles.
* **Examples to support:**
  * `Attack.on.Titan.S04E05.1080p.mkv` -> Title: "Attack on Titan", Season: 4, Episode: 5, Resolution: "1080p"
  * `ون بيس الحلقة 1086 4k.mp4` -> Title: "ون بيس", Season: 1, Episode: 1086, Resolution: "4K"
* **Live Folder Watching:** Implement directory monitoring using native Go libraries or file system events to instantly index new files without a full disk rescan.

#### B. Offline Metadata Scraper
* Integrate TMDB API and MyAnimeList API to fetch banners, posters, genres, release year, cast, and plots.
* Download and save all image assets (posters, banners) locally on the server (e.g., in `/assets/images/`) so the system works 100% offline.

#### C. Local HTTP Streaming Server
* Build an HTTP file server in Go that supports **HTTP Range Requests** (crucial for video scrubbing/seeking).
* Extract internal subtitle tracks and convert them to WebVTT format on-the-fly to stream them alongside the video to the browser.

#### D. Migration Wizard & Copy Engine
* **Migration Wizard:** Scan unstructured drives and present a preview diff showing how the files will be structured on disk. 
* **Copy Engine:** Create a multi-threaded copy engine in Go with pause/resume support, ETA calculation, transfer speeds, and SHA-256 checksum validation to prevent file corruption.

---

### 4. Database Schema (PostgreSQL)

```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name_ar VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE media_items (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES categories(id),
    title_ar VARCHAR(255),
    title_en VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'movie', 'series', 'anime'
    plot_ar TEXT,
    plot_en TEXT,
    release_year INT,
    rating NUMERIC(3, 1),
    poster_path VARCHAR(500),
    banner_path VARCHAR(500),
    genres VARCHAR(255)[],
    status VARCHAR(50) DEFAULT 'completed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE seasons (
    id SERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE CASCADE,
    season_number INT NOT NULL,
    title_ar VARCHAR(150),
    title_en VARCHAR(150),
    UNIQUE(media_item_id, season_number)
);

CREATE TABLE video_files (
    id SERIAL PRIMARY KEY,
    media_item_id INT REFERENCES media_items(id) ON DELETE CASCADE,
    season_id INT REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INT,
    title_ar VARCHAR(255),
    title_en VARCHAR(255),
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    duration INT,
    resolution VARCHAR(50),
    video_codec VARCHAR(50),
    audio_tracks JSONB,
    subtitles JSONB,
    checksum VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE storage_disks (
    id SERIAL PRIMARY KEY,
    disk_letter CHAR(1) UNIQUE NOT NULL,
    disk_label VARCHAR(100),
    total_space BIGINT NOT NULL,
    free_space BIGINT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_scanned TIMESTAMP
);
```

---

### 5. Suggested Folder Structure

```
nexora-project/
├── server/
│   ├── cmd/
│   │   └── api/             # Go Entry Point
│   ├── internal/
│   │   ├── scanner/         # Disk Scanning, Regex Parser
│   │   ├── api/             # REST endpoints, Streaming logic
│   │   ├── metadata/        # MAL/TMDB Scrapers, Local image caching
│   │   ├── db/              # Postgres DB connections & queries
│   │   └── search/          # Meilisearch indexing service
│   ├── go.mod
│   └── main.go
├── client/
│   ├── public/
│   ├── src/
│   │   ├── components/      # Glassmorphic UI widgets, VideoPlayer
│   │   ├── pages/           # Dashboard, MediaDetails, CategoryView
│   │   ├── context/         # Auth, MediaContext
│   │   ├── assets/          # Styles, global animations
│   │   ├── App.jsx
│   │   └── main.jsx
│   ├── package.json
│   └── tailwind.config.js
└── README.md
```

---

### 6. Phase-by-Phase Development Instructions

#### Phase 1: Database & Scanner Core (Go)
1. Initialize the PostgreSQL schema.
2. Build the File Tree Scanner in Go to list files from directory paths.
3. Write robust tests for the Regex name parser to verify English and Arabic titles.
4. Integrate Meilisearch: write a sync task that indexes database updates into Meilisearch.

#### Phase 2: Metadata Fetcher & Media Processor (Go)
1. Create the Scraper client using TMDB/MAL. Cache images locally to the disk.
2. Integrate `ffmpeg` wrappers to verify video health and generate thumbnails.
3. Implement HTTP Range requests handler for fast seeking/streaming.

#### Phase 3: React Frontend Setup
1. Scaffold the React app using Vite & Tailwind CSS.
2. Build the glassmorphic layouts: sidebar navigation, hero cards, neon border hover states.
3. Create the Category page and implement instant search query forwarding to Meilisearch.
4. Build the Media Details page: seasons/episodes grid, descriptions, ratings.

#### Phase 4: Video Player & Integration
1. Integrate Plyr.js. Configure it to stream videos from the Go API.
2. Implement audio track switching and subtitle injection.
3. Build the Admin Dashboard displaying disk statuses, duplicate file scanner, and copy migration tasks.
4. Run integration tests (Multi-client concurrent streaming).
