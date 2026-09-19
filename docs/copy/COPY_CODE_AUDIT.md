# تقرير تدقيق كود قسم النسخ (Copy / Transfer)

> التدقيق الأصلي تحليلي فقط. بعد اعتماد المالك، نُفّذ تنظيف الأكواد الميتة والجيل الأول، والنتيجة موثّقة في [القسم 9](#9-سجل-التنظيف-المنفّذ). أرقام الأسطر أدناه تشير إلى الحالة قبل التنظيف.
> التاريخ: 2026-09-18.

---

## 0. الهدف والمنهجية

الهدف: رصد ثلاث فئات داخل قسم النسخ بالكامل (خادم + عميل):

1. **الأكواد المتكررة** (Duplication).
2. **الأكواد غير المستخدمة** (Dead code).
3. **العمليات غير المنطقية** (Illogical / buggy logic).

المنهجية: قراءة كاملة للملفات + تتبّع callers/callees داخل المستودع + التحقق من غياب المراجع لكل رمز ميت.

---

## 1. نطاق الفحص

### 1.1 الخادم

| الملف | الأسطر | الدور |
|---|---|---|
| `server/internal/copybridge/service.go` | 897 | خدمة النسخ المحلية (jobs, sources, URL streaming) |
| `server/internal/copybridge/server.go` | 363 | HTTP routes + CORS middleware |
| `server/internal/copybridge/config.go` | 82 | الإعدادات + قراءة `.env` |
| `server/cmd/copybridge/main.go` | 114 | نقطة تشغيل الجسر (console / Windows service) |
| `server/internal/transfer/service.go` | 1597 | خدمة v1 + المحرك v2 + legacy helpers |
| `server/internal/transfer/engine.go` | 159 | `TransferEngine` (v2) |
| `server/internal/transfer/worker.go` | 319 | `DeviceWorker` (v2) |
| `server/internal/transfer/scheduler.go` | 64 | توجيه المهام لكل جهاز (v2) |
| `server/internal/transfer/models.go` | 108 | `TransferJob` (v1) |
| `server/internal/transfer/models_v2.go` | 76 | `TransferJobV2` / `TransferFile` (v2) |
| `server/internal/transfer/errors.go` | 152 | أخطاء مُهيكلة موحّدة |
| `server/internal/transfer/events.go` | 66 | `eventBroker` (SSE) |
| `server/internal/transfer/verify.go` | 84 | التحقق بعد النسخ |
| `server/internal/transfer/paths.go` | 56 | تنظيف المسارات |
| `server/internal/transfer/progress.go` | 77 | timeout / formatBytes / parse size |
| `server/internal/transfer/backend_helpers.go` | 59 | مساعدات المسارات |
| `server/internal/transfer/android_backend.go` | 308 | MTP عبر PowerShell |
| `server/internal/transfer/go_ios_backend.go` | 306 | AFC/HouseArrest |
| `server/internal/transfer/storage_backend.go` | 201 | تخزين محلي/USB |
| `server/internal/transfer/removable_windows.go` | 110 | اكتشاف أقراص USB |
| `server/internal/api/handlers_transfer.go` | 342 | النسخة المدمجة من هاندلرز النقل |

### 1.2 العميل

| الملف | الأسطر | الدور |
|---|---|---|
| `client/src/context/TransferContext.jsx` | 314 | حالة النسخ + SSE |
| `client/src/components/transfer/TransferModal.jsx` | 1092 | نافذة النسخ الحية |
| `client/src/components/transfer/MiniTransferCenter.jsx` | 279 | مركز التقدّم المصغّر |
| `client/src/components/transfer/FloatingCopyButton.jsx` | 57 | زر عائم (غير مستورد) |
| `client/src/components/CopyToPhoneModal.jsx` | 672 | نافذة الجيل الأول (غير مستوردة) |
| `client/src/pages/admin/AdminTransferPage.jsx` | 284 | شاشة الإدارة + polling |
| `client/src/lib/api.js` | 601 | مغلّفات الاتصال بالجسر |
| `client/src/lib/transferDevices.js` | 38 | تطبيع/ترتيب الأجهزة |

---

## 2. الخريطة المعمارية الحالية

### 2.1 جيلان متوازيان داخل `transfer`

- **v2 (المسار المفضّل):** `TransferEngine` (`engine.go`) ← `Scheduler` (`scheduler.go`) ← `DeviceWorker` (`worker.go`) ← `TransferBackend` (`backend.go`) مع `Verifier` (`verify.go`) وأخطاء `TransferError`.
- **v1 (المسار القديم):** `Service` + `copyToMTPDevice` / `copyToIOSDeviceGoIOS` / `copyToLocalPath`، يُستدعى فقط كـ fallback من `runTransferV2` عند فشل `buildV2Job` أو رفض `Submit`.

المشكلة: المحرك v2 مُفعّل، لكن models v1 هي التي تُعرض عبر REST (`TransferJob`)، ويتم نسخ تقدّم v2 إلى v1 عبر `mirrorV2Progress` (`server/internal/transfer/service.go:807`). النتيجة: طبقتان + جسر مزامنة دائم.

### 2.2 مساران للنسخ في الخادم

- **copybridge** (`127.0.0.1:32145`): هو المستخدم من الواجهة فعليًا. يعيد تعريف routes + handlers + broker + updateJob خاصته.
- **api المدمج** (`handlers_transfer.go` + `api/server.go:303-315`): نسخة موازية لنفس المسارات، تُفعّل فقط عند `NEXORA_SERVER_USB_TRANSFER=true` (`server/internal/app/app.go:79`).

### 2.3 مساران داخل copybridge

- مصادر **URL**: تُبثّ مباشرة عبر `streamRemoteToDevice` (`copybridge/service.go:373`).
- مصادر **محلية**: تمرّ عبر `s.transfer.StartCopy` ثم تُتابع بـ polling كل 300ms عبر `followChildJob` (`copybridge/service.go:459`).

---

## 3. الأكواد المتكررة (Duplication)

### 3.1 الخادم

| # | البند | الموقع | النسخة المكررة | ملاحظات |
|---|---|---|---|---|
| D1 | `eventBroker` (subscribe/publish) | `server/internal/transfer/events.go:31-66` | `server/internal/copybridge/service.go:864-897` | نسخ حرفي تقريبًا لنفس البنية والمنطق |
| D2 | Routes + Handlers النقل | `server/internal/copybridge/server.go:32-224` | `server/internal/api/handlers_transfer.go:64-342` + `api/server.go:303-315` | 11 مسارًا مكررًا بنفس الأسماء |
| D3 | سكربتات MTP PowerShell | `transfer/service.go:1116-1193` و`1247-1281` | `transfer/android_backend.go:65-103` و`129-176` | منطق `Shell.Application` / `NewFolder` / `CopyHere` مكرر |
| D4 | اشتقاق نوع الجهاز | `copybridge/service.go:625` (`inferDeviceType`) | `transfer/service.go:961` (`deviceTypeOf`) و`transfer/service.go:1487` (`isIOSUDID`) | ثلاث قواعد مختلفة لنفس القرار |
| D5 | `updateJob` + نشر الحدث | `transfer/service.go:1447-1459` | `copybridge/service.go:572-583` | نسخ منطقي (قفل + نسخة + publish) |
| D6 | مساعدات المسارات | `transfer/paths.go` | `transfer/backend_helpers.go` و`verify.go:68` و`copybridge/service.go:682,762,771` | تداخل كبير: `safeRelativePath` / `splitRelativePath` / `splitRemoteDir` / `joinRemotePath` / `safeRelativeSub` / `safeStageName` / `fileNameFromAnyPath` |
| D7 | دوال متطابقة فعليًا | `copybridge/service.go:807` (`indexedSize`) | `copybridge/service.go:814` (`indexedInt64`) | نفس الجسم تمامًا |
| D8 | دوال متطابقة فعليًا | `copybridge/service.go:792` (`sumSourceSizes`) | `copybridge/service.go:718` (`recomputeTotalBytes`) | نفس المنطق (`sum(max(size,0))`) |
| D9 | `firstNonEmpty` | `transfer/service.go:1534` | `metadata/tmdb.go:849` و`db/repository_metadata.go:546` | ثلاث نسخ متطابقة |
| D10 | `safePathComponent` | `transfer/paths.go:12` | `migration/service.go:473` | نسخة شبه مطابقة |
| D11 | اكتشاف أجهزة iOS الوهمية | `transfer/service.go:219-237` (`filterWindowsIOSPlaceholders`) | `client/src/lib/transferDevices.js:1-21` | منطق مصنّف مرتين على طرفين |

### 3.2 العميل

| # | البند | المواقع | ملاحظات |
|---|---|---|---|
| D12 | `formatBytes` | `CopyToPhoneModal.jsx:17`, `transfer/TransferModal.jsx:14`, `transfer/MiniTransferCenter.jsx:4`, `transfer/FloatingCopyButton.jsx:4`, `pages/admin/AdminTransferPage.jsx:7` | 5 نسخ محلية، مع وجود نسخة مشتركة أصلًا في `pages/admin/adminConstants.js:86` |
| D13 | خريطة تسمية الطور | `CopyToPhoneModal.jsx:30` (`getTransferPhaseLabel`) | `MiniTransferCenter.jsx:37` (`getPhaseBadge`) — تغطية مختلفة (الثاني يدعم `retrying`/`waiting_device`) |
| D14 | تصنيف الجهاز | `transferDevices.js:1,10` | `TransferModal.jsx:130-138`, `CopyToPhoneModal.jsx:195,264`, `AdminTransferPage.jsx:127-129` — قواعد غير متسقة |
| D15 | اختيار الجهاز الافتراضي | `TransferModal.jsx:95-102` (SSE) | `TransferModal.jsx:112-119` (HTTP) — منطق مطابق |
| D16 | جلب/تطبيع الأجهزة | `TransferModal.jsx:91-127` | `CopyToPhoneModal.jsx:76-101`, `AdminTransferPage.jsx:28-36` |
| D17 | مسندات "قيد التشغيل" | `TransferContext.jsx:264-271` | `MiniTransferCenter.jsx:75-84`, `MiniTransferCenter.jsx:91-98`, `MiniTransferCenter.jsx:199-203`, `AdminTransferPage.jsx:200` — خمس قوائم بمحتويات مختلفة |
| D18 | `checkBridge` | `TransferContext.jsx:26-34` | `CopyToPhoneModal.jsx:70-74` |

---

## 4. الأكواد غير المستخدمة (Dead code)

### 4.1 الخادم

| الرمز | الموقع | حالة الاستخدام |
|---|---|---|
| `StatDevicePath` | `transfer/service.go:927` | مذكور في الواجهة `api/server.go:143` لكن لا handler يستدعيه؛ المرجع الوحيد اختبار `browser_test.go:48` |
| `v2Outcome` | `transfer/service.go:876` | المرجع الوحيد اختبار `service_bridge_test.go:90-100` |
| `extractJSONValue` | `transfer/service.go:1461` | صفر مراجع |
| `min` | `transfer/service.go:1480` | صفر مراجع (ويتعارض مع `min` المدمجة في Go) |
| `copyToIOSDevice` | `transfer/service.go:1301` | صفر مراجع (يغلف `copyToIOSDeviceGoIOS`) |
| `resolveTool` | `transfer/backend_helpers.go:27` | المرجع الوحيد `backend_test.go:60` |
| `Scheduler.Stop` | `transfer/scheduler.go:54` | صفر مراجع؛ `Service.Close` (`service.go:67`) لا يوقف المحرك/الجدولة |
| `DeviceWorker.Stop` | `transfer/worker.go:57` | يُستدعى فقط من `Scheduler.Stop` الميت |
| `PhaseQueued` | `models_v2.go:9` | صفر مراجع (يُضبط الطور كنص حرفي) |
| `PhaseVerifying` | `models_v2.go:15` | صفر مراجع |
| `PhaseWaitingDevice` | `models_v2.go:16` | صفر مراجع |
| `PhasePaused` | `models_v2.go:20` | صفر مراجع |
| `StatusPaused` | `models.go:71` | صفر مراجع |
| `StatusWaiting` | `models.go:72` | صفر مراجع |
| ثوابت أخطاء | `errors.go:38-49` | `CodeDeviceDisconnected`, `CodeAFCTimeout`, `CodeMuxError`, `CodeDiskFull`, `CodeDestinationNotFound`, `CodePermissionDenied`, `CodeUnsupportedOperation`, `CodeInvalidDestination` بلا مراجع خارج `errors.go` |
| دوال أخطاء | `errors.go:72-110` | `ErrDeviceLost`, `ErrAFCTimeout`, `ErrAFCIO`, `ErrDiskFull`, `ErrDestinationNotFound`, `ErrPermissionDenied`, `ErrUnsupported` بلا مراجع; `ErrCancelled` مُستبدل ببناء حرفي في `worker.go:279` |
| `IsDeviceLost` | `errors.go:136` | صفر مراجع |
| مسار legacy | `service.go:975` (`executeTransferLegacy`) و`service.go:993` (`copyOneLegacy`) | يُستدعى فقط كـ fallback نادر من `runTransferV2` (`service.go:642-660`) |

> ملاحظة إيجابية: `queryMTPFileSize` (`service.go:1243`) ليس مكررًا؛ فهو مستخدم فعليًا من `android_backend.go:202,286`.

### 4.2 العميل

| الرمز | الموقع | حالة الاستخدام |
|---|---|---|
| `CopyToPhoneModal` | `components/CopyToPhoneModal.jsx:49` | صفر importers |
| `FloatingCopyButton` | `components/transfer/FloatingCopyButton.jsx:17` | صفر importers |
| `getTransferAppFolders` | `lib/api.js:499` | المستدعي الوحيد `CopyToPhoneModal.jsx:157` (ميت) |
| `getTransferJob` | `lib/api.js:514` | المستدعي الوحيد `CopyToPhoneModal.jsx:174` (ميت) |
| `isRealIOSDevice` / `isWindowsIOSPlaceholder` | `lib/transferDevices.js:1,10` | مُصدّرة لكن لا أحد يستوردها خارجيًا (تُستخدم داخليًا فقط) |
| `toggleFileSelection` | `context/TransferContext.jsx:144` (يُعرض :279) | صفر مستهلكين |
| `hasRunningJobs` | `context/TransferContext.jsx:264` (يُعرض :290) | صفر مستهلكين |
| `refreshJobs` | `context/TransferContext.jsx:300` | صفر مستهلكين (`fetchJobs` يُستخدم داخليًا) |
| `clearSelection` (كـ API عام) | `context/TransferContext.jsx:173` | المستهلك الخارجي الوحيد `FloatingCopyButton.jsx:49` (ميت); مستخدم داخليًا في `:239` |

---

## 5. العمليات غير المنطقية (Illogical / buggy logic)

### 5.1 الخادم

| # | المشكلة | الدليل | الأثر |
|---|---|---|---|
| S1 | `streamRemoteToDevice` يعيد اشتقاق نوع الجهاز بـ `inferDeviceType(req.DeviceID)` متجاهلًا النوع المحلول في `StartCopy` | `copybridge/service.go:399` مقابل `copybridge/service.go:137-148` | UDID خام (40 خانة سِت عشرية) لا يبدأ بـ `ios_` يُصنّف `android` فيُختار `AndroidBackend` خطأً، رغم أن `transfer` يملك `isIOSUDID` |
| S2 | نظاما أحداث منفصلان + polling للـ child | `copybridge/service.go:89-97` (يرحّل `EventDevices` فقط) و`:459` (polling كل 300ms) | مهمتان متزامنتان على نفس البيانات; الجسر لا يستفيد من أحداث `transfer` للمهام المحلية |
| S3 | حساب تقدّم مصادر URL يضيف `source.Size` لا البايتات الفعلية | `copybridge/service.go:309` | إن كانت `Size=0` (شائع لـ URL) يبقى التقدّم 0 وقد ينتهي `Transferred = max(totalBytes, completedBytes) = 0` (`:362`) |
| S4 | تعليق "Zero-Spool Direct Stream" بينما Android يكتب الملف كاملًا في temp | `copybridge/service.go:294-295` مقابل `android_backend.go:230-259` | مضاعفة استخدام القرص وزمن إضافي للملفات الكبيرة |
| S5 | `transfer.Service.CancelJob` يضبط `Status` فقط دون `Phase`/`Error`/`CompletedAt` | `transfer/service.go:1429-1445` مقابل `copybridge/service.go:549-566` | payload غير متسق للواجهة (فرع mini ينظر إلى `phase`) |
| S6 | احتمال فقدان حدث الإكمال في `runTransferV2` | `transfer/service.go:657-684` | يتم الاشتراك بعد تنفيذ `Submit`; مهمة سريعة قد تنتهي بين `mirrorV2Progress` والاشتراك → انتظار بلا نهاية (لا يوجد deadline على `ctx`) |
| S7 | `queryMTPFileSize` يعيد `exists=true` عند فشل تحليل الحجم | `transfer/service.go:1293-1294` و`android_backend.go:202-210` | يُعامَل الملف كموجود بحجم 0 → يستمر الـ polling حتى المهلة بدل الفشل المبكر |
| S8 | `corsAllow` يفحص `"*"` مرّتين | `copybridge/server.go:272-276` ثم `containsConfiguredOrigin` (`:289-300`) | تكرار غير ضار لكنه زائد |
| S9 | `isLANOrigin` يعتبر `0.0.0.0` (IsUnspecified) أصلًا مسموحًا | `copybridge/server.go:349` | قيمة Origin غير منطقية تُقبل |
| S10 | `buildV2Job` يجعل `total=1` عند غياب كل الأحجام | `transfer/service.go:790-792` | رياضيات تقدّم مضلّلة |
| S11 | طفح `documentsPath`/`joinRemotePath` بتنسيقين للمسار | `go_ios_backend.go:208-216`, `verify.go:68` | خطر ازدواج الفواصل عند دمج مسارات iOS |
| S12 | `Service.Close` لا يوقف `Scheduler`/`DeviceWorker` | `transfer/service.go:67-75` | تسريب goroutines عند إيقاف الجسر (يظهر أثره خصوصًا في وضع الخدمة) |
| S13 | الطور `"streaming"` غير معروف في خريطة أطوار العميل | `copybridge/service.go:299` مقابل `MiniTransferCenter.jsx:45-60` | يظهر "جاري النقل" العام بدل وصف دقيق |

### 5.2 العميل

| # | المشكلة | الدليل | الأثر |
|---|---|---|---|
| C1 | وسم المهام `queued/retrying/waiting_device` كـ "فشل" | `AdminTransferPage.jsx:200` (`isRunning` يغطي processing/pending فقط) و`:223` | عرض مضلّل + لا زر إلغاء لهذه الحالات |
| C2 | فرع `cancelled` غير قابل للوصول | `MiniTransferCenter.jsx:41` مقابل الفلتر `:75-84` الذي يستبعد `cancelled` | لا تظهر المهام الملغاة أبدًا؛ وإذا كانت وحدها يختفي المركز (`:86-88`) |
| C3 | ثلاثة آليات جلب لنفس البيانات | SSE في `TransferContext.jsx:77` + polling كل 3s في `AdminTransferPage.jsx:38-42` (مع `getMediaList` كل 3s) | مصدرا حقيقة متعارضان + حِمل شبكي زائد |
| C4 | جلب مكرر بعد بدء النسخ | `TransferContext.jsx:236` (`await fetchJobs()`) رغم وصول حدث `job` عبر SSE | طلب شبكة زائد لكل عملية |
| C5 | `customBundleId` غير معرّف أصلًا في المكوّن | يُستعمل في `CopyToPhoneModal.jsx:265,665` ولا يوجد `useState` له في الملف | `ReferenceError` عند أي استخدام فعلي (ميت حاليًا) |
| C6 | مصدران للأجهزة داخل النافذة (SSE + HTTP) | `TransferModal.jsx:91-103` و`:106-127` | سباق: رد HTTP قديم قد يطمس تحديث SSE; و`eject` يعيد الجلب فوق البث (`:303`) |
| C7 | ترتيب/اعتماديات effect غير سليمة | `TransferModal.jsx:184-190` (يستدعي `loadDirectory` المعرّفة لاحقًا في `:193` وليست في deps) | يعمل بالحظ؛ أي تغيير في هوية الدالة يكسر التحديث |
| C8 | خمس قوائم مختلفة لحالة "قيد التشغيل" | `TransferContext.jsx:264`, `MiniTransferCenter.jsx:75,91,199`, `AdminTransferPage.jsx:200` | اختلاف سلوكي بين المكوّنات لنفس المهمة |
| C9 | تصنيف جهاز غير متسق | `transferDevices.js:1-21` مقابل `TransferModal.jsx:130-138` مقابل `AdminTransferPage.jsx:127-129` | جهاز "iPhone" من نوع `ios_mtp_*` يُعامل iOS في مكان وغير iOS في آخر |
| C10 | قراءة حقول v1/v2 غير متسقة | `AdminTransferPage.jsx:231-232` (v1 فقط) مقابل `MiniTransferCenter.jsx:101-105` (الهجين) | حقول مفقودة لمهام v2 في شاشة الإدارة |
| C11 | إرسال مفاتيح v1 و v2 معًا | `TransferContext.jsx:217-228` | يجمّد العميل في حالة "متعدد الملفات بشكل v1"؛ صيانة مزدوجة |
| C12 | `refreshData` غير مدرجة في deps | `AdminTransferPage.jsx:38-42` | تعمل لكنها هشّة تجاه تغييرات نطاق الدالة |

---

## 6. جدول الأولوية (خطورة x جهد)

| الأولوية | البند | الخطورة | الجهد | التوصية |
|---|---|---|---|---|
| P0 | C1 — وسم "فشل" الخاطئ في شاشة الإدارة | عالية | منخفض | توحيد مسند حالة واحد مشترك |
| P0 | S1 — سوء تصنيف UDID في `streamRemoteToDevice` | عالية | منخفض | تمرير النوع المحلول أو استخدام `isIOSUDID` |
| P1 | S4 — Android PutStream يكتب كامل الملف في temp | عالية | متوسط | إبقاء spool أو توثيق أنه مقصود، وإزالة تعليق Zero-Spool |
| P1 | S6 — احتمال تعليق `runTransferV2` | عالية | متوسط | اشتراك قبل `Submit` أو deadline/mirror مباشر |
| P1 | C3 — SSE + polling متعارضان | متوسطة | متوسط | الاعتماد على سياق SSE في شاشة الإدارة |
| P1 | C5 — `customBundleId` المفقود | عالية (لو فُعّل) | منخفض | حذف المكوّن الميت أو إصلاحه |
| P2 | D1/D2/D5 — تكرار الوسيط/الهاندلرز/updateJob | متوسطة | متوسط | توحيد طبقة أحداث وهاندلرز واحدة |
| P2 | D4 — تعدد قواعد نوع الجهاز | متوسطة | منخفض | دالة واحدة مشتركة |
| P2 | D6/D7/D8/D9 — مساعدات مكررة/متطابقة | منخفضة | منخفض | تجميع في حزمة utilities واحدة |
| P2 | C2 — فرع `cancelled` الميت | منخفضة | منخفض | إدراج `cancelled` أو حذف الفرع |
| P2 | S5 — CancelJob ناقص في `transfer` | متوسطة | منخفض | مواءمة الحقول مع copybridge |
| P3 | D12/D13/D14/D15/D16/D17/D18 — تكرارات العميل | منخفضة | متوسط | استخراج utilities مشتركة |
| P3 | القسم 4 بالكامل — أكواد ميتة | منخفضة | منخفض | حذف تدريجي بعد تأكيد المالك |
| P3 | S7/S9/S10/S12/S13 — تحسينات منطقية فرعية | منخفضة | منخفض | إصلاحات صغيرة مجمّعة |

> ملاحظة حاكمة: أغلب الأكواد الميتة متروكة كمرجع أو fallback. لا يُحذف منها شيء دون موافقة المالك (تصنيف Architectural حسب `AGENTS.md`).

---

## 7. ملاحظات ختامية

- هذا التقرير لا يقترح إعادة تصميم؛ يصف الوضع الحالي فقط مع توصيات قابلة للتنفيذ لاحقًا.
- الجيل v2 موجود ومُستخدم فعليًا للمسارات المحلية، بينما copybridge يبقي مسارًا مستقلًا لمصادر URL. أي توحيد بينهما يعد تغييرًا معماريًا يستلزم ADR وموافقة.
- الجيل v1 (مكوّنات العميل `CopyToPhoneModal` و`FloatingCopyButton`) مصنّف ميت ولم يُحذف.

---

## 8. ملحق المراجع السريعة

| المرجع | الوصف |
|---|---|
| `server/internal/copybridge/service.go:625` | `inferDeviceType` |
| `server/internal/copybridge/service.go:399` | إعادة اشتقاق نوع الجهاز |
| `server/internal/copybridge/service.go:864-897` | `eventBroker` المكرر |
| `server/internal/copybridge/service.go:459` | `followChildJob` (polling) |
| `server/internal/api/handlers_transfer.go:64-342` | هاندلرز API المكررة |
| `server/internal/transfer/service.go:807` | `mirrorV2Progress` |
| `server/internal/transfer/service.go:876` | `v2Outcome` (ميت) |
| `server/internal/transfer/service.go:927` | `StatDevicePath` (ميت) |
| `server/internal/transfer/service.go:1461` | `extractJSONValue` (ميت) |
| `server/internal/transfer/service.go:1480` | `min` (ميت) |
| `server/internal/transfer/scheduler.go:54` | `Scheduler.Stop` (ميت) |
| `server/internal/transfer/android_backend.go:230-259` | `PutStream` spool |
| `server/internal/transfer/errors.go:38-110` | ثوابت/دوال أخطاء بلا مراجع |
| `client/src/components/CopyToPhoneModal.jsx:49` | مكوّن v1 ميت |
| `client/src/components/transfer/FloatingCopyButton.jsx:17` | مكوّن ميت |
| `client/src/context/TransferContext.jsx:217-228` | payload هجين v1/v2 |
| `client/src/context/TransferContext.jsx:236` | جلب مكرر بعد البدء |
| `client/src/pages/admin/AdminTransferPage.jsx:200-223` | وسم الحالة الخاطئ |
| `client/src/components/transfer/MiniTransferCenter.jsx:41` | فرع `cancelled` الميت |
| `client/src/components/transfer/TransferModal.jsx:184-193` | ترتيب effect |

---

## 9. سجل التنظيف المنفّذ

نُفّذ التنظيف بتاريخ 2026-09-18 (المالك طلب "اعتمد الطريقة المستخدمة واحذف غير المستخدم"). التحقق: `go build ./...`، `go vet ./...`، `go test ./internal/transfer/... ./internal/copybridge/... ./internal/api/...`، و`npm run build` — كلها ناجحة.

### 9.1 الخادم

- حُذف مسار النسخ v1 بالكامل من `transfer/service.go`: `executeTransferLegacy`، `copyOneLegacy`، `copyToIOSDeviceGoIOS`، `copyToMTPDevice`، `waitForMTPFile`، `copyToIOSDevice`، `copyToLocalPath`، `updateProgress`. أصبح `runTransferV2` المسار الوحيد ويعيد خطأً واضحًا عند تعذّر بناء مهمة v2 بدل الـ fallback.
- حُذف `v2Outcome`، `extractJSONValue`، `min`، و`StatDevicePath` (مع إزالته من واجهة `internal/api/server.go`).
- حُذف `resolveTool` من `backend_helpers.go`.
- حُذفت ثوابت `PhaseQueued/PhaseVerifying/PhaseWaitingDevice/PhasePaused` و`StatusPaused/StatusWaiting`.
- حُذفت ثوابت الأخطاء غير المستخدمة (`CodeDeviceDisconnected`, `CodeAFCTimeout`, `CodeMuxError`, `CodeDiskFull`, `CodeDestinationNotFound`, `CodePermissionDenied`, `CodeUnsupportedOperation`, `CodeInvalidDestination`) ودوالها (`ErrDeviceLost`, `ErrAFCTimeout`, `ErrAFCIO`, `ErrDiskFull`, `ErrDestinationNotFound`, `ErrPermissionDenied`, `ErrCancelled`, `ErrUnsupported`) و`IsDeviceLost`.
- بدل حذف `Scheduler.Stop`/`DeviceWorker.Stop` (S12)، تم ربطهما في `Service.Close`؛ وجُعل `DeviceWorker.Stop` آمنًا للاستدعاء المتكرر (`sync.Once`)، و`Scheduler.Stop` يفرّغ registry فعليًا.
- أُبقي `queryMTPFileSize` و`powershellOutputHasError` لأنهما مستخدمان من `android_backend.go`.

### 9.2 العميل

- حُذف المكوّنان الميتان: `components/CopyToPhoneModal.jsx` و`components/transfer/FloatingCopyButton.jsx`.
- حُذفت مغلّفات `lib/api.js`: `getTransferAppFolders` و`getTransferJob`.
- أُزيل `export` عن `isRealIOSDevice`/`isWindowsIOSPlaceholder` (استخدام داخلي فقط).
- من `context/TransferContext.jsx` حُذفت الرموز غير المستهلكة: `isFileSelected`، `toggleFileSelection`، `hasRunningJobs`، وإظهار `clearSelection` و`refreshJobs` في قيمة الـ context (بقيت الاستخدامات الداخلية).

### 9.3 ما تبقّى (تغييرات لاحقة، لم تُنفّذ)

- تكرار الوسيط/الهاندلرز (D1/D2/D3/D5/D6) وتوحيد قواعد نوع الجهاز (D4/S1).
- إصلاحات منطقية: S3/S4/S5/S6/S7/S9/S10/S11/S13 و C1/C2/C3/C6/C7.
- توحيد `formatBytes` (D12) — أُبقي كما هو حفاظًا على تنسيق العرض الحالي.
