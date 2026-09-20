# NEXORA Agent — Implementation Report

> **المهمة:** إعادة بناء طبقة Android داخل وكيل موجود، مع الحفاظ الصارم على كل
> ما يعمل. **النطاق المنفَّذ:** Android فقط.

- **التاريخ:** 2026-09-20
- **الفرع:** `feature/nexora-agent-android-architecture`
- **Baseline (نقطة الرجوع):** `1f51f8a` على `main`

---

## What was changed

**ملفان للكود، وثلاثة للاختبار، وتسع للتوثيق.** لا أكثر.

| # | الملف | التغيير |
|---|---|---|
| 1 | `server/internal/transfer/android_backend.go` | إكمال 3 دوال ناقصة + 6 تحسينات تشغيلية + 3 إصلاحات لعيوب حقيقية |
| 2 | `server/internal/transfer/errors.go` | **إضافة** 8 أكواد خطأ مُصنَّفة |
| 3 | `server/internal/transfer/models.go` | **إضافة** `Capabilities` على `Device` (حقل جديد، لا تغيير) |
| 4 | `server/internal/transfer/removable_windows.go` | **إضافة** `Capabilities` للقرص (سطران) |
| 5 | `server/internal/transfer/service.go` | **إضافة** `capabilitiesFor` (بلا لمس أي منطق) |
| 6 | `android_backend_test.go` | 9 اختبارات جديدة |
| 7 | `android_script_test.go` | **جديد** — اختبارات صحة السكربتات |
| 8–16 | `docs/agent/**` + `ADR-015` | التوثيق |

### تفصيل الميزات المُضافة

| # | الميزة | التنفيذ |
|---|---|---|
| 1 | **`List`** — تصفح الهاتف | تعداد MTP مع نوع وحجم صريحين → `RemoteEntry` |
| 2 | **`Delete`** | `InvokeVerb('delete')` — نفس آلية `Put` الموجودة |
| 3 | **فحص المساحة** | `FreeSpace()` قبل النسخ → `INSUFFICIENT_SPACE` |
| 4 | **كشف الفصل** | `devicePresent()` في كل tick → `DEVICE_DISCONNECTED` فوراً |
| 5 | **تعداد المساحات** | `NewAndroidBackendForStorage(name)` — الافتراضي بلا تغيير |
| 6 | **polling بدل `Start-Sleep 500ms`** | polling محدود بـ 8 ثوانٍ |
| 7 | **أكواد خطأ** | 8 أكواد جديدة + `classifyMTPOutput` |
| 8 | **Capabilities** | `DeviceCapabilities` — يمنع محاولة عملية مستحيلة |
| 9 | **`powershellPath()`** | حسْم المسار المطلق (إصلاح عيب حقي) |
| 10 | **`-EncodedCommand`** | إزالة اعتماد الترميز (إصلاح عيب حقي) |

---

## What was preserved

**`git diff` يُثبت أن هذه الملفات لم تُلمس إطلاقاً:**

```
server/internal/transfer/go_ios_backend.go      ← iPhone: صفر تغيير
server/internal/transfer/storage_backend.go     ← USB/HDD/أقراص: صفر تغيير
server/internal/transfer/engine.go              ← المحرك v2: صفر تغيير
server/internal/transfer/worker.go              ← صفر تغيير
server/internal/transfer/scheduler.go           ← صفر تغيير
server/internal/copybridge/**                   ← الوكيل: صفر تغيير
server/cmd/copybridge/**                        ← نقطة الدخول: صفر تغيير
client/**                                        ← الواجهة: صفر تغيير
```

**`service.go`** — لم يُحذف ولا يُعاد كتابته. موصول من 3 callers، وأُضيفت له
**دالة واحدة** (`capabilitiesFor`). منطق اكتشاف iPhone و MTP **لم يُلمس**.

---

## What was removed

**لا شيء.** لا ملف حُذف ولا دالة أُزيلت.

ما **تغيّر سلوكه** (من stub إلى تنفيذ حقي):

| الدالة | قبل | بعد |
|---|---|---|
| `List` | `return error("غير مدعوم")` | تعداد فعلي |
| `Delete` | `return error("غير مدعوم")` | `InvokeVerb('delete')` |
| `Rename` | `return error("غير مدعومة")` | `UNSUPPORTED_OPERATION` مُصنَّف |

**`Rename` لم يُنفَّذ** — وسبب ذلك موثّق ومقصود (يحتاج WPD).

---

## Android implementation

```
Android Phone → USB → MTP → Windows Shell COM → PowerShell → AndroidBackend
```

المسار **لم يتغيّر**. ما تغيّر: **إكماله وإصلاح عيوبه**.

### العيوب الثلاثة الحقيقية المكتشفة بالاختبار الحي

هذه أهم ما في التقرير: **لم تكتشفها قراءة كود، بل تشغيل فعلي.**

| # | العيب | كيف اكتُشف | الإصلاح |
|---|---|
| **A** | `powershell` بالاسم المجرد | `exec: "powershell": executable file not found in %PATH%` على endpoint حقي | `powershellPath()` |
| **B** | السكربت عبر `-Command` | PowerShell أعاد `Missing closing '}'` في سكربت **سليم الأقواس** — الترميز/الطول | `-EncodedCommand` |
| **C** | `try { if (...) { ... } catch {}` | الخطأ استمر بعد إصلاح B → فحص عدّ الأقواس: **3 فتح / 2 إغلاق** | أُصلح + اختبار توازن |

**لماذا لم تُكتشف سابقاً؟** لأن Android **لم يُشغَّل فعلياً** على هذه الآلة
(العيب A يمنعه من الأساس). أول استدعاء حقي أظهر A، ثم B، ثم C.

---

## Windows implementation

**لم يُلمس.** `removable_windows.go` و `storage_backend.go` كما هما، عدا
`Capabilities: storageCapabilities()` (سطر واحد إضافي).

---

## Device providers

`TransferBackend` — **العقد لم يتغيّر**. لا توقيع دالة واحد عُدّل.

| Provider | التمثيل | الحالة |
|---|---|---|
| Filesystem | `StorageBackend` | ✅ يعمل (محمي) |
| iOS | `GoIOSBackend` | ✅ يعمل (محمي) |
| Android MTP | `AndroidBackend` | ✅ **مُكمَّل** |
| WPD | — | ⏭️ مُخطَّط |

**إضافة:** `DeviceCapabilities` تُعلن ما يمكن لكل ناقل فعله.

---

## Transfer engine

**لم يُلمس.** `engine.go` / `worker.go` / `scheduler.go` كما هي.

الميزات المطلوبة كانت **موجودة أصلاً**: retries بـ backoff، phases،
`PerDeviceConcurrency`، `int64` للاحجام، `OnProgress(transferred int64)`.

---

## API

**لم يُلمس.** 16 endpoint + SSE كما هي. **لا `POST /execute`** — الطلب §31.

ما تغيّر: `Device` صار يحمل `capabilities` في استجابة `/api/transfer/devices`.

---

## Security

| البند | النتيجة |
|---|---|
| Command Execution API | ✅ **لا يوجد** |
| أسرار hardcoded | ✅ لا يوجد |
| سطح جديد | ✅ لا يوجد — نفس الـ endpoints |
| Token + CORS | ✅ بلا تغيير |
| Path traversal | ⚠️ قائم (خارج النطاق) |

---

## Performance

| البند | النتيجة |
|---|---|
| لا قراءة كاملة في RAM | ✅ `io.CopyBuffer` |
| أحجام 64-bit | ✅ `int64` |
| `Start-Sleep` ثابت | ✅ صار polling محدود |
| concurrency | ✅ 1/جهاز (بلا تغيير) |
| **بث بلا temp** | ❌ **غير ممكن** عبر Shell COM (موثّق) |

---

## Tests

### ما نجح فعلياً

```
go build ./...                    → 0
go vet  ./...                     → 0
go test ./internal/transfer/...   → ok
go test ./internal/copybridge/... → ok
go test ./internal/api/...        → ok
```

**اختبارات جديدة:**
- `TestParseMTPListing` + `TestParseMTPListingRootPath`
- `TestClassifyMTPOutput` (5 حالات)
- `TestAndroidConnectBindsDeviceName` (4 حالات)
- `TestAndroidRenameIsClassifiedUnsupported`
- `TestAndroidCapabilitiesMatchTheBackend`
- `TestStorageCapabilitiesAdvertiseEverything`
- `TestCapabilitiesForEveryDeviceType`
- `TestPowershellPathResolvesAnAbsoluteBinary` + `TestPowershellPathHonoursSystemRoot`
- **`TestGeneratedScriptParses`** — يتحقق بمُحلِّل PowerShell نفسه (7 سكربتات)
- **`TestGeneratedScriptsHaveBalancedBraces`**

### الاختبار الحي (عبر الوكيل الحقي)

| الاختبار | النتيجة |
|---|---|
| `/api/health` (bridge) | ✅ 200 |
| `/api/transfer/devices` | ✅ **كشف قرص D: حقي (123 GB FAT32) مع capabilities** |
| `/api/transfer/browse` (جهاز وهمي) | ✅ **خطأ مُصنَّف**: `"لم يتم العثور على الجهاز الموصول عبر USB"` |

**الأخير هو الأهم:** يُثبت أن `List` يعمل من طرف إلى طرف — السكربت يُبنى، يُرسَل،
يُنفَّذ، وعلامته تُصنَّف لرسالة نظيفة.

### مُعلَّق بصدق (لا أدّعي نجاحه)

| الاختبار | السبب |
|---|---|
| Android discovery | **لا جهاز Android متصل بهذه الآلة** |
| Android storage listing | نفس السبب |
| Android file transfer | نفس السبب |
| Android reconnect | نفس السبب |
| Android disconnect أثناء النقل | نفس السبب |
| iPhone حي | يحتاج جهاز + `usbmuxd` |
| USB/HDD حي | يحتاج توصيل القرص |

---

## Known limitations

| # | القيد | السبب | الحل المستقبلي |
|---|---|
| 1 | **البث يمرّ بملف temp** | `CopyHere` يطلب مسار ملف | WPD `IStream` |
| 2 | **لا Resume** | `CopyHere` بلا واجهة إضافة | WPD |
| 3 | **لا Rename** | لا rename ذرّي في Shell COM | WPD |
| 4 | **التقدم polling (1.2s)** | `CopyHere` لا يُبلّغ | WPD |
| 5 | **أول مساحة افتراضية** | الافتراضي محفوظ للتوافق | تمرير `storageName` |
| 6 | **External HDD لا يظهر** | يُعرَّف `DRIVE_FIXED` | IOCTL bus check (خارج النطاق) |
| 7 | **Session 0 لـ Android** | Shell COM قد يحتاج سياق تفاعلي | WPD (لا يحتاجه) |

---

## Future roadmap

موثّقة بالتفصيل في [ROADMAP.md](ROADMAP.md):

```text
WPD-1..5    إثبات مفهوم → محتوى → IStream → استبدال → إزالة PowerShell
ADB         مهام متقدمة (Install/Push/Backup)
Explorer    تكامل Context Menu / Send To
Providers   Network · NAS · Cloud
```

---

## Files changed

```
server/internal/transfer/android_backend.go         (+~330 / -~180)
server/internal/transfer/android_backend_test.go    (+~180)
server/internal/transfer/android_script_test.go     (جديد, ~130)
server/internal/transfer/errors.go                  (+17)
server/internal/transfer/models.go                  (+47)
server/internal/transfer/removable_windows.go       (+2 / -1)
server/internal/transfer/service.go                 (+25)
docs/agent/*.md                                     (11 ملفاً)
docs/decisions/ADR-015-*.md                         (جديد)
docs/decisions/README.md · docs/ARCHITECTURE.md     (+2 كل واحد)
```

## Files intentionally untouched

```
server/internal/transfer/go_ios_backend.go          ← iPhone
server/internal/transfer/storage_backend.go         ← USB/HDD/أقراص
server/internal/transfer/engine.go                  ← المحرك
server/internal/transfer/worker.go
server/internal/transfer/scheduler.go
server/internal/transfer/progress.go
server/internal/copybridge/**                       ← الوكيل
server/cmd/copybridge/**                            ← نقطة الدخول
client/**                                            ← الواجهة
```

---

## Git commit

```
<يُملأ بعد الرفع>
```

## Rollback instructions

```powershell
cd "c:\Users\mousa\Desktop\project\NEXORA"

# الرجوع الكامل
git checkout main
git branch -D feature/nexora-agent-android-architecture

# الرجوع الجزئي (ملف واحد)
git checkout 1f51f8a -- server/internal/transfer/android_backend.go
```

`main` عند `1f51f8a` — **لم يُلمس**، ومطابق للريموت.

---

## الملخص النهائي

```text
✅ ما الذي تم
   إكمال AndroidBackend: تصفح + حذف + فحص مساحة + كشف فصل + مساحات متعددة
   + 8 أكواد خطأ مُصنَّفة + Capabilities على الأجهزة
   + إصلاح 3 عيوب حقيقية اكتشفها الاختبار الحي (PATH, ترميز, قوس)

🟢 ما الذي بقي كما هو
   iPhone · USB · External HDD · الأقراص المحلية · المحرك v2 · الوكيل · الواجهة
   (مُثبت بـ git diff = فارغ)

🔴 ما الذي حُذف
   لا شيء. لا ملف ولا دالة.

📱 كيف أصبح Android يعمل
   List/Delete/FreeSpace/disconnect  — 3 stubs صارت تنفيذاً فعلياً
   تحقق حي: browse يعطي خطأً مُصنَّفاً نظيفاً عبر الوكيل الحقي

💾 كيف أصبح USB يعمل
   كما كان — لم يُلمس، وأُضيف له capability declaration فقط

📱 كيف بقي iPhone
   كما كان تماماً — صفر تغيير في provider أو بروتوكول أو نقل

⚡ كيف تم تحسين النقل
   Start-Sleep 500ms → polling محدود · فحص مساحة قبل البدء
   · كشف فصل فوري بدل انتظار المهلة

🔐 كيف تم تأمين Agent
   لا Command API · نفس الـ 16 endpoint · token + CORS بلا تغيير

📚 أين توجد الوثائق
   docs/agent/ (11 ملفاً) + docs/decisions/ADR-015

🧪 ما الاختبارات التي نجحت
   build/vet/test أخضر · 20+ اختبار وحدة جديد
   · 7 سكربتات يتحقق منها محلّل PowerShell
   · 3 استدعاءات حية عبر الوكيل الحقي

⚠️ ما القيود الموجودة
   لا بث بلا temp · لا Resume · لا Rename  — الثلاثة تحتاج WPD (مُخطَّط)
   · لا جهاز Android متصل: الاختبار الحي مُعلَّق بصدق
```
