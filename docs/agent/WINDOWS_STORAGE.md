# NEXORA Agent — Windows Storage

> **الحالة:** يعمل، مُختبر، ومحمي. هذا الملف يوثّق ولا يغيّر.
> **الملفات المرجعية:** `removable_windows.go` · `storage_backend.go`

---

## 1. الطبقات الثلاث للوصول إلى التخزين

```text
Windows Storage Access
        │
        ├── 1. Win32 Volume API   ← اكتشاف الأقراص (fast path)
        │      kernel32.dll
        │
        ├── 2. Win32 / os File I/O ← النقل الفعلي (Go os package فوق Win32)
        │      storage_backend.go
        │
        └── 3. Windows Shell COM   ← الأجهزة المحمولة فقط (MTP)
               android_backend.go
```

**قاعدة معمارية:** كل طبقة تُستخدم لما تصلح له. لا نستخدم Shell COM لقرص عادي
(أبطأ وأثقل)، ولا Win32 لجهاز MTP (لا يعمل).

---

## 2. اكتشاف الأقراص — Win32 مباشر

`server/internal/transfer/removable_windows.go` (بنية `//go:build windows`).

### الدوال المستخدمة

| الدالة | DLL | الغرض |
|---|---|---|
| `GetLogicalDrives` | kernel32 | قناع 32-bit لكل حرف قرص موجود |
| `GetDriveTypeW` | kernel32 | نوع القرص (`DRIVE_REMOVABLE`=2) |
| `GetDiskFreeSpaceExW` | kernel32 | المتاح/الإجمالي (int64) |
| `GetVolumeInformationW` | kernel32 | اسم الحجم + نظام الملفات |

### الخصائص المُقاسة (من تعليق الكود)

```go
// discoverRemovableDrivesWin32 queries Windows kernel32 directly.
// It finishes in under 1ms, requires 0% CPU, and strictly isolates
// removable USB flash drives, completely ignoring internal fixed disks.
```

### القيود المتعمدة

```go
const driveRemovable = 2 // DRIVE_REMOVABLE

// Never treat system drive C: as a removable drive
if letter == "C" || letter == "c" { continue }

// Check drive type: strictly removable (USB flash drive / memory stick)
if dtype != driveRemovable { continue }
```

| القرار | السبب |
|---|---|
| تجاهل C: صريحاً | حماية من كتابة على قرص النظام |
| `DRIVE_REMOVABLE` فقط | لا External HDD يُظهر كـ `DRIVE_FIXED` |
| تجاهل الأقراص الثابتة | منع عرض أقراص داخلية للمستخدم |

> **⚠️ قيد موثّق:** External HDD/SSD الذي يُعرّفه Windows كـ `DRIVE_FIXED` (وهو
> الشائع للأقراص الكبيرة) **لن يظهر** حالياً. مُدرَج في [ROADMAP](ROADMAP.md).

### المعرّف المُنتَج

```go
ID: "disk_" + letter     // مثال: "disk_E"
```

| الحقل | المصدر |
|---|---|
| `Name` | اسم الحجم، أو `"ذاكرة USB (E:)"` كبديل |
| `Model` | ثابت `"USB Storage"` |
| `Type` | `DeviceStorage` |
| `FreeSpace`/`TotalSpace` | `GetDiskFreeSpaceExW` (int64) |
| `FileSystem` | `GetVolumeInformationW` (مثال: `NTFS`, `exFAT`) |

---

## 3. النقل إلى التخزين — `StorageBackend`

`storage_backend.go` — **مكتمل 100%** (كل دوال `TransferBackend` مُنفَّذة).

| الدالة | التنفيذ |
|---|---|
| `Connect` | يتحقق من وجود المسار (`os.Stat`) |
| `List` | `os.ReadDir` |
| `Stat` | `os.Stat` → `RemoteEntry` |
| `Mkdir` | `os.MkdirAll` (ينشئ الآباء تلقائياً) |
| `Put` | `os.Open` + `io.CopyBuffer` بمخزن 4MB |
| `PutStream` | `io.CopyBuffer` مباشر — **بلا ملف temp** ✅ |
| `Delete` | `os.Remove` |
| `Rename` | `os.Rename` |
| `Close` | لا موارد معلّقة |

> **`PutStream` هنا هو المرجع الصحيح** (بلا spool). Android يفشل في هذا — لأن
> Shell COM **يطلب مسار ملف** بينما Win32 يكتب في مجرى مفتوح.

---

## 4. Streaming والمخزن

```go
const defaultBufferSize = 4 * 1024 * 1024   // 4 MB
```

`TransferConfig.BufferSize` قابل للتعديل. الكود يستخدم `io.CopyBuffer` — أي
**مخزن واحد مُعاد استخدامه**، لا تخصيص لكل قطعة.

> **مطابق للطلب §12 و§28:** لا يُقرأ الملف كاملاً في RAM، المخزن محدود، الزمن
> الحقي بالـ bytes (`OnProgress(transferred int64)`).

---

## 5. أمان الملفات والنظام

| المبدأ | التطبيق |
|---|---|
| لا تجاوز UAC | يُشغَّل كخدمة أو Console بـ `-debug` |
| لا تعطيل Defender | غير مستخدم إطلاقاً |
| لا registry hacks | غير مستخدم للصلاحيات |
| صلاحيات طبيعية | `sc.exe` بصلاحيات النظام/المستخدم |
| مسارات مُتحقَّقة | `mediaPathAllowed` (allowlist) في السيرفر |
| لا حذف في الفحص | مبدأ ADR-010 |

**الطلب §16-17:** الوكيل له قدرة محلية عالية، لكن **API = وظائف محددة**،
**Authorization = يتحكم**، **Audit = يسجّل**. راجع [SECURITY.md](SECURITY.md).

---

## 6. مصادر (Microsoft الرسمية)

- [File Management Functions](https://learn.microsoft.com/en-us/windows/win32/fileio/file-management-functions)
- [GetDriveTypeW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getdrivetypew)
- [GetDiskFreeSpaceExW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getdiskfreespaceexw)
- [GetVolumeInformationW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getvolumeinformationw)
- [Windows File Systems](https://learn.microsoft.com/en-us/windows/win32/fileio/file-systems)
