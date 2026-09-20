# NEXORA Architecture

> هذه وثيقة للمعمارية **الحالية** كما يثبتها `PROJECT_ANALYSIS.md` والكود. لا تعتمد المقترحات أو الميزات الموسومة `[PLANNED]` أو `[INFERRED]` كتنفيذ أو قرار مستقبلي ملزم.

## 1. Current Architecture

NEXORA تطبيق مركزي لإدارة كتالوج وسائط محلية وبثها. لا توجد microservices أو تطبيق عميل سطح مكتب ضمن الكود الحالي؛ العميل متصفح ويب، والخادم Go هو نقطة API والوصول إلى الملفات.

```text
Browser
  ↓
React Application
  ├─ JSON requests ──────────────────────────────┐
  ├─ native <video> media requests ───────────────┼──► Go API
  ├─ preview image requests ──────────────────────┤      ├─ PostgreSQL
  └─ USB device requests ───► Local Copy Bridge ──┘      ├─ Meilisearch
                              (127.0.0.1:32145)          ├─ Media Storage / filesystem
                                   │                     ├─ Asset image directory
                                   └─ USB Storage /      ├─ FFmpeg / FFprobe executables
                                      Android MTP /      ├─ TMDB / MAL when configured
                                      iOS AFC            └─ scanner, watcher, TMDB queue worker
```

- React/Vite يقدم الواجهة، ويستخدم `client/src/lib/api.js` لاستهلاك API.
- Go (`net/http`) يقدم REST API، streaming، الوصول إلى PostgreSQL، وorchestration لعمليات الوسائط والـ metadata.
- **Local Copy Bridge** خدمة Go محلية اختيارية على جهاز العميل (`server/cmd/copybridge`) تربط USB وتنسخ إليه عبر دفق من السيرفر المركزي. السيرفر المركزي لا يرى أجهزة العميل.
- PostgreSQL هو المخزن الدائم للكتالوج والملفات الوصفية والإعدادات والعلاقات.
- Meilisearch يحتفظ بفهرس بحث مشتق من بيانات PostgreSQL.
- ملفات الفيديو الأصلية تبقى على نظام الملفات؛ قاعدة البيانات تحتفظ بالـ paths وmetadata فقط.
- الصور المشتقة/المخزنة والـ preview JPEGs تحفظ تحت `NEXORA_ASSET_IMAGE_DIR`.

## 2. Architectural Principles

هذه المبادئ تصف النظام الحالي أو تحرس حدوده الحالية، وليست تفويضًا لتغيير أي جزء منه:

1. **PostgreSQL هو مصدر الحقيقة للكتالوج.**
   جداول media/season/file/metadata/settings تحتفظ بالبيانات الدائمة.
2. **Meilisearch فهرس بحث مشتق وقابل لإعادة البناء.**
   تتم مزامنته من `ListSearchDocuments`، وليس مصدر الحقيقة الوحيد للعمل أو الملف.
3. **ملفات الفيديو تبقى على filesystem.**
   لا تخزن bytes الفيديو كـ PostgreSQL BLOBs.
4. **المتصفح هو playback/decoding engine الحالي.**
   يستخدم التطبيق `<video>` أصليًا؛ React يوفر presentation and controls.
5. **Go مسؤول عن API والوصول إلى الملف وstreaming وbusiness orchestration.**
   لا يصل frontend إلى filesystem أو DB أو executables مباشرة.
6. **FFmpeg وFFprobe أدوات processing لا محرك تشغيل أساسي.**
   تستخدم للفحص والتحقق وتوليد الصورة، ولا تقع في stream hot path الحالي.
7. **مسار streaming الحار لا يحمل الفيديو كاملًا إلى ذاكرة الخادم.**
   يمرر Go file handle إلى `http.ServeContent`.
8. **البيانات المشتقة ينبغي أن تبقى قابلة لإعادة التكوين حيث يثبت ذلك.**
   فهرس البحث، preview thumbnails، وcached artwork ليست source of truth للكتالوج.
9. **الواجهة لا تتعامل مع paths محلية للخادم كصلاحية أو مصدر بيانات.**
   تستخدم API/URLs التي يصدرها الخادم فقط.
10. **مدّ النظام الموجود قبل إنشاء نظام موازٍ.**
    توجد طبقات API/repository/scanner/cache/metadata/streaming قائمة يجب فحصها قبل إنشاء بدائل.

## 3. Boundaries

### 3.1 Browser and React Application

**المسؤوليات:** UI، navigation، state الخاص بالعرض، تفاعل المستخدم، استهلاك JSON API، التحكم بالـ native video، وbrowser-local state مثل theme وwatch progress.

**لا مسؤولية له عن:** filesystem access، SQL/database access، تنفيذ FFmpeg/FFprobe، أو تقرير صلاحية path على الخادم.

**الحد الفعلي:** `client/src/lib/api.js` هو طبقة طلبات JSON العامة، بينما `<video>` و`<img>` يصلان إلى URLs يقدمها Go/Vite proxy في التطوير.

### 3.2 Go Backend

**المسؤوليات:** HTTP API، business logic، PostgreSQL repository access، Meilisearch integration، فتح الملفات وتقديمها، فحص/معالجة media عبر executables، metadata orchestration، ومهام watcher/TMDB queue الحالية.

**الحد الفعلي:** لا ينبغي نقل قواعد الملفات أو database/metadata orchestration إلى React؛ React لا يملك وصولًا آمنًا أو مباشرًا لهذه الموارد.

### 3.3 PostgreSQL

**Source of truth لـ:** media catalogue، relationships، file paths وtechnical metadata، metadata snapshots، settings، TMDB usage/queue، وpersistent system data.

**ليس مخزنًا لـ:** video bytes أو preview JPEGs أو browser-local watch progress.

### 3.4 Meilisearch

**المسؤولية:** fast text search وsearch-specific index settings.

**ليس مصدر الحقيقة:** يمكن إعادة مزامنته من PostgreSQL عبر endpoint/workflow الموجود. لا ينبغي أن ينفرد ببيانات لا يمكن بناؤها من المصدر الدائم.

### 3.5 Media Storage

**المسؤولية:** video files الأصلية، subtitle files الخارجية، وartwork المصدرية المحلية.

**مهم:** وجود ملف على القرص لا يعني وحده أنه داخل نموذج صلاحية صحيح. سلوك `mediaPathAllowed` الحالي موثق كمخاطرة معروفة في `PROJECT_ANALYSIS.md` ولا تغيره هذه الوثيقة.

### 3.6 Asset Image Storage

**المسؤولية:** artwork المحفوظ من metadata providers وpreview thumbnails المولدة. هذه ملفات derived/cache وليست كتالوجًا authoritative.

### 3.7 FFmpeg / FFprobe

**المسؤوليات الحالية:** inspection، verification، thumbnail generation، وSRT→WebVTT conversion يقع في طبقة media (لا يتطلب FFmpeg في المسار الحالي).

**خارج النطاق الحالي:** Transcoding/HLS/DASH ليس جزءًا منفذًا من streaming architecture. لا يدخل FFmpeg في playback hot path بلا تحليل وقرار معماري صريح لاحقًا.

### 3.8 External Metadata Providers

TMDB هو provider الأساسي عند توفر الإعدادات؛ MAL fallback خاص بالـ anime. تحفظ النتائج/snapshots والصور محليًا وفق الكود والإعدادات. فشل network أو credentials ليس سببًا لاستبدال الكتالوج المحلي كمصدر حقيقة.

### 3.9 Local Copy Bridge

**المسؤوليات:** تشغيل خادم HTTP محلي على `127.0.0.1:32145`، تعداد أجهزة USB وAndroid (MTP) وiOS (AFC) على جهاز العميل، تصفحها وإنشاء المجلدات وإخراجها، واستقبال طلبات النسخ وجدولة مهامها وتحديث تقدمها عبر SSE.

**لا مسؤولية له عن:** كتالوج الوسائط، أو مصدر الحقيقة للملفات، أو تشغيل الفيديو، أو تخزين الملفات بشكل دائم. هو **قناة نقل محلية** فقط.

**الحد الفعلي:** `server/cmd/copybridge` (binary + Windows service `NEXORACopyBridge`) و`server/internal/copybridge` (HTTP/SSE/jobs) و`server/internal/transfer` (backends). الواجهة تصل إليه عبر `client/src/lib/api.js` و`TransferContext.jsx`، والسيرفر المركزي يبقى مصدر `/api/stream/file/{id}`.

## 4. Layer Responsibilities

| Layer | مسؤول عن | ليس مسؤولًا عن |
|---|---|---|
| Frontend | UI، API consumption، browser playback controls، local presentation state | filesystem، DB، FFmpeg، authorization logic |
| Go API | API، business rules، streaming، file access، media/metadata orchestration | browser codec decoding أو browser UI state |
| PostgreSQL | catalogue and persistent data | media bytes وpreview file cache |
| Meilisearch | fast derived search | authoritative catalogue ownership |
| Media storage | original content and external sidecar assets | API authorization وcatalogue identity وحده |
| Local Copy Bridge | USB/MTP/AFC device access والنسخ المباشر إلى أجهزة العميل | catalogue ownership، streaming، أو permanent storage |
| FFmpeg/FFprobe | inspection/verification/derived processing | regular streaming/playback engine |

## 5. Data Flow

### 5.1 Catalogue read/search

```text
React page
  → api.js
  → Go handler
  → PostgreSQL (catalogue reads) or Meilisearch (search)
  → JSON response
  → browser/session in-memory cache where applicable
```

- client cache: GET عامة فقط، ذاكرة + `sessionStorage`، بمهلات قصيرة.
- server catalogue cache: L1 in-memory قصير الأجل لبعض catalogue reads، ويمسح عند non-GET.
- search: Go يرسل request إلى Meilisearch ويعيد SearchResult؛ index مبني من PostgreSQL documents.

### 5.2 Indexing
الفهرسة مسار staged مع حدود واضحة، وتكتب فقط ما تغيّر. التفاصيل الكاملة والقرار في [ADR-009](decisions/ADR-009-incremental-indexing-pipeline.md).

```text
POST /api/index   { roots, mode, inspect, syncSearch }
  → START SCAN SESSION        (scan_sessions: status = running)
  → LoadKnownFiles(roots)     (video_files row: size, mtime, file_id)
  → DISCOVERY                 per-root goroutines, bounded channel, ignore rules, symlink policy
  → METADATA WORKERS          stat, fingerprint, parse, classify, artwork (directory-cached)
  → IDENTITY / CHANGE         path → file_id → size+mtime  ⇒ new | changed | renamed | unchanged
  → emit (single goroutine)   unchanged files are skipped unless mode = full
  → BATCH PERSISTENCE         one transaction per 256 files, multi-row upsert
  → FFprobe inspection        only for new/changed files when inspect = true
  → RECONCILIATION            unobserved records ⇒ MISSING (root readable) or UNAVAILABLE (root not readable)
  → FINISH SCAN SESSION       (status, duration, structured stats JSONB, per-root state)
  → search sync               ListSearchDocuments → Meilisearch
```

الأنماط المتاحة: `full` و`incremental` (افتراضي) و`reconcile` و`recovery`. الفهرسة لا تعني جلب TMDB ولا إنتاج previews تلقائيًا في هذا المسار.

### 5.2.1 مسار الكتابة الواحد (ResolutionSession)
**قاعدة إلزامية:** أي ملف يصل الكتالوج يمر عبر **Entity Resolution**، مهما كان مصدره.

المسارات الثلاثة تتشارك جلسة واحدة (`db.ResolutionSession`):

```text
POST /api/index        ─┐
watcher (debounced)    ─┼─► ResolutionSession.Ingest ─► ResolveAndIngest ─► works/seasons/episodes
scheduler (reconcile)  ─┘
```

السبب: لو كتب أي مسار ملفًا مباشرة، لأعاد إنشاء الأعمال المشوّهة التي وُجد القرار لمنعها. لذلك:

- `ingestWithoutResolution` (المسار القديم) **غير موصول بأي مسار تشغيلي** ويحمل تحذيرًا صريحًا.
- الجلسة تحتفظ بـ `works` **بمؤشر** كي يرى الملف التالي العمل الذي أنشأه السابق. بدون ذلك، `01.mkv` و`02.mkv` في مجلد واحد يُنشئان عملين منفصلين.
- الجلسة تحتفظ بخريطة الـ aliases وتحدّثها عند تعلّم alias جديد، فيصبح الاسم المؤكد مطابقة تامة في الملف التالي.

### 5.2.2 واجهة التحكم في الفحص (Control Center)
الفحص يُدار تعاونيًا، وPause ليس مرادفًا لـ Cancel. المرجع: [ADR-011](decisions/ADR-011-search-projection-and-scan-control.md).

```text
RUNNING ──(طلب pause)──► PAUSING ──(لا worker وسط عنصر)──► PAUSED
   ▲                                                        │
   └────────────────(resume)── RESUMING ◄───────────────────┘
```

**الضمانة الأساسية:** بوابة الـ pause تقع **قبل سحب عنصر عمل جديد**، وليس في منتصف عنصر:

```go
for visit := range candidates {
    // بوابة الـ pause قبل سحب العمل، لذلك فحص متوقف لا يتخلى عن ملف نصف معالج
    if !control.wait(workCtx) { return }
    control.markWorker(workerID, RoleMetadata, visit.Path)
    file, ok := s.processCandidate(...)
}
```

النتيجة: **لا يمكن لـ pause أن يُنتج صفًا نصف مكتوب** في قاعدة البيانات.

- `PAUSING` منفصل عن `PAUSED` حتى لا تدّعي الواجهة التوقف بينما workers ما زالت تُنهي عملها.
- **Cancel عملية مختلفة:** يُطلق أي worker محجوب بالـ pause (وإلا لن يُلاحظ الإلغاء)، وينهي الفحص بحالة `CANCELLED`. استخدام Cancel للتوقف المؤقت يمحو counters وcursor التي يحفظها Pause عمدًا.

**رؤية كل worker:**
```json
{"id": 1, "role": "metadata", "active": true,
 "currentPath": "/media/Disk1/Show/S01E04.mkv", "processed": 812}
```
الأدوار: `discovery` · `metadata` · `persistence` · `idle`. الترتيب ثابت بالمعرّف حتى لا تتغير القائمة مع كل poll.

### 5.2.3 Search Projection
الفهرس البحثي **projection قابل لإعادة البناء**، وليس مصدر حقيقة. المرجع: [ADR-011](decisions/ADR-011-search-projection-and-scan-control.md).

```text
PostgreSQL (media_items)
  ↓ ListSearchDocumentPage(afterID, limit)   keyset pagination بمعرّف تصاعدي
  ↓ IndexDocuments(page)                     يُرسل لكل صفحة
  ↓ SaveProjectionCursor(afterID)             التقدم يُحفظ بعد كل صفحة
```

أربع خصائص مُثبتة باختبارات:

| الخاصية | التفصيل |
|---|---|
| **بلا حد أقصى** | الصفحات تُقرأ حتى نهاية الكتالوج. مُثبت على **25,000 مستند** (الحد القديم 10,000) |
| **ذاكرة ثابتة** | صفحة واحدة فقط في الذاكرة؛ الاستهلاك مستقل عن حجم المكتبة |
| **قابل للاستكمال** | الـ cursor في `search_projection_state` ويُكتب بعد كل صفحة، فإعادة التشغيل تستكمل |
| **قابل لإعادة البناء من DB** | لا يقرأ نظام الملفات إطلاقًا — إسقاط الفهرس وإعادته لا يمس أي ملف وسائط |

**Keyset pagination وليس offset:** الـ cursor `id > afterID` مع `ORDER BY id`. المؤشر بالإزاحة يُسقط أو يكرر صفوفًا عند الإدراج أثناء إعادة البناء، وهو الحالة الطبيعية لمكتبة حيّة.

**التحديث المستهدف:** `ProjectWork` يُفهرس المستندات المعطاة فقط ولا يلمس أي صفحة — إضافة حلقة واحدة تُحدّث مستندًا واحدًا، لا تُعيد بناء فهرس مليون عمل.

### 5.2.4 Logical Media Model و Entity Resolution
الملف الفيزيائي ليس العمل. المرجع الكامل في [ADR-010](decisions/ADR-010-logical-media-model.md).

```text
Filesystem File
  ↓ parse (اسم/مسار)
Media Candidate
  ↓ ENTITY RESOLUTION
      evidence → ranked candidates → decision
Logical Work → Season → Episode → Physical File
```

المرحلة الحاسمة هي **Entity Resolution** وتحكمها القواعد التالية من الكود:

- **محوران مستقلان**: `media_type` (movie | series | unknown) منفصل عن `category` (movies/series/anime/kids/...). `anime` تصنيف ونوعه غالبًا `series` لكن فيلم الأنمي يبقى `movie`.
- **لا تُنشئ Work فورًا**: ملف جديد يُقارَن بالكيانات الموجودة أولًا (alias متعلَّم ثم مرشحون مُقيَّمون). الإنشاء هو الملاذ الأخير، وفقط عند تأكد `media_type` وقابلية العنوان للاستخدام، ويُوسم الكيان `provisional`.
- **عند عدم اليقين**: يُكتب الملف في `resolution_queue` بحالة `needs_review` مع المرشحين ودرجاتهم وأسبابها. البيانات غير المؤكدة أفضل من بيانات خاطئة.
- **`unknown` نتيجة صحيحة** لنوع الوسائط عندما تتقارب أدلة الفيلم والمسلسل، وليس تخمينًا.
- **ذاكرة قواعدية بلا AI**: `media_aliases` تربط كل تهجئة مؤكدة بعمل واحد (`UNIQUE(alias_normalized)`)، فتصبح `Breaking Bad` و`Breaking.Bad` و`بريكنغ باد` عملًا واحدًا. التقاطع مع كلمة مفتاحية أو تشابه < 0.9 لا يُتعلَّم تلقائيًا.
- **حل المجموعة (Batch)**: لكل مجلد ملخص محدود (حجم، أشقاء حلقات، أرقام حلقات، توقيع الموسم، عنوان توافقي من ≥2 ملف) ليتمكن النظام من قراءة `01..04.mkv` كفصل واحد. الملخص محدود بـ 20,000 مجلد مع إخلاء الأقدم.
- **المصدر يحدد الأولوية**: `admin > tmdb > database > resolver > parser > filesystem`. قرار المسؤول لا يعيد الكتابة عليه أي Full Scan.
- **الحذف غير موجود**: الكيانات المتكررة تُدمج عبر `merged_into_id` مع الحفاظ على السجل للتدقيق.

ملاحظات تشغيلية مثبتة من الكود:

- **Root isolation**: كل media root له حالة مستقلة داخل `scan_roots`; قرص مفقود يسجل `unavailable` ولا يوقف بقية الأقراص، وتصبح حالة الجلسة `partial`.
- **لا يتم قراءة محتوى الملف أثناء الفحص**: الهوية تعتمد على `file_id` (فهرس ملفات Windows / inode على POSIX) ثم `size + mod_time`. الـ hashing يبقى عملية صريحة منفصلة.
- **إعادة التسمية تُعالج كـ move** على نفس السجل (`ActionMovePath`) وليس حذفًا وإنشاءً، فتبقى الهوية وتقدم المشاهدة.
- **الحذف غير موجود داخل الفحص**: السجل يصبح `missing` أو `unavailable` فقط، والتنظيف عملية منفصلة بسياسة عمر أدنى ورفض عند تعذر الوصول لأي root وحد أقصى لكل تشغيل.
- **Watcher هو المسار السريع فقط**: يعمل مع debounce وstability check، وحدث remove/rename لا يُنفّذ كحذف. Reconciliation الدوري (افتراضيًا كل 15 دقيقة) هو مصدر الحقيقة لأنه يغطي ما فاته fsnotify أثناء التوقف.
- **جلستان فحص متزامنتان غير مسموحتين**: الطلب الثاني يعيد `409 Conflict`.

### 5.3 Metadata enrichment

```text
explicit enrich / TMDB queue job
  → metadata service
  → TMDB or MAL when configured
  → PostgreSQL snapshots/metadata/provider relations
  → local or remote artwork result according to settings
```

### 5.4 Related-title graph

بعد إثراء عمل من TMDB، يستخرج Go `recommendations` و`similar` من الـ snapshot إلى `media_related_titles`. تحفظ العلاقة بواسطة `provider + target_external_id + target_kind`، لا بالعنوان. عند `GET /api/media/{id}/related` يطابق PostgreSQL هذا الـ ID مع `media_items.metadata_external_id`، فيعيد `local_media_id` للأعمال المتاحة و`local=false` للأعمال قيد الإضافة. لا يستدعي هذا المسار TMDB أثناء التصفح.

## 6. Media Flow

### 6.1 File identity and access

```text
Filesystem path
  → scanner parses path/name
  → PostgreSQL video_files record
  → file ID returned with media detail
  → stream endpoint resolves ID back to file path
```

`video_files` stores file metadata such as size, duration, resolution, codecs and inspected tracks, but not bytes.

### 6.2 Subtitles

```text
Video file path
  → FindExternalSubtitles adjacent files
  → /subtitles JSON list
  → /subtitles/{id} stream
  → <track> in native video
```

SRT is converted to WebVTT in response. Other advertised formats are not all converted by current code; see Known Risks.

## 7. Streaming Flow

```text
User chooses file
  → RealVideoPlayerModal constructs API stream URL
  → VideoPlayer passes URL to native <video>
  → browser requests GET /api/stream/file/{id} (often with Range)
  → Go looks up path in PostgreSQL
  → serveCataloguePath (DB-resolved, trusted) or mediaPathAllowed for client-supplied ?path=
  → os.Open + file.Stat
  → Accept-Ranges: bytes + http.ServeContent
  → browser decodes supported container/codec
```

- `http.ServeContent` is responsible for HTTP Range parsing and partial responses.
- The server gives it an open `*os.File`; it does not allocate an in-memory copy of the full video.
- Current direct streaming has no HLS/DASH manifest, adaptive bitrate ladder, or server-side transcoding.

### 7.1 Timeline preview flow

```text
timeline hover
  → 180 ms client debounce
  → 10-second timestamp bucket
  → /api/stream/file/{id}/preview?at={bucket}
  → existing disk JPEG OR FFmpeg single-frame generation
  → redirect to /assets/images/previews/...
```

### 7.2 Copy-to-device (USB) flow

```text
User selects media + target device in browser
  → browser calls local Copy Bridge POST /api/transfer/copy
  → bridge prepares sources: source_url (/api/stream/file/{id}) or local source_path
  → bridge opens source stream (HTTP Range from central server, or local file)
  → backend.Connect + backend.Mkdir(remoteDir)
  → backend.PutStream(reader, size, remotePath)
       ├─ Storage: direct write, zero-spool
       ├─ iOS (AFC): direct write with resume/buffer, folder created first
       └─ Android (MTP): spool temp file named after target, Shell CopyHere, delete
  → progress + phase pushed to browser via SSE /api/transfer/events
```

## 8. Caching Responsibilities

| Cache | Owner | Purpose | Rebuild/invalidation model |
|---|---|---|---|
| API read cache | Browser `api.js` | reduce short-repeat GET requests | TTL; explicit `invalidateAPICache` |
| Catalogue response cache | Go API | reduce repeated public catalogue reads | TTL; clear on non-GET |
| Search index | Meilisearch | fast searching | sync/rebuild from PostgreSQL documents |
| Artwork cache | Asset image directory | local serving of provider images | existing file reuse; provider fetch can regenerate |
| Preview cache | Asset image directory | timeline JPEG previews | generated on first request per bucket |
| Watch progress | Browser localStorage | resume on same browser profile | local key per file/source; not shared persistent data |

Redis is not assigned a runtime caching responsibility in current application code.

## 9. Storage Responsibilities

| Store | Owns | Durability/role |
|---|---|---|
| PostgreSQL | catalogue, paths, metadata, settings, queues | authoritative persistent application state |
| Media filesystem | original video and external subtitle/artwork files | original media source |
| Asset image directory | cached provider art and preview JPEGs | derived/regenerable files |
| Browser local/session storage | theme/font, watch position, short API cache | per-browser presentation state/cache |
| Meilisearch volume | derived search index | rebuildable from PostgreSQL |

## 10. Integration Boundaries

- **Vite development proxy:** `/api` و`/assets` تذهب إلى Go upstream في التطوير. Production static hosting topology غير مثبت في repository.
- **PostgreSQL:** يصل إليه Go فقط عبر repository/`database/sql`; browser لا يتصل به.
- **Meilisearch:** يصل إليه Go `search.Client`; browser يستخدم `/api/search`.
- **TMDB/MAL:** يصل إليهما Go metadata layer، لا React مباشرة.
- **Filesystem/FFmpeg/FFprobe:** ينفذها Go host فقط.
- **Local Copy Bridge:** browser يتصل بـ `127.0.0.1:32145` مباشرة (loopback)، والـ bridge يقرأ من السيرفر المركزي عبر `/api/stream/file/{id}` أو من ملف محلي. الأجهزة لا تُنقل إلى السيرفر المركزي.
- **Docker Compose:** يشغّل PostgreSQL وMeilisearch وRedis؛ لا يشغّل frontend أو Go API في ملف Compose الحالي.

## 11. Decision References

القرارات الموجودة والقابلة للتحقق موثقة في [`decisions/README.md`](decisions/README.md):

- ADR-001: Native HTML5 Video Player with Custom React Controls
- ADR-002: Direct HTTP Range Streaming through Go `http.ServeContent`
- ADR-003: Filesystem-Based Original Media Storage
- ADR-004: PostgreSQL as Catalogue Source of Truth
- ADR-005: Meilisearch as Derived Search Index
- ADR-006: FFmpeg and FFprobe as External Media Processing Tools
- ADR-007: Provider-ID Related Titles Graph
- ADR-008: Local Copy Bridge for USB Device Transfer
- ADR-009: Incremental, Fault-Tolerant Media Indexing Pipeline
- ADR-010: Logical Media Model and Entity Resolution
- ADR-011: Rebuildable Search Projection and Cooperative Scan Control
- ADR-012: Local Episode Enrichment and a Separate Episode Search Index
- ADR-013: Catalogue Consolidation — Duplicates, Container Titles and Unlinked Files
## 12. Known Risks and Non-Decisions

هذه ليست أوامر إصلاح تلقائية:

- path authorization: `handleStream` (بمسار من العميل `?path=`) و`handleStreamImage` يخضعان لـ `mediaPathAllowed`، بينما البث بمعرّف قاعدة البيانات (`/api/stream/file/{id}`) يستخدم مسار الكتالوج مباشرة عبر `serveCataloguePath` ولا يقبل مسارًا من العميل.
- admin authentication الحالي token ثابت وroute protection غير ظاهر لمعظم admin writes.
- preview thumbnail generation لا يستخدم single-flight/lock للطلب الأول المتزامن.
- subtitle formats الخارجية لا تتحول كلها إلى WebVTT.
- watch progress محلي للمتصفح فقط.
- watcher لا يظهر أنه يحذف database record عند file removal.
- Copy Bridge: الافتراضي يعمل كـ `LocalSystem` (Session 0)، وبعض إصدارات ويندوز قد تتطلب جلسة مستخدم تفاعلية لـ Android MTP (Shell COM).
- Copy Bridge: نسخ Android MTP يستهلك مساحة مؤقتة بحجم الملف (قيد Shell COM)، ويجب أن تتوفر مساحة في `NEXORA_COPY_BRIDGE_TEMP_DIR`.
- Copy Bridge: لا يوجد token مشترك للأوامر الحساسة — مُؤجَّل بقرار المالك؛ الحماية الحالية هي loopback + تضييق CORS.
- توجد فجوة بين بعض docs القديمة والكود الحالي، مثل Plyr/Redis/Transcoding.

راجع [`PROJECT_ANALYSIS.md`](PROJECT_ANALYSIS.md) قبل تغيير أي من هذه المناطق.
