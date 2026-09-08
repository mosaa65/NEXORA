# NEXORA Project Analysis

> نطاق هذه الوثيقة: قراءة وتحليل الـ Repository كما هو في 30 أغسطس 2026. لا تعد هذه الوثيقة تصميمًا بديلًا ولا تغيّر سلوك النظام.
>
> دلالة التصنيفات: **[VERIFIED]** مقروء من الكود الحالي، **[INFERRED]** استنتاج محدود مبني على الكود، **[PLANNED]** اتجاه مستقبلي فقط، **[UNKNOWN]** لا يمكن إثباته من الـ Repository وحده.

## 1. Executive Summary

- [VERIFIED] NEXORA نظام Web لإدارة مكتبة وسائط محلية وبثها. الواجهة React/Vite، والخادم Go `net/http`، وPostgreSQL هو مخزن الكتالوج، وMeilisearch محرك البحث.
- [VERIFIED] المسار الرئيسي للتشغيل هو: ملفات فيديو على القرص ← فهرسة إلى PostgreSQL ← واجهة React تستعلم الـ API ← عنصر HTML5 `<video>` يطلب الملف من خادم Go عبر HTTP.
- [VERIFIED] المشغل الحالي ليس Plyr: إنه عنصر `<video>` أصلي مع واجهة تحكم React مخصصة في `client/src/components/VideoPlayer.jsx`.
- [INFERRED] التصميم مناسب مبدئيًا لمكتبة LAN مركزية؛ إذ يتجنب Transcoding وHLS ويقرأ الملف من القرص عند الطلب.
- [VERIFIED] توجد إمكانات أوسع من المشاهدة: فهرسة، مراقبة تغيرات المجلدات، جودة، كشف تكرار، نقل آمن بالـ checksum، وTMDB/MAL وكتالوج محلي للعلاقات.

## 2. Project Purpose

- [VERIFIED] `README.md` وإعدادات التشغيل تصف النظام كمكتبة وبث LAN للاستراحات/الشبكات المحلية، مع تصفح أعمال وحلقات وملفات موجودة على أقراص الخادم.
- [VERIFIED] الكود يقبل `NEXORA_MEDIA_ROOTS` كقائمة جذور وسائط، ويفهرس امتدادات فيديو محددة، ويحفظ مساراتها الفعلية في `video_files`.
- [INFERRED] الهدف التشغيلي الأساسي هو أن يكون الخادم هو نقطة الوصول المركزية للملفات وmetadata؛ لا يوجد عميل سطح مكتب خاص ضمن الـ Repository.

## 3. Current Technology Stack

| المجال | الحالة | التقنية الفعلية | ملاحظات |
|---|---|---|---|
| Frontend | Implemented | React 18، React Router، Vite 5، Tailwind/PostCSS، Framer Motion | [VERIFIED] من `client/package.json` وملفات `src`. |
| Backend | Implemented | Go 1.22، `net/http` و`http.ServeMux` | [VERIFIED] لا يظهر إطار ويب خارجي. |
| Database | Implemented | PostgreSQL 16، driver `pgx/v5` عبر `database/sql` | [VERIFIED] من `go.mod` و`compose.yml`. |
| Search | Implemented | Meilisearch v1.11 عبر REST | [VERIFIED] `internal/search/client.go`. |
| Cache | Implemented | ذاكرة Go، `sessionStorage`، `localStorage`، وملفات cache على القرص | [VERIFIED] موضح أدناه. |
| Media utilities | Implemented / environment-dependent | FFmpeg وFFprobe كبرامج خارجية | [VERIFIED] المسارات من البيئة؛ توفرهما الفعلي في بيئة إنتاج غير مثبت من الكود. |
| File watcher | Implemented | `fsnotify` | [VERIFIED] يعمل عند وجود media roots. |
| Redis | Unknown / unused by code | Redis container في Compose فقط | [VERIFIED] لا يوجد عميل Redis أو استخدام له في كود Go/React الحالي. |
| Plyr | Dependency unused | `plyr` في `package.json` وبعض CSS قديم | [VERIFIED] لا يوجد import أو إنشاء Plyr في `client/src`. |

## 4. Repository Structure

```text
client/                         React SPA وVite
  src/components/               واجهات مشتركة، ومنها VideoPlayer
  src/pages/                    صفحات العميل والإدارة
  src/lib/api.js                عميل HTTP والـ browser cache
  src/context/                  ThemeContext
server/
  cmd/api/main.go + main.go     نقطتا بدء متكافئتان
  internal/api/                 REST handlers، streaming، middleware/cache
  internal/app/                 تركيب التطبيق والتشغيل graceful shutdown
  internal/config/              تحميل البيئة
  internal/db/                  migrations وrepository PostgreSQL
  internal/scanner/             walk/parser/watcher للوسائط
  internal/media/               FFmpeg/FFprobe والترجمات الخارجية
  internal/metadata/            TMDB/MAL وcache للصور
  internal/search/              عميل Meilisearch
  internal/quality/             تقارير الجودة والتحقق
  internal/migration/           معاينة/نسخ ملفات مع checksum
  internal/disks/               اكتشاف الأقراص، مع تنفيذ Windows
  migrations/                   SQL migrations مرتبة بالاسم
compose.yml                     PostgreSQL + Meilisearch + Redis
docs/                           توثيق موجود ودراسات سابقة
test-media-library/             مكتبة اختبار محلية
```

- [VERIFIED] يوجد 35 ملف Go و9 ملفات Go test و51 ملف JSX تحت المصدر وقت التحليل.
- [VERIFIED] توجد وثائق كثيرة سابقًا تحت `docs/`؛ هذه الوثيقة تقارنها بالكود بدل اعتبارها مصدر الحقيقة تلقائيًا.

## 5. Application Entry Points

- [VERIFIED] الواجهة تبدأ من `client/src/main.jsx`: تنشئ React root وتلف `App` بـ `AppErrorBoundary` و`ThemeProvider` و`React.StrictMode`.
- [VERIFIED] `client/vite.config.js` يشغّل dev server على `0.0.0.0:5173` ويعمل proxy لـ `/api` و`/assets` إلى `NEXORA_API_UPSTREAM` أو `127.0.0.1:8080`.
- [VERIFIED] `server/main.go` و`server/cmd/api/main.go` كلاهما يستدعيان `app.Run()`.
- [VERIFIED] `internal/app/app.go` يحمّل config، يفتح PostgreSQL، يشغّل migrations، ينشئ الخدمات، يبدأ watcher اختياريًا، ثم يستمع HTTP على `NEXORA_HTTP_ADDR` (افتراضيًا `:8080`).
- [VERIFIED] shutdown يلتقط `SIGINT`/`SIGTERM` ويمنح HTTP server مهلة 10 ثوانٍ للإغلاق.

## 6. System Architecture

```text
Browser (React SPA)
  ├─ catalogue/search/admin JSON ───────────────► Go REST API
  ├─ video bytes via native <video> ────────────► Go stream endpoint
  └─ preview image request ─────────────────────► Go + FFmpeg cache

Go API
  ├─ PostgreSQL: catalogue, files, metadata, settings, queue
  ├─ Meilisearch: indexed search documents
  ├─ Local/attached disks: video, subtitle, artwork source files
  ├─ Asset directory: TMDB/MAL images and preview JPEGs
  ├─ FFmpeg/FFprobe executables
  └─ TMDB / MAL (only when configured and explicitly invoked/queued)
```

- [VERIFIED] الخادم يمرر requests إلى interfaces للخزانة والبحث والـ metadata والمعالجة، مما يجعل handlers قابلة للاختبار بواسطة mocks.
- [INFERRED] البنية Centralized server/thin browser client وليست microservices؛ Compose لا يشغّل Go API أو Vite، بل يعتمد تشغيلهما خارج Compose.

## 7. Frontend Architecture

- [VERIFIED] `App.jsx` يستخدم `HashRouter` وroutes عامة وإدارية، ويحمل health/categories ويحتفظ بـ search query ونتائج البحث ونافذة المشغل في state محلي.
- [VERIFIED] إدارة الحالة ليست Redux/Zustand: هي React `useState`/`useEffect`/`useDeferredValue` وContext واحد للـ theme/font.
- [VERIFIED] `ThemeContext` يحفظ `nexora_theme` و`nexora_font` في `localStorage` ويضع `data-theme` و`data-font` على `<html>`.
- [VERIFIED] `client/src/lib/api.js` هو طبقة HTTP الواحدة: يرسل JSON، يضيف `Content-Type`، ويحفظ GET العامة في Map ذاكرية و`sessionStorage` لمدة دقيقتين (health 15 ثانية)، مع deduplication للطلبات المتزامنة.
- [VERIFIED] طلبات `/api/admin/*` و`/api/stream/*` مستثناة من cache الواجهة.
- [VERIFIED] هناك واجهتا layout: `CustomerCinemaLayout` و`AdminPortalLayout`؛ كلاهما يعتمد React Router `Outlet`.
- [VERIFIED] صفحة دليل الممثلين تستخدم `FilterToolbar` نفسه الموجود في النظام؛ البحث محلي في الأسماء العربية/الإنجليزية، مع فلترة حد أدنى للأعمال المحلية و`known_for_department` المخزن من TMDB، وفرز الظهور/الأعمال/الشعبية/الاسم للنتائج المحمّلة. لا تحفظ بنية الأشخاص الحالية جنسية أو بلد ميلاد.
- [VERIFIED] صفحة الشخص تعرض بيانات `people` المتاحة حاليًا (الصورة، الاسم، التخصص، الشهرة وعدد الأعمال المحلية) ثم قائمة الأعمال المحلية المرتبطة عبر `media_credits`؛ لا تعرض سيرة أو جنسية أو تاريخ ميلاد لأن هذه الحقول ليست ضمن response الحالي.
- [VERIFIED] مسار تفاصيل media في `App.jsx` يستخدم حاليًا `mockLibrary` أو fallback ثابتًا في `MediaDetailsRouteWrapper` بدل استدعاء `getMediaDetail` داخل ذلك wrapper. نافذة التشغيل الفعلية (`RealVideoPlayerModal`) تستدعي `getMediaDetail` و`getFileSubtitles`.
- [VERIFIED] صفحة تفاصيل العمل تعرض أبرز 24 ممثلًا وفق `billing order` في TMDB، مع شارة تبين إجمالي طاقم العمل؛ بطاقات الممثلين تنقل إلى `/person/tmdb-person-{TMDB ID}`. دليل الأشخاص يستخدم معيار billing نفسه ولا يعرض كل background credits؛ صفحة الشخص تستعلم أعماله المرتبطة محليًا من `media_credits`، ولا تعتمد على مطابقة الاسم.

## 8. Backend Architecture

- [VERIFIED] `api.NewServer` ينشئ `http.ServeMux` ويغلّفه بـ middleware واحد، ويبدأ goroutine لمعالجة طابور TMDB.
- [VERIFIED] Repository واحد في `internal/db/repository.go` يحتوي SQL للوصول إلى الكتالوج والملفات والإعدادات والعلاقات.
- [VERIFIED] إعداد `database/sql`: حد أقصى 25 اتصالًا، 10 idle، وعمر اتصال 30 دقيقة.
- [VERIFIED] Scanner يفصل اكتشاف المسارات عن قراءة metadata: workers متوازية، لكن callback `emit` يستهلك النتيجة في goroutine واحد لحماية ingest الحالي.
- [VERIFIED] Event watcher يعمل في goroutine عند توفر roots ويعيد فهرسة ملف فيديو عند create/modify؛ حدث remove يسجّل event لكنه لا يظهر في `app.Run` أنه يحذف صف قاعدة البيانات.

## 9. API Architecture

- [VERIFIED] API هو REST/JSON عبر `http.ServeMux` وpath patterns في Go، ولا توجد GraphQL أو WebSocket endpoints.
- [VERIFIED] مجموعات endpoints الفعلية تشمل: health، categories، media CRUD/detail/files/metadata/enrich، search، hubs/showcases/franchises/people، indexing/scan/ingest، quality/checksum/migration، streaming/subtitles/previews، TMDB settings/queue، disks/system browse، وadmin login/session/logout.
- [VERIFIED] المسارات الدقيقة المربوطة في `server/internal/api/server.go` هي مصدر الحقيقة؛ `docs/api/ENDPOINTS.md` مرجع مساعد وقد لا يطابق كل تغيير حديث.
- [VERIFIED] middleware يسمح CORS بـ `Access-Control-Allow-Origin: *`، ويعلن methods `GET, POST, OPTIONS` رغم وجود handlers فعلية لـ `PUT` و`DELETE`.

## 10. Data Flow

### Catalog/index flow

```text
Admin/UI → POST /api/index
  → scanner.Walk(roots)
  → batches of 128 files → Repository.IngestScannedFiles
  → FFprobe Inspect لكل ملف في الدفعة
  → UpdateVideoTechnicalDetails
  → ListSearchDocuments → Meilisearch IndexDocuments
```

- [VERIFIED] الفهرسة لا تجلب TMDB ولا تستخرج الترجمات كجزء من `handleIndex`؛ التعليق والكود يفصلان ذلك عن first pass.
- [VERIFIED] scan يدعم `.mp4,.mkv,.avi,.mov,.wmv,.m4v,.webm,.ts,.m2ts` ويستخرج الاسم/الموسم/الحلقة/resolution من المسار والاسم عبر parser.

### Playback flow

```text
User → RealVideoPlayerModal → GET /api/media/{id}
  → current file → <video src="/api/stream/file/{fileId}">
  → GET /api/stream/file/{fileId} (عادة مع Range)
  → Repository.GetVideoFilePath → serveMediaPath → os.Open → http.ServeContent
```

- [VERIFIED] عند عدم وجود `id` للملف يبني modal بديلًا `/api/stream?path=...`.

## 11. Storage and File System Architecture

- [VERIFIED] ملفات الفيديو لا تخزن BLOB داخل PostgreSQL؛ `video_files.file_path` يربط الصف بالملف الفعلي، مع الحجم ومدة/codec/مسارات الصوت والترجمات الفنية كبيانات وصفية.
- [VERIFIED] roots تأتي من `NEXORA_MEDIA_ROOTS`، وتفصل بـ OS path-list separator؛ Windows هو البيئة المستهدفة بوضوح لوجود `manager_windows.go`.
- [VERIFIED] Asset directory (`NEXORA_ASSET_IMAGE_DIR`، افتراضي `assets/images`) يحوي صور metadata cached وpreview JPEGs ويخدمها Go تحت `/assets/images/`.
- [VERIFIED] `mediaPathAllowed` يقارن المسار المطلق مع configured roots، لكنه أيضًا يسمح **بأي مسار موجود على القرص** بعد فشل المطابقة. هذه حقيقة مهمة للأمان وليست allowlist صارمة كما يوحي الاسم.
- [VERIFIED] `handleStreamImage` يفتح صورة محلية ويعمل `io.Copy` مع browser cache يوم واحد.

## 12. Media Library Architecture

- [VERIFIED] core schema: `categories` → `media_items` → `seasons` و`video_files`، مع `storage_disks`.
- [VERIFIED] ingest يجمع الملفات عبر normalized title/type/year، ينشئ/يحدث media item وseason، ويحفظ مسار الملف وحجمه والحلقة. يوجد unique index لمسار الملف وidentity مركب للـ media.
- [VERIFIED] scanner يبحث عن artwork محلي في مجلد الفيديو ثم الأب، ويعطي أولوية لأسماء مثل poster/cover/folder/banner.
- [VERIFIED] هناك جداول وrepository لعلاقات provider collections وpeople وcredits وrelated titles وsmart hubs وshowcases، وهي منفصلة عن ملفات الفيديو نفسها.
- [VERIFIED] مكتبة `test-media-library/` موجودة للاختبار؛ لا تثبت وجود محتوى إنتاجي.

## 13. Video Streaming Architecture

- [VERIFIED] البث المباشر متاح بـ `GET /api/stream?path=...` و`GET /api/stream/file/{id}`.
- [VERIFIED] endpoint بالـ ID يقرأ المسار من PostgreSQL أولًا؛ endpoint بالـ path يستقبل المسار مباشرة ثم يمر على `mediaPathAllowed`.
- [VERIFIED] `serveMediaPath` يفتح الملف بـ `os.Open`، يحصل على stat، يحدد MIME حسب extension إن عرفه، يضع `Accept-Ranges: bytes`، ثم يستدعي `http.ServeContent`.
- [VERIFIED] لا يوجد HLS/DASH manifest ولا Transcoding في stream path الحالي.
- [INFERRED] هذا يناسب LAN وملفات يقدر browser على فك ترميز codecs/containers الخاصة بها؛ التوافق النهائي مع MKV أو codec معين يعتمد على المتصفح والجهاز.

## 14. Video Player Architecture

- [VERIFIED] المكوّن هو `client/src/components/VideoPlayer.jsx` ويستخدم `<video playsInline preload="metadata">` مع `<source src={src}>` و`<track>` للترجمات.
- [VERIFIED] يوجد UI مخصص فوق `<video>`: Play/Pause، تقديم/رجوع 10 ثوانٍ، شريط تقدم HTML range، volume/mute، speed (0.75–2x)، subtitles، PiP عند دعمه، وFullscreen API.
- [VERIFIED] الضغط على الفيديو يبدل play/pause؛ double click يطلب seek يسار/يمين حسب نصف الصورة.
- [VERIFIED] controls تظهر عند mouse move/enter/focus وتختفي بعد 2.5 ثانية من التشغيل أو عند خروج الماوس، مع animation CSS.
- [VERIFIED] keyboard shortcuts المطبقة بعد التركيز داخل المشغل: Space/K، الأسهم/J/L، M، F، C.
- [VERIFIED] القائمة الجانبية في modal تسرد كل الملفات/الحلقات؛ وفي fullscreen فقط يظهر زر تحت timeline لقائمة الحلقات المتبقية، ويستدعي `onSelectFile` عند اختيار حلقة.
- [VERIFIED] عند `ended` يستدعي `onNext`، والذي يختار العنصر التالي من القائمة إن وجد.
- [VERIFIED] لا توجد حاليًا ميزة intro/recap/credits segment في code path؛ تمت إزالة migration المنشئة وبقي migration `0019_remove_playback_segments.sql` كـ `DROP TABLE IF EXISTS`.

## 15. HTTP Range Implementation

- [VERIFIED] لا توجد parser يدوي لـ `Range` في المشروع. Go `http.ServeContent` هو المنفذ الذي يفسر Range requests ويرسل partial responses عند طلبها.
- [VERIFIED] `Accept-Ranges: bytes` يضاف صراحة قبل `ServeContent`، والـ file handle يبقى مفتوحًا حتى نهاية الاستجابة عبر `defer file.Close()`.
- [VERIFIED] لا يقرأ handler الملف كاملًا في slice/ذاكرة قبل الإرسال؛ يعطي `*os.File` مباشرة إلى `ServeContent`.
- [INFERRED] عندما يرسل `<video>` طلب Range صالحًا، فإن `ServeContent` يعيد HTTP `206 Partial Content` بالـ headers القياسية. هذا سلوك Go القياسي وليس status مُعيّنًا يدويًا في الكود.
- [UNKNOWN] لا توجد قياسات throughput أو عدد اتصالات متزامنة/ملفات 4K في الـ Repository، لذلك لا يمكن إثبات سعة إنتاجية معينة.

## 16. FFmpeg / FFprobe Usage

- [VERIFIED] المسارات تأتي من `NEXORA_FFMPEG_PATH` و`NEXORA_FFPROBE_PATH`؛ الافتراض `ffmpeg`/`ffprobe` من PATH.
- [VERIFIED] FFprobe يشغّل `-show_streams -show_format -of json` ويستخرج duration، أول video codec/resolution، ومسارات audio وsubtitle، ثم يحفظها في PostgreSQL JSONB.
- [VERIFIED] FFmpeg يستخدم للتحقق الكامل `-v error -i file -f null -`، وتوليد thumbnail واحد عبر `-ss … -frames:v 1 -q:v 2`.
- [VERIFIED] streaming نفسه لا يستدعي FFmpeg؛ لا decoding/transcoding لكل عميل.
- [UNKNOWN] لا يمكن إثبات أن FFmpeg/FFprobe مثبتان وصالحان على كل جهاز نشر من الـ Repository وحده.

## 17. Preview Thumbnail System

```text
mousemove over timeline
  → client calculates timestamp
  → floor(time / 10) * 10 and 180 ms timer
  → GET /api/stream/file/{id}/preview?at={bucket}
  → if JPEG is absent: FFmpeg generates it
  → redirect 307 to /assets/images/previews/{id}/{bucket}.jpg
```

- [VERIFIED] `VideoPlayer` يعمل debounce بـ `setTimeout(..., 180)` ويمنع إعادة طلب bucket نفسه بواسطة `previewBucket`.
- [VERIFIED] backend يعيد تقريب timestamp لكل 10 ثوانٍ حتى لو أرسل العميل رقمًا مختلفًا.
- [VERIFIED] cache على القرص هو `AssetImageDir/previews/<fileID>/<second>.jpg`؛ بعد وجود الملف لا يعيد تشغيل FFmpeg، ويرسل redirect مع `Cache-Control: public, max-age=31536000, immutable`.
- [VERIFIED] لا توجد pre-generation jobs للـ preview thumbnails وقت الفهرسة.
- [VERIFIED] لا توجد lock/single-flight حول إنشاء نفس thumbnail؛ طلبان متزامنان لأول bucket قد يشغلان FFmpeg أكثر من مرة على نفس output.

## 18. Subtitle System

- [VERIFIED] `GET /api/stream/file/{id}/subtitles` يبحث فقط عن subtitle **خارجي** بجوار الفيديو (`.srt,.vtt,.ass,.sub`) ويطابق الاسم الأساسي بقاعدة contains/prefix/equality.
- [VERIFIED] endpoint آخر يفتح الملف، ويحوّل SRT إلى WebVTT في الاستجابة؛ VTT والامتدادات الأخرى تنسخ كما هي مع `Content-Type: text/vtt`.
- [VERIFIED] `VideoPlayer` يحصل على قائمة الترجمات في modal ويضيف `<track kind="subtitles">`؛ زر CC يدوّر text tracks بين showing/disabled.
- [VERIFIED] FFprobe يسجل مسارات subtitles المضمنة في الفيديو كـ metadata، لكن لا يوجد endpoint في الكود الحالي لاستخراج embedded subtitle track وتحويله إلى WebVTT.
- [INFERRED] إرسال ASS أو SUB خام مع MIME `text/vtt` قد لا يعمل في المتصفح؛ ذلك ليس تحويلًا فعليًا إلى VTT في الكود.

## 19. Watch Progress System

- [VERIFIED] التقدم لا يحفظ في PostgreSQL. `VideoPlayer` يكتب `nexora:playback:{fileId|src}` في `localStorage`.
- [VERIFIED] يحفظ كل 10 ثوانٍ تقريبًا، وعند pause/unmount/ended. عند الوصول إلى 95% يعد playback مكتملًا ويخزن position=0.
- [VERIFIED] عند metadata يطلب استئنافًا فقط إذا كان الموضع أكبر من 30 ثانية وأبعد من آخر 30 ثانية ولم يكتمل.
- [INFERRED] التقدم مرتبط بمتصفح/ملف تعريف المستخدم المحلي ولا يتزامن بين أجهزة الاستراحة أو مستخدمين.

## 20. Episode Navigation System

- [VERIFIED] modal يحول seasons/episodes إلى `allPlayableItems` أو يستخدم direct files، ويحدد العنصر الحالي والحلقة التالية بالترتيب الحالي للمصفوفة.
- [VERIFIED] `onEnded` يشغل التالية تلقائيًا إذا كانت موجودة؛ لا يوجد countdown أو تأكيد أو منطق skip intro قائم.
- [VERIFIED] قائمة fullscreen تعرض فقط `playlist.slice(currentIndex + 1)`، ولا تظهر خارج fullscreen.
- [UNKNOWN] لا يمكن تأكيد أن ترتيب query من backend يطابق دائمًا ترتيب بث مناسب في كل أنواع الملفات دون فحص بيانات واقعية لكل حالة.

## 21. Database Architecture

- [VERIFIED] migrations تعمل تلقائيًا عند startup، مرتبة lexicographically، وكل ملف غير مسجل في `schema_migrations` ينفذ داخل transaction ثم يسجل اسمه.
- [VERIFIED] الجداول الجوهرية: categories، media_items، seasons، video_files، storage_disks، metadata_snapshots، tmdb_settings، tmdb_usage_log، hub_definitions، provider_collections، media_collection_links، collection/person snapshots، people، media_credits، tmdb_refresh_queue.
- [VERIFIED] Postgres يحتفظ بالـ catalogue والـ metadata والـ paths والتقارير/الإعدادات، لا bytes الفيديو ولا progress.
- [VERIFIED] تتوفر indexes لمسار الفيديو، identity العمل، relations، metadata facets (GIN)، genres (GIN)، والـ queue.
- [VERIFIED] `0019_remove_playback_segments.sql` يحذف جدولًا اختياريًا فقط؛ لا يوجد ملف `0018_add_playback_segments.sql` في الشجرة الحالية.

## 22. Caching Architecture

| الطبقة | الحالة | التنفيذ |
|---|---|---|
| Browser API cache | Implemented | [VERIFIED] Map + `sessionStorage` في `api.js`، GET عامة، 2 دقيقة/health 15 ثانية. |
| Server catalogue cache | Implemented | [VERIFIED] L1 in-memory، 256 entries كحد أعلى، TTL 20 أو45 ثانية؛ يمسح عند أي non-GET. |
| Artwork | Implemented | [VERIFIED] TMDB/MAL cache محلي atomically تحت AssetImageDir، مع خيار remote mode. |
| Preview thumbnails | Implemented | [VERIFIED] JPEG دائم باسم fileId/time bucket وعلى القرص؛ browser cache سنة. |
| Redis | Missing in application code | [VERIFIED] service موجود في Compose فقط. |
| CDN/reverse proxy cache | Unknown | [UNKNOWN] لا توجد إعدادات deployment أو reverse proxy ضمن الشجرة. |

## 23. Concurrency and Background Processing

- [VERIFIED] scanner يطلق walkers للـ roots وworkers بعدد `NEXORA_SCAN_WORKERS` (افتراضي 8) مع channels bounded، وcallback serial.
- [VERIFIED] filesystem watcher يعمل في goroutine ويراجع root غير المتصل كل 5 ثوانٍ، ويشاهد recursively وفق `NEXORA_WATCH_RECURSIVE`.
- [VERIFIED] TMDB queue ticker كل 10 ثوانٍ؛ إذا auto-refresh مفعّل، يضيف stale items، ويشغل من 1 إلى 4 workers ثم ينتظرها.
- [VERIFIED] HTTP server يعالج requests وفق نموذج Go المعتاد لكل request goroutine.
- [VERIFIED] لا توجد worker queue للـ preview generation أو locking لكل cache key.

## 24. External Services

- [VERIFIED] TMDB قابل للضبط بمفتاح API أو Bearer token؛ يجلب details/candidates/configuration وصورًا، مع usage log وsettings/queue محلية.
- [VERIFIED] MAL موجود كـ fallback خاص بالـ anime إن كان configured؛ TMDB هو canonical provider في `metadata.Service` عندما يكون متاحًا.
- [VERIFIED] Meilisearch يستخدم للبحث وفهرسة `MediaDocument`، ويضبط searchable/filterable/sortable attributes عبر API.
- [VERIFIED] PostgreSQL وMeilisearch وRedis مُعرّفة في Compose؛ البيانات persistent في named volumes.
- [UNKNOWN] مفاتيح TMDB/MAL الفعلية، سياسات rate limit الحقيقية، والوصول للإنترنت في الإنتاج لا يمكن التحقق منها دون تشغيل/إعدادات سرية.

## 25. Current Architectural Decisions

| القرار | التقييم القائم على الكود | الحفاظ عليه؟ |
|---|---|---|
| React SPA + Go API | [VERIFIED] فصل واضح بين UI وعمليات القرص/البيانات؛ مناسب للمتصفح كعميل خفيف. | [INFERRED] نعم، لا يظهر نقص يستدعي استبداله. |
| Native HTML5 `<video>` + custom controls | [VERIFIED] يستفيد من pipeline المتصفح وFullscreen/PiP/text tracks بدون dependency runtime. | [INFERRED] نعم حاليًا؛ Plyr موجود لكنه غير مستخدم، فلا ينبغي إضافته بلا حاجة مثبتة. |
| Direct HTTP Range streaming | [VERIFIED] Go يفتح الملف ويستخدم `ServeContent` بلا تحميل كامل/بدون transcoding. | [INFERRED] مناسب لشبكة LAN وملفات متوافقة مع العملاء. |
| FFmpeg/FFprobe CLI | [VERIFIED] يقدمان inspect/verify/thumbnail خارج stream hot path. | [INFERRED] مناسب، مع الحاجة لإدارة concurrency لاحقًا عند زيادة الحمل. |
| PostgreSQL + Meilisearch | [VERIFIED] Postgres هو authoritative catalogue؛ Meilisearch index للبحث. | [INFERRED] مناسب للبحث العربي/الإنجليزي السريع؛ يتطلب التعامل مع تأخر مزامنة index. |
| Disk-based media library | [VERIFIED] paths تبقى على أقراص الخادم ولا تنسخ إلى DB. | [INFERRED] متسق مع مكتبة كبيرة محلية؛ يعتمد على استقرار mount letters/paths. |
| Browser-local watch progress | [VERIFIED] بسيط ولا يحتاج users table. | [INFERRED] مناسب لتجربة جهاز واحد، لا لتقدم موحد متعدد الأجهزة. |

## 26. Existing Strengths

- [VERIFIED] stream لا يحمل الفيلم كاملًا في ذاكرة Go ويستخدم file handle + `ServeContent`.
- [VERIFIED] الفهرسة bounded batching وتفحص metadata تقنيًا عبر FFprobe.
- [VERIFIED] cache متعدد الطبقات يقلل تكرار DB/API/FFmpeg في الحالات الطبيعية.
- [VERIFIED] migrations transactional وschema مفهرس للكتالوج الأساسي.
- [VERIFIED] preview timeline يطبق debounce و10-second buckets وdisk cache.
- [VERIFIED] metadata artwork يخزن atomically، وتوجد queue قابلة للتقييد لـ TMDB.
- [VERIFIED] اختبارات Go موجودة للـ API/scanner/media/metadata/migration/quality؛ وجودها لا يثبت وحده تغطية كاملة.

## 27. Current Technical Risks

- [VERIFIED] `mediaPathAllowed` يسمح أي مسار موجود على الجهاز حتى إن كان خارج `MediaRoots`; endpoints تعتمد عليه للبث/الصور وبعض عمليات الملفات.
- [VERIFIED] admin login لديه بيانات افتراضية `admin`/`admin123` إذا لم تضبط البيئة، ويعيد token ثابتًا؛ route middleware لا يتحقق من هذا token لحماية بقية admin write endpoints.
- [VERIFIED] CORS مفتوح `*`، وmethods المعلنة لا تشمل PUT/DELETE رغم استعمالهما.
- [VERIFIED] لا يوجد auth/authorization ظاهر حول streaming أو غالبية endpoints الإدارية.
- [VERIFIED] race محتمل لتوليد preview نفسه مع طلبات متزامنة؛ لا توجد single-flight/lock.
- [VERIFIED] event watcher لا يحذف record عند remove في `app.Run` callback.
- [VERIFIED] `README.md` يذكر Plyr/MediaInfo وخصائص لا تظهر جميعها كما هي في code path الحالي؛ التباين قد يسبب قرارات تشغيلية خاطئة.

## 28. Bottlenecks and Scalability Concerns

- [VERIFIED] أول hover لكل 10-second bucket يستدعي FFmpeg synchronous داخل request؛ عدة مستخدمين/لقطات جديدة قد تنافس CPU/disk.
- [VERIFIED] كل index يمر على كل ملف وFFprobe لكل ملف، ثم يطلب حتى 10,000 search document للمزامنة؛ ذلك عمل ثقيل لمكتبات ضخمة.
- [VERIFIED] server cache يحتفظ body responses في الذاكرة حتى 256 key؛ TTL قصير ولكن أحجام responses الكبيرة تؤثر على الذاكرة.
- [INFERRED] direct file serving سيضع حمل القراءة على الأقراص والشبكة مع كل عميل؛ لا يوجد ABR/transcoding أو CDN في الكود ليعالج clients بقدرات مختلفة.
- [UNKNOWN] لا توجد benchmark أو profile أو أرقام الأجهزة/عدد العملاء لتحديد bottleneck الفعلي.

## 29. Technical Debt

- [VERIFIED] dependency `plyr` وCSS `.plyr*` موجودان دون استخدام في المشغل الحالي.
- [VERIFIED] Redis مشغّل في Compose دون integration برمجية ظاهرة.
- [VERIFIED] `App.jsx` يجمع بيانات حقيقية وmock/fallback في مسار تفاصيل الوسائط، ما يخلق مصدرين للحقيقة في الواجهة.
- [VERIFIED] auth الحالي شكلي من ناحية route protection وليس session/auth system مكتملًا.
- [VERIFIED] docs API/README تتضمن أوصافًا أقدم/أوسع من الكود الحالي، ويجب مواءمتها بعد كل تغيير لاحق.
- [VERIFIED] subtitle pipeline الخارجي يعلن ASS/SUB كـ VTT دون تحويل ظاهر في الكود.

## 30. Missing Information / Unknown Areas

- [UNKNOWN] topology النشر الفعلية: هل Vite static build يقدم من Go/reverse proxy أم dev server؟
- [UNKNOWN] عدد العملاء المتزامنين، bitrates، سرعة الشبكة، وعمر/نوع أقراص الخادم.
- [UNKNOWN] codecs/containers الفعلية في المكتبة وهل كل browsers المستهدفة تدعمها native.
- [UNKNOWN] سياسة النسخ الاحتياطي، استعادة PostgreSQL، وحماية AssetImageDir.
- [UNKNOWN] من يملك صلاحية admin فعليًا، وهل التطبيق معزول على LAN موثوقة أم متاح لأجهزة غير موثوقة.
- [UNKNOWN] هل migration `0019` طُبقت بالفعل في كل قواعد البيانات الموجودة؛ وجود الملف لا يثبت حالة DB runtime.

## 31. Implemented vs Planned Features

| النظام | الحالة | كيف يعمل | الملفات الأساسية | ملاحظات |
|---|---|---|---|---|
| Catalog + file ingest | Implemented | scanner → repository → PostgreSQL | `scanner`, `repository`, `server.go` | [VERIFIED] |
| Search | Implemented | Meilisearch sync/search | `search/client.go` | [VERIFIED] يحتاج service عاملًا. |
| Direct video streaming | Implemented | `ServeContent`/Range | `api/server.go` | [VERIFIED] |
| Custom player | Implemented | Native `<video>` + React controls | `VideoPlayer.jsx` | [VERIFIED] |
| Timeline previews | Implemented | FFmpeg lazy disk cache | `VideoPlayer.jsx`, `server.go` | [VERIFIED] |
| External subtitles | Partial | listing + SRT→VTT | `media/subtitles.go` | [VERIFIED] embedded extraction غير منفذ. |
| Watch progress | Implemented, local-only | Browser localStorage | `VideoPlayer.jsx` | [VERIFIED] |
| Next episode / fullscreen list | Implemented | current array order and callbacks | `App.jsx`, `VideoPlayer.jsx` | [VERIFIED] |
| Related / similar titles | Implemented | TMDB-ID-backed relation graph مع مطابقة local/pending | `repository_related.go`, `MediaDetailsPage.jsx` | [VERIFIED] لا يستدعي TMDB أثناء التصفح. |
| TMDB/MAL enrichment | Implemented, configuration-dependent | HTTP clients + DB snapshots/cache | `metadata/*` | [VERIFIED] يحتاج credentials/network. |
| TMDB automatic refresh | Implemented, disabled by default | 10-second queue runner | `api/server.go` | [VERIFIED] settings default false. |
| Redis cache/queue | Missing | Compose service only | `compose.yml` | [VERIFIED] |
| HLS/DASH/transcoding/ABR | Missing | لا توجد manifests أو pipeline | API/media code | [VERIFIED] |
| Multi-user server-side progress | Missing | لا schema/API ظاهر | DB/player | [VERIFIED] |
| Production capacity proof | Unknown | لا benchmarks | repository | [UNKNOWN] |

## 32. Recommended Architecture Direction

- [INFERRED] حافظوا على مسار LAN الحالي: React browser client + Go direct Range server + PostgreSQL catalogue + Meilisearch؛ الكود الحالي يثبت أنه مبني حول هذا المسار.
- [INFERRED] اعتبروا PostgreSQL مصدر الحقيقة للكتالوج، وMeilisearch index قابلًا لإعادة البناء، وAssetImageDir/preview cache بيانات قابلة لإعادة الإنشاء وليست المصدر الوحيد.
- [INFERRED] أي تطوير playback لاحق ينبغي أن يظل خلف props/API مستقرة في `VideoPlayer` وأن لا يفرض framework فيديو إضافيًا قبل ظهور حاجة قابلة للقياس.
- [INFERRED] قبل توسيع الوصول خارج LAN، يجب اعتبار الأمن والـ path policy/auth من المتطلبات الأساسية، لا تحسينات UI.

## 33. Proposed Future Improvements

> هذه مقترحات لاحقة فقط؛ لم تُنفذ ضمن مهمة التحليل.

1. [PLANNED] وضع حدود/queue وsingle-flight لتوليد preview thumbnails، أو pre-generate اختياريًا وقت الفهرسة للمكتبات المستخدمة بكثافة.
2. [PLANNED] فصل path authorization الصارم عن وجود الملف على القرص، وإضافة auth حقيقي وحماية routes الإدارية إذا كان الوصول غير موثوق.
3. [PLANNED] تحويل subtitle formats غير WebVTT فعليًا أو تقييد القائمة إلى formats قابلة للعرض في browser.
4. [PLANNED] تحديد سياسة واضحة للـ codec compatibility وقرار transcoding/HLS فقط إذا أثبتت الأجهزة المستهدفة عدم دعم الملفات الأصلية.
5. [PLANNED] إزالة أو تفعيل dependencies/services غير المستخدمة (Plyr/Redis) بعد قرار مقصود، وتحديث README/API docs لتطابق code.
6. [PLANNED] إن كان المستخدم ينتقل بين أجهزة، تصميم watch progress server-side بمفهوم مستخدم/جهاز وسياسة خصوصية واضحة.
7. [PLANNED] إضافة قياسات تشغيلية: latency للـ range/preview، FFmpeg queue depth، disk throughput، وعدد streams متزامنة قبل تغيير architecture.

## 34. Files and Components Map

| الملف/المجلد | المسؤولية |
|---|---|
| `client/src/main.jsx` | Bootstrap React/providers/error boundary |
| `client/src/App.jsx` | routes، global UI state، modal integration |
| `client/src/lib/api.js` | API client وbrowser response cache |
| `client/src/components/VideoPlayer.jsx` | native video controls/progress/preview/progress persistence/fullscreen episodes |
| `client/src/layouts/*` | customer/admin shells |
| `server/internal/app/app.go` | startup composition، DB migrations، watcher، HTTP lifecycle |
| `server/internal/api/server.go` | endpoints، stream، previews، subtitles، middleware |
| `server/internal/api/cache.go` | server L1 catalogue response cache |
| `server/internal/db/migrate.go` | migration runner |
| `server/internal/db/repository.go` | SQL persistence/queries |
| `server/internal/scanner/*` | scan، parse، artwork discovery، FS watcher |
| `server/internal/media/processor.go` | FFprobe/FFmpeg inspect/verify/thumbnail |
| `server/internal/media/subtitles.go` | external subtitle discovery وSRT→WebVTT |
| `server/internal/metadata/*` | TMDB/MAL lookup/cache/settings |
| `server/internal/search/client.go` | Meilisearch indexing/search |
| `server/internal/quality/*` | duplicates/missing/corruption reports |
| `server/internal/migration/*` | file copy/resume/checksum workflow |
| `server/migrations/*.sql` | PostgreSQL schema evolution |
| `compose.yml` | local PostgreSQL/Meilisearch/Redis services |

## Questions for Project Owner

1. [UNKNOWN] هل سيبقى النظام محصورًا داخل LAN موثوقة، أم سيُفتح لاحقًا عبر الإنترنت أو Wi‑Fi للضيوف؟ هذا يغير أولويات auth وCORS وpath access.
2. [UNKNOWN] ما العدد المتوقع للمشاهدين المتزامنين، وأقصى bitrate/resolution فعلي للملفات؟ يلزم ذلك قبل اختيار سياسة preview generation أو أي Transcoding.
3. [UNKNOWN] هل أجهزة العملاء متجانسة (Chrome/Windows) أم تشمل Smart TVs/iOS/Android؟ ذلك يحدد مدى كفاية direct MP4/MKV playback.
4. [UNKNOWN] ما حجم المكتبة المستهدف وعدد/نوع الأقراص، وهل تتغير أحرف الأقراص أو تستبدل؟ ذلك يؤثر على paths والفهرسة والمراقبة.
5. [UNKNOWN] هل يراد حفظ «أكمل المشاهدة» لكل مستخدم عبر كل الأجهزة أم لكل جهاز فقط؟
6. [UNKNOWN] هل مطلوب دعم embedded subtitles وASS/SUB، أم يكفي WebVTT/SRT خارجي؟
7. [UNKNOWN] ما سياسة backup/restore لقاعدة البيانات والصور cached، ومن المسؤول عن إدارة مفاتيح TMDB/MAL؟
