# NEXORA Agent — Architecture

> **الحالة:** معمارية قائمة، موثّقة من الكود الفعلي. لا افتراضات.
> **يعتمد على:** [MIGRATION_BASELINE](MIGRATION_BASELINE.md) · ADR-008 (Copy Bridge)

---

## 1. الصورة الكاملة

```text
                    NEXORA WEB (React/Vite)
                            │
                            │ HTTP/SSE  ← يعمل محليًا
                            ▼
                 NEXORA SERVER  (server/cmd/api :8080)
                            │
              ┌─────────────┴──────────────┐
              │                            │
       Media / Metadata / Auth      Catalogue / Search
              │                            │
              │                            │
              ▼                            ▼
     PostgreSQL · Meilisearch         الملفات على القرص
              │
              │  Browser → 127.0.0.1:32145  للعمليات المحلية فقط
              ▼
     ┌──────────────────────────────┐
     │      NEXORA AGENT            │
     │      (cmd/copybridge)        │
     │      Go · Windows Service    │
     │      127.0.0.1:32145         │
     └──────────────┬───────────────┘
                    │
          ┌─────────┼─────────────┐
          │         │             │
          ▼         ▼             ▼
     TransferBackend (interface موحّد)
          │         │             │
          ▼         ▼             ▼
     StorageBackend  AndroidBackend  IOSBackend
     (Win32 I/O)     (Shell COM/MTP) (go-ios AFC)
          │         │             │
          ▼         ▼             ▼
      C: D: E:    Android       iPhone
      USB Flash   Phones
      HDD/SSD
```

**المفتاح:** طبقة `TransferBackend` هي الحد الفاصل. الواجهة (Frontend) لا تعرف
شيئاً عن Shell COM أو AFC أو Win32 — ترى `Device` و`TransferJob` فقط.

---

## 2. المكونات الفعلية (أدلة من الكود)

| المكون | الملف | المسؤولية |
|---|---|---|
| **Agent** | `server/cmd/copybridge/main.go` | نقطة دخول: Windows Service أو `-debug` Console |
| **Agent service** | `server/cmd/copybridge/service_windows.go` | ربط `golang.org/x/sys/windows/svc` |
| **Agent API** | `server/internal/copybridge/server.go` | 13 endpoint + SSE + CORS + token |
| **Transfer facade** | `server/internal/transfer/service.go` | اكتشاف الأجهزة + الواجهة العامة |
| **Transfer engine v2** | `server/internal/transfer/engine.go` | جدولة، إعادة محاولة، تحقق |
| **Worker** | `server/internal/transfer/worker.go` | تنفيذ ملف واحد |
| **Scheduler** | `server/internal/transfer/scheduler.go` | عدالة بين الأجهزة |
| **Backend contract** | `server/internal/transfer/backend.go` | `TransferBackend` interface |
| **Storage backend** | `server/internal/transfer/storage_backend.go` | Win32/os I/O — USB, HDD, الأقراص |
| **Android backend** | `server/internal/transfer/android_backend.go` | Shell COM + MTP |
| **iOS backend** | `server/internal/transfer/go_ios_backend.go` | go-ios AFC (**محمي**) |
| **Errors** | `server/internal/transfer/errors.go` | `TransferError` مُصنَّف |

---

## 3. تدفق اكتشاف الأجهزة (Device Flow)

```text
DiscoverDevices(ctx)
    │
    ├──► 1. iOS            discoverIOSDevices()          مهلة 2s
    │         └─► go-ios/usbmuxd
    │
    ├──► 2. Windows MTP    discoverWindowsMTPDevices()   مهلة 2.5s
    │         └─► PowerShell Shell.Application, NameSpace(17)
    │         └─► filterWindowsIOSPlaceholders()  ← يزيل ما ظهر كـ iPhone
    │
    └──► 3. Removable      discoverRemovableDrives()      < 1ms
              └─► kernel32: GetLogicalDrives, GetDriveTypeW,
                  GetDiskFreeSpaceExW, GetVolumeInformationW
```

**ترتيب مهم:** iOS يُكتشف **قبل** MTP، ثم يُفلتر شبائه iOS من نتائج MTP. هذا
يمنع ظهور iPhone مرتين (مرة كجهاز iOS، ومرة كجهاز MTP).

### نموذج الجهاز الحالي

```go
type Device struct {
    ID         string     `json:"id"`         // "disk_E" | "mtp_<name>" | "ios_<udid>"
    Name       string     `json:"name"`
    Model      string     `json:"model"`
    Type       DeviceType `json:"type"`       // android | ios | storage
    Status     string     `json:"status"`
    FreeSpace  int64      `json:"free_space"`
    TotalSpace int64      `json:"total_space"`
    FileSystem string     `json:"file_system,omitempty"`
}
```

> **ملاحظة معمارية (فجوة موثّقة):** `Device` المسطّح لا يعبّر عن جهاز له **عدة
> مساحات تخزين** (`storages[]` كما في الطلب §9). Android الحالي يفترض أن **أول**
> مساحة هي الهدف (`$storage.Items() | Select-Object -First 1`). هذا قيد حقي
> مُوثَّق في [ANDROID.md](ANDROID.md) وفي [ROADMAP.md](ROADMAP.md).

---

## 4. تدفق النقل (Transfer Flow)

```text
1. الواجهة ترسل طلب النسخ
        │
        ▼
2. Agent API يستقبل  POST /api/transfer/copy
        │   └─► التحقق من الأصل (corsAllow)
        │   └─► التحقق من الرمز (tokenMatches, constant-time)
        ▼
3. TransferEngine يجدول مهمة (TransferJobV2)
        │   └─► PerDeviceConcurrency = 1  (منع تضارب I/O على نفس USB)
        ▼
4. Worker ينفّذ الملف الواحد
        │
        ├─► PhasePreparing           تحقق من المصدر
        ├─► PhaseConnecting          backend.Connect()
        ├─► PhaseCheckingDestination backend.Mkdir() + فحص المساحة
        ├─► PhaseCopying             backend.Put() / PutStream()
        │       └─► OnProgress(transferred int64)  → SSE فوري
        ├─► PhaseRetrying            عند فشل قابل للإعادة (حتى 3 مرات، backoff 500ms→4s)
        └─► PhaseCompleted           تحقق (size أو SHA-256)
        │
        ▼
5. SSE يبث الحدث → الواجهة تحدّث التقدم لحظياً
```

### نموذج المهمة v2

```go
type TransferJobV2 struct {
    ID               string
    DeviceID         string
    DeviceName       string
    DeviceType       DeviceType
    Destination      TransferDestination
    Files            []TransferFile
    TotalBytes       int64              // 64-bit
    TransferredBytes int64              // 64-bit
    CurrentFileIndex int
    Progress         float64            // 0..100
    SpeedBps         int64
    // ... ETASeconds, Phase, Status
}
```

> **مطابق للطلب §12:** الأحجام `int64` (64-bit) في كل الطبقات. لا `int` لأحجام
> الملفات.

---

## 5. حدود الطبقات — ماذا يرى من؟

| الطبقة | ترى | لا ترى |
|---|---|---|
| **Frontend (React)** | `Device`, `TransferJobV2`, أحداث SSE | Shell COM، AFC، Win32، MTP |
| **Agent API** | طلبات REST، رموز، مناشئ | تفاصيل البروتوكول لكل جهاز |
| **TransferEngine** | `TransferBackend`، `Device`, `PutOptions` | كيف يُنفّذ كل backend عمله |
| **Backend** | مسارات، واجهات نظام | حالة الواجهة، قائمة الانتظار |

**الفائدة المعمارية:** إضافة `NetworkProvider` أو `AndroidADBProvider` مستقبلاً
تعني **إضافة backend جديد** فقط — بلا لمس الواجهة أو المحرك.

---

## 6. قرار: WPD مُخطَّط لا مُنفَّذ

المطلوب في الطلب (§6, §8) أن يكون WPD/MTP هو محرك Android. **قرار المالك (خيار أ):
WPD يبقى مُخطَّطاً، ويُكمل Shell COM المعزول.**

**السبب الموثّق:**
1. لا يوجد أي كود WPD في المشروع حالياً — بناؤه **من الصفر**.
2. WPD في Go يحتاج COM interop يدوي (`IPortableDevice`, `IPortableDeviceValues`,
   `IStream`) أو مكتبة خارجية بلا اعتماد موثوق.
3. Shell COM **يعمل فعلاً ومُختبر** مع أجهزة حقيقية.
4. الواجهة معزولة بـ `TransferBackend` أصلاً — **فالاستبدال لاحقاً لا يلمس الواجهة**.

راجع [ROADMAP.md](ROADMAP.md) §WPD للخطة التفصيلية.

---

## 7. مصادر (Microsoft الرسمية)

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [WPD Application Programming Interface](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/wpd-application-programming-interface)
- [Win32 File Management Functions](https://learn.microsoft.com/en-us/windows/win32/fileio/file-management-functions)
- [Windows Shell Extensions](https://learn.microsoft.com/en-us/windows/win32/shell/shell-exts)
