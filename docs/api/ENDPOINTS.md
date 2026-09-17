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
| `/api/search/sync` | `POST` | مزامنة قاعدة البيانات بالكامل مع محرك البحث الفوري |
| `/api/indexer/scan` | `POST` | بدء فحص وفهرسة مسار مجلد محدد وإدخال البيانات |
| `/api/indexer/browse` | `POST` | استعراض مجلدات نظام الملفات لاختيار المسارات |

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
