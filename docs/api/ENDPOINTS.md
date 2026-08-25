# 📡 NEXORA API Endpoints Reference

> دليل نقاط الاتصال البرمجية الخاصة بخادم NEXORA (Go Backend API).

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
| `/api/media` | `POST` | إنشاء عمل جديد يدوياً |
| `/api/media/:id` | `PUT` | تعديل بيانات عمل |
| `/api/media/:id` | `DELETE` | حذف عمل وملفاته |
| `/api/media/:id/enrich` | `POST` | جلب وتحديث بيانات العمل التلقائية من TMDB/MAL |
| `/api/media/classify-folders` | `POST` | التصنيف التلقائي للبلد والنوع من أسماء المجلدات الفيزيائية |

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
| `/api/franchises?limit=24` | `GET` | السلاسل التي تضم جزأين محليين على الأقل، مرتبة محلياً |
| `/api/franchises/:slug` | `GET` | بيانات سلسلة واحدة بالعربية والإنجليزية عند توفرهما |
| `/api/franchises/:slug/media` | `GET` | الأعمال المتوفرة في هذه السلسلة فقط |
| `/api/people?limit=24` | `GET` | الأشخاص الذين لديهم عملان محليان على الأقل |
| `/api/people/:slug` | `GET` | بيانات شخص محفوظة محلياً |
| `/api/people/:slug/media` | `GET` | أعمال الشخص الموجودة في مكتبة NEXORA فقط |
| `/api/admin/catalog/sync-relations` | `POST` | إعادة استخراج السلاسل والاعتمادات من `metadata_snapshots` بدون شبكة؛ آمن للتكرار |
| `/api/admin/franchises/:id` | `PUT` | إبراز/إخفاء/ترتيب سلسلة محلية (`is_featured`, `is_hidden`, `sort_priority`) |
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
| `/api/stream/file/:fileId` | `GET` | بث مباشر لملف الفيديو مع دعم HTTP Range |
| `/api/stream/file/:fileId/subtitles/:index` | `GET` | استخراج وتوفير ملف الترجمة WebVTT |
