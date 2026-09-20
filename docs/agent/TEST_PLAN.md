# NEXORA Agent — Test Plan

> **المبدأ (الطلب §26):** النجاح ليس "الـ build مرّ". النجاح **اختبار حي**.
> **القيد الصادق:** لا يوجد جهاز Android متصل في بيئة التطوير — الاختبارات
> الحية للأندرويد **مُعلَّقة** وتُوثَّق كذلك.

---

## 1. اختبارات عدم التراجع (إلزامية — الطلب §27)

يجب أن تُثبت أن ما يعمل **ما زال يعمل** بعد كل تغيير:

| # | الهدف | الاختبار | الحالة |
|---|---|
| R1 | **iPhone** | `go test ./internal/transfer/... -short` (اختبارات AFC) | ✅ تمر |
| R2 | **USB Flash** | `discoverRemovableDrives` — وحدة + حي | ✅ يعمل |
| R3 | **External HDD** | `StorageBackend` كامل | ✅ يعمل |
| R4 | **Windows Drives** | `StorageBackend.Put`/`List`/`Delete` | ✅ يعمل |
| R5 | **Copy Bridge** | `internal/copybridge` tests | ✅ تمر |
| R6 | **المحرك v2** | `engine_test.go` | ✅ تمر |

### الأوامر

```powershell
cd "c:\Users\mousa\Desktop\project\NEXORA\server"
& "C:\Users\mousa\go\bin\go.exe" build ./...
& "C:\Users\mousa\go\bin\go.exe" vet ./...
& "C:\Users\mousa\go\bin\go.exe" test ./internal/transfer/... -short
& "C:\Users\mousa\go\bin\go.exe" test ./internal/copybridge/...
```

**البوابة:** لا يُعلن أي تغيير مكتملاً إن فشل أحد هذه.

---

## 2. اختبارات Android (Phase 7)

### 2.1 اختبارات وحدة (بلا جهاز) — قابلة الآن

| # | الهدف | ماذا تختبر |
|---|---|---|
| A1 | `targetParts` | تقسيم المسارات (`Movies/Season 1` → `['Movies','Season 1']`) |
| A2 | `androidDestinationParts` | فصل dir/base للمسارات بنهايات مختلفة |
| A3 | PowerShell escaping | `'` → `''` في أسماء الأجهزة والملفات |
| A4 | `Connect` — استخراج اسم الجهاز | `mtp_Samsung_A55` → `Samsung A55` |
| A5 | `Connect` — `Device.Name` له الأولوية | كما ينص الكود |
| A6 | بناء السكربت | containing device name, paths, file name |
| A7 | `Stat` — مسار فارغ | لا panic |
| A8 | أكواد `TransferError` | التصنيف الصحيح |

### 2.2 اختبارات حية (تحتاج جهاز) — تُوثَّق لا تُدّعى

| # | السيناريو | الطلب |
|---|---|---|
| A9 | Android discovery | §26 Test 5 |
| A10 | Android storage discovery | §26 Test 5 |
| A11 | Android folder browsing | §35 |
| A12 | Android file transfer | §26 Test 5 |
| A13 | **Android reconnect** | §26 Test 6 |
| A14 | **Android disconnect أثناء النقل** | §26 Test 7 |

> **سياسة الصدق:** لا أدّعي نجاح اختبار لم أُشغّله. تُسجَّل كـ **"مُعلَّقة — تحتاج
> جهاز Android متصل"** في `IMPLEMENTATION_REPORT.md`.

---

## 3. السيناريوهات الكاملة (الطلب §26)

| # | السيناريو | الجهاز المطلوب | الحالة |
|---|---|
| 1 | C: local folder | ✅ متاح | قابل الآن |
| 2 | D: drive | ⚠️ إن وُجد | قابل |
| 3 | USB Flash | ⚠️ يحتاج توصيل | قابل |
| 4 | External HDD | ⚠️ يحتاج توصيل | قابل |
| 5 | Android phone | 🔴 **غير متاح** | مُعلّق |
| 6 | Android reconnect | 🔴 **غير متاح** | مُعلّق |
| 7 | Android disconnect أثناء النقل | 🔴 **غير متاح** | مُعلّق |
| 8 | Large media file | ✅ متاح | قابل (ملف حقي) |
| 9 | Insufficient space | ✅ متاح | قابل (محاكاة) |
| 10 | Duplicate file | ✅ متاح | قابل |
| 11 | Existing folder | ✅ متاح | قابل |
| 12 | Create new folder | ✅ متاح | قابل |
| 13 | Cancel | ✅ متاح | قابل |
| 14 | Pause / Resume | ⚠️ جزئي | مُعلّق (غير مُنفَّذ) |
| 15 | Agent restart | ✅ متاح | قابل |
| 16 | Frontend reconnect | ✅ متاح | قابل |
| 17 | Multiple devices simultaneously | ⚠️ | قابل جزئياً |

---

## 4. اختبارات الأداء (الطلب §28)

| القياس | الهدف | الطريقة |
|---|---|---|
| No full-file-in-RAM | إلزامي | مراجعة كود + قياس RSS أثناء نقل |
| Streaming buffer | 4 MB محدود | `io.CopyBuffer` |
| 64-bit sizes | إلزامي | مراجعة + ملف > 4 GB |
| Accurate progress | بالبايت | مقارنة `OnProgress` بالحجم الفعلي |
| Low idle CPU | ~0% | قياس وقت الخمول |
| High throughput | مقارب للقرص | نقل 10 GB وقياس |
| Bounded concurrency | 1/جهاز | `PerDeviceConcurrency` |

### قياس الملف الكبير

```powershell
# نقل ملف كبير من C: إلى قرص آخر، ومراقبة RSS وCPU
# الأداة: server/cmd/transfertest
```

---

## 5. مسح أمني (الطلب §31) — راجع [SECURITY.md](SECURITY.md) §7

```text
[ ] موقع آخر لا يستخدم Agent
[ ] جهاز على الشبكة لا يستخدم Agent
[ ] Copy بدون Authorization مرفوض (إن جُعل الرمز إلزامياً)
[ ] Path Traversal مرفوض
[ ] destination خبيث مرفوض
[ ] لا Command Execution API
[ ] لا أسرار hardcoded
```

---

## 6. ما لا يُختبر (صراحة)

| البند | السبب |
|---|---|
| iPhone حقي | يحتاج جهاز + `usbmuxd` |
| ADB | غير مُنفَّذ |
| WPD | غير مُنفَّذ |
| Resume | غير مُنفَّذ |
| Session 0 مع Android | يحتاج تثبيت خدمة |

---

## 7. مصادر

- [go test](https://pkg.go.dev/testing)
- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
