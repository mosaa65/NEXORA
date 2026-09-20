# NEXORA Agent — Transfer Engine

> **الحالة:** المحرك v2 (`engine.go` + `worker.go` + `scheduler.go`) قائم ويعمل.
> هذا الملف يوثّق ما هو موجود، ويحدد الفجوات.

---

## 1. البنية

```text
TransferEngine
    ├── Scheduler          (scheduler.go)     — توزيع عادل، per-device
    ├── Worker pool        (worker.go)        — تنفيذ ملف واحد
    ├── Config             (engine.go)        — buffer, retries, verify
    └── Jobs               (models_v2.go)     — TransferJobV2, TransferFile
```

### الإعدادات الافتراضية

```go
func DefaultTransferConfig() TransferConfig {
    return TransferConfig{
        BufferSize:           defaultBufferSize,      // 4 MB
        MaxRetries:           3,
        RetryInitialDelay:    500 * time.Millisecond,
        RetryMaxDelay:        4 * time.Second,
        VerifyMode:           VerifySize,             // سريع (افتراضي)
        PerDeviceConcurrency: 1,                      // منع تضارب I/O
    }
}
```

| الإعداد | القيمة | السبب |
|---|---|---|
| `BufferSize` | 4 MB | توازن ذاكرة/إنتاجية (الأصل §33) |
| `MaxRetries` | 3 | يحتمل انقطاعاً عابراً |
| `RetryInitialDelay` → `RetryMaxDelay` | 500ms → 4s | backoff تدريجي |
| `VerifyMode` | `size` | SHA-256 لمئات GB مكلف |
| `PerDeviceConcurrency` | **1** | **منع تضارب USB الواحد** |

> **مطابق للطلب §28:** *«لا تستخدم concurrency مفرطاً يقتل سرعة القرص أو MTP»*.
> القيمة 1 لكل جهاز، وقابلة للتغيير في `TransferConfig`.

---

## 2. مراحل المهمة (Phases)

```go
PhasePreparing           // تحقق من المصدر، تجهيز
PhaseConnecting          // backend.Connect()
PhaseCheckingDestination // Mkdir + فحص الوجهة
PhaseCopying             // النقل الفعلي
PhaseRetrying            // فشل قابل للإعادة
PhaseCompleted
PhaseFailed
PhaseCancelled
```

الواجهة تُظهر المرحلة الحالية — الطلب §20 يطلب هذا صراحة.

---

## 3. النقل — Streaming بلا قراءة كاملة

```go
// StorageBackend.Put (المرجع الصحيح)
sourceFile, _ := os.Open(source)
io.CopyBuffer(dst, sourceFile, buffer)     // مخزن واحد مُعاد استخدامه
```

**الضمانات:**
- ✅ لا قراءة كاملة في RAM
- ✅ `int64` لكل الأحجام (64-bit)
- ✅ `OnProgress(transferred int64)` — **بالبايت لا بالنسبة**
- ✅ الاستمرار عند الملفات الكبيرة بلا حدود

**الاستثناء الموثّق:** `AndroidBackend.PutStream` يمرّ بملف temp — فجوة مُدرَجة.

---

## 4. Progress

```go
type PutOptions struct {
    BufferSize   int64
    ResumeOffset int64
    Overwrite    bool
    OnProgress   func(transferred int64)   // تراكمي، بالبايت
}
```

**آلية الحساب** (في `service.go` / `progress.go`):

```text
BytesTransferred = مجموع المكتمل + المُبلَّغ حالياً
SpeedBps         = Δbytes / Δt
SpeedMBps        = SpeedBps / (1024*1024)
ETASeconds       = (TotalBytes - Transferred) / SpeedBps
Progress (%)     = Transferred / TotalBytes * 100
```

> **مطابق للطلب §12:** Progress حقي بالـ bytes لا بالنسبة فقط، وسرعة وETA محسوبان.

---

## 5. Pause / Resume / Cancel

### Cancel — مُنفَّذ

```go
type TransferJob struct {
    // ...
    cancelFunc context.CancelFunc
}
```

الإلغاء عبر `context.WithCancel` → يُوقف الـ worker ويُنظّف (يحذف `.part`).

### Pause/Resume — جزئي

`TransferPhase` لا تحتوي `paused`، لكن ADR-011 يوثّق Pause/Resume **للفحص**
(`/api/scan/pause`) لا للنقل. للنقل: `ResumeOffset` **موجود في البنية**.

> **قيد موثّق:** Resume غير مُنفَّذ للنقل. الحقل موجود استعداداً. راجع §8.

### Resume — التصميم المطلوب (الطلب §13)

الطلب يطلب عزله في `TransferStrategy`:

```go
// مستقبلاً — تصميم فقط، لا تنفيذ الآن
type TransferStrategy interface {
    CanResume() bool
    Resume(dst string, offset int64) error
}
```

**الحقيقة:** `CopyHere` (Android) **لا يدعم Resume** — لا واجهة لإضافة لمقطع.
`StorageBackend` يمكنه Resume (`os.OpenFile` بـ `O_APPEND` + `Seek`).
`IOSBackend` (AFC) يدعم الكتابة عند موقع.

**لهذا يجب أن يكون Resume لكل-provider لا عاماً.**

---

## 6. Retry و Verification

### Retry

```go
MaxRetries:        3
RetryInitialDelay: 500ms
RetryMaxDelay:     4s
```

يُعاد المحاولة فقط عند `TransferError.Retryable == true`. الأخطاء الدائمة
(صلاحيات، مساحة) **لا تُعاد**.

### Verification

```go
type VerifyMode string
const (
    VerifySize   VerifyMode = "size"   // remote size == source size
    VerifyStrict VerifyMode = "strict" // SHA-256 كامل
)
```

| الوضع | التكلفة | الاستخدام |
|---|---|---|
| `size` | O(1) | افتراضي، كافٍ للنقل العادي |
| `strict` | O(n) قراءة كاملة | عندما تكون السلامة حرجة |

---

## 7. الأخطاء المُصنَّفة

```go
type TransferError struct {
    Code       string   // "SOURCE_NOT_FOUND"
    Message    string
    Retryable  bool     // هل يُعاد؟
    DeviceLost bool     // هل فُقد الجهاز؟
    Cause      error
}
```

الأكواد الموجودة: `SOURCE_NOT_FOUND`, `VERIFY_FAILED`, `USER_CANCELLED`,
`AFC_IO_ERROR`.

**الطلب §29 يطلب أكواداً أكثر** — مُدرَجة في §8 كفجوة.

---

## 8. الفجوات (نطاق العمل)

| # | الفجوة | الطلب | الحالة |
|---|---|
| 1 | أكواد خطأ ناقصة (`INSUFFICIENT_SPACE`, `MTP_ERROR`, `DEVICE_DISCONNECTED`) | §29 | تُضاف |
| 2 | `Pause`/`Resume` للنقل | §13 | `ResumeOffset` موجود، التنفيذ لا |
| 3 | `TransferStrategy` ليست معرَّفة | §13 | تُصمَّم |
| 4 | لا فحص مساحة قبل البدء | §12 | يُضاف |
| 5 | Android `PutStream` يمرّ بملف temp | §12 | **غير قابل للإصلاح عبر Shell COM** |
| 6 | لا كشف فصل أثناء النقل (Android) | §12 | يُضاف |
| 7 | Logging منظم (structured) | §30 | يُتحقق |

---

## 9. مصادر

- [Win32 File Management](https://learn.microsoft.com/en-us/windows/win32/fileio/file-management-functions)
- [Folder.CopyHere](https://learn.microsoft.com/en-us/windows/win32/shell/folder-copyhere) — لا يدعم الإضافة (Resume)
- [io.CopyBuffer (Go)](https://pkg.go.dev/io#CopyBuffer)
