# 📡 NEXORA API Endpoints Reference

> دليل نقاط الاتصال البرمجية الخاصة بخادم NEXORA (Go Backend API).
>
> **ملاحظة مهمة:** مسارات النقل عبر USB (`/api/transfer/*`) و`/api/health` الخاصة بالنسخ تُدار بواسطة **خدمة NEXORA Copy Bridge المحلية** على `http://127.0.0.1:32145` وليست من مسارات السيرفر المركزي — راجع قسم **Copy Bridge** بالأسفل. نسخة السيرفر المركزي من `/api/transfer/*` معطّلة افتراضيًا وتُفعّل بـ `NEXORA_SERVER_USB_TRANSFER=true`.

---

## 🏥 فحص النظام والحالة (System & Health)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/health` | `GET` | فحص صحة الخادم، قاعدة البيانات، والخدمات المرتبطة |
| `/api/disks` | `GET` | جلب قائمة الأقراص وسعات التخزين المتاحة |
| `/api/disks/scan` | `POST` | إعادة مسح الأقراص الموصولة بالسيرفر |

---

## 🗂️ الأقسام والتصنيفات (Categories)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/categories` | `GET` | جلب كافة الأقسام مع إحصائيات الأعمال والملفات |
| `/api/categories` | `POST` | إضافة قسم جديد |
| `/api/categories/:id` | `PUT` | تحديث بيانات قسم |
| `/api/categories/:id` | `DELETE` | حذف قسم |

---

## 🎬 الأعمال والوسائط (Media Items & CMS)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/media` | `GET` | استعلام الأعمال مع دعم الفلترة (category, sort, q, limit, offset) |
| `/api/media/:id` | `GET` | جلب التفاصيل الشاملة لعمل ما (المواسم والحلقات والملفات) |
| `/api/media/:id/related?limit=18` | `GET` | توصيات وأعمال مشابهة محفوظة محليًا من TMDB؛ تحدد `local_media_id` و`local` بالـ TMDB ID عند توفر العمل في المكتبة |
| `/api/media` | `POST` | إنشاء عمل جديد يدوياً |
| `/api/media/:id` | `PUT` | تعديل بيانات عمل |
| `/api/media/:id` | `DELETE` | حذف عمل وملفاته |
| `/api/media/:id/enrich` | `POST` | جلب وتحديث بيانات العمل التلقائية من TMDB/MAL |
| `/api/media/classify-folders` | `POST` | التصنيف التلقائي للبلد والنوع من أسماء المجلدات الفيزيائية |

## TMDB والتحكم بالتحديث والطابور

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/tmdb/candidates?title=...&type=...&year=...` | `GET` | عرض مرشحي TMDB لاختيار العمل الصحيح يدويًا قبل الإثراء |
| `/api/media/:id/enrich?tmdb_id=...` | `POST` | إثراء عمل بمعرف TMDB معتمد، بدل إعادة اختيار نتيجة البحث |
| `/api/tmdb/settings` | `GET/PUT` | إعدادات اللغة والصور وسياسة التحديث التلقائي والطابور |
| `/api/tmdb/queue` | `GET/POST` | عرض مهام التحديث أو إضافة عمل إلى الطابور |
| `/api/tmdb/queue/:id/cancel` | `POST` | إلغاء مهمة قيد الانتظار |
| `/api/tmdb/usage/history?days=90` | `GET` | ملخص استهلاك يومي للطلبات والبيانات والصور والنجاح والفشل |

---

## 🔎 البحث والمزامنة (Search & Indexing)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/search` | `GET` | البحث الفوري اللحظي عبر Meilisearch (`?q=...`) |
| `/api/search/sync` | `POST` | إعادة بناء فهرس البحث كـ projection من PostgreSQL (admin). `?reset=true` يبدأ من الصفر، وبدونه **يستكمل** من الـ cursor |
| `/api/index` | `POST` | تشغيل مسار الفهرسة (admin). الحقول: `roots`, `mode` (`full`/`incremental`، الافتراضي `incremental`), `inspect`, `syncSearch` |
| `/api/ingest` | `POST` | مرادف لـ `/api/index` (توافق مع العملاء القدامى) |
| `/api/index/preview` | `POST` | معاينة Dry-Run بدون أي كتابة لقاعدة البيانات |
| `/api/scan/status` | `GET` | حالة الفحص الجاري وتقدمه الحقي + `control` + `workers`، أو آخر جلسة مكتملة |
| `/api/scan/workers` | `GET` | تفصيل كل worker: الدور + الملف الحالي + العدد المُعالج |
| `/api/scan/pause` | `POST` | طلب توقف تعاوني (admin) — يعيد `pausing` غالبًا |
| `/api/scan/resume` | `POST` | رفع التوقف ومتابعة من نفس النقطة (admin) |
| `/api/scan/cancel` | `POST` | طلب إلغاء الفحص الجاري (admin) — عملية مختلفة عن Pause |

### واجهات الإدارة (React)

| المسار | المكوّن | ما يعرضه |
|---|---|---|
| `/admin/indexer` | `ScanControlCenter` | تقدم حقي + جدول العوامل + Pause/Resume/Cancel + اختيار وضع الفهرسة |
| `/admin/review` | `ResolutionReviewCenter` | قائمة المراجعة مجمّعة بالسبب + المرشحين بتفصيل الأدلة + نموذج القرار |

> الـ polling في مركز التحكم **يتوقف** عند انتهاء الفحص أو إغلاق الصفحة، فلا يستمر طلب شبكي بلا داعٍ. والإلغاء يطلب **تأكيدًا صريحًا** لأنه يفقد العدّادات، بينما الإيقاف المؤقت لا يفقد شيئًا.
| `/api/scan/interrupted` | `POST` | جلسات فحص لم تكتمل (انقطاع السيرفر) (admin) |

### التحكم في الفحص (Pause / Resume / Cancel)

```text
RUNNING ──(pause)──► PAUSING ──(لا worker وسط عنصر)──► PAUSED
   ▲                                                    │
   └──────────────(resume)── RESUMING ◄─────────────────┘
```

| العملية | المعنى | ماذا يُحفظ |
|---|---|---|
| **Pause** | تعليق قابل للاستكمال | counters + cursor + per-root state + worker state |
| **Resume** | متابعة من نفس النقطة | لا يُفقد أي عمل |
| **Cancel** | إنهاء نهائي بحالة `CANCELLED` | يُطلق أي worker محجوب بالـ pause أولًا |

**Pause ليس بديلًا عن Cancel.** استخدام Cancel للتوقف المؤقت يمحو التقدم الذي يحفظه Pause عمدًا. تفاصيل القرار: [ADR-011](../decisions/ADR-011-search-projection-and-scan-control.md).

**الضمانة:** بوابة الـ pause تقع قبل سحب عنصر عمل جديد، لذلك **لا يمكن لـ pause أن يُنتج صفًا نصف مكتوب** — الـ worker الذي بدأ ملفًا يُكمله.

#### استجابة `/api/scan/workers`
```json
{
  "running": true,
  "scanState": "running",
  "workers": [
    {"id": 0, "role": "discovery",   "active": true,  "currentPath": "D:/Media", "processed": 12},
    {"id": 1, "role": "metadata",    "active": true,  "currentPath": "D:/Media/Show/S01E04.mkv", "processed": 812},
    {"id": 2, "role": "persistence",  "active": true,  "currentPath": "D:/Media/Movies/a.mkv", "processed": 811},
    {"id": 3, "role": "idle",        "active": false, "processed": 790}
  ],
  "activeWorkers": 3,
  "idleWorkers": 1,
  "draining": 0
}
```
الأدوار: `discovery` (يمشي المجلدات) · `metadata` (يحلل ملفًا) · `persistence` (يكتب) · `idle`. الترتيب ثابت بالمعرّف.
| `/api/resolution/queue` | `GET` | الملفات التي رفض Entity Resolution الحسم فيها (admin). معاملات: `reason`, `limit` (حد 500) |
| `/api/resolution/stats` | `GET` | عدد الملفات المنتظرة مراجعة، مُرتّبة حسب السبب (admin) |
| `/api/resolution/queue/{id}/decide` | `POST` | قرار المشغّل على ملف غامض (admin). الحقول: `action` (attach/create/ignore/mark_movie/move_season), `work_id`, `season`, `episode`, `new_title`, `learn_alias` |

### قائمة المراجعة (Resolution Queue)

هذه النقاط هي ما يجعل ملفًا غامضًا **قابلًا للمعالجة** بدل أن يبقى عالقًا. الرد يُضمّن المرشحين مع تفصيل أدلتهم (`breakdown`) حتى يرى المشغّل سبب رفض الحسم، ويختار بدل أن يُعيد الكتابة.

قرار المشغّل يُسجّل للتدقيق (`decided_by`، `decided_at`) وعند `learn_alias=true` يُرقّى إلى alias دائم في `media_aliases`، فلا يتكرر نفس الغموض. هذا هو ما يجعل المكتبة **تتعلم بلا AI**.

`POST /api/index` يعيد `report` منظّمًا (مجلدات، ملفات، جديد/متغير/منقول/غير متغير، أخطاء مصنفة، throughput) و`rootStates` لكل قرص و`missing`. جلسة فحص واحدة تعمل في الوقت الواحد؛ الطلب المتزامن يعيد `409 Conflict`.

> ملاحظة: `GET /api/scan` القديم (قائمة JSON لكل الملفات) أُزيل لأنه لم يُستخدم من الواجهة ولا يمكنه التعامل مع مكتبات ضخمة. راجع [ADR-009](../decisions/ADR-009-incremental-indexing-pipeline.md).

---

## 🎞️ السلاسل والأشخاص المحليون (Offline Catalogue Graph)

> هذه المسارات لا تستدعي TMDB أثناء العرض. تقرأ من PostgreSQL والعلاقات المستخرجة سابقاً فقط.

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/franchises?limit=24` | `GET` | السلاسل المرتبطة بعمل محلي واحد أو أكثر، مرتبة محلياً |
| `/api/franchises/:slug` | `GET` | بيانات سلسلة واحدة بالعربية والإنجليزية عند توفرهما |
| `/api/franchises/:slug/media` | `GET` | الأعمال المتوفرة في هذه السلسلة فقط |
| `/api/people?limit=24` | `GET` | الأشخاص الذين لديهم عملان محليان على الأقل |
| `/api/people/:slug` | `GET` | بيانات شخص محفوظة محلياً |
| `/api/people/:slug/media` | `GET` | أعمال الشخص الموجودة في مكتبة NEXORA فقط |
| `/api/admin/catalog/sync-relations` | `POST` | إعادة استخراج السلاسل والأشخاص والأعمال المرتبطة من `metadata_snapshots` بدون شبكة؛ آمن للتكرار |
| `/api/admin/franchises/:id` | `PUT` | إبراز/إخفاء/ترتيب سلسلة محلية (`is_featured`, `is_hidden`, `sort_priority`) |
| `/api/admin/franchises/:id/refresh` | `POST` | تحديث كامل ومقصود لسلسلة واحدة من TMDB باللغتين `en-US` و`ar-SA`، ثم حفظ النصوص والأجزاء والصور محلياً |
| `/api/admin/franchises/refresh-missing?limit=24` | `POST` | ترقية دفعة محدودة من السلاسل القديمة التي لا تحمل لقطة إنجليزية صالحة؛ لا يُستدعى أثناء التصفح |
| `/api/admin/people/:id` | `PUT` | إبراز/إخفاء/ترتيب شخص محلي بالحقول نفسها |

---

## 🛡️ الجودة والترتيب والنقل (Quality & Migration)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/quality/report` | `GET` | تقرير الحلقات الناقصة، الملفات المكررة، والمعطوبة |
| `/api/quality/checksums` | `POST` | تشغيل حساب بصمات SHA-256 للملفات |
| `/api/migration/preview` | `POST` | توليد معاينة لخطة إعادة التنظيم الفيزيائي |
| `/api/migration/copy` | `POST` | تنفيذ النقل الآمن للملفات مع التحقق من البصمة |

---

## 🍿 البث والترجمات (Streaming & Subtitles)

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/stream?path=...` | `GET` | بث ملف بمسار يُقدّمه العميل، ويخضع لفحص `mediaPathAllowed` |
| `/api/stream/file/:fileId` | `GET` | بث مباشر لملف الفيديو مع دعم HTTP Range؛ يُستخرج المسار من الكتالوج (`serveCataloguePath`) ولا يقبل مسارًا من العميل |
| `/api/stream/file/:fileId/subtitles` | `GET` | قائمة الترجمات الخارجية المرافقة للملف |
| `/api/stream/file/:fileId/subtitles/:index` | `GET` | استخراج وتوفير ملف الترجمة WebVTT |
| `/api/stream/file/:fileId/preview?at=...` | `GET` | صورة معاينة زمنية (JPEG) للـ timeline |

---

## 🔌 Copy Bridge (خدمة محلية على جهاز العميل)

> Base URL: `http://127.0.0.1:32145`. الخدمة loopback-only افتراضيًا. الطلبات المتقاطعة من مناشئ غير مصرح بها تُرفض بـ `403` (loopback وعناوين LAN الخاصة مسموحة افتراضيًا؛ التحكم عبر `NEXORA_COPY_BRIDGE_CORS_ORIGIN`).

| المسار | الطريقة | الوصف |
|--------|---------|-------|
| `/api/health` | `GET` | فحص حالة خدمة الـ Bridge (يُستخدم لكشف الاتصال في الواجهة) |
| `/api/transfer/devices` | `GET` | قائمة أجهزة USB / Android / iOS الموصولة بجهاز العميل |
| `/api/transfer/device-apps?device_id=...` | `GET` | تطبيقات iOS التي تدعم File Sharing |
| `/api/transfer/device-app-folders?device_id=...&bundle_id=...` | `GET` | مجلدات تطبيق iOS |
| `/api/transfer/browse?device_id=...&path=...` | `GET` | تصفح مسار على الجهاز |
| `/api/transfer/mkdir` | `POST` | إنشاء مجلد على الجهاز |
| `/api/transfer/eject` | `POST` | إخراج/فصل الجهاز بأمان |
| `/api/transfer/copy` | `POST` | بدء مهمة نسخ (يدعم `source_url` و`source_urls`، و`file_id`/`file_ids`، والتقدم) |
| `/api/transfer/jobs` | `GET` | قائمة مهام النسخ |
| `/api/transfer/job/:id` | `GET` | تفاصيل مهمة واحدة |
| `/api/transfer/cancel/:id` | `POST` | إلغاء مهمة |
| `/api/transfer/events` | `GET` (SSE) | بث مباشر لتقدم المهام وأحداث الأجهزة |

**سلوك النسخ حسب نوع الجهاز:** USB Storage وiOS يبثّان الملف مباشرة (zero-spool)؛ Android MTP يخزّن الملف مؤقتًا في `NEXORA_COPY_BRIDGE_TEMP_DIR` ثم ينسخه (قيد Shell COM).
