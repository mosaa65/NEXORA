# NEXORA Agent — Migration Baseline

> **الغرض:** تسجيل حالة المشروع **قبل** أي تغيير في طبقة الأجهزة، وتحديد ما هو
> محمي، وما سيُغيَّر، وكيف نرجع. هذا الملف هو نقطة الرجوع المرجعية.

- **التاريخ:** 2026-09-20
- **Checkpoint (نقطة الرجوع):** `1f51f8a` على `main` — مطابق لـ `origin/main`
- **الفرع الجديد:** `feature/nexora-agent-android-architecture`
- **الحالة عند التسجيل:** شجرة عمل نظيفة، `go build`/`go vet` = 0، `go test ./internal/transfer/... -short` = `ok 20.1s`

---

## 1. لغة المشروع ونقاط التشغيل

| العنصر | الواقع الفعلي |
|---|---|
| لغة الخادم والوكيل | **Go** (`server/`) |
| لغة الواجهة | **React + Vite** (`client/`) |
| قاعدة البيانات | **PostgreSQL** (Docker، منفذ `15432`) |
| فهرس البحث | **Meilisearch** (Docker، منفذ `7700`) |
| Cache | **Redis** (Docker، منفذ `6379`) |
| نقطة تشغيل السيرفر المركزي | `server/cmd/api` → `nexora-api.exe`، **يُشغَّل من داخل `server/`** ليقرأ `.env` الصحيح |
| نقطة تشغيل الوكيل المحلي | `server/cmd/copybridge` → منفذ `127.0.0.1:32145` |
| الخدمات | `compose.yml` في الجذر: `docker compose up -d` |

---

## 2. أهم اكتشاف: "NEXORA Agent" **موجود بالفعل** باسم `copybridge`

المطلوب في الطلب (§1) هو "مكوّن مستقل يعمل محليًا على جهاز Windows" — وهذا
**مُنفَّذ فعلاً** وموثّق في ADR-008:

```
server/cmd/copybridge/main.go            ← نقطة الدخول: Windows Service أو Console (-debug)
server/cmd/copybridge/service_windows.go ← ربط golang.org/x/sys/windows/svc
server/cmd/copybridge/service_other.go    ← بناء غير ويندوز
server/internal/copybridge/               ← 6 ملفات
```

| القدرة المطلوبة في الطلب | الحالة الفعلية في `copybridge` |
|---|---|
| استماع محلي فقط | **`127.0.0.1:32145`** ✅ |
| مصادقة | `NEXORA_COPY_BRIDGE_TOKEN` (اختياري) + `tokenMatches` **constant-time** ✅ |
| CORS مُقيَّد | `corsAllow` يرفض المناشئ العامة بـ `403` ✅ |
| بث الأحداث | **SSE** على `/api/transfer/events` ✅ |
| اكتشاف + نسخ | مندوب إلى `internal/transfer` ✅ |
| Windows Service رسمي | `svc.Handler` عبر `golang.org/x/sys/windows/svc` ✅ |
| تثبيت/إزالة | ADR-008 (سكريبت `sc.exe` + أوامر مدمجة) ✅ |

**الخلاصة:** لا نبني Agent من الصفر. **نوسّع وكيلاً يعمل.**

---

## 3. البنية الفعلية لطبقة النقل (المرجع للحقيقة)

### 3.1 التجريد موجود مسبقاً

`server/internal/transfer/backend.go` (69 سطراً) يعرّف **العقد الموحّد**:

```go
type TransferBackend interface {
    Connect(ctx, Device, TransferDestination) error
    List(ctx, path) ([]RemoteEntry, error)
    Stat(ctx, path) (RemoteEntry, error)
    Mkdir(ctx, path) error
    Put(ctx, source, destination, PutOptions) error
    PutStream(ctx, reader, size, destination, PutOptions) error
    Delete(ctx, path) error
    Rename(ctx, oldPath, newPath) error
    Close() error
}
```

مع أنواع مشتركة: `TransferDestination` (فيه `RemotePath` موحّد)، `RemoteEntry`،
`PutOptions` (**فيه `ResumeOffset` بالفعل**، و`OnProgress func(transferred int64)`).

> **هذا هو `DeviceProvider` العملي الموجود.** لا نبتكر طبقة موازية — نُوسّعه.

### 3.2 المحرك v2 موجود ومتقدّم

`engine.go` (136 سطراً) + `models_v2.go` + `worker.go` (282 سطراً) + `scheduler.go`:

```go
type TransferConfig struct {
    BufferSize           int64          // 4 MB افتراضي
    MaxRetries           int            // 3
    RetryInitialDelay    time.Duration  // 500ms
    RetryMaxDelay        time.Duration  // 4s
    VerifyMode           VerifyMode     // size | strict (SHA-256)
    PerDeviceConcurrency int            // 1 لكل جهاز
}
```

`TransferJobV2` يحمل: `Files []TransferFile` (متعدد الملفات)، `TotalBytes`/
`TransferredBytes` (**`int64` — 64-bit، مطابق §12**)، `SpeedBps`، `ETASeconds`،
`CurrentFileIndex`، وحالات دقيقة `TransferPhase` (`preparing` → `connecting` →
`checking_destination` → `copying` → `retrying` → `completed`/`failed`/`cancelled`).

### 3.3 أخطاء مُصنَّفة موجودة

`errors.go` يعرّف `TransferError{Code, Message, Retryable, DeviceLost, Cause}`
مع أكواد مثل `SOURCE_NOT_FOUND`, `VERIFY_FAILED`, `USER_CANCELLED`,
`AFC_IO_ERROR`.

### 3.4 `service.go` **ليس كوداً ميتاً**

`service.go` (1071 سطراً) هو **الواجهة العامة** أمام المحرك — وليس نسخة قديمة.
أُثبت أنه **موصول من 3 أماكن**:

| Caller | الملف | السطر |
|---|---|---|
| السيرفر المركزي | `server/internal/app.go` | 91 |
| الوكيل المحلي | `server/internal/copybridge/service.go` | 70 |
| أداة اختبار | `server/cmd/transfertest/main.go` | 45 |

وهو نفسه يستدعي المحرك v2 (`newV2Engine`) في `service.go:142`.

> **⚠️ قرار موثّق:** `service.go` **لا يُحذف ولا يُعاد كتابته** في هذه المهمة.
> حذفه يكسر السيرفر المركزي والوكيل وأداة الاختبار معاً.

---

## 4. Backends الفعلية —ها يعمل وماذا ينقص

| Backend | الملف | الحالة | `List` | `Delete` | `Rename` |
|---|---|
| **Storage** (USB/HDD/أقراص) | `storage_backend.go` (174) | **مكتمل** | ✅ `os.ReadDir` | ✅ | ✅ `os.Rename` |
| **iOS** (iPhone) | `go_ios_backend.go` (281) | **مكتمل ومُختبر** | ✅ | — | — |
| **Android** (MTP) | `android_backend.go` (282) | **ناقص** 🔴 | ❌ | ❌ | ❌ |

### Android الحالي — الطريقة الفعلية

**لا يوجد أي ملف WPD في المشروع.** الطريقة الحالية:

| الطبقة | التقنية الفعلية |
|---|---|
| الاكتشاف | PowerShell `Shell.Application` + `NameSpace(17)` (مجلد "أجهزة الكمبيوتر") |
| تحديد الجهاز | مطابقة `$item.Name` ثم `$item.Type -like '*Portable*'` أو مسار `::` |
| إنشاء مجلد | `$folder.NewFolder(name)` + `Start-Sleep 500ms` + إعادة المسح |
| **النسخ** | **`$finalTarget.CopyHere(source, 16)` — غير متزامن، بلا قيمة راجعة** |
| **التقدم** | **Polling حجم الملف كل `1200ms`** (`waitForMTPFile`) |
| المهلة | `transferTimeout(wantSize)` |
| `PutStream` | يكتب الملف كاملاً في ملف temp أولاً ثم `CopyHere` (**يخالف §12**) |

### الفجوات الحقيقية في Android (نطاق العمل)

```text
1. List()      → يرجع خطأ "غير مدعوم"   → لا يمكن تصفح الهاتف
2. Delete()    → يرجع خطأ "غير مدعوم"
3. Rename()    → يرجع خطأ "غير مدعوم"
4. PutStream() → يمرّ بملف temp على القرص (spool) بدل البث المباشر
5. لا كشف فصل الجهاز (disconnect) بشكل صريح
6. لا تحقق من المساحة قبل البدء
7. الأخطاء نصية عربية بلا أكواد مُصنَّفة
```

---

## 5. ما سيُغيَّر في هذه المهمة

### داخل النطاق — يُلمس
| الملف/المسار | نوع التغيير |
|---|---|
| `server/internal/transfer/android_backend.go` | إكمال `List`/`Delete`/`Rename`، إزالة spool من `PutStream`، كشف الفصل، استشارة المساحة |
| `server/internal/transfer/android_backend_test.go` | اختبارات جديدة |
| `docs/agent/**` (11 ملفاً) | إنشاء |
| `docs/decisions/ADR-015-*.md` | إنشاء (قرار معمارية الوكيل والأندرويد) |

### خارج النطاق — **محمي، لا يُلمس**
| الملف | السبب |
|---|---|
| `server/internal/transfer/go_ios_backend.go` | **iPhone يعمل — قاعدة غير قابلة للتفاوض (الطلب §24)** |
| `server/internal/transfer/removable_windows.go` | USB/الأقراص تعمل (Win32 مباشر، <1ms) |
| `server/internal/transfer/storage_backend.go` | مكتمل ومستخدم لـ USB/HDD/الأقراص |
| `server/internal/transfer/service.go` | الواجهة الموصولة من 3 أماكن |
| `server/internal/transfer/engine.go` + `worker.go` + `scheduler.go` | المحرك v2 يعمل |
| `server/internal/copybridge/**` | الوكيل يعمل (ADR-008) |
| `server/cmd/copybridge/**` | نقطة دخول الخدمة |
| `client/**` | لا إعادة تصميم واجهة الآن (الطلب §21) |

---

## 6. المخاطر

| # | الخطر | الدرجة | التخفيف |
|---|---|
| 1 | كسر iPhone بتغيير مشترك | **عالية** | `go_ios_backend.go` لا يُلمس؛ اختبار عدم-تراجع إلزامي |
| 2 | كسر USB/الأقراص | **عالية** | `storage_backend.go` و`removable_windows.go` لا تُلمس |
| 3 | تغيير عقد `TransferBackend` يكسر 3 backends | **متوسطة** | العقد **يُوسَّع لا يُغيَّر**؛ إضافات اختيارية فقط |
| 4 | `CopyHere` غير متزامن بلا إشارة إكمال | **متوسطة** | الاستمرار في polling الحجم (مُثبت)، مع تحسين الأخطاء |
| 5 | لا جهاز Android متصل للاختبار الحي | **عالية** | اختبارات وحدة للـ parsing والمسارات؛ الاختبار الحي يُوثَّق كـ"مُعلَّق" بصدق |
| 6 | بناء WPD من الصفر = مخاطرة عالية ووقت طويل | **عالية** | **قرار المالك: WPD مُخطَّط لا مُنفَّذ (خيار أ)** |
| 7 | `go fmt ./...` على ويندوز يلوّث 25 ملفاً بـ CRLF | **متوسطة (بيئية)** | **لا يُشغَّل**؛ التحقق بـ `go vet` |

---

## 7. طريقة الرجوع (Rollback)

### أ. الرجوع الكامل
```powershell
cd "c:\Users\mousa\Desktop\project\NEXORA"
git checkout main
git branch -D feature/nexora-agent-android-architecture
```
`main` عند `1f51f8a` — مطابق للريموت. لا شيء يُفقد.

### ب. الرجوع الجزئي (ملف واحد)
```powershell
git checkout 1f51f8a -- server/internal/transfer/android_backend.go
```

### ج. إثبات أن الـ checkpoint سليم قبل البدء
```powershell
git rev-parse HEAD          # يجب = 1f51f8a قبل أول تعديل
git status --porcelain      # يجب أن يكون فارغاً
```

### د. التحقق من عدم التراجع بعد أي تغيير
```powershell
cd server
& "C:\Users\mousa\go\bin\go.exe" build ./...
& "C:\Users\mousa\go\bin\go.exe" vet ./...
& "C:\Users\mousa\go\bin\go.exe" test ./internal/transfer/... -short
```

---

## 8. قيد البيئة (يجب معرفته لكل من يعمل هنا)

| الأمر | المسار الصحيح على هذه الآلة |
|---|---|
| git | `C:\Program Files\Git\cmd\git.exe` (ليس في PATH) |
| go | `C:\Users\mousa\go\bin\go.exe` (ليس في PATH) |
| npm/node | `C:\Program Files\nodejs\npm.cmd` (ليس في PATH) |
| الخدمات | `docker compose up -d` من الجذر |
| السيرفر | `cd server; ..\nexora-api.exe` |

**⚠️ لا تشغّل `go fmt ./...` على ويندوز** — يحوّل 25 ملفاً من LF إلى CRLF بلا
تغيير منطقي (مُتحقَّق: النص متطابق حرفياً بعد توحيد نهايات الأسطر).

---

## 9. المصادر التقنية (الطلب §34)

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices) — مُخطَّط (خيار أ)
- [WPD Application Programming Interface](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/wpd-application-programming-interface)
- [Win32 File Management](https://learn.microsoft.com/en-us/windows/win32/fileio/file-management-functions)
- [Windows Shell Extensions](https://learn.microsoft.com/en-us/windows/win32/shell/shell-exts)
- مكتبة Go الحالية لـ iOS: `github.com/danielpaulus/go-ios` (مثبتة في `server/go.mod`، **لا تُلمس**)
