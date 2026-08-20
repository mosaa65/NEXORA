<div align="center">

![Inama Soft logo](public/inama-soft-logo.ico)

# NEXORA

**LAN-First Smart Media Library Management System**

![Status: development](https://img.shields.io/badge/status-development-2563EB)
![License: MIT](https://img.shields.io/badge/license-MIT-6B7280)
![Version: 0.1.0](https://img.shields.io/badge/version-v0.1.0-7C3AED)
![Main language: Go](https://img.shields.io/badge/main%20language-Go-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-18.3-61DAFB?logo=react&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-5.3-646CFF?logo=vite&logoColor=white)
![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Meilisearch](https://img.shields.io/badge/Meilisearch-v1.11-FF5A5F?logo=meilisearch&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)

</div>

> **Hero view — Operations & Media Library Console**
>
> [![NEXORA Media Library Dashboard](Screenshots/image.png)](Screenshots/image.png)
>
> A single operational view for 100+ TB LAN media streaming, instant bilingual search, disk monitoring, and automated catalog management.

---

## Table of Contents | فهرس المحتويات

- [Overview](#overview--نظرة-عامة)
- [Quick Start](#quick-start--بدء-سريع)
- [Quick Facts](#quick-facts--حقائق-سريعة)
- [Why This Project?](#why-this-project--لماذا-هذا-المشروع)
- [System Scope](#system-scope--نطاق-النظام)
- [Screenshots](#screenshots--لقطات-الشاشة)
- [Key Features](#key-features--الميزات-الرئيسية)
- [Module Overview](#module-overview--نظرة-عامة-على-الوحدات)
- [System Workflow](#system-workflow--سير-العمل)
- [Engineering Highlights](#engineering-highlights--نقاط-الإبداع-والتميز)
- [Technology Stack](#technology-stack)
- [Architecture Overview](#architecture-overview--نظرة-عامة-على-المعمارية)
- [Engineering Decisions](#engineering-decisions--القرارات-الهندسية)
- [Performance Considerations](#performance-considerations--اعتبارات-الأداء)
- [Technical Challenges](#technical-challenges--التحديات-التقنية)
- [UI/UX Design](#uiux-design)
- [Installation & Configuration](#installation--configuration)
- [Project Structure](#project-structure)
- [Services Provided](#services-provided)
- [API Overview](#api-overview)
- [Database Overview](#database-overview--نظرة-عامة-على-قاعدة-البيانات)
- [Security](#security--الأمان)
- [Deployment](#deployment--النشر)
- [Roadmap](#roadmap--خارطة-الطريق)
- [Development Team](#development-team)

---

## Overview | نظرة عامة

🇺🇸 **English**

NEXORA is a high-performance, LAN-first media library management and streaming system designed for internet lounges, gaming cafes, and local entertainment networks. Built to manage 100+ Terabytes of storage across Movies, Series, Anime, Kids, Plays, and Documentaries, NEXORA eliminates tedious file-system navigation. It combines an ultra-fast Go backend with an offline metadata scraper, instant Meilisearch indexer, HTTP Range video streaming server, multi-threaded migration engine, and a glassmorphic React frontend with a Netflix/Plex-grade user experience.

🇸🇦 **العربية**

نظام NEXORA هو منصة متكاملة عالية الأداء لإدارة وبث مكتبات الوسائط عبر الشبكة المحلية (LAN)، صُمّم خصيصاً للاستراحات وصالات الألعاب وشبكات الترفيه المحلية التي تدير وسائط تخزين ضخمة تتجاوز 100 تيرابايت (أفلام، مسلسلات، أنمي، كرتون، وثائقيات، مسرحيات). يقضي النظام على عناء التنقل اليدوي المعقد في الأقراص الصلبة، حيث يدمج خلفية برمجية فائقة السرعة بلغة Go مع محرك تنقيب محلي عن البيانات الوصفية، ومحرك بحث فوري عبر Meilisearch، وخادم بث فيديو يدعم طلبات HTTP Range، ومحرك نقل متعدد المسارات، وواجهة ويب سينمائية زجاجية تقدم تجربة تصفح تضاهي المنصات العالمية مثل Netflix وPlex.

---

## Quick Start | بدء سريع

🇺🇸 **English**

NEXORA uses Docker Compose for core infrastructure services (PostgreSQL, Meilisearch, Redis) alongside a Go backend server and a React Vite client.

🇸🇦 **العربية**

يعتمد NEXORA على Docker Compose لتشغيل خدمات البنية التحتية (PostgreSQL وMeilisearch وRedis) إلى جانب خادم Go وواجهة React Vite.

```bash
# 1. Clone the repository
git clone https://github.com/mosaa65/NEXORA.git
cd NEXORA

# 2. Start PostgreSQL, Meilisearch, and Redis containers
docker compose up -d postgres meilisearch redis

# 3. Start the Go Backend Server
cd server
cp ../.env.example .env
go run ./cmd/api

# 4. In a new terminal, start the React Client
cd client
npm install
npm run dev
```

Configure environment variables in `.env` as documented in [Installation & Configuration](#installation--configuration).

---

## Quick Facts | حقائق سريعة

| Item | Value |
| --- | --- |
| Project type | Smart Media Library Management & Local Streaming System for LAN Networks |
| Architecture | Centralized Go Server with Thin-Client Zero-Install React Frontend |
| Frontend | React 18, Vite 5, Tailwind CSS 3, Framer Motion, Plyr.js |
| Backend | Go 1.22 (net/http, fsnotify, pgx v5), FFmpeg, MediaInfo CLI |
| Database | PostgreSQL 16 Alpine (Core schema, relationships, indexes) |
| Search & Cache | Meilisearch v1.11 (Instant Arabic/English Search), Redis 7 |
| Deployment | Local LAN Server Deployment (`http://localhost:8080` API / `http://localhost:5173` UI) |
| License | MIT License |

---

## Why This Project? | لماذا هذا المشروع؟

🇺🇸 **English**

LAN lounges and gaming centers face severe friction managing tens or hundreds of terabytes of media across loose external hard drives. Traditional file-explorer browsing leads to missing episode confusion, corrupted file playback errors, broken subtitles, and slow file access. NEXORA solves this by converting unstructured drive folders into a single centralized database with automated metadata enrichment, local thumbnail caching, on-the-fly WebVTT subtitle extraction, instant typo-tolerant search, and high-concurrency LAN streaming without requiring software installations on client terminals.

🇸🇦 **العربية**

تتواجه صالات الألعاب والاستراحات تحديات تشغيلية كبيرة عند إدارة عشرات أو مئات التيرابايتات من وسائط الترفيه الموزعة على أقراص صلبة متعددة. يتسبب التنقل التقليدي عبر مجلدات نظام التشغيل في ضياع الحلقات، وتشغيل الملفات التالفة، وغياب الترجمات، وبطء الوصول. يحل NEXORA هذه المشكلة بتحويل المجلدات العشوائية إلى قاعدة بيانات مركزية موحدة مع إثراء تلقائي للبيانات الوصفية، وتوليد الصور المصغرة، واستخراج الترجمات فورياً بتنسيق WebVTT، وبحث فوري تسامحي، وبث عالي السرعة عبر الشبكة المحلية دون الحاجة لتثبيت أي برامج على أجهزة العملاء.

---

## System Scope | نطاق النظام

🇺🇸 **English**

- **Media Cataloging:** Automated scanning of local hard drives (`D:\`, `E:\`, etc.), smart Arabic/English Regex parsing for titles, seasons, episodes, resolution, and codecs.
- **Metadata Fetching:** Automated retrieval from TMDB and MyAnimeList APIs with local image asset caching for 100% offline operation.
- **High-Performance Streaming:** Native Go HTTP Range Server supporting high-bitrate 4K/1080p video streaming, seeking, audio track identification, and dynamic WebVTT subtitle injection.
- **Disk & Storage Management:** Active disk space monitoring, S.M.A.R.T health alerts, duplicate file detection via SHA-256 checksums, and missing episode gap identification.
- **Migration & Organization:** Multi-threaded copy engine with pause/resume, progress tracking, speed calculation, and physical drive folder reorganization wizard.
- **Client Experience:** Zero-install web browser access, glassmorphic dark UI, instant Meilisearch queries, category filtering, and embedded Plyr video player.

🇸🇦 **العربية**

- **فهرسة الوسائط:** مسح تلقائي لأقراص التخزين المحلية، وتحليل اسم الملف باللغتين العربية والإنجليزية لاستخراج العنوان والموسم والحلقة والدقة والأكواد الفنية.
- **جلب البيانات الوصفية:** سحب تلقائي للبيانات والصور من TMDB وMyAnimeList وحفظها محلياً للعمل الكلي دون اتصال بالإنترنت.
- **بث عالي الأداء:** خادم Go يدعم طلبات HTTP Range لتشغيل وسحب مقاطع الفيديو بدقة 4K و1080p بسلاسة، مع التعرف على مسارات الصوت واستخراج الترجمات فورياً بتنسيق WebVTT.
- **إدارة الأقراص والتخزين:** مراقبة المساحات التخزينية، وتنبيهات صحة الأقراص، واكتشاف الملفات المكررة عبر بصمة SHA-256، وتحديد الحلقات المفقودة.
- **محرك النقل والتنظيم:** محرك نقل متعدد المسارات يدعم الإيقاف والاستكمال ومراقبة السرعة مع معالج إعادة هيكلة المجلدات في الأقراص الصلبة.
- **تجربة العملاء:** تصفح فوري عبر المتصفح بدون تثبيت، واجهة زجاجية داكنة فاخرة، بحث فوري عبر Meilisearch، تصفية الفئات، ومشغل فيديو Plyr مدمج.

---

## Screenshots | لقطات الشاشة

🇺🇸 **English**

Select any image to view it at full size. Captured directly from the running NEXORA web environment.

🇸🇦 **العربية**

اضغط على أي صورة لعرضها بالحجم الكامل. التقاطات مباشرة من بيئة عمل NEXORA.

### Dashboard & Catalog | لوحة التحكم والفهرس

[![NEXORA Dashboard](Screenshots/image.png)](Screenshots/image.png)

<sub>Dashboard — Media library statistics, storage usage, and featured recommendations</sub>

### Details & Streaming | تفاصيل العمل والبث

| Media Details | Player & Video Experience |
| --- | --- |
| [![NEXORA Media Details](Screenshots/image%20copy.png)](Screenshots/image%20copy.png)<br><sub>Media details, seasons, episodes, and plot information</sub> | [![NEXORA Admin & Streaming](<Screenshots/ChatGPT Image Jun 30, 2026, 02_44_06 AM.png>)](<Screenshots/ChatGPT Image Jun 30, 2026, 02_44_06 AM.png>)<br><sub>Video player with WebVTT subtitles, audio track support, and admin console</sub> |

---

## Key Features | الميزات الرئيسية

🇺🇸 **English**

- ⚡ **Zero-Latency Search:** Typo-tolerant instant search in Arabic and English powered by Meilisearch v1.11.
- 🚀 **Go Concurrency Engine:** Parallel filesystem scanner utilizing Go goroutines to parse 100+ TB drives in seconds.
- 🎬 **Smart Filename Parser:** Robust Regex engine capable of parsing complex release names (`Attack.on.Titan.S04E05.1080p.mkv` and `ون بيس الحلقة 1086 4k.mp4`).
- 📺 **HTTP Range Streaming:** Seamless scrubbing and seeking in HTML5 Plyr player without pre-buffering full files.
- 🌐 **100% Offline Resilience:** Local disk caching for TMDB/MAL posters, banners, and extracted WebVTT subtitles.
- 🗃️ **Multi-Threaded Migration Engine:** Local file copy with SHA-256 integrity checksum validation and pause/resume support.
- 🎨 **Glassmorphism UI:** Modern Royal Purple and Electric Blue dark-mode theme built with React, Tailwind CSS, and Framer Motion micro-animations.

🇸🇦 **العربية**

- ⚡ **بحث لحظي معدوم التأخير:** بحث فوري تسامحي يدعم العربية والإنجليزية بسرعة استجابة تقل عن 10 ميلي ثانية بواسطة Meilisearch.
- 🚀 **محرك تزامن بلغ Go:** ماسح أقراص متوازي يعتمد على الـ Goroutines لفهرسة أقراص التخزين الضخمة في ثوانٍ معدودة.
- 🎬 **محلل أسماء ذكي:** محرك Regex متطور يفهم تسميات الملفات المعقدة باللغتين العربية والإنجليزية.
- 📺 **بث مجزأ عبر HTTP Range:** تقديم ورجع فوري للفيديو في مشغل Plyr دون الحاجة لتحميل الملف كاملاً.
- 🌐 **عمل كامل بدون إنترنت:** تخزين كاش محلي للبوسترات والبانرات والترجمات على خادم السيرفر.
- 🗃️ **محرك نقل موثوق:** نقل محلي للملفات بين الأقراص مع التحقق من سلامة البصمة SHA-256 وإمكانية الإيقاف والاستكمال.
- 🎨 **واجهة زجاجية فاخرة:** تصميم سينمائي عصري باللون البنفسجي والأزرق الكهربائي مع مؤشرات دقيقة وتنقل سلس.

---

## Module Overview | نظرة عامة على الوحدات

🇺🇸 **English**

The modules below represent the core architectural building blocks of NEXORA.

🇸🇦 **العربية**

تُمثل الوحدات التالية المكونات المعمارية الأساسية لنظام NEXORA.

| Module | Purpose | Responsibilities and Main Capabilities |
| --- | --- | --- |
| Go API Core (`server/internal/api`) | REST API & Streaming Server | Handles HTTP endpoints, CORS headers, video stream chunking, and search proxies. |
| Smart Scanner (`server/internal/scanner`) | Hard Drive Scanner & Watcher | Recursively traverses directory paths, listens to `fsnotify` events, and executes Regex title parsing. |
| Metadata Service (`server/internal/metadata`) | Scraper & Asset Cache | Queries TMDB/MAL APIs, downloads posters/banners, and stores image assets locally on disk. |
| Media Processor (`server/internal/media`) | FFmpeg & Video Verification | Integrates FFmpeg CLI to generate video thumbnails, check video health, and inspect stream codecs. |
| Search Engine (`server/internal/search`) | Meilisearch Integration | Syncs PostgreSQL records into Meilisearch indexes and executes fast typo-tolerant client searches. |
| Migration Engine (`server/internal/migration`) | File Reorganization & Copy | Previews target drive directory structures, resumes interrupted copies, atomically publishes only SHA-256-verified destinations, and can remove a verified source on request. |
| Database Layer (`server/internal/db`) | Persistence & Migrations | Manages PostgreSQL connection pooling via `pgx/v5` and executes SQL migrations. |
| React Client (`client/src`) | User & Admin Web Interface | Single Page Application presenting the dashboard, category views, details modal, Plyr video player, and admin disk console. |

---

## System Workflow | سير العمل

🇺🇸 **English**

The diagram below illustrates the end-to-end flow from client video search to Go HTTP Range streaming execution.

🇸🇦 **العربية**

يوضح المخطط أدناه سير العمل الكامل من بحث الزبون عن الفيديو إلى تنفيذ البث عبر خادم Go HTTP Range.

```mermaid
flowchart LR
    A[Client Browser / PC] --> B[React SPA Interface]
    B --> C{User Action}
    C -->|Instant Search| D[Meilisearch Engine]
    C -->|Browse Media| E[Go REST API]
    C -->|Play Video| F[Go HTTP Range Streamer]
    E --> G[(PostgreSQL Database)]
    F --> H[Local Hard Disk Storage D:\ / E:\]
    H --> F
    F -->|Video Chunks + WebVTT Subtitles| I[Plyr HTML5 Player]
```

---

## Engineering Highlights | نقاط الإبداع والتميز

🇺🇸 **English**

- **Goroutine-Powered Disk Scanning:** Scanning hundreds of thousands of files across 100+ TB drives is distributed across worker pools, preventing I/O bottlenecks.
- **Zero-Copy HTTP Range Streaming:** Video files are streamed directly from disk to client socket buffers using native Go HTTP Range support without loading entire files into memory.
- **Dynamic WebVTT Subtitle Extraction:** Embedded subtitle tracks are extracted on-the-fly via FFmpeg and served as WebVTT streams alongside HTML5 video players.
- **Deterministic SHA-256 Checksum Validation:** Local drive transfers utilize streaming cryptographic hashes to verify destination integrity before marking migration tasks complete.

🇸🇦 **العربية**

- **مسح أقراص متعدد المسارات عبر Goroutines:** توزيـع مسح مئات الآلاف من الملفات عبر أقراص 100+ تيرابايت على مصفوفة عمال لتفادي اختناقات قراءة القرص.
- **بث مباشر دون استهلاك ذاكرة (Zero-Copy HTTP Range):** بث مقاطع الفيديو مباشرة من القرص الصلب إلى المقبس دون تحميل الملف كاملاً في ذاكرة السيرفر.
- **استخراج فوري للترجمات المدمجة:** قراءة مسارات الترجمة المدمجة داخل ملفات MKV وتوليد ملفات WebVTT ديناميكياً لدعم المشغل الويب.
- **تحقق قاطع من سلامة النقل عبر SHA-256:** فحص بصمة الملفات المنقولة بين الأقراص لضمان عدم تلف أي بايت أثناء النقل والتنظيم.

---

## Technology Stack

### Programming Languages

| Category | Technology | Version / Evidence |
| --- | --- | --- |
| Backend Language | Go (Golang) | 1.22 (`go.mod`) |
| Frontend Language | JavaScript (ESNext) / JSX | Node.js ecosystem, React 18 |

### Frontend & UI

| Category | Technology | Version / Evidence |
| --- | --- | --- |
| Frontend Framework | React | ^18.3.1 (`client/package.json`) |
| Build Tool | Vite | ^5.3.4 (`client/package.json`) |
| Styling & Design | Tailwind CSS & PostCSS | ^3.4.6 / ^8.4.39 |
| Micro-Animations | Framer Motion | ^11.3.8 |
| Video Player Engine | Plyr.js | ^3.7.8 |

### Backend, Database, and Infrastructure

| Category | Technology | Version / Evidence |
| --- | --- | --- |
| Database Engine | PostgreSQL | 16 Alpine (`compose.yml` port 15432) |
| Search Engine | Meilisearch | v1.11 (`compose.yml` port 7700) |
| Cache & Session Store | Redis | 7 Alpine (`compose.yml` port 6379) |
| Database Driver | pgx v5 | v5.7.1 (`server/go.mod`) |
| File System Watcher | fsnotify | v1.8.0 (`server/go.mod`) |
| Media Utilities | FFmpeg & MediaInfo | System CLI execution (`.env.example`) |

---

## Architecture Overview | نظرة عامة على المعمارية

🇺🇸 **English**

NEXORA adopts a centralized server with thin-client architecture tailored for local network environments. The Go backend acts as a single compiled binary hosting disk scanner tasks, PostgreSQL data access, Meilisearch synchronization, and HTTP video streaming. Clients connect via standard web browsers to the Vite-served React Single Page Application. No client installation is required, and all assets (posters, banners, subtitles) are served directly from the server's local directory.

🇸🇦 **العربية**

يعتمد NEXORA بنية خادم مركزي وعملاء خفاف (Centralized Server & Thin Clients) مخصصة للشبكات المحلية. يعمل خادم Go كملف تنفيذي واحد يجمع وظائف مسح الأقراص، وقاعدة بيانات PostgreSQL، ومحرك Meilisearch، وبث الفيديو. يتصل العملاء عبر المتصفحات بسيطة بواجهة React دون الحاجة لأي تثبيت، وتُقدم جميع الوسائط والأغلفة محلياً من السيرفر.

```mermaid
flowchart TB
    subgraph Client PCs / Mobile Devices
        UI[React SPA Client - Plyr Player]
    end

    subgraph Central LAN Server
        API[Go net/http Server]
        Scan[Smart Scanner & fsnotify]
        Meta[TMDB / MAL Scraper]
        FF[FFmpeg / MediaInfo Engine]
    end

    subgraph Infrastructure
        PG[(PostgreSQL 16)]
        MS[(Meilisearch v1.11)]
        RD[(Redis 7)]
        HD[(Local Hard Disks D:\ E:\])
    end

    UI <-->|HTTP REST / Streaming| API
    API <--> PG
    API <--> MS
    API <--> RD
    Scan --> HD
    FF --> HD
    API --> HD
```

---

## Engineering Decisions | القرارات الهندسية

🇺🇸 **English**

The technical decisions below reflect the architecture implemented in the NEXORA repository.

🇸🇦 **العربية**

تعكس القرارات الفنية التالية المعمارية المنفذة في مستودع مشروع NEXORA.

| Decision | Repository Evidence | Engineering Rationale |
| --- | --- | --- |
| Choice of Go over Node.js/Python for Backend | `server/cmd/api/main.go`, `go.mod` | Go offers low memory usage, compiled speed, and lightweight goroutine concurrency for parsing 100+ TB disks without system lag. |
| Rejection of Flutter Desktop in favor of React Web | `client/package.json` | Web applications eliminate client PC installation/maintenance overhead and provide superior video subtitle/audio track controls. |
| Meilisearch for Instant Search | `compose.yml`, `internal/search` | Meilisearch delivers sub-10ms typo-tolerant search across both Arabic and English media titles. |
| HTTP Range Streaming vs HLS Transcoding | `internal/api/server.go` (`GET /api/stream`) | Direct HTTP Range requests allow instant video seeking on local LAN networks without heavy GPU CPU transcoding overhead. |
| Local Image Asset Caching | `internal/metadata/cache.go` | Downloading posters and banners locally ensures 100% functionality even during internet disconnects. |

---

## Performance Considerations | اعتبارات الأداء

🇺🇸 **English**

The following performance characteristics are enforced in the codebase design.

🇸🇦 **العربية**

تُطبق خصائص الأداء التالية في تصميم الهيكل البرمجي للنظام.

| Evidence | Implementation Detail | Practical Effect / Boundary |
| --- | --- | --- |
| Database Indexing | Unique index on `video_files(file_path)` and compound index on `media_items(type, title_en, release_year)`. | Prevents duplicate file ingestion and accelerates title lookup queries. |
| Multi-Worker Scanner | `NEXORA_SCAN_WORKERS=8` concurrent scan workers configured. | Enables high-speed parallel directory traversal across multiple physical hard drives. |
| Local Image Caching | Image assets stored under `assets/images/`. | Offloads external CDN hits and ensures instant poster rendering on client devices. |
| Redis Session Caching | Redis 7 container included in `compose.yml`. | Prepares system for session management and API response caching under heavy client concurrency. |

---

## Technical Challenges | التحديات التقنية

🇺🇸 **English**

- **Mixed Arabic/English Regex Parsing:** Filenames vary wildly across pirate releases and rips. The regex parser handles complex patterns including season/episode tags (`S04E05`, `1x02`, `الحلقة 1086`).
- **High-Bitrate Video Scrubbing:** Seeking inside 50 GB 4K video files across LAN requires compliant HTTP `206 Partial Content` Range headers implemented natively in Go.
- **Offline Resilience:** Operating in internet-isolated LAN environments requires saving all external metadata assets to server disks upon initial discovery.

🇸🇦 **العربية**

- **تحليل الأسماء المختلطة (عربي/إنجليزي):** تختلف صيغ تسميات الملفات بشكل كبير. يتناول محرك Regex التسميات المعقدة بمختلف الأشكال (مثل `S04E05` و`الحلقة 1086`).
- **تقديم الفيديو عالي البتات:** تطلب التقديم الفوري لملفات 4K ذات الأحجام الكبيرة (50 جيجابايت) تطبيقاً دقيقاً لرؤوس `206 Partial Content` في Go.
- **العمل دون إنترنت:** يتطلب التشغيل في شبكات محليّة معزولة حفظ جميع الملصقات والبيانات الوصفية على خادم السيرفر فور جلبها.

---

## UI/UX Design

| Element | Tool/Library |
| --- | --- |
| Color System | Royal Purple (`#7C3AED`), Electric Blue (`#2563EB`), Charcoal Black (`#0F172A`) |
| Styling Engine | Tailwind CSS v3.4 |
| Design Pattern | Glassmorphic cards (`backdrop-blur-md`), Subtle Neon Borders |
| Icon Library | Lucide React / Custom SVG Component (`client/src/components/Icon.jsx`) |
| Animations | Framer Motion v11 micro-animations for card hovers and page transitions |
| Media Player | Plyr.js HTML5 Video Player |

---

## Installation & Configuration

1. Ensure Go 1.22+, Node.js 18+, and Docker Desktop are installed on the host system.
2. Clone the repository and navigate to the project folder:

```bash
git clone https://github.com/mosaa65/NEXORA.git
cd NEXORA
```

3. Start background infrastructure via Docker Compose:

```bash
docker compose up -d
```

4. Create `.env` file from `.env.example`:

```bash
# Server Configuration
NEXORA_HTTP_ADDR=:8080
NEXORA_DATABASE_URL=postgres://nexora:nexora@localhost:15432/nexora?sslmode=disable
NEXORA_MEDIA_ROOTS=D:\Media;E:\Media
NEXORA_MEILI_HOST=http://127.0.0.1:7700
NEXORA_ASSET_IMAGE_DIR=assets/images
```

5. Build and run the Go API backend:

```bash
cd server
go mod tidy
go test ./...
go run ./cmd/api
```

6. In a separate terminal, start the React client:

```bash
cd client
npm install
npm run dev
```

---

## Project Structure

```text
NEXORA/
├── client/
│   ├── public/
│   ├── src/
│   │   ├── components/      # GlassCard, MediaCard, VideoPlayer, Sidebar, TopBar
│   │   ├── context/         # MediaContext & Auth states
│   │   ├── pages/           # DashboardPage, CategoryPage, MediaDetailsPage, AdminPage
│   │   ├── App.jsx
│   │   └── main.jsx
│   ├── package.json
│   ├── tailwind.config.js
│   └── vite.config.js
├── server/
│   ├── cmd/
│   │   └── api/             # Go Application Entry Point
│   ├── internal/
│   │   ├── api/             # HTTP Handlers & Streaming endpoints
│   │   ├── app/             # Application lifecycle
│   │   ├── db/              # Postgres DB connection & queries
│   │   ├── media/           # FFmpeg video processor & thumbnail generator
│   │   ├── metadata/        # TMDB/MAL scrapers & image caching
│   │   ├── migration/       # File preview & multi-threaded copy engine
│   │   ├── scanner/         # Disk scanner, Regex parser, fsnotify watcher
│   │   └── search/          # Meilisearch indexing client
│   ├── migrations/          # 0001_init_schema.sql
│   ├── go.mod
│   └── go.sum
├── Screenshots/             # Application UI Screenshots
├── compose.yml              # Docker Compose for PostgreSQL, Meilisearch, Redis
├── .env.example             # Global Environment Specification
└── README.md
```

---

## Services Provided

| Service | Value Delivered |
| --- | --- |
| Automated Media Ingestion | Eliminates manual file entry by scanning hard drives and populating database records automatically. |
| Instant Catalog Search | Empowers lounge customers to find any movie or episode in milliseconds in Arabic or English. |
| Offline Metadata Enrichment | Provides full posters, banners, plot summaries, and IMDB ratings without needing continuous internet access. |
| High-Speed LAN Streaming | Streams 4K and 1080p content across local network PCs smoothly with full seeking support. |
| Storage Migration & Cleanup | Helps lounge administrators organize chaotic hard drives, detect duplicates, and identify missing episodes. |

---

## API Overview

> **API Architecture:** NEXORA exposes RESTful endpoints for client interface interaction, search query proxying, file ingestion, and HTTP Range video streaming.

| Endpoint | Method | Description |
| --- | --- | --- |
| `/api/health` | `GET` | Health check verifying PostgreSQL and Go API status. |
| `/api/categories` | `GET` | Retrieves category counts for dashboard display. |
| `/api/search` | `GET` | Forwards instant search queries (`?q=...`) to Meilisearch index. |
| `/api/media/{id}/files` | `GET` | Returns list of video files associated with a media item. |
| `/api/media/inspect` | `POST` | Uses FFprobe to extract and persist duration, resolution, codec, audio tracks, and subtitles for `{ "fileId": 42 }`. |
| `/api/media/verify` | `POST` | Fully decodes a file with FFmpeg and persists its `healthy` / `corrupted` result for `{ "fileId": 42 }`. A direct `{ "path": "..." }` check is also supported. |
| `/api/library/duplicates` | `GET` | Returns groups of files with matching verified SHA-256 checksums. |
| `/api/library/missing-episodes` | `GET` | Reports missing episode numbers between 1 and the highest indexed episode per season. |
| `/api/library/corrupted` | `GET` | Lists indexed files whose latest full FFmpeg verification detected corruption. |
| `/api/scan` | `GET` | Triggers directory file scan and returns parsed Regex metadata. |
| `/api/ingest` | `POST` | Streams scan results into PostgreSQL in bounded batches, avoiding a full-library in-memory file list. |
| `/api/index` | `POST` | Runs bounded-batch ingest, technical FFprobe inspection, and Meilisearch synchronization for one or more media roots. |
| `/api/search/sync` | `POST` | Syncs PostgreSQL media records into Meilisearch index. |
| `/api/media/checksums` | `POST` | Calculates and stores SHA-256 checksums. Pass `{ "mediaItemId": 12 }` to limit work to one title, or `{}` for all indexed files. |
| `/api/stream` | `GET` | Streams local video file (`?path=...`) with HTTP Range headers. |
| `/api/stream/file/{id}` | `GET` | Streams imported video file by database ID with Range headers. |
| `/api/migration/preview` | `POST` | Generates disk reorganization diff preview. |
| `/api/migration/copy` | `POST` | Copies files to a target directory via resumable `.nexora-part` files and SHA-256 validation. Include `"removeSource": true` only to remove originals after verification. |

---

## Database Overview | نظرة عامة على قاعدة البيانات

🇺🇸 **English**

NEXORA utilizes a relational PostgreSQL schema designed for high query performance and clean domain separation. Categories contain media items (Movies, Series, Anime, etc.); Series and Anime link to Seasons, which own individual Video Files. Physical storage locations are monitored via `storage_disks`.

Indexes are applied on `video_files(file_path)`, `media_items(category_id)`, `media_items(type)`, and a compound unique index on `media_items(type, LOWER(title_en), COALESCE(release_year, 0))` to prevent duplicate entries during drive scans.

🇸🇦 **العربية**

يستخدم NEXORA مخطط PostgreSQL علائقي يضمن أداءً عالياً في الاستعلام وتفصيلاً واضحاً بين الكيانات. تحوي الفئات الأعمال الفنية (أفلام، مسلسلات، أنمي)، وترتبط المسلسلات والأنمي بالمواسم التي تملك ملفات الفيديو. وتُراقب أقراص التخزين عبر جدول `storage_disks`.

تُطبق الفهارس على `video_files(file_path)` و`media_items(category_id)` و`media_items(type)` وفهرس فريد مركّب لمنع تكرار إدخال الوسائط أثناء عمليات المسح.

```mermaid
erDiagram
    CATEGORIES ||--o{ MEDIA_ITEMS : contains
    MEDIA_ITEMS ||--o{ SEASONS : has
    MEDIA_ITEMS ||--o{ VIDEO_FILES : includes
    SEASONS ||--o{ VIDEO_FILES : holds
    STORAGE_DISKS ||--o{ VIDEO_FILES : stores
```

---

## Security | الأمان

🇺🇸 **English**

- **LAN Network Isolation:** Designed for internal lounge deployment behind firewalls; external ports are not exposed publicly.
- **Path Traversal Protection:** Streaming and migration endpoints validate source and target paths against configured `NEXORA_MEDIA_ROOTS` to prevent directory traversal attacks.
- **Isolated Credentials:** Sensitive database and API keys are managed exclusively via `.env` environment variables.
- **Input Sanitization:** Search and scan parameters are validated before database execution.

🇸🇦 **العربية**

- **عزل الشبكة المحلية:** صُمّم للنشر الداخلي خلف جدران الحماية بالاستراحات؛ ولا تُعرّض المنافذ للخارج.
- **الحماية من مسارات الملفات الضارة:** تتحقق نقاط البث والنقل من مسارات المصدر والوجهة مقابل المجلدات المسموحة `NEXORA_MEDIA_ROOTS` لتفادي هجمات القفز بين المجلدات.
- **عزل الاعتمادات:** تُدار الاعتمادات الحساسة عبر ملف `.env` المستثنى من المستودع.
- **تعقيم المدخلات:** يتم التحقق من كافة معلمات الاستعلام والبحث قبل التنفيذ في قاعدة البيانات.

---

## Deployment | النشر

🇺🇸 **English**

NEXORA is built for deployment on a central LAN server machine inside internet lounges or gaming centers.

**Deployment Steps:**
1. Configure host server with PostgreSQL 16, Meilisearch v1.11, and Redis 7 (via `docker compose up -d`).
2. Build the Go API production binary: `cd server && go build -o nexora-api.exe ./cmd/api`.
3. Build the static React frontend bundle: `cd client && npm run build`.
4. Host the API service on port `8080` and serve static client files via Vite preview or Nginx on local IP.

**Local Server URL:** `http://localhost:8080` (Backend API) | `http://localhost:5173` (Frontend Web Console)

🇸🇦 **العربية**

صُمم NEXORA للنشر على خادم مركزي داخل الاستراحة أو صالة الألعاب.

**خطوات النشر:**
1. إعداد الخادم الرئيسي بالخدمات (PostgreSQL وMeilisearch وRedis) عبر Docker Compose.
2. بناء الملف التنفيذي لخادم Go: `cd server && go build -o nexora-api.exe ./cmd/api`.
3. بناء واجهة الويب: `cd client && npm run build`.
4. تشغيل خادم Go على المنفذ `8080` وتقديم واجهة الويب على العنوان المحلي.

**العنوان المحلي:** `http://localhost:8080` (الخلفية البرمجية) | `http://localhost:5173` (واجهة المستخدم)

---

## Roadmap | خارطة الطريق

🇺🇸 **English**

- [ ] Hardware GPU Transcoding support (NVIDIA NVENC / Intel QuickSync) for mobile client playback.
- [ ] TV / Kiosk Remote Control navigation support (Gamepad & Android TV remote).
- [ ] User Watch Progress synchronization across lounge terminals.
- [ ] Automated Discord/Telegram notifications for disk space alerts and corrupted media.
- [ ] Multi-lounge centralized catalog syncing.

🇸🇦 **العربية**

- [ ] دعم الترميز المباشر باستخدام كارت الشاشة (NVIDIA NVENC / Intel QuickSync) للهواتف.
- [ ] دعم التنقل عبر الريموت كنترول ويد ألعاب الكونسول (نمط TV / Kiosk Mode).
- [ ] مزامنة تقدم مشاهدة الزبون عبر أجهزة الاستراحة المختلفة.
- [ ] إشعارات تلقائية عبر تلجرام أو ديسكورد عند امتلاء الأقراص أو اكتشاف ملفات تالفة.
- [ ] مزامنة الفهرس المركزي بين فروع ومقرات متعددة.

---

## Development Team

| Name | Responsibilities |
| --- | --- |
| **المهندس موسى** (Mousa Gamil Al-Awadhi) | Technical Leadership, System Architecture, Backend Engineering, Frontend Engineering, Database Design, Documentation |

---

<div align="center">

![Inama Soft logo](public/inama-soft-logo.ico)

**Made with ❤️ by Inama Soft — Collaborative Development Group**

Mousa Gamil Al-Awadhi

Ibb, Yemen · [mousa.mc13@gmail.com](mailto:mousa.mc13@gmail.com) · [+967 772 217 218](tel:+967772217218)

[Website](https://inma-soft.vercel.app) · [LinkedIn](https://www.linkedin.com/in/mousa-al-awadhi-6518633a8) · [GitHub](https://github.com/mosaa65) · [Live Project](https://github.com/mosaa65/NEXORA)

تم التطوير بواسطة فريق Inama Soft © 2026

</div>
