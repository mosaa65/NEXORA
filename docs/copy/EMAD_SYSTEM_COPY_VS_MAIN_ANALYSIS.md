# تحليل شامل: فرع `emad_system_copy` مقابل `main` + فهم مشروع NEXORA وقسم النسخ عبر USB

> إعداد: تحليل مقارن كامل للفروع + مراجعة التوثيق + تفكيك نظام النسخ عبر USB.
> المرجع التقني: `git diff emad_system_copy main`, `git log`, ملفات `docs/`، وفحص الكود الفعلي في الفروع.
> التاريخ: 2026-09-09

---

## 1. الإجابة المختصرة

- **`emad_system_copy`** هو فرع العمل الذي بُنيت فيه **ميزة النسخ عبر USB / نقل الملفات للأجهزة** (Transfer Engine v2 + iOS عبر go-ios + Android MTP + أقراص USB).
- **`main`** يحتوي هذا الفرع **مدمجًا بالكامل** (`emad_system_copy` سلف لـ `main`، وليس العكس)، ويضيف فوقه:
  1. نظام **العناوين/الأشخاص ذات الصلة** (PR #3 + ADR-007).
  2. **إصلاحات transfer**: دعم `file_ids` متعددة + حل المسار خادمًا + تفضيل تخزين USB.
  3. **إصلاح SSE**: منع gzip لتدفق الأحداث الحية (commit `a1ca7f9`).
  4. **إصلاح الكاش**: استثناء `/api/transfer/` + احترام `cache:"no-store"` (ميزة من `main` — أنتجت المشكلة 1 في `SOLVED_ISSUES.md` ثم حُلّت).
  5. **إصلاحات تنقل وتمرير**: إيقاف تمرير المتصفح التلقائي، استعادة موضع التمرير، `PageBackButton` عبر `navigate(-1)`.
  6. **إصلاحات بطاقات وواجهات**: `FilterToolbar`, `ShowcaseHero`, `HubBannerCard`, `UnifiedMediaCard`, `TopBar`, `CustomerCinemaLayout`, `DashboardPage`, وغيرها.
  7. **تحسينات أداء/خادم**: إغلاق Redis عند الفشل، TMDB queue متعددة الـ workers، `repository.go` موسّع.

> **الخلاصة**: لا يوجد كود نسخ عبر USB فريد في `main` غير موجود في `emad_system_copy`؛ كل القسم متطابق تمامًا بين الفرعين. الفروقات كلها **إضافات فوق النسخ**.

---

## 2. طوبولوجيا الفروع (Git Topology)

| البند | القيمة |
|---|---|
| الفرعان المحليان | `emad_system_copy`, `main` |
| الفرعان على origin | `origin/emad_system_copy`, `origin/feature/emad_system_copy`, `origin/main` |
| merge base بينهما | `d20c9336fbc6659d12390b40f831611481716526` |
| هل `emad_system_copy` سلف لـ `main`؟ | **نعم** (`git merge-base --is-ancestor emad_system_copy main` exit 0) |
| Commits موجودة في `emad_system_copy` وليست في `main` | **صفر** |
| Commits موجودة في `main` وليست في `emad_system_copy` | ~20 (في القائمة أدناه) |
| الملفات المختلفة بين الفرعين | 53 ملفًا (بها إضافات `main` فقط) |

### Commits الخاصة بـ `main` (فوق `emad_system_copy`) — المهمة

| Commit | المعنى |
|---|---|
| `a1ca7f9` | fix(api): منع gzip لتدفق SSE progress + فلاش مباشر لاصق gzip |
| `b5cd7bb` | merge: دمج إضافات USB المحلية فوق origin/main (مع حذف test media الضخم) |
| `d6a75b7` | Merge PR #3 من `mosaa65/feat/related-titles-and-people` |
| `ea9085d` | Merge origin/emad_system_copy into main — دمج نظام النسخ دون regressions |
| `25a3d96` | fix(transfer): حل المسارات المفقودة بتعريض `file_path`, دعم `file_ids`, وتفضيل تخزين USB |
| `9c36761` | feat: إضافة اكتشاف العناوين والأشخاص ذات الصلة |
| `a0bf537` | fix(navigation): تعطيل تمرير المتصفح التلقائي |
| `7cf54b2` | fix(scroll): استعادة موضع التمرير عند الرجوع |
| `ecf8957` | fix(navigation): إصلاح التنقل والكاش على مستوى النظام |
| `8b6f65c`, `9f68efa`, `feaa37d`, `9173971`, `7053d8e`, `fcb5935` | إصلاحات بطاقات وعروض وصفحات وHero |
| `533dfe2` | fix: تصحيح checksum لـ `golang.org/x/sync` في go.sum |

---

## 3. الفرق الكامل في الملفات (53 ملفًا)

```text
M  .env.example
M  .gitignore
M  client/src/App.jsx
A  client/src/components/CopyToPhoneModal.jsx
M  client/src/components/FilterToolbar.jsx
M  client/src/components/HubBannerCard.jsx
M  client/src/components/PageBackButton.jsx
A  client/src/components/RelatedMediaRail.jsx
M  client/src/components/ShowcaseHero.jsx
M  client/src/components/TopBar.jsx
M  client/src/components/UnifiedMediaCard.jsx
M  client/src/components/TransferModal.jsx        (ضمن transfer)
A  client/src/components/transfer/FloatingCopyButton.jsx
M  client/src/context/NavigationStateContext.jsx
M  client/src/context/TransferContext.jsx
M  client/src/hooks/useScrollRestoration.js
M  client/src/layouts/CustomerCinemaLayout.jsx
M  client/src/lib/api.js
M  client/src/main.jsx
M  client/src/pages/{CategoryPage,DashboardPage,DirectoryPage,MediaDetailsPage,PersonPage,SmartHubPage}.jsx
M  client/src/pages/admin/AdminSmartHubsPage.jsx
M  client/vite.config.js
M  docs/ARCHITECTURE.md
M  docs/PROJECT_ANALYSIS.md
M  docs/api/ENDPOINTS.md
A  docs/decisions/ADR-007-provider-id-related-titles.md
M  docs/decisions/README.md
M  server/go.mod, server/go.sum
M  server/internal/api/{cache.go, gzip.go, handlers_media.go, handlers_transfer.go, server.go, server_test.go}
M  server/internal/db/{repository.go, repository_catalog.go, repository_media.go, repository_metadata.go}
A  server/internal/db/repository_metadata_test.go
A  server/internal/db/repository_related.go
M  server/internal/transfer/service.go
A  server/migrations/0021_add_media_related_titles.sql
A  server/migrations/0021_remove_duplicate_arabic_series_hub.sql
```

---

## 4. تفصيل الفروقات الجوهرية (main فوق emad_system_copy)

### 4.1 نظام العناوين والأشخاص ذات الصلة (Related Titles / People)

- جديد بالكامل في `main`: `server/internal/db/repository_related.go` + migration `0021_add_media_related_titles.sql`.
- ميزة مطابقة الهوية بـ (provider + TMDB external ID + kind) بدل العنوان، مع إرجاع `local: true/false` و`local_media_id`.
- واجهة جديدة: `client/src/components/RelatedMediaRail.jsx` في `MediaDetailsPage`.
- موثق في `docs/decisions/ADR-007-provider-id-related-titles.md`.
- كما أُضيف migration لإزالة تكرار «العربية» في المحاور: `0021_remove_duplicate_arabic_series_hub.sql`.
- ⚠️ **مشكلة ملاحظة**: ملفا migration يتشاركان نفس الرقم `0021` — ترتيب lexicographic قد يجعل سلوك التطبيق يعتمد على ترتيب الأسماء، وهو ما يستحق التفتيش.

### 4.2 إصلاحات النسخ عبر USB في `main`

| الملف | التغيير |
|---|---|
| `handlers_transfer.go` | دعم `FileIDs []int64` متعددة، وحلّ المسارات خادمًا عبر `GetVideoFilePath`، وتراجع `SourcePath` إلى أول `SourcePaths` |
| `service.go` | اسم iOS افتراضي `iPhone_%d` بدل اسم فارغ؛ قراءة `GetValue("TotalDataAvailable")`/`GetValue("TotalDiskCapacity")` بدون بادئة `com.apple.disk_usage`؛ وجهة افتراضية `"Movies"` لأقراص USB عند غياب `TargetFolder` |

### 4.3 إصلاح SSE الحية (commit `a1ca7f9`)

- `server/internal/api/gzip.go`: استثناء `text/event-stream` من الضغط + تنفيذ `Flush()` (http.Flusher) لدفق أحداث progress فورًا.

### 4.4 إصلاح كاش الواجهة (سياق التحليل)

- في عصر `emad_system_copy` كان `client/src/lib/api.js` يعتبر أي GET (بما فيها `/api/transfer/*`) قابلاً للتخزين لمدة دقيقتين **ويتجاهل** `cache:"no-store"` → نسبة النسخ تتجمد.
- في `main` أصبح: `isCacheableRequest = GET && cache!=="no-store" && !/api/admin/ && !/api/stream/ && !/api/transfer/`.
- هذه معالجة المشكلة 1 في `docs/copy/SOLVED_ISSUES.md`.
- `CopyToPhoneModal.jsx` (في `main`) مكوّن مرجعي غير مستورد (لا يزال polling على `job/{id}` كل 500ms).

### 4.5 Redis صامد عند الفشل

- `server/internal/api/cache.go`: عند عدم وصول Redis → `_ = rdb.Close()` + `redisActive=false` (تعمل L1-Memory فقط بدل الاحتفاظ بعميل معطوب).

### 4.6 التنقل والتمرير (Scroll Restoration)

- `main.jsx`: `window.history.scrollRestoration = "manual"`.
- `App.jsx`: ScrollManager يمرر للأعلى فقط عند PUSH/REPLACE، وليس POP.
- `useScrollRestoration.js` + `NavigationStateContext.jsx`: حفظ موضع التمرير وعدم الكتابة فوق قيمة موجبة بـ 0.
- `PageBackButton.jsx`: يستخدم `useNavigate` + `navigate(-1)` بدل `history.back()`.

### 4.7 واجهات وعروض

- `FilterToolbar.jsx`: خاصيتا `formatLabel`/`statusLabel`.
- `ShowcaseHero.jsx`: `DEFAULT_FALLBACK_SLIDES`.
- `MediaDetailsPage.jsx`: `handleBack` عبر `navigate(-1)`، `RelatedMediaRail`، `FEATURED_CAST_LIMIT = 24`.
- `HubBannerCard`, `UnifiedMediaCard`, `TopBar`, `CustomerCinemaLayout`, `DashboardPage`, `CategoryPage`, `DirectoryPage`, `PersonPage`, `SmartHubPage`, `AdminSmartHubsPage`: تحسينات عرض/تنقل.

### 4.8 الخادم (`server.go` / repository)

- واجهة repository موسّعة: `UpdateCategory` ترجع error، `ListCollections`, `SaveCollection`, `DeleteCollection`, `ListSearchDocuments`, `ListVideoFiles`, `GetVideoFileIDByPath`, `ListRelatedMedia` — وإزالة `GetSubtitlesForFile`/`GetSubtitleByID`.
- حلقة TMDB queue: إضافة `AutoRefreshEnabled` + `EnqueueStaleTMDBRefreshes` + تشغيل 1–4 workers متوازية بـ WaitGroup.
- `repository.go`: `VideoFile.FilePath` أصبح `json:"file_path,omitempty"` بدل `json:"-"`.

---

## 5. فهم المشروع NEXORA (من docs)

- **ما هو**: نظام ويب لإدارة مكتبة وسائط محلية LAN وبثها. الواجهة React 18/Vite، الخادم Go 1.22 `net/http`، PostgreSQL 16 مصدر الحقيقة، Meilisearch فهرس بحث مشتق، ومتصفح يلعب عبر `<video>` أصلي.
- **المسار الرئيسي**: ملفات فيديو على القرص ← فهرس في PostgreSQL ← React يستعلم API ← `<video>` يطلب من Go عبر `http.ServeContent` (Range مباشر).
- **قرارات معمارية (ADRs موجودة)**:
  - ADR-001 Native HTML5 + controls React مخصصة (وليس Plyr).
  - ADR-002 Direct HTTP Range streaming عبر Go `ServeContent`.
  - ADR-003 Filesystem-Based Media Storage (لا BLOBs).
  - ADR-004 PostgreSQL ككتالوج مصدر الحقيقة.
  - ADR-005 Meilisearch فهرس مشتق قابل لإعادة البناء.
  - ADR-006 FFmpeg/FFprobe أدوات معالجة خارج hot path.
  - ADR-007 Related Titles Graph بروابط provider-ID.
- **مخاطر معروفة (no-touch بدون موافقة)**:
  - `mediaPathAllowed` يسمح أي مسار موجود على القرص خارج roots.
  - Admin token ثابت/افتراضي `admin/admin123`.
  - Preview race عند أول طلب متزامن.
  - Subtitle formats خارجية تُرسل كـ VTT دون تحويل فعلي.
  - Watch progress محلي (localStorage).
  - Watcher لا يحذف سجل DB عند حذف الملف.
  - فجوة توثيق (Plyr/Redis/Transcoding في docs قديمة).

---

## 6. تفكيك قسم النسخ عبر USB (كما في الوثائق والكود المتطابق بين الفرعين)

### 6.1 المعمارية (ثلاث طبقات)

1. **اكتشاف الأجهزة** (`service.go`):
   - iOS: `goios.ListDevices()` عبر usbmuxd (بديل libimobiledevice الخارجي).
   - Android/iOS-placeholder: PowerShell `Shell.Application` (MTP/WPD).
   - أقراص USB: `Get-Volume` + DriveType=Removable → `disk_<letter>` (fallback فقط).
   - مراقبة كل 1.5 ثانية + كاش ساخن 4 ثوانٍ + `scanDevicesInternal` بالترتيب.
2. **الوصول للنسخ** (واجهة `TransferBackend`):
   - `GoIOSBackend` — house_arrest + AFC مباشرة في Go، Resume عبر APPEND.
   - `AndroidBackend` — `CopyHere(16)` غير متزامن + انتظار حجم الملف كل 1200ms.
   - `StorageBackend` — `os.Open → os.Create → io.Copy` مع Resume.
3. **جدولة وتقدم** (v2):
   - `TransferEngine` + `Scheduler` (worker لكل جهاز) + `DeviceWorker` + `Verifier` (حجم افتراضي) + `TransferError` مصنّف.
   - تقدم EMA (0.7 قديم + 0.3 لحظي) ≥ 500ms، مرآة على REST كل 250ms.

### 6.2 واجهات REST الخاصة بالنقل (متطابقة بين الفرعين)

| الطريقة | المسار | الغرض |
|---|---|---|
| GET | `/api/transfer/devices` | الأجهزة |
| GET | `/api/transfer/device-apps` | تطبيقات iOS (File Sharing) |
| GET | `/api/transfer/device-app-folders` | مجلدات تطبيق iOS |
| POST | `/api/transfer/copy` | بدء نسخ |
| GET | `/api/transfer/jobs` | كل الوظائف |
| GET | `/api/transfer/job/{id}` | وظيفة واحدة |
| POST | `/api/transfer/cancel/{id}` | إلغاء |
| GET | `/api/transfer/events` | **SSE** (بذر لقطات + `event:` مسمى + keep-alive 20s) |
| GET | `/api/transfer/browse` | تصفح مجلد سفلي |
| POST | `/api/transfer/mkdir` | إنشاء مجلد |
| POST | `/api/transfer/eject` | طرد جهاز |

> ملاحظة: وثيقة `USB_COPY_SYSTEM_ANALYSIS.md` (2026-09-04) ما زالت تصف «لا يوجد SSE — polling فقط»، بينما **الكود الحالي يحتوي SSE** (`events.go`, `handleTransferEvents`, `SetNotifier`, `SubscribeEvents`) في **كلا الفرعين**. هذا انحراف توثيقي يجب تحديثه.

### 6.3 تدفق البيانات في الواجهة (React)

```
MediaDetailsPage / AdminTransferPage / FloatingCopyButton
        └─ openTransferModal(file...)
                ▼
        TransferContext (EventSource SSE واحد + startTransfer/cancelJob)
                │
                ├─ TransferModal (اختيار جهاز/تطبيق/مجلد)
                └─ MiniTransferCenter (تقدم لحظي)
                │
                └─ lib/api.js → POST /api/transfer/copy , cancel, mkdir...
```

- **قاعدة**: كل قراءة حية عبر SSE؛ كل أمر عبر HTTP صريح؛ لا polling للبيانات الحيوية (عدا `CopyToPhoneModal` و`AdminTransferPage`).
- حالة الوظائف: `queued/pending/processing/retrying/waiting_device/connecting/checking_destination/copying/verifying/completed/failed/cancelled`.

### 6.4 القضايا المحلولة (SOLVED_ISSUES.md)

1. تجمد نسبة النسخ 1% — كاش `api.js` + تجاهل `no-store` → استثناء `/api/transfer/`.
2. «جاري إرسال الطلب…» للنسخة الثانية — `isSubmitting` لم يُعد → useEffect عند الفتح + بعد النجاح.
3. اختفاء شريط التقدم في النسخة الثانية — `dismissed` محلية دائمة → رُفعت إلى `TransferContext` + `setCenterDismissed`.
4. النسبة ثابتة بعد الانتقال لـ SSE — تباين `v2.ID` ≠ `job.ID` يمنع نشر الأحداث → `v2.ID = job.ID` في `runTransferV2` + updater نقي في React.
5. polling القديم مع الكاش → نظام SSE حي (بذر + دمج).

---

## 7. توثيق موجود يستحق الاهتمام (أوضاع خاصة إلى emad_system_copy)

- `USB_COPY_SYSTEM_ANALYSIS.md` (2026-09-04) — تحليل معمّق مسبق، **قبل** إصلاحات الكاش ودمج SSE في `main`. موظوعه «المشكلة = كاش الواجهة؛ الإصلاح مقترح فقط» — لكنه **نُفّذ بالفعل** لاحقًا (المشكلة 1 في SOLVED_ISSUES.md). يُقترح تحديث القسمين 5.1/6.5.
- `NEXORA_TRANSFER_ENGINE_V2_DEVELOPMENT_PLAN.md` — 2995 سطرًا، أغلبها (قصد 9, 20, 582–618, 1881–1903, 2884–2935) **تاريخي** مبني على `afcclient` القديم رغم أن التنفيذ الفعلي أصبح go-ios.

---

## 8. التوصيات

1. **حل تعارض رقم migration `0021`** (ملفان يحملان نفس الرقم) بترقية أحدهما.
2. **تحديث** `USB_COPY_SYSTEM_ANALYSIS.md` ليذكر أن SSE هو النقل الحي الحالي (بدل «polling فقط»)، وأن إصلاح الكاش نُفّذ.
3. **توثيق REST الفعلي** في `docs/api/ENDPOINTS.md` ليشمل `/api/transfer/*` وخاصة `events` و`eject` (لا يظهر قسم transfer في ENDPOINTS.md حاليًا).
4. **التحقق**: `go build ./...`, `go test ./internal/transfer/...`, `npm --prefix client run build` لمن يؤكد سلوك الفرعين قبل أي دمج/تفريع.
5. **مراجعة الإبقاء على** `CopyToPhoneModal.jsx` غير المستورد أو تحويله ليعتمد على `TransferContext`.

---

## 9. الأسئلة المفتوحة للمالك

1. هل يُراد إغلاق/حذف فروع `emad_system_copy` و`feature/emad_system_copy` بعد التأكد من الدمج، أم إبقاؤها للتتبع؟
2. هل يُراد توحيد رقمي migrations `0021` الآن (طلب منفصل صغير)؟
3. هل `CopyToPhoneModal.jsx` قابل للحذف أم يُحتفظ به كمرجع مقصود؟

---

## 10. الخلاصة

- `emad_system_copy` = **مصدر نظام النسخ عبر USB**، وهو مدمج بالكامل في `main` (صفر commits فريدة).
- `main` = `emad_system_copy` **+** ميزات/titles/people + إصلاح نقل + إصلاح kاش/SSE + إصلاحات تنقل/تمرير + إصلاحات بطاقات + تحسينات خادم.
- قسم النسخ عبر USB متطابق تمامًا بين الفرعين؛ أي «فرق في النسخ» يُعزى لتغييرات `service.go`/`handlers_transfer.go` الصغيرة المذكورة في 4.2 فقط.
- الانحرافات التوثيقية الرئيسية: `USB_COPY_SYSTEM_ANALYSIS.md` (لا SSE) و`NEXORA_TRANSFER_ENGINE_V2_DEVELOPMENT_PLAN.md` (afcclient تاريخي) — وكلاهما لا يعكسان الكود الحالي الكامل.

---

*نهاية التحليل. كل الفروقات مؤكدة بأمر `git diff emad_system_copy main` والـ git log أعلاه.*