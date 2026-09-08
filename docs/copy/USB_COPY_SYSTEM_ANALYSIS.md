# تحليل نظام النسخ عبر USB في NEXORA

> **المستند**: تحليل معمّق لقسم النسخ عبر USB (اكتشاف الأجهزة، الوصول للملفات، النسخ) في مشروع NEXORA.
> **نطاق التحليل**: `server/internal/transfer/*` و `server/internal/api/server.go` و `client/src/**` (الواجهة).
> **الدولة الحالية**: تم الانتقال الكامل من أدوات `libimobiledevice` الخارجية إلى مكتبة Go-native **`go-ios`**، مع بقايا كود ووثائق وتوثيقات ما زالت تشير إلى الأدوات القديمة.
> **تاريخ التحليل**: 2026-09-04

---

## 1. ملخص تنفيذي

يتكون نظام النسخ عبر USB في NEXORA من ثلاث طبقات تعمل معاً:

1. **طبقة اكتشاف الأجهزة** (`Service.discoverIOSDevices` + `discoverWindowsMTPDevices` + `discoverRemovableDrives`) — تكتشف iPhone/iPad عبر مكتبة go-ios (`usbmuxd` native)، وأجهزة Android عبر PowerShell Shell.Application (MTP/WPD)، ووحدات التخزين الخارجية عبر `Get-Volume`.
2. **طبقة الوصول للنسخ** — ثلاثة Backends تحقق واجهة `TransferBackend` موحّدة:
   - `GoIOSBackend` (iOS): اتصال AFC عبر `house_arrest` + `afc` مباشرة داخل Go (بدون عمليات خارجية).
   - `AndroidBackend` (Android MTP): نسخ عبر `PowerShell CopyHere(16)` ثم استطلاع حجم الملف على الهاتف.
   - `StorageBackend` (قرص USB): نسخ مباشر عبر Go `io` العادي.
3. **طبقة الجدولة والتقدم** — محرك v2 (`TransferEngine` + `Scheduler` + `DeviceWorker`) يدير وظائف متعددة الملفات، إعادة محاولة، تحقق، وحساب السرعة/نسبة التقدم.

**أهم خلاصة**: تحوّل iOS من "استدعاء عمليات `afcclient` الخارجية" إلى "مكتبة go-ios مدمجة" — وهذا يزيل الاعتماد على وجود `.tools\libimobiledevice` في النظام، ويزيل مشكلة تعقيد إنشاء عمليات جديدة لكل ملف.

---

## 2. اكتشاف الأجهزة (Device Discovery)

### 2.1 iOS — go-ios / usbmuxd (مسار جديد)

الوظيفة: `Service.discoverIOSDevices` في `server/internal/transfer/service.go:325-372`

```
func (s *Service) discoverIOSDevices(ctx context.Context) ([]Device, error) {
    devices, err := goios.ListDevices()   // 1. enumerate عبر usbmuxd (native)
    ...
    for _, entry := range devices.DeviceList {
        udid := entry.Properties.SerialNumber
        // اسم الجهاز عبر Lockdown Session (بدل idevicename الخارجي)
        if lockdown, err := goios.ConnectLockdownWithSession(entry); err == nil {
            if values, err := lockdown.GetValues(); err == nil {
                name = values.Value.DeviceName
            }
        }
    }
}
```

- **قبل**: `getIDeviceTool("idevice_id") -l` (عملية خارجية) + `idevicename -u <UDID>` (عملية خارجية ثانية لكل جهاز).
- **بعد**: `goios.ListDevices()` يعتمد على usbmuxd مباشرة في Go، واسم الجهاز يُقرأ عبر Lockdown session.
- الـ Device ID الناتج بصيغة `ios_<index>_<UDID>` (مثال: `ios_0_00008030-001450161E00802E`).

### 2.2 Android / iPhone عبر Windows MTP (WPD)

الوظيفة: `Service.discoverWindowsMTPDevices` في `service.go:192-290`

- يشغّل سكربت PowerShell يستخدم `Shell.Application` (اسم فضاء `17` = My Computer) ويجمع العناصر التي يطابق نوعها:
  - `*Portable*`, `*Camera*`, `*هاتف*`, `*جهاز*`, أو البادئة `::` (ممرات shell).
  - وأسماء `iPhone/Apple/iPad/Android/Galaxy/Huawei/Redmi/Xiaomi`.
- **كاشف نوع**: `windowsPortableDeviceKind` (`service.go:311-323`) → إن احتوى الاسم "iphone/apple/ipad" يُصنّف `ios` (placeholder)، وإلا `android` (MTP).
- الـ Device ID: `mtp_<index>_<Name>` للأندرويد، `ios_mtp_<index>_<Name>` لـ iPhone.
- **أهمية**: هذه أجهزة iOS من نوع *placeholder* (تُستخدم قبل أن يفعّل المستخدم الثقة / قبل اتصال AFC)، ويتم **فلترتها** إذا وُجد جهاز iOS حقيقي عبر go-ios (دالة `filterWindowsIOSPlaceholders` في `service.go:172-190`).

### 2.3 وحدات تخزين USB الخارجية

الوظيفة: `Service.discoverRemovableDrives` في `service.go:442-478`

- `Get-Volume | Where DriveType = 'Removable'` → Device ID: `disk_<DriveLetter>` (مثال: `disk_E`).
- تُستخدم فقط كـ **fallback** عندما لا يُكتشف أي جهاز iOS أو Android (لمنع إغراق القائمة برموز محركات الأقراص).

### 2.4 ترتيب الفحص

`scanDevicesInternal` في `service.go:139-170`:

```
1. goios.ListDevices()              → أجهزة iOS حقيقية
2. (Windows) MTP discovery          → Android + iOS placeholders
   (تُفلتر placeholders إذا وُجد iOS حقيقي)
3. (Windows, fallback) Removable    → أقراص USB إذا لم يوجد شيء
```

المراقبة: `startDeviceMonitor` (`service.go:63-78`) يفحص كل 1.5 ثانية ويخزّن في كاش ساخن `cachedDevices`. `ListDevices` يعيد الكاش فوراً إذا كان عمره < 4 ثوانٍ.

---

## 3. الوصول إلى الملفات والنسخ (Backends)

### 3.1 الواجهة الموحدة `TransferBackend`

`server/internal/transfer/backend.go:54-78`

```
Connect(ctx, device, destination) error
List(ctx, path) ([]RemoteEntry, error)
Stat(ctx, path) (RemoteEntry, error)
Mkdir(ctx, path) error
Put(ctx, source, destination, opts PutOptions) error
Delete(ctx, path) error
Rename(ctx, oldPath, newPath) error
Close() error
```

اختيار الـ backend: `newV2Engine` في `service.go:95-110`:

```
iOS + UDID حقيقي  → NewGoIOSBackend()
storage (disk_)   → NewStorageBackend()
ما عدا ذلك        → NewAndroidBackend()
```

### 3.2 iOS — `GoIOSBackend` (go-ios native) ✅ الجديد

`server/internal/transfer/go_ios_backend.go`

- **الاتصال**: `house_arrest.New(entry, appID)` يفتح AFC مباشرة على مجلد Documents للتطبيق (بدل `afcclient --documents`).
- **النسخ**: `client.Open(dest, afc.WRITE_ONLY_CREATE_TRUNC)` ثم بث مباشر بذاكرة `defaultBufferSize` (4MB محدودة 512KB في PUT iOS) مع `OnProgress` بعد كل كتابة.
- **الاستئناف**: يفتح بـ `afc.WRITE_ONLY_CREATE_APPEND` إذا وُجد `ResumeOffset > 0`.
- **عمليات محدودة**: `Rename` غير مدعوم (يرجع خطأ "go-ios AFC لا يوفر إعادة تسمية مباشرة")، والفرق الأكبر: **لا regex** ولا parsing نصي للمخرجات — كل شيء typed API.

### 3.3 Android (MTP) — `AndroidBackend` (PowerShell)

`server/internal/transfer/android_backend.go`

- **النسخ**: `$finalTarget.CopyHere(source, 16)` (16 = no UI). هذا غير متزامن — يعيد فوراً، ثم `waitForMTPFile` يستطلع حجم الملف على الهاتف كل 1200ms عبر `queryMTPFileSize` حتى يصل إلى حجم المصدر.
- **التحقق**: بحجم الملف فقط (يقرأ `System.Size` أو `item.Size`).
- الحد الأقصى للانتظار: `transferTimeout(size)` في `progress.go:15-28` (90 ثانية حد أدنى، و (الحجم/4MB)+45 ثانية، و90 دقيقة حد أقصى).

### 3.4 قرص USB — `StorageBackend` (نقل Go خالص)

`server/internal/transfer/storage_backend.go`

- نسخ مباشر `os.Open → os.Create → io.Copy` بذاكرة 4MB، مع دعم `ResumeOffset` (seek على الوجهة والمصدر).
- عمليات كاملة: إعادة تسمية وحذف مدعومان عبر `os.Rename` / `os.Remove`.

### 3.5 المسار القديم الاحتياطي (Legacy)

`executeTransferLegacy` في `service.go:882-929` + `copyOneLegacy` يوجّه إلى:

- `copyToIOSDeviceGoIOS` (`service.go:936-974`) — يستخدم نفس `GoIOSBackend`.
- `copyToMTPDevice` (`service.go:976-1102`) — نفس أسلوب AndroidBackend.
- `copyToLocalPath` (`service.go:1201-1270`) — نفس أسلوب StorageBackend لكن في service.

يُستخدم هذا المسار فقط عندما يتعذر وصف وجهة v2 أو يرفض المحرك الوظيفة.

---

## 4. محرك الجدولة والتقدم (v2 Engine)

### 4.1 المكونات

| الملف | الدور |
|---|---|
| `engine.go` | `TransferEngine`: يسجّل الوظائف النشطة، بنية `TransferConfig` (ذاكرة 4MB، 3 محاولات، verify size) |
| `scheduler.go` | `Scheduler`: يوجّه كل وظيفة إلى Worker خاص بالجهاز (`workerKey` = `device:<DeviceID>`) |
| `worker.go` | `DeviceWorker`: يستهلك قائمة الانتظار تسلسلياً، ينفّذ الوظيفة، يحسب السرعة/التقدم |
| `verify.go` | `Verifier`: تحقق حجم (الافتراضي) أو صارم (hash، حالياً يقع على الحجم) |
| `errors.go` | `TransferError` مُصنّف: `DEVICE_DISCONNECTED`, `AFC_TIMEOUT`, `AFC_IO_ERROR`, `MUX_ERROR`, `DISK_FULL`... |
| `progress.go` | `transferTimeout`, `formatBytes`, `parseHumanOrNumericSize` |

### 4.2 تدفق وظيفة النسخ (`worker.go`)

```
processJob:
  1. backend = engine.backendFor(destination)
  2. backend.Connect(...)
  3. backend.Mkdir(remotePath)            // إنشاء مجلدات الوجهة
  4. لكل ملف:
       opts.ResumeOffset = remoteResumeOffset(...)
       backend.Put(source, dest, opts)   // OnProgress يُحدّث job
       engine.verify.Verify(...)         // تحقق الحجم
  5. job.Status = completed, Progress=100
```

### 4.3 حساب التقدم والسرعة (`worker.go:222-266`)

- `onFileProgress`: يجمع أحجام الملفات المنجزة + البايتات المنقولة للحالي → `job.TransferredBytes` → `job.Progress = done/TotalBytes*100`.
- السرعة: تحدَّث كل ≥ 500ms بـ EMA smooth (0.7 القديمة + 0.3 اللحظية) → `SpeedBps`, و`ETA = remaining/SpeedBps`.
- `mirrorV2Progress` في `service.go:755-787` يعكس هذه القيم على `TransferJob` القديم (REST) كل 250ms عبر ticker في `runTransferV2`.

---

## 5. مسار REST والواجهة الأمامية

### 5.1 الـ Endpoints (في `server/internal/api/server.go:302-312`)

| الطريقة | المسار | الـ Handler |
|---|---|---|
| GET | `/api/transfer/devices` | `handleTransferDevices` (خدمة: `ListDevices`) |
| GET | `/api/transfer/device-apps` | `handleTransferDeviceApps` (`ListDeviceApps`) |
| GET | `/api/transfer/device-app-folders` | `handleTransferDeviceAppFolders` (`ListAppFolders`) |
| POST | `/api/transfer/copy` | `handleTransferCopy` (`StartCopy`) |
| GET | `/api/transfer/jobs` | `handleTransferJobsList` (`ListJobs`) |
| GET | `/api/transfer/job/{id}` | `handleTransferJobGet` (`GetJob`) |
| POST | `/api/transfer/cancel/{id}` | `handleTransferJobCancel` (`CancelJob`) |
| GET | `/api/transfer/browse` | `handleTransferBrowse` (`ListDevicePath`) |
| POST | `/api/transfer/mkdir` | `handleTransferMkdir` (`CreateDeviceFolder`) |

ملاحظات:
- لا يوجد WebSocket/SSE — التقدم يصل للواجهة عبر **Polling** HTTP فقط.
- `go.mod` يحتوي `github.com/gorilla/websocket v1.5.3` كاعتماد **غير مستخدم** (indirect).

### 5.2 الواجهة (React)

- `TransferContext.jsx`: `fetchJobs()` كل 800ms أثناء النسخ (Polling على `GET /api/transfer/jobs`).
- `MiniTransferCenter.jsx`: يقرأ `job.progress` مباشرة ويعرض `(progress.toFixed(0)%)` وشريط عرضه `width: progress%`.
- `TransferModal.jsx`: نافذة الاختيار/البدء فقط (أجهزة، تطبيقات، مجلدات).

---

## 6. 🔴 مشكلة "عدم ظهور نسبة النسخ" — التحليل الجذري

### 6.1 ما وجده التحليل

الخادم يُنتج `progress` بشكل صحيح من `transferred_bytes/total_bytes` (`worker.go:239`) ويعكسه على REST كل 250ms. المشكلة محصورة في **طبقة الكاش في واجهة العميل**:

### 6.2 السبب الجذري: `client/src/lib/api.js`

1. **`READ_CACHE_TTL = 2*60*1000`** (سطر 6): أي استجابة GET تُخزَّن في الذاكرة + `sessionStorage` لمدة **دقيقتين**.
2. **`isCacheableRequest`** (سطر 16-20): تعتبر أي GET (عدا `/api/admin/` و`/api/stream/`) **قابلاً للتخزين** — **بما فيها `/api/transfer/jobs` و`/api/transfer/job/{id}` و`/api/transfer/devices`**.
3. **`requestJSON`** (سطر 55-63): إذا وجد كاشاً صالحاً يرجعه فوراً **دون أي fetch فعلي**.
4. **`cache: "no-store"`** الذي يُمرَّر في دوال النقل (`getTransferJobs`, `getTransferDevices`, ... في الأسطر 421-464) **لا تُقرؤه `requestJSON` إطلاقاً** — يذهب فقط كخيار fetch ولا يُستخدم كعتبة تخطّي للكاش الذاتي.

### 6.3 تسلسل الفشل

1. المستخدم يضغط "نسخ" → POST `/api/transfer/copy` → `invalidateAPICache()` يمسح الكاش.
2. `startTransfer` يستدعي `fetchJobs()` مرة → لقطة جيدة أولية (`progress ≈ 1`).
3. **هذه اللقطة تُخزَّن 120 ثانية**.
4. كل استطلاعات الـ 800ms/4000ms التالية تُعيد **نفس اللقطة المجمّدة** → النسبة تلتصق عند 1%/0% ولا تتحرك، والسرعة والوقت المتبقي يظهران صفراً.
5. بعد انتهاء الـ TTL قد يقفز التقدم فجأة إلى قيمة لاحقة، لكن تجربة المستخدم خلال النسخ تكون "النسبة لا تظهر".

### 6.4 جهات ذات صلة

- نفس الخلل **يجمّد كشف الأجهزة الحية** في `TransferModal` (Polling كل 1.5 ثانية) — فلن تظهر الأجهزة الموصولة حديثاً خلال دقيقتين.
- الخادم لا يخزّن `/api/transfer/*` في كاشه (`cache.go:71-82` ترجع 0 لكل ما عدا الفهارس والداشبورد) — **المشكلة client-side خالصة**.

### 6.5 الإصلاح المقترح (لم يُنفَّذ — توثيق فقط)

في `client/src/lib/api.js`:

```js
function isCacheableRequest(path, options) {
  return (!options.method || options.method.toUpperCase() === "GET")
    && !path.startsWith("/api/admin/")
    && !path.startsWith("/api/stream/")
    && !path.startsWith("/api/transfer/");      // ← إضافة: مسارات النقل لا تُخزَّن
}
```

أو جعل `requestJSON` تحترم `options.cache === "no-store"`:

```js
const cacheable = isCacheableRequest(path, options) && options.cache !== "no-store";
```

> ملاحظة ثانوية: على Android MTP، تقدم النسبة يعدو قفزات كل 1200ms (استطلاع حجم الملف على الهاتف) فهو ليس سلساً مثل iOS، لكنه يعمل.

---

## 7. 🔎 الكود المكرر (Duplicated Code)

### 7.1 سكربت PowerShell MTP مكرر شبه حرفي

أسلوب "إيجاد الجهاز → GetFolder → إنشاء مجلدات → WaitForFile" **مكرر في 4 مواضع**:

| الموقع | الغرض |
|---|---|
| `service.go:1012-1098` (`copyToMTPDevice`) | نسخ legacy |
| `service.go:1143-1177` (`queryMTPFileSize`) | استطلاع حجم legacy |
| `android_backend.go:61-99` (`Mkdir`) | إنشاء مجلدات |
| `android_backend.go:125-172` (`Put`) | نسخ عبر الـ backend |

كما أن منطق "إيجاد الجهاز (`$deviceItem = $null; foreach ...`) ثم `$firstStorage`" مكرر في كل سكربت. **مقترح**: دالة PS مشتركة مستخرجة (template) أو برنامج Go واحد يعيد المسار.

### 7.2 منطق "انتظار ظهور/اكتمال الملف" مكرر

- `service.go:1104-1137` (`waitForMTPFile`)
- `android_backend.go:188-218` (`waitForMTPFile`)

نفس الحلقات، ونفس منطق خطأ "توقف نقل USB قبل اكتمال الملف". يمكن استخراجهما لدالة واحدة مشتركة.

### 7.3 منطق الاستئناف/التحقق

- `remoteResumeOffset` و`verifySize` في `verify.go` يستخدمهما كل من `worker.go` والمسارات القديمة؛ موحّد جيداً لكن في legacy يوجد `entry.Size != job.FileSize` يدوي (خدمة `copyToIOSDeviceGoIOS`).

### 7.4 schema الرد بين `jobs` و `job/{id}`

- `handleTransferJobsList` يغلّف بـ `{"jobs": [...]}` بينما `handleTransferJobGet` يرد **بالكائن مباشرة**. هذا تباين غير موثق يجب تثبيته (مغلف واحد موحّد).

### 7.5 استطلاع التقدم: ثلاث آليات

- `TransferContext` كل 2500→800ms على `jobs`.
- `CopyToPhoneModal.jsx` (غير مستخدم!) كل 500ms على `job/{id}`.
- `AdminTransferPage` كل 3000ms.
توحيدها في `TransferContext` يقلل الازدواجية، وربما إزالة `CopyToPhoneModal` غير المستورد.

---

## 8. ✅ تنظيف بقايا libimobiledevice (مكتمل)

> **الحالة الآن**: أُزيل كل كود/إعداد يشير إلى أدوات `libimobiledevice` الخارجية ولم يعد `Options.IDevicePath` موجوداً. ما تبقّى هو توثيق تاريخي فقط.

### 8.1 العاملة (أُزيلت بالكامل)

| الموقع | الوصف | الحالة |
|---|---|---|
| `server/internal/config/config.go` | حقل `IDevicePath` + قراءة `NEXORA_IDEVICE_PATH` | **مُزال** |
| `server/internal/transfer/models.go` | حقل `Options.IDevicePath` | **مُزال** |
| `server/internal/app/app.go` | تمرير `cfg.IDevicePath` إلى `transfer.NewService` | **مُزال** |
| `.env.example` | سطر `NEXORA_IDEVICE_PATH=.tools\libimobiledevice` | **مُزال** |
| `service.go` | `isLibimobileDeviceID` / `parseLibimobileDeviceID` | **أُعيدت تسميتها** إلى `isIOSUDID` / `parseIOSUDID` (المنطق نفسه محفوظ — يميّز iOS UDID حقيقي من placeholder) |

### 8.2 أسماء/توثيق قديم (تسميات مضللة لا تؤثر في السلوك)

| الموقع | الملاحظة |
|---|---|
| `service.go` | دوال `isIOSUDID` / `parseIOSUDID` / `isLikelyAppleUDID` — منطقها = "هل هو UDID حقيقي؟" (تحقق hex/طول). |
| `service_test.go` | اختبار `TestParseIOSUDIDRejectsWindowsPlaceholder` برسالة محايدة. |
| `docs/copy/NEXORA_TRANSFER_ENGINE_V2_DEVELOPMENT_PLAN.md` | وثيقة التخطيط تحتوي فقرات `afcclient` تفصيلية (سطور 9, 20, 582-618, 1881-1903, 2884-2935) — مبنية على الافتراض القديم بأننا نعتمد `afcclient` التفاعلي. تاريخية (التنفيذ الفعلي عبر go-ios). |
| `.merge-backups/local-changes-before-github-merge.patch` | أرشيف — لا يحتاج تعديل. |

### 8.3 اعتماد go-ios في `go.mod`

`server/go.mod:16`: `github.com/danielpaulus/go-ios v1.3.2` — **مُدرج كـ `// indirect` رغم استخدامه مباشرة** في `go_ios_backend.go` و`service.go`. يجب نقله إلى كتلة `require` المباشرة.

### 8.4 ملفات قديمة محذوفة من الـ PR الحالي (غير متتبّعة/محذوفة)

| الملف | الوصف | الحالة الحالية |
|---|---|---|
| `server/internal/transfer/afc_manager.go` | `SessionManager` + `CommandDispatcher` لإدارة عمليات `afcclient` | محذوف |
| `server/internal/transfer/afc_session.go` | جلسة `afcclient` تفاعلية (stdin/stdout) | محذوف |
| `server/internal/transfer/afc_session_test.go` | اختبارات الجلسة | محذوف |
| `server/internal/transfer/ios_backend.go` | الـ backend القديم عبر `afcclient` + `parseAFCLongListing` | محذوف |
| `server/internal/transfer/local_path_windows.go` | تحويل مسار لـ 8.3 (ShortPathName) لـ afcclient | محذوف |
| `server/internal/transfer/local_path_other.go` | تنفيذ بديل (غير Windows) | محذوف |
| `server/internal/transfer/zz_parse_long_test.go` | اختبارات parsing `ls -l` | محذوف |

استُبدلت جميعها بـ `go_ios_backend.go` + `go_ios_backend_test.go` (جديدان، غير متتبّعين بعد — `git status` يُظهرهما كـ Untracked).

---

## 9. ملاحظات إضافية / تحسينات مقترحة

1. ✅ **إزالة حقل `IDevicePath`** من `Options`/`Config`/`app.go`/`.env.example` — **تم** (لم يعد go-ios يحتاجه).
2. **نقل `go-ios` من `// indirect` إلى الاعتماد المباشر** في `go.mod`.
3. ✅ **إعادة تسمية** دوال `isLibimobileDeviceID`/`parseLibimobileDeviceID` إلى `isIOSUDID`/`parseIOSUDID` — **تم**.
4. **إصلاح كاش الواجهة** (القسم 6.5) — وهو المشكلة العملية الوحيدة الملموسة المذكورة (نسبة النسخ لا تظهر).
5. **توحيد سكربتات PowerShell** المتفشية في 4 مواضع في أداة/قالب واحد.
6. **توحيد الاستجابة** بين `jobs` (مغلّف) و `job/{id}` (غير مغلّف).
7. **مراعاة أن `CopyHere(16)` على MTP غير متزامن** فلن يكون التقدم سلساً؛ يمكن مستقبلاً اللجوء إلى `WPD` native أو أسلوب `Get-Item` دوري بالفعل — الحل الحالي مقبول.
8. **`CopyToPhoneModal.jsx` غير مستورَد** — إما حذفه أو دمجه في `TransferContext` ليكون مصدر استطلاع واحد.

---

## 10. مخطط التدفق (سياقي)

```
[React UI]
   TransferModal ── GET /api/transfer/devices ──► Service.discoverIOSDevices (goios.ListDevices)
   │                                                   + discoverWindowsMTPDevices (PowerShell)
   │                                                   + discoverRemovableDrives (PowerShell)
   │ POST /api/transfer/copy ──► StartCopy ──► TransferEngine.Submit ─► Scheduler
   │                                                    │  (worker per device)
   │                                                    ▼
   │                                              DeviceWorker.processJob
   │                                                    │
   │                          ┌─────────────────────────┼─────────────────────────┐
   │                          ▼                         ▼                         ▼
   │                   GoIOSBackend             AndroidBackend             StorageBackend
   │              (house_arrest → AFC)     (PowerShell CopyHere)          (Go io copy)
   │                          │                         │                         │
   │                          └────────── TransferBackend.Put ──────────────────┘
   │                                          │
   │                                   OnProgress(done)
   │                                          ▼
   │                              worker.onFileProgress → job.Progress
   │                                          │
   MiniTransferCenter ◄── Poll 800ms ── GET /api/transfer/jobs ── mirrorV2Progress (250ms ticker)
```

---

## 11. الخلاصة

- **What changed**: iOS نقل الملفات أصبح عبر `go-ios` (مكتبة Go) — `house_arrest` + `afc` داخلياً، بلا اعتماد على `.tools/libimobiledevice`.
- **ما زال يعمل بآخرها**: خيار `NEXORA_IDEVICE_PATH` وأسماء/تسميات `libimobile*` ووثيقة التخطيط القديمة.
- **مشكلة نسبة النسخ**: سببها كاش `requestJSON` في `client/src/lib/api.js` الذي يتجاهل `cache:"no-store"` ويجمّد استجابات `/api/transfer/*` لمدة 120 ثانية.
- **الخطوة الموصى بها التالية**: إصلاح كاش الواجهة (بند 6.5)، ثم تنظيف بقايا libimobiledevice (بند 8).

---

*نهاية التحليل.*