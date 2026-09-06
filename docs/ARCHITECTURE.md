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
  └─ preview image requests ──────────────────────┘      ├─ PostgreSQL
                                                           ├─ Meilisearch
                                                           ├─ Media Storage / filesystem
                                                           ├─ Asset image directory
                                                           ├─ FFmpeg / FFprobe executables
                                                           ├─ TMDB / MAL when configured
                                                           └─ scanner, filesystem watcher, TMDB queue worker
```

- React/Vite يقدم الواجهة، ويستخدم `client/src/lib/api.js` لاستهلاك API.
- Go (`net/http`) يقدم REST API، streaming، الوصول إلى PostgreSQL، وorchestration لعمليات الوسائط والـ metadata.
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

## 4. Layer Responsibilities

| Layer | مسؤول عن | ليس مسؤولًا عن |
|---|---|---|
| Frontend | UI، API consumption، browser playback controls، local presentation state | filesystem، DB، FFmpeg، authorization logic |
| Go API | API، business rules، streaming، file access، media/metadata orchestration | browser codec decoding أو browser UI state |
| PostgreSQL | catalogue and persistent data | media bytes وpreview file cache |
| Meilisearch | fast derived search | authoritative catalogue ownership |
| Media storage | original content and external sidecar assets | API authorization وcatalogue identity وحده |
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

```text
POST /api/index
  → scanner.Walk(media roots)
  → Repository.IngestScannedFiles in batches
  → FFprobe inspection
  → PostgreSQL technical metadata update
  → PostgreSQL search documents
  → Meilisearch indexing
```

الفهرسة لا تعني جلب TMDB أو إنتاج previews تلقائيًا في المسار الحالي.

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
  → mediaPathAllowed
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
- **Docker Compose:** يشغّل PostgreSQL وMeilisearch وRedis؛ لا يشغّل frontend أو Go API في ملف Compose الحالي.

## 11. Decision References

القرارات الموجودة والقابلة للتحقق موثقة في [`decisions/README.md`](decisions/README.md):

- ADR-001: Native HTML5 Video Player with Custom React Controls
- ADR-002: Direct HTTP Range Streaming through Go `http.ServeContent`
- ADR-003: Filesystem-Based Original Media Storage
- ADR-004: PostgreSQL as Catalogue Source of Truth
- ADR-005: Meilisearch as Derived Search Index
- ADR-006: FFmpeg and FFprobe as External Media Processing Tools

## 12. Known Risks and Non-Decisions

هذه ليست أوامر إصلاح تلقائية:

- path authorization الحالي يسمح مسارًا موجودًا خارج configured roots في `mediaPathAllowed`.
- admin authentication الحالي token ثابت وroute protection غير ظاهر لمعظم admin writes.
- preview thumbnail generation لا يستخدم single-flight/lock للطلب الأول المتزامن.
- subtitle formats الخارجية لا تتحول كلها إلى WebVTT.
- watch progress محلي للمتصفح فقط.
- watcher لا يظهر أنه يحذف database record عند file removal.
- توجد فجوة بين بعض docs القديمة والكود الحالي، مثل Plyr/Redis/Transcoding.

راجع [`PROJECT_ANALYSIS.md`](PROJECT_ANALYSIS.md) قبل تغيير أي من هذه المناطق.
