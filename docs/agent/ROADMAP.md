# NEXORA Agent — Roadmap

> **المبدأ الحاكم:** `PRESERVE WHAT WORKS · ISOLATE WHAT CHANGES · DOCUMENT EVERYTHING · TEST BEFORE DECLARING SUCCESS`

---

## المرحلة الحالية — Agent موجود + Android يُكمَّل

### ما هو موجود بالفعل (لا نبنيه)

| القدرة | الملف | الحالة |
|---|---|---|
| Windows Service | `cmd/copybridge/` | ✅ |
| API محلي (12 endpoint + SSE) | `internal/copybridge/server.go` | ✅ |
| Token + CORS | نفسه | ✅ |
| TransferBackend (تجريد) | `internal/transfer/backend.go` | ✅ |
| Transfer Engine v2 | `engine.go` + `worker.go` + `scheduler.go` | ✅ |
| Storage backend (USB/HDD/أقراص) | `storage_backend.go` | ✅ 100% |
| iOS backend | `go_ios_backend.go` | ✅ **محمي** |
| Errors مُصنَّفة | `errors.go` | ✅ جزئي |

### نطاق العمل (Phase 6–9)

| # | المهمة | الجدوى |
|---|---|---|
| 1 | `AndroidBackend.List` — تصفح الهاتف | قابلة |
| 2 | `AndroidBackend.Delete` | قابلة (`InvokeVerb('delete')` مستخدم فعلاً) |
| 3 | فحص المساحة قبل البدء | قابلة |
| 4 | كشف فصل الجهاز أثناء النقل | قابلة |
| 5 | تعداد كل مساحات التخزين (لا الأولى فقط) | قابلة |
| 6 | استبدال `Start-Sleep 500ms` بـ polling بحد | قابلة |
| 7 | أكواد خطأ `TransferError` للأندرويد | قابلة |
| 8 | `Capabilities` على `Device` | قابلة |
| 9 | Logging منظم | قابلة |

### غير قابلة عبر Shell COM (موثّقة)

| # | المهمة | السبب |
|---|---|---|
| A | `PutStream` بلا ملف temp | `CopyHere` **يطلب مسار ملف حقي** |
| B | `Resume` للنقل | `CopyHere` لا يدعم الإضافة |
| C | `Rename` موثوق | Shell COM لا يوفّر rename ذرّياً على MTP |

> **هذه الثلاثة تحتاج WPD.** لا نحاول حلولاً خطرة (الطلب §33).

---

## WPD — مُخطَّط (قرار المالك: خيار أ)

### لماذا WPD في النهاية؟

يحل الثلاثة "غير قابلة" أعلاه عبر `IPortableDeviceResources` + `IStream`:

```text
Shell COM (الحالي)          WPD (المستقبلي)
─────────────────           ────────────────
CopyHere (بلا تحكم)    →    IStream (كتابة مباشرة)
لا Resume             →    WriteAt(offset) ممكن
لا تقدم حقي           →    bytes مكتوبة فعلاً
يحتاج temp للبث       →    بث مباشر من مجرى
يحتاج سياق تفاعلي     →    يعمل في Session 0
```

### خطة التنفيذ

```text
WPD-1  إثبات مفهوم: IPortableDeviceManager → تعداد الأجهزة
WPD-2  المحتوى: IPortableDeviceContent → تصفح
WPD-3  البث: IPortableDeviceResources → IStream (Read/Write)
WPD-4  استبدال AndroidBackend
WPD-5  إزالة الاعتماد على PowerShell
```

**البوابة:** لا يبدأ WPD قبل أن يعمل Shell COM بالكامل (`List`, `Delete`,
`SpaceCheck`) — ليكون **المرجع السلوكي** الذي يطابقه WPD.

**المخاطرة:** COM interop يدوي في Go (`syscall` + `unsafe` + vtables) أو مكتبة
خارجية بلا اعتماد موثوق. راجع [ANDROID.md](ANDROID.md) §3.

---

## ADB — مُخطَّط

### لماذا؟

MTP **لا يصل** إلى:
- `/data/data/<app>` — بيانات التطبيقات
- `/sdcard/Android/data/<app>` — Android 11+
- `/sdcard/Android/obb` — محجوب

### التصميم (interfaces فقط — لا تنفيذ الآن)

```text
AndroidADBProvider (مستقبلاً)
    ├── Install / Uninstall APK
    ├── Push / Pull
    ├── Backup / Restore app data
    ├── DeviceInfo / PackageInfo
    ├── Logcat
    └── Diagnostics
```

> الطلب §14: *«لا تنفذ ADB الآن إلا إذا كان موجودًا أصلًا»* — **غير موجود**. مُخطَّط.

---

## Explorers Integration — مُخطَّط

الطلب §6 يذكر `Windows Shell` **للتكامل فقط** (لا كمحرك نقل):

```text
Context Menu      — "انسخ إلى NEXORA"
Open With         — فتح من NEXORA
Send To           — إرسال إلى جهاز
Shell Extensions  — تكامل Explorer
```

> **غير مُنفَّذ**، وغير مطلوب في هذه الجولة.

---

## Providers مستقبلية (الطلب §32)

التجريد `TransferBackend` يسمح بإضافتها **بلا لمس الواجهة أو المحرك**:

| Provider | الحالة | ملاحظة |
|---|---|---|
| `StorageBackend` | ✅ موجود | USB, HDD, أقراص |
| `AndroidBackend` (Shell COM) | ✅ موجود | يُكمَّل |
| `IOSBackend` (AFC) | ✅ موجود | **محمي** |
| `WPDProvider` | ⏭️ مُخطَّط | يستبدل Android |
| `AndroidADBProvider` | ⏭️ مُخطَّط | مهام متقدمة |
| `NetworkProvider` | ⏭️ مُخطَّط | SMB/NFS |
| `NASProvider` | ⏭️ مُخطَّط | — |
| `CloudStorageProvider` | ⏭️ مُخطَّط | — |

**الفائدة:** إضافة backend جديد = ملف جديد + تسجيل في `BackendFactory`.
**صفر تغيير** في الواجهة، API، أو المحرك.

---

## قيد معروف: External HDD لا يظهر

`discoverRemovableDrives` يقبل `DRIVE_REMOVABLE` فقط. External HDD يُعرّفه
Windows غالباً كـ `DRIVE_FIXED` → **لا يظهر**.

**الحل المُقترح (لا يُنفَّذ الآن):**
```text
- قبول DRIVE_FIXED أيضاً، لكن:
  - استبعاد C: والنظام صراحةً
  - التحقق من BusType عبر IOCTL_STORAGE_QUERY_PROPERTY == BusTypeUsb
```

> **مؤجّل** لأن الطلب §2 يمنع لمس منطق USB/الأقراص العامل حالياً.

---

## مصادر

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [IPortableDeviceResources](https://learn.microsoft.com/en-us/windows/win32/api/portabledeviceapi/nn-portabledeviceapi-iportabledeviceresources)
- [Android Storage](https://developer.android.com/training/data-storage)
- [Windows Shell Extensions](https://learn.microsoft.com/en-us/windows/win32/shell/shell-exts)
