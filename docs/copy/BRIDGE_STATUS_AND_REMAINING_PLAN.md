# NEXORA Copy Bridge — تقرير الحالة وخطة إكمال ما تبقى

> المرجع: `plan.md` — خطة معمارية وتنفيذية لخدمة NEXORA Copy Bridge.
> التاريخ: 2026-09-15 · **آخر تحديث: 2026-09-17** (الإكتمال الفعلي للخطوات 1-6 + توثيق ADR-008 + إصلاح الخللين المكتشفين)

## 0. ملخص الحالة الحالية (بعد التنفيذ)

**الكود يبني الآن بنجاح** (`go build ./...` ✅، `go vet` ✅، `go test` ✅) بعد إغلاق ثلاث كُتب فرعية كانت تمنع البناء:

1. **`copybridge/service.go`** — أُزيل التعريفان المكرران لدالة `streamRemoteToDevice`، وتبقّت نسخة واحد فقط (الـ `keeper` — الدفق المباشر zero-spool) تُستدعى من `runBridgeJob` لمسارات URL، وأُزيلت `stageRemoteSource` نهائيًا (لا fallback تنزيل-ثمّ-نسخ). المصدر بلا مسار محلي أو URL يفشل بصوت عالٍ بدل الصمت.
2. **`transfer/backend.go`** — أُضيفت `PutStream(ctx, reader io.Reader, size, destination, opts) error` إلى واجهة `TransferBackend`.
3. **`transfer/go_ios_backend.go`** — نُفّذ `GoIOSBackend.PutStream` (دفق `io.Reader` → AFC مع Resume/Buffer)، و`StorageBackend.PutStream`/`AndroidBackend.PutStream` موجودتان أصلًا.
4. **`transfer/service.go`** — تصدير `Service.GetBackendFor` (تغليف `browserBackendFor` الموجود) ليفتتح الـ keeper أياً من الخلفيات الثلاث عبر نقطة إدخال واحدة.

### اكتمل في هذه الجولة ✅
- **الخطوة 4 — CORS:** أُضيف `corsAllow` في `copybridge/server.go`: loopback دائمًا مسموح، وعناوين LAN الخاصة (RFC 1918 / link-local / ULA) افتراضيًا، ورفض `403` للمناشئ العامة غير المصرّح بها، مع `NEXORA_COPY_BRIDGE_CORS_ORIGIN` (قائمة أو `*`). تم اختباره في `server_test.go`.
- **الخطوة 5 — إشعار الواجهة:** أُضيف `getBridgeHealth`/`BRIDGE_OFFLINE_MESSAGE`/`isBridgeOfflineError` في `api.js`، وحالة `bridgeOnline` + `checkBridge` في `TransferContext.jsx`، وشريط عربي واضح في `CopyToPhoneModal.jsx` و`TransferModal.jsx` مع زر إعادة المحاولة.
- **الخطوة 6 — التوثيق:** `README.md` (تثبيت + قيد Session 0 + Android MTP)، `docs/ARCHITECTURE.md` (حدود + مخطط + copy flow)، `docs/PROJECT_ANALYSIS.md` (مخاطر/قرارات/خريطة ملفات)، `docs/api/ENDPOINTS.md` (قسم Copy Bridge)، و**ADR-008** في `docs/decisions/`.
- **إصلاح الخللين المكتشفين أثناء المراجعة:**
  1. **مسار Android:** كان مسار الوجهة يُمرَّر كاملًا إلى `AndroidBackend.Put` الذي يعتبره مجلدات ويستخرج اسم الملف من مسار المصدر (ملف spool عشوائي `nexora-mtp-stream-*.bin`)، فينشئ مجلدًا باسم الفيلم وينسخ داخله ملفًا باسم عشوائي. الآن `androidDestinationParts` تفصل المجلد عن الاسم، ويُعاد تسمية ملف الـ spool باسم الوجهة، و`leafName` أصبح آمنًا على مسارات Windows. مغطى بـ `android_backend_test.go`.
  2. **مجلد iOS:** كان مسار الدفق يفتح الوجهة كاملة دون `Mkdir`، بينما AFC يرفض الكتابة في مجلد غير موجود. الآن `streamRemoteToDevice` يستدعي `backend.Mkdir(ctx, remoteDir)` قبل `PutStream` لكل الخلفيات (idempotent)، وبناء المسار موحّد في `remoteTargetPath`/`safeRelativeSub` (مغطى بـ `service_test.go`).

### لا يتبقى من الخطة الأصلية
- (مؤجل بقرار المالك) **token اختياري** `NEXORA_COPY_BRIDGE_TOKEN` للأوامر الحساسة — لم يُنفّذ، وكُتب كـ non-decision في ADR-008.
- (مرحلة لاحقة) **Tray Agent** بديلًا للخدمة.

### إصلاح إضافي خارج نطاق الخطة
- **403 عند البث بمعرّف قاعدة البيانات:** `/api/stream/file/{id}` كان يمر على `mediaPathAllowed` ويرفض الملفات المستخرجة من الكتالوج إن كانت خارج `NEXORA_MEDIA_ROOTS` (وهو ما كان يفشل النسخ عبر الـ Bridge). أُضيف `serveCataloguePath` في `handlers_stream.go` يتخطى الفحص للبث المستخرج من الكتالوج فقط، مع إبقاء `handleStream` (مسار من العميل) و`handleStreamImage` على الفحص الصارم.

---

## 1. الملخص التنفيذي

تم تنفيذ **معظم الهيكل** المطلوب في `plan.md` عبر أربع مراحل: بناء خادم محلي (Bridge) بـ Windows Service، تنظيف السيرفر المركزي، وربط واجهة المتصفح. لكن **المرحلة 2 (محول النقل المباشر Zero-Spool) غير مكتملة والكود لا يُبنى** (`go build ./...` سيفشل) بسبب:

1. دالة `streamRemoteToDevice` معرّفة **مرتين** بنفس التوقيع.
2. استدعاء دوال غير موجودة: `transfer.Service.GetBackendFor` و `transfer.Service.GetBackendForDevice`.
3. استدعاء `PutStream` على واجهة `TransferBackend` التي **لا تحتويها** — وليست مضافة لها، ولا منفّذة في `GoIOSBackend`.

بمعنى آخر: البنية مركّبة لكن محرك الدفق المباشر حاول تنفيذ "Phase 2 Upgrade" أولًا ولم يُنهَ، ما ترك نسختين متضاربتين من نفس الدالة ضمن ملف واحد.

---

## 2. ما تم إنجازه وتقييم مطابقته للخطة

### المرحلة 1 — بناء ثنائي خدمة الويندوز النظيف : **مكتمل وصحيح** ✅

| البند من `plan.md` | الحالة | الملف/الدوال |
|---|---|---|
| فحص وضع التشغيل (SCM / Console تفاعلي) | ✅ | `server/cmd/copybridge/main.go` — `isWindowsService()`, `-debug` |
| أوامر `-install/-uninstall/-start/-stop/-service` | ✅ | `main.go:19-24` + `service_windows.go` (via `golang.org/x/sys/windows/svc/mgr`) |
| `svc.Handler` (واجهة دورة حياة الخدمة) | ✅ | `service_windows.go:27-69` — `Execute`, Stop/Shutdown Interrogate |
| إدارة دورة الحياة عبر SCM | ✅ | `installService/uninstallService/startService/stopService` |
| بوابات الأنظمة غير الويندوزية | ✅ | `service_other.go` |
| سكريبت التثبيت `sc.exe` + netsh firewall | ✅ | `scripts/install-bridge-service.bat` (تحقق Admin، failure reset، Port 32145) |
| سكريبت الإزالة | ✅ | `scripts/uninstall-bridge-service.bat` |

### المرحلة 2 — خادم الـ Bridge المحلي : **ناقص وغير قابل للبناء** ❌

| البند | الحالة | الموضع |
|---|---|---|
| `127.0.0.1:32145` افتراضيًا | ✅ | `copybridge/config.go:27` |
| خادم HTTP بكل المسارات (devices/apps/folders/browse/mkdir/eject/copy/jobs/job/cancel/events/health) | ✅ | `copybridge/server.go:29-43` |
| SSE (بذر لقطات + keep-alive 20s) | ✅ | `copybridge/server.go:176-221` |
| CORS | ⚠️ | `server.go:223-240` — يفترض `*`؛ يُوصى بتضييقه (قسم 4) |
| إدارة المهام والحياة والتقدم | ✅ (جزئيًا) | `copybridge/service.go` — `StartCopy/runBridgeJob/followChildJob` |
| **Zero-Spool Direct Stream (المحول المباشر)** | ❌ **لا يُبنى** | `copybridge/service.go` — انظر القسم 3 |
| التنظيف التلقائي للـ temp | ✅ | `cleanupTempRoot` + `defer os.RemoveAll` |

### المرحلة 3 — تنظيف السيرفر المركزي : **مكتمل** ✅

- `server/internal/app/app.go:76-84`: `transferService` أصبح اختياريًا عبر `NEXORA_SERVER_USB_TRANSFER=true` — السيرفر لم يعد يفحص أقراص/هواتف السيرفر للنوافذ.
- `server/internal/api/handlers_transfer.go`: كل handlers حصلت على حماية `if s.transfer == nil` وترجع قوائم فارغة/خطأ — أي نقاط `/api/transfer/*` في السيرفر المركزي أصبحت "اختيارية/معطلة افتراضيًا" دون تسريب للأجهزة المحلية.
- `/api/stream/file/{id}` (`handlers_stream.go:70`) **ليست محمية بأدمن** وتستخدم `http.ServeContent` (تدعم Range) → مصدر مثالي يجلب منه الـ Bridge الملفات عبر LAN.

### المرحلة 4 — ربط واجهة المتصفح : **مكتمل** (مع تحسين بسيط متبقٍ) ✅

| البند | الحالة | الموضع |
|---|---|---|
| `BRIDGE_BASE_URL = http://127.0.0.1:32145` + قابلية التخصيص `VITE_COPY_BRIDGE_URL` | ✅ | `client/src/lib/api.js:440-449` |
| توجيه دوال النقل كلها للـ Bridge مع رسالة عربية ودية عند عدم التشغيل | ✅ | `api.js:452-525` (`requestBridgeJSON`) |
| SSE من الـ Bridge بدل السيرفر | ✅ | `client/src/context/TransferContext.jsx:62` |
| إرسال `source_url/source_urls` (نقطة البث بالسيرفر المركزي) عند النسخ | ✅ | `TransferContext.jsx:191-197`, `CopyToPhoneModal.jsx:204-208` |
| إزالة إدخال Bundle ID اليدوي واعتماد قائمة التطبيقات | ✅ | `CopyToPhoneModal.jsx` |
| إشعار "الخدمة غير مشغّلة" داخل النافذة | ⚠️ | يوجد خطأ ودّي في `requestBridgeJSON` لكن لا يوجد Banner صريح؛ يُستكمل في قسم 4 |

**ملحوظة:** `TransferModal.jsx` و `AdminTransferPage.jsx` تستخدمان `api.js` الحالية، وبالتالي تتصلان تلقائيًا بالـ Bridge دون تعديل.

---

## 3. العوائق الحرجة — الكود لا يُبنى (المطلوب حله أولًا)

كلها في `server/internal/copybridge/service.go`:

### 3.1 تكرار دالة `streamRemoteToDevice` (خطأ تصريف فوري)
- **سطر 387** و **سطر 493**: تعريفان لدالة بنفس التوقيع:
  `func (s *Service) streamRemoteToDevice(ctx, jobID, source, req, completedBytes, spanStart, spanEnd) error`
- النتيجة: `streamRemoteToDevice redeclared in this block` — يفشل البناء فورًا.
- السبب الجذري: عمل من جولتين — الجولة الأولى (سطر 387) تبني المسار البعيد يدويًا لكل نوع جهاز وتستدعي `GetBackendFor`، والثانية (سطر 493) تستدعي `GetBackendForDevice`. اللازمة: **دمجهما في دالة واحدة** تأخذ `TransferDestination` من خادمة واحدة، وتستخدم المسار البعيد من الـ destination.

### 3.2 دوال Backend غير موجودة في `transfer.Service`
- `service.go:444` → `s.transfer.GetBackendFor(ctx, ...)`
- `service.go:515` → `s.transfer.GetBackendForDevice(deviceID, ...)`
- لا يوجد أي اسم مشابه في `server/internal/transfer` (البحث الشامل أكّد صفر وجود). الأقرب موجود و**غير مُصدَّر**: `browserBackendFor` في `service.go:894`.

### 3.3 `PutStream` غير موجودة في واجهة `TransferBackend`
- `backend.go:54-79` — الواجهة تعرّف `Put` فقط.
- `PutStream` أُضيفت كطريقة **خارجية** على `*StorageBackend` (`storage_backend.go:142`) و`*AndroidBackend` (`android_backend.go:223`) لكنها ليست في الواجهة، و`GoIOSBackend` **لا تملكها أصلًا**.
- النتيجة: استدعاء `backend.PutStream(...)` على قيمة من نوع `transfer.TransferBackend` خطأ تصريف: `backend.PutStream undefined`.

### 3.4 أثر جانبي عند إصلاح 3.3
- بمجرد إضافة `PutStream` إلى الواجهة سيضطر `GoIOSBackend` لتنفيذها وإلا يفشل البناء (مبدأ الواجهات). يجب كتابة تنفيذ دفق `io.Reader → AFC` له (شبيه `GoIOSBackend.Put` لكن القراءة من `io.Reader` بدل `os.Open`).

### 3.5 تحذير اقترانه بـ Android MTP
- `AndroidBackend.PutStream` (**voluntary**) ينزّل إلى ملف temp ثم `CopyHere(16)` — لأن واجهة Shell COM تتطلب ملفًا محليًا. أي: صفر تخزين مؤقت يسري فعليًا على **Storage** فقط، وطول الملف الكامل يُخزَّن مؤقتًا في `os.TempDir` للأندرويد. يجب اعتماد هذا كسلوك موثق ومقبول (قيد Shell COM)، مع تنظيفه تلقائيًا — أو جعله داخل `cfg.TempDir` الموحّد.

---

## 4. خطة تنفيذ ما تبقى (بالتفصيل)

### الخطوة 1 — إصلاح العائق الحرج في `copybridge/service.go` (أولوية قصوى)
1. **حذف** التعريف الثاني `streamRemoteToDevice` (سطر 493–574) والاحتفاظ بنسخة واحدة نظيفة.
2. **إعادة الكتابة** لدالة واحدة تأخذ القرار من مكان واحد:
   - فتح الاتصال: `http.NewRequestWithContext(ctx, GET, source.URL)` مع التحقق من `ContentLength` وتأخذ بعين الاعتبار `resp.StatusCode` (200 أو 206).
   - تحديد المسار البعيد من `TransferDestination.RemotePath` (من الخطوة 2) وليس الحساب اليدوي.
   - استدعاء `backend.PutStream(ctx, resp.Body, size, remotePath, putOpts)` مع `OnProgress` يحدّث `job.Progress/Transferred/SpeedBps/ETASeconds`.
3. **إزالة المسار القديم** `stageRemoteSource` (التنزيل كاملًا ثم النسخ) من مسار URL — أو إبقاؤه fallback محصور بـ `source_path` محلي فقط، مع التعليق على السبب.

### الخطوة 2 — إكمال دعم الخلفيات (واجهة + تنفيذ iOS)
1. **إضافة** `PutStream(ctx, reader io.Reader, size int64, destination string, opts PutOptions) error` إلى واجهة `TransferBackend` في `backend.go:54-79`.
2. **تنفيذ** `GoIOSBackend.PutStream` في `go_ios_backend.go`:
   - مشابه `Put` (سطر 145) لكن القراءة من `reader` بدل `os.Open(source)`, مع `ResumeOffset` اختياري، و`bufferSize` محدود بـ 512KB (قيود AFC الحالية).
3. التحقق أن كل الآلي التالية تنفّذ الواجهة (ستظهر بالبناء): `StorageBackend` ✅، `AndroidBackend` ✅، `GoIOSBackend` (الجديد).

### الخطوة 3 — فتح مخرج خلفية موحّد في `transfer.Service`
1. **تصدير** خادمة واحدة (تغليف `browserBackendFor`) بمعنى صريح، مثل:
   `func (s *Service) OpenBackend(ctx context.Context, deviceID, appID string, deviceType DeviceType) (TransferBackend, TransferDestination, error)`
2. تعديل `copybridge/service.go` ليستدعي هذه الخادمة الواحدة فقط (تحل محل `GetBackendFor` و`GetBackendForDevice`)، حيث تملأ `TransferDestination` وتمنح `RemotePath` الجاهز (مثال: أقراص → `E:\Movies\...`، أندرويد → `Download/...`، iOS → `/Documents/...`).

### الخطوة 4 — تهيئة CORS وأمان الـ Bridge (تحسين مطلوب من الخطة §6.2.3)
1. **تضييق** `CORSOrigin` الافتراضي بدل `*`: السماح بـ `http://localhost:*` وعناوين الـ LAN المخصّصة (الإعداد عبر `NEXORA_COPY_BRIDGE_CORS_ORIGIN`). لأن الربط على `127.0.0.1` لا يمنع أي موقع محلي من إصدار أوامر نسخ/إخراج.
2. (اختياري، توصية) إضافة **token مشترك** اختياري `NEXORA_COPY_BRIDGE_TOKEN` يُطلب في هيدر `Authorization` لأوامر `POST /copy` و `POST /eject` و `POST /mkdir`، مع تمريره من `api.js`.

### الخطوة 5 — واجهات: إشعار واضح عند تعطّل الـ Bridge
- في `CopyToPhoneModal.jsx` و`TransferModal.jsx`: عند فشل `getTransferDevices()` بسبب `requestBridgeJSON` عرض شريط توضيحي عربي مباشر:
  *"يرجى التأكد من تشغيل خدمة NEXORA Copy Bridge على هذا الجهاز (تثبيت: `scripts/install-bridge-service.bat`)"* — بدل الاعتماد على رسالة الخطأ الخام فقط.

### الخطوة 6 — توثيق القيد التشغيلي (Session 0) من `plan.md` §7
- إضافة ملاحظة في `README`/دليل التثبيت: الخدمة الافتراضية تعمل بـ `LocalSystem` (Session 0) — ممتازة للملفات Storage وiOS عبر usbmuxd، لكن **Android MTP (Shell COM)** قد يتطلب سياق مستخدم تفاعلي في بعض إصدارات ويندوز؛ يتم التبديل عبر `sc.exe config NEXORACopyBridge obj= ".\Administrator" password= "..."` أو تشغيل `-debug` يدويًا للتجربة.

---

## 5. خطة التحقق (بعد التنفيذ)

### بناء واختبار آلي
```pwsh
# 1. بناء الكل
cd server
go build ./...

# 2. بناء ثنائي الـ Bridge
go build -o nexora-bridge.exe ./cmd/copybridge

# 3. اختبارات حزمة النقل
go test -v ./internal/transfer/...

# 4. اختبارات الحزم الجديدة
go vet ./internal/copybridge/... ./cmd/copybridge/...
```
(ملاحظة: الغرض من بيئة التطوير هذه لا يتضمن `go` في المسار؛ تأكد من تثبيت Go ≥ 1.22)

### تحقق يدوي (من `plan.md` §8)
1. السيرفر المركزي على جهاز، ومتصفح من جهاز آخر — قائمة الأجهزة فارغة ما دام الـ Bridge غير موصول بأجهزة على العميل.
2. توصيل فلاشة/هاتف على **جهاز العميل** → يظهر فورًا محليًا عبر SSE، ولا يظهر لأي جهاز آخر.
3. نسخ فيلم ~2GB ومراقبة السرعة والـ ETA حتى الاكتمال.
4. تجربة iOS (تطبيق يدعم File Sharing) و Android (سيلاحظ spool مؤقت في temp ثم نسخ) و Storage (صفر spool).

---

## 6. التوثيق المطلوب تحديثه بعد الإتمام

- `docs/PROJECT_ANALYSIS.md` — نقل مخاطر USB من مصدر السيرفر إلى الـ Bridge المحلي.
- `docs/ARCHITECTURE.md` — مخطط الـ Bridge (Localhost Service + Stream Pipe).
- `docs/api/ENDPOINTS.md` + `docs/copy/EMAD_SYSTEM_COPY_VS_MAIN_ANALYSIS.md` — توثيق `/api/transfer/*` في الـ Bridge لا السيرفر.
- `docs/decisions/` — ADR جديد (أو ملحق ADR-002) لقرار "النسخ عبر خدمة محلية على جهاز العميل بدل سيرفر مركزي".
- `README.md` — خطوات التثبيت `install-bridge-service.bat` وقيد Session 0.
- `plan.md` — تحديث علامة الحالة بعد الإكمال (لا يُرفع القسم 5 من الخطة إلى واقع دون تنفيذ قابل للتحقق).

---

## 7. أسئلة مفتوحة للمالك

1. هل نبقى على **fallback التنزيل المؤقت** (`stageRemoteSource`) كخيار للشبكات الضعيفة أم نزيله نهائيًا (الصارم صفر spool)؟
2. هل الـ Bridge يحتاج **token أمان** (الخطوة 4.2) في هذه النسخة أم يؤجل؟
3. هل نضيف **Tray Agent** (خطة §7 كخيار بديل للخدمة) الآن أم يبقى كمرحلة لاحقة؟

---

*نهاية التقرير. التحليل مبني على فحص الكود الفعلي (primary sources) في الحالة الحالية للفرع `main`.*