# NEXORA Agent — Android

> **الحالة:** Android الحالي يعمل عبر **Windows Shell COM**، **لا WPD**.
> WPD مُخطَّط لا مُنفَّذ — قرار المالك (خيار أ). راجع [ROADMAP](ROADMAP.md).

---

## 1. الطريقة الفعلية الحالية (من الكود)

**لا يوجد أي ملف WPD في المشروع.** Android يُنفَّذ في
`server/internal/transfer/android_backend.go` (282 سطراً) عبر PowerShell.

### 1.1 الاكتشاف

الدالة `discoverWindowsMTPDevices` (في `service.go`) تشغّل PowerShell:

```powershell
$shell = New-Object -ComObject Shell.Application
$myComp = $shell.NameSpace(17)      # 17 = "أجهزة الكمبيوتر" (ssfDRIVES)
```

`NameSpace(17)` هو **مساحة أسماء Windows Shell** التي تعرض:
- الأقراص العادية (C:, D:, ...)
- الأقراص القابلة للإزالة
- **الأجهزة المحمولة (MTP)** — تظهر كعناصر بمسار يبدأ بـ `::`

كشف الجهاز المحمول يعتمد على:
```powershell
$item.Type -like '*Portable*'   # "Portable Device" / "جهاز محمول"
$item.Type -like '*هاتف*'
$item.Type -like '*جهاز*'
$item.Path.StartsWith("::")     # معرّف Shell Namespace، ليس مسار ملف
```

**معرّف الجهاز المُنتَج:** `mtp_<name>` (يُبنى في `service.go`).

### 1.2 الوصول للتخزين

```powershell
$storage = $deviceItem.GetFolder
$firstStorage = $storage.Items() | Select-Object -First 1   # ⚠️ أول مساحة فقط
$folder = $firstStorage.GetFolder
```

> **⚠️ قيد موثّق:** يُفترض أن **أول** مساحة تخزين هي الهدف دائماً. جهاز له ذاكرة
> داخلية + كارت SD سيُكتب على **الأولى فقط** بلا اختيار. انظر §5 و[ROADMAP](ROADMAP.md).

### 1.3 إنشاء المجلدات

```powershell
foreach ($name in 'Movies|Season 1'.Split('|')) {
    foreach ($item in $folder.Items()) {
        if ($item.Name -eq $name) { $next = $item.GetFolder; break }
    }
    if (-not $next) {
        $folder.NewFolder($name)
        Start-Sleep -Milliseconds 500      # ⚠️ انتظار ثابت
        # إعادة مسح للعثور على المجلد الجديد
    }
    $folder = $next
}
```

**مشكلة موثّقة:** `Start-Sleep 500ms` هو **تعويض عن indexation MTP البطيء**، لا
ضمان. على جهاز بطيء قد يفشل العثور على المجلد المنشأ.

### 1.4 النسخ — القيد الأساسي

```powershell
$finalTarget.CopyHere($sourcePath, 16)   # 16 = لا نافذة تقدم، لا تأكيد
```

**`CopyHere` غير متزامن تماماً ويرجع فوراً.** لا يعيد قيمة. لا واجهة تقدم. لا
إشعار إكمال. اعتماداً على MSDN: العملية تُنفَّذ بواسطة Explorer في خيط آخر.

### 1.5 التقدم — Polling

بما أن `CopyHere` لا يُبلّغ، الحل الحالي **يقيس حجم الملف الهدف دورياً**:

```go
ticker := time.NewTicker(1200 * time.Millisecond)   // كل 1.2s
// ...
size, exists, err := queryMTPFileSize(ctx, deviceName, targetParts, fileName)
if onProgress != nil { onProgress(size) }
if size >= wantSize { return nil }                    // اكتمل
```

**المهلة:** `transferTimeout(wantSize)`، وعند تجاوزها:
- إن وُجد الملف: `"توقف نقل USB قبل اكتمال الملف: وصل X من Y"`
- إن لم يوجد: `"لم يظهر الملف داخل الهاتف بعد بدء النسخ خلال Z"`

### 1.6 `PutStream` — يخالف قاعدة الملفات الكبيرة

```go
func (b *AndroidBackend) PutStream(ctx, reader, size, destination, opts) error {
    tempFile, _ := os.CreateTemp("", "nexora-mtp-stream-*.bin")
    io.Copy(tempFile, reader)        // ⚠️ يكتب الملف كاملاً على القرص أولاً
    // ثم Put(tempPath, ...)
}
```

> **⚠️ يخالف الطلب §12:** "لا تقرأ الملف كله إلى RAM / استخدم streaming".
> هنا **لا يُقرأ للـ RAM لكن يُكتب كاملاً لقرص temp**. لملف 80 GB هذا يعني
> **80 GB كتابة إضافية على C:** ووقت مضاعف. مُوثَّق كفجوة، ومُدرَج في نطاق العمل.

---

## 2. الفجوات الحقيقية (نطاق العمل)

| # | الفجوة | الأثر | الحالة |
|---|---|
| 1 | `List()` يرجع خطأ "غير مدعوم" | لا تصفح للهاتف من الواجهة | ✅ **مُغلقة** |
| 2 | `Delete()` غير مدعوم | لا إدارة ملفات | ✅ **مُغلقة** |
| 3 | `Rename()` غير مدعوم | لا إعادة تسمية | ⏭️ **مُوثَّقة** — تحتاج WPD |
| 4 | `PutStream` يمرّ بملف temp | كتابة مضاعفة، مساحة C: | ⏭️ **مُوثَّقة** — تحتاج WPD |
| 5 | لا كشف فصل الجهاز | المهمة تعلق حتى المهلة | ✅ **مُغلقة** |
| 6 | لا فحص مساحة قبل البدء | فشل بعد بدء النسخ | ✅ **مُغلقة** |
| 7 | `Start-Sleep 500ms` بدل التحقق | فشل على أجهزة بطيئة | ✅ **مُغلقة** — polling محدود |
| 8 | أول مساحة فقط | لا SD card | ✅ **مُغلقة** — `NewAndroidBackendForStorage` |
| 9 | الأخطاء نصية بلا أكواد | لا retry ذكي | ✅ **مُغلقة** — 8 أكواد جديدة |

### عيوب حقيقية كشفها الاختبار الحي (لا القراءة)

ثلاثة عيوب لم تكن ظاهرة في الكود ولم تُلاحظ في المراجعة، ظهرت فقط عند التنفيذ الفعلي:

| # | العيب | الأثر | الإصلاح |
|---|---|
| A | `powershell` يُستدعى بالاسم المجرد | **فشل كامل** على أي عملية لا تحتوي `System32` في `PATH` (خدمة، مُشغّل مقيّد) | `powershellPath()` يحل المسار المطلق |
| B | السكربت يُمرّر عبر `-Command` | سكربت طويل بعربية يُقتطع أو يُفسَّر خطأً → `Missing closing '}'` في سكربت سليم | `-EncodedCommand` (UTF-16LE base64) |
| C | `try { if (...) { ... } catch {}` | قوس واحد ناقص → خطأ نحوي حقي في `List` | أُصلح، و**اختبار يتحقق من التوازن** |

**الدرس:** العيوب الثلاثة **لم تكن لتظهر في أي اختبار وحدة** — اكتشفها تشغيل
الوكيل الحقي واستدعاء endpoint فعلي. ولذلك أُضيف `TestGeneratedScriptParses`
ليُشغّل **محلّل PowerShell نفسه** على كل سكربت مُولَّد.

**النقطة 4 حرجة:** البث المباشر **مستحيل** عبر `CopyHere` لأن Windows Shell API
**تطلب مسار ملف حقي** كمصدر. الحل الوحيد هو WPD (`IStream`).

---

## 3. WPD — لماذا مُخطَّط لا مُنفَّذ

### ما هو WPD؟

Windows Portable Devices API — الواجهة الرسمية من Microsoft للأجهزة المحمولة
(MTP). يوفّر COM interfaces:
- `IPortableDeviceManager` — تعداد الأجهزة
- `IPortableDevice` — فتح جهاز، تعداد المحتوى
- `IPortableDeviceContent` — الملفات والمجلدات
- `IPortableDeviceResources` — **`IStream` للقراءة/الكتابة** ← يحل مشكلة البث
- `IPortableDeviceValues` — خصائص

### لماذا لا الآن؟

| السبب | التفصيل |
|---|---|
| لا كود WPD موجود | بناء من الصفر، بلا أساس |
| Go + COM | يحتاج interop يدوي (`syscall` + `unsafe` + vtables) أو مكتبة خارجية |
| المكتبات الخارجية | لا مكتبة Go معتمدة ونشطة لـ WPD |
| مخاطرة | Shell COM يعمل ومُختبر على أجهزة حقيقية |
| الوقت | أسبوع+ عمل، ومخاطرة انحدار على وظيفة تعمل |
| **العزل موجود** | `TransferBackend` يسمح بالاستبدال لاحقاً **بلا لمس الواجهة** |

### خطة WPD المستقبلية (موثّقة لا مُنفَّذة)

```text
Phase WPD-1:  إثبات مفهوم — تعداد الأجهزة عبر IPortableDeviceManager
Phase WPD-2:  تعداد المحتوى — IPortableDeviceContent
Phase WPD-3:  البث — IPortableDeviceResources + IStream
Phase WPD-4:  استبدال AndroidBackend كلياً
Phase WPD-5:  إزالة الاعتماد على PowerShell نهائياً
```

**البوابة:** لا يبدأ WPD قبل أن يعمل `List`/`Delete`/`SpaceCheck` عبر Shell COM،
لأن shell هو **المرجع السلوكي** الذي يجب أن يطابقه WPD.

---

## 4. ADB — مُخطَّط (الطلب §14, §15)

### لماذا ADB؟

MTP/WPD **لا يصل** لجميع مجلدات Android الحديثة:
- `/data/data/<app>` — بيانات التطبيقات (محمي)
- `/data/media/0/Android/data/<app>` — Android 11+ محجوب عن MTP
- `Android/obb` — محجوب عن MTP غالباً

### لماذا لا الآن؟

الطلب §14 يقول صراحة: *«لا تنفذ ADB الآن إلا إذا كان موجودًا أصلًا»*. **غير موجود.**

### التصميم المستقبلي (interfaces فقط عند الحاجة)

```text
AndroidProvider
    ├── Discover
    ├── Storages
    ├── Browse
    ├── Copy
    ├── Delete
    └── Capabilities

AndroidADBProvider (مستقبلاً)
    ├── Install / Uninstall
    ├── Push / Pull
    ├── Backup / Restore
    ├── Logcat / Diagnostics
    └── DeviceInfo / PackageInfo
```

**متطلبات ADB عند تنفيذه:** ADB مفعّل على الجهاز، USB debugging، موافقة RSA،
إصدار Android، حالة الجهاز.

---

## 5. قيود Android الحديث (يجب توثيقها — الطلب §14)

| المسار | الوصول عبر MTP | السبب |
|---|---|---|
| `/sdcard/` (المشاركة) | ✅ | مساحة مشتركة |
| `/sdcard/Movies`, `Download`, `DCIM` | ✅ | جزء من المشاركة |
| `/sdcard/Android/data/<app>` | ❌ غالباً | Android 11+ (Scoped Storage) |
| `/sdcard/Android/obb/<app>` | ❌ غالباً | محجوب عن MTP |
| `/data/data/<app>` | ❌ دائماً | محمي، يحتاج root |
| `/system/` | ❌ دائماً | للقراءة فقط، يحتاج root |

**النتيجة العملية:** NEXORA ينسخ إلى **المساحة المشتركة** (`Movies/`, `Download/`)
وهذا **متوافق** مع ما يعمل حالياً. لا نعِد بما لا نستطيع.

---

## 6. إصدارات Android

| الإصدار | ملاحظة على MTP |
|---|---|
| Android 4.x | MTP بسيط، مساحة واحدة غالباً |
| Android 5–9 | MTP مستقر، دعم SD card |
| Android 10 | Scoped Storage يبدأ — بعض القيود |
| **Android 11+** | `Android/data` و`Android/obb` محجوبان عن MTP |
| Android 13+ | `Android/data` محجوب تماماً عن MTP |

**لا نفترض بنية موحدة.** الكود يتعامل مع ما يعرضه الجهاز فعلاً.

---

## 7. مصادر (Microsoft الرسمية)

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [WPD Application Programming Interface](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/wpd-application-programming-interface)
- [IPortableDeviceResources](https://learn.microsoft.com/en-us/windows/win32/api/portabledeviceapi/nn-portabledeviceapi-iportabledeviceresources)
- [Shell.NameSpace method](https://learn.microsoft.com/en-us/windows/win32/shell/shell-namespace)
- [Folder.CopyHere method](https://learn.microsoft.com/en-us/windows/win32/shell/folder-copyhere)
