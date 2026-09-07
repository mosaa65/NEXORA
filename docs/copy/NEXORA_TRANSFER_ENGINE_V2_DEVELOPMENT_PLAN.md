# NEXORA — خطة تطوير Transfer Engine v2 ونظام النسخ عبر USB

## 1. الهدف من الوثيقة

هذه الوثيقة تمثل الخطة التطويرية المقترحة لتطوير نظام النسخ عبر USB في مشروع **NEXORA** اعتمادًا على بنية النقل الحالية ومتطلبات الواجهة وتجربة المستخدم الجديدة.

الهدف ليس إعادة بناء النظام من الصفر، بل **تطوير Transfer Engine الحالي المكتوب بـ Go** مع المحافظة على React والـ REST API والبنية الحالية قدر الإمكان، ثم نقل طبقة التنفيذ إلى محرك نقل أكثر تنظيمًا وكفاءة واعتمادية.

> **قيد تقني أساسي:** لا نضيف لغة برمجة جديدة ولا Framework جديد للمحرك. التنفيذ الأساسي يبقى بـ **Go**، مع الاستفادة من أدوات libimobiledevice الموجودة أصلًا في المشروع. بالنسبة إلى iOS، نستفيد من `afcclient` الحالي بطريقة طويلة العمر (interactive/persistent process) بدل إنشاء process جديد لكل ملف.

---

# 2. الوضع الحالي في NEXORA

النظام الحالي يدعم ثلاثة أنواع رئيسية من الوجهات:

| النوع | طريقة النقل الحالية |
|---|---|
| Android | PowerShell + COM `Shell.Application` / MTP |
| iOS | `afcclient` من libimobiledevice عبر AFC/HouseArrest |
| USB Storage | نسخ مباشر عبر Go `os` |

النظام الحالي يحتوي أصلًا على:
- اكتشاف الأجهزة.
- اكتشاف تطبيقات iOS.
- فلترة iOS placeholders القادمة من Windows MTP.
- عرض مجلدات Documents في تطبيقات iOS.
- إنشاء مجلدات.
- بدء مهام النسخ بشكل غير متزامن.
- Progress.
- Speed.
- Cancel.
- Verification.
- سجل للمهام.
- REST API مخصصة للنقل.
- مكونات React لاختيار الجهاز والتطبيق والوجهة.

هذه الأساسات يجب المحافظة عليها وتطويرها بدل التخلص منها.

---

# 3. المتطلبات الجديدة

## 3.1 تجربة المستخدم الأساسية

المستخدم يستطيع تحديد:
- حلقة واحدة أو عدة حلقات من مسلسل.
- جزء أو عدة أجزاء.
- فيلم واحد أو عدة أفلام.
- سلسلة أو عدة أعمال.
- أي مجموعة من الملفات التي يريد نسخها.

بعد تحديد عنصر واحد أو أكثر يظهر:

> **زر عائم صغير وحديث ومرن خاص بالنسخ.**

الزر لا يحتل مساحة كبيرة من الشاشة، ويزداد وضوحه عندما توجد عناصر محددة للنسخ.

عند الضغط عليه تفتح نافذة نقل متوسطة الحجم وResponsive.

---

# 4. تصميم نافذة النسخ الجديدة

## 4.1 الهيكل العام

```text
┌──────────────────────────────────────────────────────────────┐
│                     نافذة النسخ                              │
│                                                              │
│ ┌────────────────┐ ┌──────────────────────────────────────┐ │
│ │ الأجهزة        │ │                                      │ │
│ │                │ │        المحتوى الرئيسي                │ │
│ │ iPhone 15      │ │                                      │ │
│ │ iPhone 13      │ │   تطبيقات / ملفات / مجلدات            │ │
│ │ Samsung        │ │                                      │ │
│ │ USB Drive      │ │                                      │ │
│ │                │ │                                      │ │
│ │────────────────│ │                                      │ │
│ │ العناصر المحددة│ │                                      │ │
│ │ 5 ملفات        │ │                                      │ │
│ │ 7.8 GB         │ │                                      │ │
│ │                │ │                                      │ │
│ └────────────────┘ └──────────────────────────────────────┘ │
│                                                              │
│                 [  نسخ الآن  ]                               │
└──────────────────────────────────────────────────────────────┘
```

---

# 5. القسم الأيمن: الأجهزة والبيانات المحددة

## 5.1 الأجهزة

في القسم الأيمن من النافذة تظهر قائمة الأجهزة المتصلة.

مثال:

```text
الأجهزة المتصلة

● iPhone 15 Pro
  متصل عبر USB

○ Samsung S24
  متصل عبر USB

○ USB Storage (E:)
```

ويتم تحديد جهاز افتراضيًا:

> أول جهاز متاح وصالح للنقل.

إذا لم يحدد المستخدم جهازًا يدويًا، يستخدم النظام أول جهاز مناسب.

## 5.2 معلومات العناصر المحددة

أسفل قائمة الأجهزة يوجد خط فاصل واضح، ثم قسم:

```text
العناصر المحددة

5 حلقات
2 أفلام

الإجمالي:
7.8 GB

عدد الملفات:
7
```

ويجب عرض:
- عدد الملفات.
- الحجم الإجمالي.
- نوع المحتوى.
- أسماء مختصرة.
- إمكانية التحقق من العناصر قبل التنفيذ.

يمكن لاحقًا إضافة زر "عرض الكل".

---

# 6. عندما يكون الجهاز iPhone

عند اختيار iPhone:

## 6.1 عرض التطبيقات

يظهر في منتصف الواجهة:

```text
تطبيقات هذا iPhone

[VLC] VLC Media Player      [عرض]
[Infuse] Infuse             [عرض]
[Docs] Documents            [عرض]
[nPlayer] nPlayer           [عرض]
```

لكل تطبيق:
- الشعار.
- اسم التطبيق كما يظهر على الجهاز.
- Bundle ID داخليًا دون الحاجة لإظهاره للمستخدم.
- زر `عرض`.

### مهم

لا نعرض Bundle ID للمستخدم العادي.

المستخدم يرى:

> VLC

وليس:

> org.videolan.vlc-ios

---

# 7. زر "عرض" للتطبيق

عند ضغط:

```text
[عرض]
```

يتم:
1. إخفاء قائمة التطبيقات.
2. الانتقال إلى مستعرض ملفات التطبيق.
3. عرض المجلدات والملفات المتاحة للمستخدم.

مثال:

```text
VLC

← الرجوع إلى التطبيقات

Documents

📁 Movies
📁 Series
📁 Downloads
📄 video.mp4
📄 episode01.mp4
```

---

# 8. مستعرض ملفات iOS

المستعرض يجب أن يكون مشابهًا لمدير الملفات.

كل عنصر يحتوي على:
- أيقونة مجلد أو ملف.
- الاسم.
- الحجم للملفات عند توفره.
- حالة التحميل/القراءة عند الحاجة.
- سهم للمجلدات.

مثال:

```text
Documents
│
├── 📁 Movies
│    │
│    ├── 📁 Action
│    ├── 📁 Drama
│    └── 📄 sample.mp4
│
├── 📁 Series
│    │
│    ├── 📁 Breaking Bad
│    └── 📁 The Last of Us
│
└── 📁 Downloads
```

---

# 9. التنقل داخل المجلدات

يجب أن يعرف المحرك دائمًا:

```text
CurrentRemotePath
```

مثال:

```text
/Documents
/Documents/Series
/Documents/Series/Breaking Bad
/Documents/Series/Breaking Bad/Season 1
```

وعند الضغط على زر النسخ:

> النسخ يتم إلى المسار الحالي الذي وصل إليه المستخدم.

مثال:

```text
التطبيق:
VLC

المسار الحالي:
Documents/Series/Breaking Bad/Season 1
```

فإن جميع الملفات المحددة تنسخ إلى:

```text
/Documents/Series/Breaking Bad/Season 1/
```

---

# 10. إنشاء مجلد جديد

في مستعرض الملفات يجب توفير:

```text
[ + مجلد جديد ]
```

مثال:

```text
إنشاء مجلد

اسم المجلد:
[ Season 2             ]

[إلغاء] [إنشاء]
```

بعد نجاح الإنشاء:
- يظهر المجلد فورًا.
- يمكن الدخول إليه.
- يصبح هو المسار الحالي إذا اختاره المستخدم.

---

# 11. زر النسخ الرئيسي

في أسفل منتصف النافذة:

```text
                [ ⬇ نسخ 7 ملفات ]
```

الزر:
- صغير نسبيًا.
- واضح.
- Modern.
- Responsive.
- يعرض عدد الملفات.
- يعرض الحجم إن كان ذلك مناسبًا.

مثال:

```text
[ نسخ 7 ملفات • 7.8 GB ]
```

بعد الضغط:

1. إنشاء Job.
2. إدخاله إلى Scheduler.
3. بدء التنفيذ.
4. إغلاق نافذة اختيار الوجهة.
5. ظهور واجهة المهام أسفل الشاشة.

---

# 12. تجربة المستخدم بعد بدء النسخ

بعد الضغط على النسخ، تغلق نافذة الاختيار.

تظهر في أسفل الشاشة بطاقة نقل منكمشة.

مثال:

```text
┌────────────────────────────────────┐
│  نسخ 7 ملفات                       │
│  ▰▰▰▰▰▰▰▰░░░  72%                  │
│  5.6 GB / 7.8 GB                   │
│  62 MB/s • 00:34 متبقي             │
└────────────────────────────────────┘
```

وعند الحاجة يمكن توسيع البطاقة.

---

# 13. نافذة تفاصيل النقل

عند الضغط على بطاقة النسخ:

```text
┌───────────────────────────────────────┐
│ نقل الملفات                           │
│                                       │
│ الملف الحالي                          │
│ Episode 05.mp4                        │
│ 820 MB / 1.2 GB                       │
│                                       │
│ الإجمالي                              │
│ 5.6 GB / 7.8 GB                       │
│                                       │
│ السرعة: 62 MB/s                       │
│ الوقت المتبقي: 00:34                  │
│                                       │
│ [إيقاف مؤقت] [إلغاء]                  │
└───────────────────────────────────────┘
```

---

# 14. Android

عند اختيار Android يجب أن يتغير المحتوى الرئيسي إلى مدير ملفات مشابه للتخزين الداخلي.

مثال:

```text
Samsung S24

Internal Storage

📁 Android
📁 DCIM
📁 Download
📁 Movies
📁 Music
📁 Pictures
📁 Documents
```

المستخدم يدخل إلى:

```text
Movies
```

ثم:

```text
Movies / Series / Breaking Bad
```

ثم يضغط النسخ.

المسار الحالي هو الوجهة النهائية.

نفس تجربة iOS:
- تنقل.
- إنشاء مجلد.
- اختيار المسار.
- نسخ إلى المسار الحالي.

---

# 15. USB Storage

لأقراص التخزين القابلة للإزالة نفس تجربة Android تقريبًا.

مثال:

```text
USB Drive (E:)

E:\
├── Movies
├── Series
├── Music
└── Backup
```

المستخدم يحدد المجلد الحالي ثم يبدأ النسخ.

بالنسبة للأقراص المحلية، يمكن استخدام Go مباشرة دون الحاجة إلى طبقة MTP.

---

# 16. قاعدة موحدة للوجهات

يجب ألا يكون المحرك مرتبطًا بمتغيرات مثل:

```text
SubFolder
TargetFolder
```

بشكل مبعثر.

يجب إنشاء نموذج موحد:

```go
type TransferDestination struct {
    DeviceID   string
    DeviceType DeviceType

    AppID string

    RemotePath string
}
```

أمثلة:

### iOS

```text
DeviceID:
ios_<udid>

AppID:
org.videolan.vlc-ios

RemotePath:
/Documents/Series/Breaking Bad/Season 1
```

### Android

```text
DeviceID:
android_xxx

RemotePath:
Internal Storage/Movies/Breaking Bad/Season 1
```

### USB Storage

```text
DeviceID:
disk_E

RemotePath:
E:\Movies\Breaking Bad\Season 1
```

---

# 17. Transfer Engine v2

## 17.1 المبدأ

نريد أن يتحول النظام من:

```text
Request
   ↓
تشغيل أمر
   ↓
انتظار
   ↓
Polling
   ↓
انتهاء
```

إلى:

```text
Request
   ↓
TransferJob
   ↓
Scheduler
   ↓
Device Worker
   ↓
Persistent Session
   ↓
Stream
   ↓
Progress / Retry / Resume / Verify
```

---

# 18. الحفاظ على Go كطبقة المحرك

التصميم النهائي:

```text
React
  ↓
REST API
  ↓
Go Transfer Service
  ↓
Go Transfer Engine
  ├── Scheduler
  ├── Device Manager
  ├── Job Manager
  ├── Progress Engine
  ├── Retry Manager
  ├── Resume Manager
  └── Backend Registry
```

ولا نضيف Framework جديد.

---

# 19. iOS بدون إنشاء Process لكل ملف

## المشكلة الحالية

الكود الحالي يعتمد على استدعاءات `afcclient` لتنفيذ:
- `pwd`
- `ls`
- `mkdir`
- `put`
- `info`

وهذا يجعل تحسين المحرك محدودًا إذا كان كل أمر عبارة عن process مستقل.

## الحل

النسخة الحالية من `afcclient` تدعم العمل التفاعلي Interactive، أي يمكن تشغيلها كعملية طويلة العمر ثم إرسال أوامر متعددة عبر stdin بدل بدء عملية جديدة لكل أمر.

تصميم Go المقترح:

```go
type AFCSession struct {
    DeviceID string
    BundleID string

    Cmd    *exec.Cmd
    Stdin  io.WriteCloser
    Stdout io.ReadCloser
    Stderr io.ReadCloser

    Mu sync.Mutex

    Ready bool
}
```

ويتم إنشاء العملية مرة واحدة:

```text
Go
 ↓
start afcclient
 ↓
wait until session ready
```

ثم:

```text
ls
mkdir
ls
put
put
put
info
```

كلها من خلال نفس العملية.

### النتيجة

بدل:

```text
file1 → process
file2 → process
file3 → process
file4 → process
```

نصبح:

```text
AFC Session
    │
    ├── file1
    ├── file2
    ├── file3
    └── file4
```

ولا يتم إنشاء process جديد لكل ملف.

> ملاحظة: يجب بناء `AFCSession` بطريقة آمنة لأن stdout في الوضع التفاعلي يحتوي على prompt ونتائج الأوامر، لذلك نحتاج Command Dispatcher يطابق كل أمر مع نهايته ولا يسمح بتنفيذ أمرين بالتوازي داخل نفس session.

---

# 20. Session Manager

نحتاج مدير جلسات:

```go
type SessionManager struct {
    mu       sync.Mutex
    sessions map[string]*AFCSession
}
```

المفتاح الأفضل:

```text
deviceID + bundleID
```

مثال:

```text
ios_UDID::org.videolan.vlc-ios
```

بهذا يمكن الاحتفاظ بجلسة لكل:

```text
Device + App
```

---

# 21. دورة حياة جلسة iOS

```text
Disconnected
    ↓
Connecting
    ↓
Connected
    ↓
Ready
    ↓
Busy
    ↓
Ready
    ↓
...
    ↓
Disconnected
```

عند إزالة الجهاز:

```text
Ready
  ↓
DeviceLost
  ↓
Close Session
```

وعند عودته:

```text
DeviceDetected
    ↓
Reconnect
```

---

# 22. Folder Manager

ننشئ واجهة موحدة:

```go
type FileSystemBackend interface {
    List(ctx context.Context, path string) ([]RemoteEntry, error)
    Stat(ctx context.Context, path string) (RemoteEntry, error)
    Mkdir(ctx context.Context, path string) error
    Delete(ctx context.Context, path string) error
    Rename(ctx context.Context, oldPath, newPath string) error
}
```

ثم:

```text
IOSFilesystem
AndroidFilesystem
StorageFilesystem
```

---

# 23. Lazy Loading للمجلدات

لا نريد إرسال:

```text
ls كل المجلدات recursively
```

عند فتح التطبيق.

بدل ذلك:

```text
GET /device-app-folders?path=/Documents
```

ثم:

```text
GET /device-app-folders?path=/Documents/Series
```

ثم:

```text
GET /device-app-folders?path=/Documents/Series/Breaking%20Bad
```

### الفائدة

- بداية أسرع.
- ذاكرة أقل.
- ضغط أقل على الجهاز.
- تجربة أفضل عندما يكون التطبيق يحتوي على آلاف الملفات.

---

# 24. RemoteEntry

```go
type RemoteEntry struct {
    Name     string
    Path     string
    IsDir    bool
    Size     int64
    Modified time.Time
}
```

---

# 25. إدارة الملفات في الواجهة

عند فتح مجلد:

```text
Name
Type
Size
```

مثلاً:

```text
📁 Season 1
📁 Season 2
📄 Episode 01.mp4    1.2 GB
📄 Episode 02.mp4    1.4 GB
```

---

# 26. TransferJob الجديد

يجب توسيع النموذج الحالي ليستوعب عدة ملفات.

```go
type TransferJob struct {
    ID string

    DeviceID   string
    DeviceName string
    DeviceType DeviceType

    Destination TransferDestination

    Files []TransferFile

    TotalBytes       int64
    TransferredBytes int64

    CurrentFileIndex int

    Progress float64
    SpeedBps  int64
    ETA       time.Duration

    Status JobStatus
    Phase  TransferPhase

    RetryCount int

    Error *TransferError

    StartedAt   time.Time
    CompletedAt *time.Time
}
```

---

# 27. TransferFile

```go
type TransferFile struct {
    ID string

    SourcePath      string
    DestinationPath string

    Size        int64
    Transferred int64

    Status JobStatus

    RetryCount int

    ResumeOffset int64

    StartedAt   time.Time
    CompletedAt *time.Time
}
```

---

# 28. حالات المهمة

```text
pending
processing
completed
failed
cancelled
paused
waiting_device
retrying
```

ويتم استخدام phases:

```text
queued
preparing
connecting
checking_destination
copying
retrying
verifying
completed
failed
cancelled
waiting_device
```

---

# 29. Multi-file Jobs

بدل أن يكون لدينا:

```text
Job 1 = Episode 1
Job 2 = Episode 2
Job 3 = Episode 3
```

نريد:

```text
Job A
 ├── Episode 1
 ├── Episode 2
 ├── Episode 3
 ├── Episode 4
 └── Episode 5
```

وهذا مهم جدًا لتجربة المستخدم.

---

# 30. الحساب الصحيح للتقدم

إجمالي:

```text
File 1 = 1 GB
File 2 = 2 GB
File 3 = 3 GB

Total = 6 GB
```

إذا تم نقل:

```text
File 1 = 1 GB
File 2 = 0.5 GB
```

يكون:

```text
Transferred = 1.5 GB
Progress = 25%
```

وليس 50% من الملف الحالي فقط.

---

# 31. Progress الحقيقي

المحرك الجديد يجب أن يعتمد على عدد البايتات التي تم إرسالها فعليًا.

المبدأ:

```go
transferred += n
```

ثم:

```go
progress = float64(transferred) / float64(total) * 100
```

ولا يعتمد النظام أساسًا على polling لحجم الملف النهائي.

---

# 32. سرعة النقل

تستخدم نافذة زمنية حديثة:

```text
البيانات المرسلة خلال آخر 1 ثانية
```

مثال:

```text
62 MB/s
```

وليس المتوسط الكلي فقط.

ويتم حساب:

```text
Speed
ETA
```

مع smoothing بسيط لتجنب تقلب الرقم كل لحظة.

---

# 33. Buffering

## نقطة البداية المقترحة

نبدأ Benchmark على:

```text
1 MB
2 MB
4 MB
8 MB
16 MB
32 MB
```

والقيمة الافتراضية المبدئية:

```text
4 MB
```

ولا نعتبر `4 MB` رقمًا نهائيًا؛ الرقم النهائي يحدد بعد اختبار حقيقي على نفس أجهزة NEXORA.

---

# 34. لماذا 4 MB كبداية؟

لأنه حل متوازن مبدئيًا بين:
- عدد عمليات I/O.
- الذاكرة.
- نقل ملفات كبيرة.
- سهولة الضبط.

ولكن الأداء الحقيقي يجب قياسه.

---

# 35. Double Buffering

بعد نجاح الإصدار الأول يمكن إضافة:

```text
Buffer A
Buffer B
```

بحيث:

```text
Read A → Write A
Read B → Write B
Read A → Write A
...
```

لكن يتم تفعيلها بعد Benchmark وليس قبل ذلك.

---

# 36. عدم تحميل الملف كاملًا في RAM

ممنوع:

```go
os.ReadFile(...)
```

للفيديوهات الكبيرة.

نستخدم streaming:

```text
File
 ↓
Read Buffer
 ↓
Write
 ↓
Read Buffer
 ↓
Write
```

مثلاً لملف:

```text
20 GB
```

الذاكرة المستخدمة لا يجب أن تصبح:

```text
20 GB RAM
```

بل تبقى حول حجم الـ buffer.

---

# 37. Retry Manager

```go
type RetryPolicy struct {
    MaxRetries  int
    InitialWait time.Duration
    MaxWait     time.Duration
}
```

مثال:

```text
Retry 1 → 500ms
Retry 2 → 1s
Retry 3 → 2s
Retry 4 → 4s
```

---

# 38. متى نعيد المحاولة؟

### Retryable

```text
Device disconnected
USB transient error
AFC timeout
Temporary I/O failure
Mux error
```

### Non-retryable

```text
Disk full
Permission denied
Source file missing
Destination invalid
User cancelled
Unsupported operation
```

---

# 39. أنواع أخطاء النقل

```go
type TransferError struct {
    Code       string
    Message    string
    Retryable  bool
    DeviceLost bool
}
```

أمثلة:

```text
DEVICE_DISCONNECTED
AFC_TIMEOUT
AFC_IO_ERROR
MUX_ERROR
DISK_FULL
DESTINATION_NOT_FOUND
PERMISSION_DENIED
SOURCE_NOT_FOUND
VERIFY_FAILED
USER_CANCELLED
```

---

# 40. Resume

مثال:

```text
ملف = 5 GB

تم نقل = 3.2 GB

انقطع الهاتف
```

بعد عودة الجهاز:

```text
Remote size = 3.2 GB
Source size = 5 GB

ResumeOffset = 3.2 GB
```

ويبدأ من:

```text
3.2 GB
```

بدل:

```text
0
```

---

# 41. التحقق من إمكانية Resume

لا نعتمد على الحجم فقط في كل الحالات.

نخزن معلومات مثل:

```go
type TransferCheckpoint struct {
    SourceSize        int64
    SourceModifiedAt  time.Time
    DestinationPath   string
    DestinationSize   int64
}
```

وفي وضع strict يمكن استخدام hash.

---

# 42. سياسة الملفات الموجودة مسبقًا

نضيف:

```text
skip
overwrite
resume
rename
ask
```

مثلاً:

```json
{
  "conflict": "resume"
}
```

والسلوك الافتراضي يحدد حسب UX النهائي.

---

# 43. إنشاء المجلدات قبل النسخ

إذا اختار المستخدم:

```text
/Documents/Series/Breaking Bad/Season 2
```

المحرك يتحقق:

```text
Documents exists?
Series exists?
Breaking Bad exists?
Season 2 exists?
```

إن لم يوجد مجلد:

```text
mkdir
```

ثم يبدأ النسخ.

---

# 44. النقل إلى مسار المستخدم الحالي

الوجهة يجب أن تكون جزءًا من Job وليست معلومة موقتة في React فقط.

مثال:

```text
Destination:
Device = iPhone 15
App = VLC
Path = /Documents/Series/Breaking Bad/Season 2
```

بهذا إذا أعيد تحميل الواجهة، تبقى المهمة مفهومة.

---

# 45. Scheduler

نحتاج Scheduler مركزي:

```go
type TransferScheduler struct {
    jobs    chan *TransferJob
    workers map[string]*DeviceWorker
}
```

---

# 46. Worker لكل جهاز

مثال:

```text
iPhone 1
    ↓
Worker 1

iPhone 2
    ↓
Worker 2

Android 1
    ↓
Worker 3

USB E:
    ↓
Worker 4
```

---

# 47. عدة عمليات على نفس الجهاز

المطلوب الجديد:

> المستخدم يمكن أن يطلب عدة عمليات حتى على نفس الجهاز.

يتم تنفيذها بشكل منظم:

```text
iPhone 1
│
├── Job A
├── Job B
├── Job C
└── Job D
```

لكن الافتراضي:

```text
Concurrency per device = 1
```

أي لا ننقل ملفين في نفس اللحظة إلى نفس iPhone في البداية.

### لماذا؟

لأننا نريد:
- ثباتًا.
- أعلى throughput عملي.
- منع تنافس الأوامر داخل نفس AFC session.
- تجنب تلف أو تعارض في الوجهة.

ثم نختبر لاحقًا إمكانية أي concurrency أعلى.

---

# 48. عدة أجهزة تعمل بالتوازي

بينما:

```text
iPhone 1 → Job A
```

يمكن بالتوازي:

```text
iPhone 2 → Job B
```

و:

```text
USB Drive → Job C
```

وهكذا.

---

# 49. Queue داخل كل جهاز

```text
Scheduler
   │
   ├── iPhone A
   │     ├── Job 1
   │     ├── Job 2
   │     └── Job 3
   │
   ├── iPhone B
   │     ├── Job 4
   │     └── Job 5
   │
   └── Android A
         └── Job 6
```

---

# 50. عدالة الجدولة

لا يجب أن يتم احتجاز Worker واحد بلا نهاية إذا كان لديه Job ضخم.

التصميم الأول:

```text
Job واحد يُكمل ملفاته بالترتيب
```

ثم بعد الانتهاء:

```text
Job التالي
```

لاحقًا يمكن إضافة priority:

```text
high
normal
low
```

---

# 51. Cancel

الإلغاء يجب أن يكون على مستوى:

### Job

```text
Cancel Job
```

ويمكن لاحقًا دعم:

### File

```text
Cancel File
```

المطلوب الأساسي:

```text
POST /api/transfer/cancel/{id}
```

---

# 52. Cancel الحقيقي

المبدأ:

```text
React
 ↓
Cancel API
 ↓
context.Cancel()
 ↓
Worker notices cancellation
 ↓
Close current transfer
 ↓
Close file handles
 ↓
Update Job
```

النتيجة:

```text
cancelled
```

بدل انتظار Timeout.

---

# 53. التعامل مع فصل الجهاز

مثال:

```text
Copying
   ↓
iPhone removed
   ↓
DeviceLost
   ↓
Save checkpoint
   ↓
Pause / WaitingDevice
```

ثم عند عودة الجهاز:

```text
DeviceDetected
   ↓
Reconnect session
   ↓
Verify destination
   ↓
Resume
```

حسب سياسة Resume.

---

# 54. Verification

الإصدار الحالي يعتمد بدرجة أساسية على مطابقة الحجم.

يستمر هذا كخيار سريع:

```text
remote size == source size
```

لكن يمكن توفير:

```text
Strict verification
```

للتحقق الأكثر صرامة.

---

# 55. الأداء والـ I/O

يجب حذف النسخ الوسيطة قدر الإمكان.

الحالة غير المرغوبة:

```text
Original
 ↓
Temporary file
 ↓
AFC
 ↓
iPhone
```

الحالة المرغوبة:

```text
Original
 ↓
Streaming
 ↓
AFC
 ↓
iPhone
```

ويستخدم المسار المؤقت فقط عند الضرورة القصوى.

---

# 56. Security

تبقى وظائف:

```text
safePathComponent
safeRelativePath
splitRelativePath
```

ضرورية.

يجب منع:

```text
../
../../
absolute escape
invalid separators
```

على جميع أنواع الوجهات.

---

# 57. API الجديدة المقترحة

## الأجهزة

موجود:

```http
GET /api/transfer/devices
```

## تطبيقات iOS

موجود:

```http
GET /api/transfer/device-apps?device_id=
```

## مجلدات التطبيق

موجود:

```http
GET /api/transfer/device-app-folders?device_id=&bundle_id=
```

لكن نطوره ليقبل:

```text
path
```

مثلاً:

```http
GET /api/transfer/device-app-folders?device_id=...&bundle_id=...&path=/Documents/Series
```

## إنشاء مجلد

إضافة:

```http
POST /api/transfer/device-app-folder
```

## بدء Job متعدد الملفات

تطوير:

```http
POST /api/transfer/copy
```

ليدعم:

```json
{
  "device_id": "ios_xxx",
  "target_app": "org.videolan.vlc-ios",
  "target_path": "/Documents/Series/Breaking Bad/Season 1",
  "files": [
    {
      "source_path": "D:/Media/Episode01.mp4"
    },
    {
      "source_path": "D:/Media/Episode02.mp4"
    }
  ]
}
```

## Jobs

```http
GET /api/transfer/jobs
GET /api/transfer/job/{id}
POST /api/transfer/cancel/{id}
```

ويضاف لاحقًا:

```http
POST /api/transfer/pause/{id}
POST /api/transfer/resume/{id}
```

عند اعتماد Pause/Resume كـ UX مستقل.

---

# 58. واجهة حالة المهمة

يجب أن يرجع Job بيانات كافية للواجهة:

```json
{
  "id": "job_123",
  "device_id": "ios_xxx",
  "device_name": "iPhone 15",
  "status": "processing",
  "phase": "copying",
  "total_files": 7,
  "completed_files": 3,
  "current_file": "Episode 04.mp4",
  "total_bytes": 8388608000,
  "transferred_bytes": 5905580032,
  "progress": 70.42,
  "speed_bps": 65011712,
  "eta_seconds": 38
}
```

---

# 59. React State Model

واجهة النقل الجديدة تحتاج State مركزي منطقي:

```text
selectedFiles
selectedDevice
selectedDestination
currentRemotePath
jobId
jobStatus
progress
speed
eta
```

ولا نريد أن تصبح نافذة النسخ نفسها مسؤولة عن منطق النقل.

المنطق الحقيقي يبقى في Go.

---

# 60. زر النسخ العائم

يظهر فقط عندما:

```text
selectedFiles.length > 0
```

ويعرض:

```text
نسخ 5 ملفات
```

وعند زيادة التحديد:

```text
نسخ 12 ملفًا
```

ويمكن إضافة الحجم:

```text
نسخ 12 ملفًا • 9.4 GB
```

---

# 61. بعد بدء Job

تغلق نافذة تحديد الوجهة.

وتتحول المهمة إلى:

```text
Transfer Center / Mini Transfer Panel
```

في أسفل الشاشة.

يمكن أن تحتوي على أكثر من Job:

```text
┌──────────────────────────────┐
│ نسخ إلى iPhone 15       72%  │
├──────────────────────────────┤
│ نسخ إلى Samsung          43% │
├──────────────────────────────┤
│ نسخ إلى USB E:          100% │
└──────────────────────────────┘
```

---

# 62. حالات Job في الواجهة

```text
Waiting
Connecting
Copying
Verifying
Completed
Retrying
Waiting for device
Cancelled
Failed
```

---

# 63. UX عند فشل الجهاز

مثال:

```text
انقطع اتصال iPhone

تم نقل:
4.8 GB / 7.8 GB

سيتم الاستئناف عند إعادة توصيل الجهاز.

[إلغاء المهمة]
```

---

# 64. UX عند اكتمال النقل

```text
✓ اكتمل النسخ

7 ملفات
7.8 GB

إلى:
VLC / Documents / Series / Breaking Bad / Season 1
```

---

# 65. تصميم المجلدات في iOS

نلتزم فقط بالمساحة التي يكشفها التطبيق عبر File Sharing / Documents.

لا نفترض أن Bundle ID وحده يعني إمكانية الوصول.

التطبيق يجب أن تكون له مساحة قابلة للوصول عبر آليات iOS المستخدمة.

---

# 66. ملاحظة مهمة حول `afcclient`

النسخة الحالية upstream من `afcclient` تدعم:
- الوضع التفاعلي.
- `ls`.
- `info`.
- `mkdir`.
- `mv`.
- `rm`.
- `get`.
- `put`.
- الوصول إلى `--documents <appid>`.
- الوصول إلى `--container <appid>`.

كما أن upstream أضاف تحسينات حديثة للتعامل مع stdin عند استخدام الأداة في scripting، وأصبح بإمكانها العمل بطريقة أفضل مع إعادة التوجيه.

لذلك استخدام `afcclient` كـ persistent interactive process داخل Go مناسب لهدف "عدم إنشاء process جديد لكل ملف"، مع ضرورة بناء طبقة session/command dispatcher جيدة.

---

# 67. تقدم `afcclient`

النسخة الحالية من `afcclient` نفسها تتعامل مع `put` بقراءة الملف على دفعات وكتابة الدفعات عبر AFC، وتوفر progress محليًا للأحجام الكبيرة.

لكن بالنسبة إلى NEXORA لا يجب ربط واجهة المستخدم مباشرة بمخرجات progress النصية للأداة.

الأفضل أن يكون لدينا **Go Job Progress Model** مستقل.

---

# 68. استراتيجية Progress الموصى بها

### المستوى الأول

```text
Go knows:
source size
current operation
current transferred bytes
```

### المستوى الثاني

للاستخدام مع `afcclient` interactive:
- يقرأ Go نتائج الأمر.
- يراقب العملية.
- ويستخدم checkpoints/stat عند الحاجة للتحقق.

### المستوى الثالث

بعد استقرار المحرك:
- نزيل polling غير الضروري.
- نعتمد على أحداث نقل أفضل كلما سمحت طبقة النقل.

---

# 69. التوافق مع النسخة الحالية

لا نحذف مباشرة:

```text
copyToIOSDevice
copyToMTPDevice
copyToLocalPath
```

بل نحولها تدريجيًا إلى backends.

مثلاً:

```text
copyToIOSDevice
      ↓
IOSBackend.Put
```

و:

```text
copyToMTPDevice
      ↓
AndroidBackend.Put
```

و:

```text
copyToLocalPath
      ↓
StorageBackend.Put
```

---

# 70. هيكل الملفات الجديد المقترح

```text
server/internal/transfer/
│
├── service.go
├── models.go
├── errors.go
│
├── engine/
│   ├── engine.go
│   ├── scheduler.go
│   ├── worker.go
│   ├── progress.go
│   ├── retry.go
│   ├── resume.go
│   └── verify.go
│
├── sessions/
│   ├── manager.go
│   └── afc_session.go
│
├── ios/
│   ├── backend.go
│   ├── filesystem.go
│   ├── apps.go
│   └── paths.go
│
├── android/
│   ├── backend.go
│   └── mtp.go
│
├── storage/
│   └── backend.go
│
└── tests/
    ├── engine_test.go
    ├── scheduler_test.go
    ├── retry_test.go
    ├── resume_test.go
    └── paths_test.go
```

---

# 71. أهم Interfaces

## TransferBackend

```go
type TransferBackend interface {
    Connect(ctx context.Context, device Device, destination TransferDestination) error
    List(ctx context.Context, path string) ([]RemoteEntry, error)
    Stat(ctx context.Context, path string) (RemoteEntry, error)
    Mkdir(ctx context.Context, path string) error
    Put(ctx context.Context, source, destination string, opts PutOptions) error
    Delete(ctx context.Context, path string) error
    Rename(ctx context.Context, oldPath, newPath string) error
    Close() error
}
```

---

# 72. PutOptions

```go
type PutOptions struct {
    BufferSize   int64
    ResumeOffset int64
    Overwrite    bool

    OnProgress func(transferred int64)
}
```

---

# 73. Transfer Engine

```go
type TransferEngine struct {
    scheduler *Scheduler
    devices   *DeviceManager
    sessions  *SessionManager

    config TransferConfig
}
```

---

# 74. Transfer Config

```go
type TransferConfig struct {
    BufferSize int64

    MaxRetries int

    RetryInitialDelay time.Duration
    RetryMaxDelay     time.Duration

    VerifyMode VerifyMode

    PerDeviceConcurrency int
}
```

البداية:

```text
BufferSize = 4 MB
MaxRetries = 3
PerDeviceConcurrency = 1
```

ثم Benchmark وتعديل القيم.

---

# 75. عدم التوازي داخل AFC Session

الـ Session الواحدة تحتوي على:

```go
Mu sync.Mutex
```

ويتم تنفيذ command واحد في كل لحظة.

مثال:

```text
Session
  │
  ├── ls      ← running
  │
  └── put     ← waits
```

ولا يحدث:

```text
put A
put B
```

في نفس الوقت داخل Session واحدة.

---

# 76. تحسين أداء قراءة المجلدات

بدلاً من:

```text
Start process
ls
exit

Start process
ls
exit
```

نستخدم:

```text
Persistent session

ls
ls
ls
mkdir
ls
```

وهذا أيضًا يسرع Folder Browser وليس النسخ فقط.

---

# 77. التعدد المطلوب في المستخدم

المستخدم قد ينفذ:

### العملية الأولى

```text
5 حلقات → iPhone VLC
```

### العملية الثانية

```text
فيلم → iPhone VLC
```

### العملية الثالثة

```text
سلسلة كاملة → Samsung
```

وهذه كلها تدخل Scheduler.

---

# 78. مثال على Scheduler حقيقي

```text
Queue

Job A → iPhone 1
Job B → iPhone 1
Job C → iPhone 2
Job D → USB E:
Job E → iPhone 1
```

التنفيذ:

```text
iPhone 1:
A → B → E

iPhone 2:
C

USB E:
D
```

في نفس الوقت.

---

# 79. منع تضارب المهام

إذا كانت مهمتان تستهدفان نفس الملف:

```text
Job A → VLC/Movies/video.mp4
Job B → VLC/Movies/video.mp4
```

يجب أن يملك Scheduler/Backend سياسة تضارب.

مثلاً:

```text
queue
or
reject
```

ولا يسمح لنقلين بالكتابة في نفس المسار في الوقت نفسه.

---

# 80. Device Lock

كل جهاز لديه:

```go
DeviceWorker
```

وهو المسؤول عن ترتيب المهام لذلك الجهاز.

هذا يمنع:

```text
Job A وJob B
```

من العبث بنفس Session.

---

# 81. أخطاء الإلغاء مقابل أخطاء الاتصال

يجب عدم اعتبار:

```text
context.Canceled
```

فشلًا.

الواجهة يجب أن تظهر:

```text
Cancelled
```

وليس:

```text
Failed
```

---

# 82. سجلات النقل

كل Job يحتوي على:
- StartedAt.
- CompletedAt.
- Error.
- RetryCount.
- Device.
- Destination.
- TotalBytes.
- TransferredBytes.

هذا يجعل صفحة سجل النقل الحالية أفضل.

---

# 83. Logging

يتم استخدام logs منظمة في Go:

```text
transfer.start
transfer.connect
transfer.file.start
transfer.retry
transfer.device_lost
transfer.resume
transfer.verify
transfer.complete
```

مع:

```text
job_id
device_id
file_id
```

بدون تسجيل بيانات حساسة غير ضرورية.

---

# 84. اختبار الأداء

نحتاج Benchmark منفصل.

## Buffer benchmark

اختبار:

```text
1 MB
2 MB
4 MB
8 MB
16 MB
32 MB
```

على:

```text
500 MB
2 GB
10 GB
```

---

# 85. مقاييس Benchmark

نسجل:

```text
Throughput MB/s
CPU %
RAM MB
Total time
Retries
Disconnect recovery
Verification time
```

---

# 86. اختبار الأجهزة

يجب اختبار:

```text
iPhone حديث
iPhone أقدم
Android Samsung
Android Xiaomi
Android Huawei
USB SSD
USB HDD
```

قدر الإمكان ضمن الأجهزة المتوفرة.

---

# 87. اختبار الملفات

```text
100 MB
500 MB
1 GB
5 GB
10 GB
20 GB
```

وملفات:
- عربي.
- إنجليزي.
- أسماء طويلة.
- مسارات عميقة.
- مسافات.
- رموز.
- Unicode.

---

# 88. اختبارات الأخطاء

يجب محاكاة:

```text
Device disconnect
USB reconnect
Destination missing
Destination full
Source missing
Permission denied
Existing file
Partial file
Cancel during copy
Cancel during retry
```

---

# 89. اختبارات Multi-job

مثال:

```text
3 Jobs
نفس الجهاز
```

ثم:

```text
3 Jobs
3 أجهزة
```

ثم:

```text
10 Jobs
3 أجهزة
```

للتأكد من:
- Queue.
- ترتيب الأولوية.
- عدم وجود race conditions.
- عدم تسرب الموارد.

---

# 90. اختبارات Session

يجب التأكد من:

```text
Start session
ls
mkdir
put
put
put
ls
close
```

دون إعادة تشغيل `afcclient` بين الأوامر.

---

# 91. Recovery

في حال توقف `afcclient`:

```text
Session broken
 ↓
detect
 ↓
close
 ↓
recreate
 ↓
resume/retry
```

والمحرك لا ينهار.

---

# 92. إدارة المصادر

يجب التأكد من إغلاق:
- ملفات المصدر.
- handles.
- stdin.
- stdout.
- stderr.
- process.
- sessions.

عند:
```text
complete
failed
cancelled
device disconnected
```

---

# 93. التوافق مع Windows

يستمر المشروع باستخدام:

```text
local_path_windows.go
```

للمشكلات الخاصة بمسارات Windows.

لكن يجب تخفيض الاعتماد على temporary staging متى ما أمكن.

---

# 94. مسار التطوير المقترح

## المرحلة 1 — Refactor فقط

لا نغير UX.

نحول:

```text
service.go
```

إلى:
- models.
- engine.
- backend.
- scheduler.

الهدف:
> المحافظة على السلوك الحالي.

---

# 95. المرحلة 2 — Persistent AFC Session

نضيف:

```text
AFCSession
SessionManager
CommandDispatcher
```

ونستخدم `afcclient` interactive.

الهدف:
> لا process جديد لكل ملف أو كل أمر.

---

# 96. المرحلة 3 — File Browser

نضيف:
- lazy folder listing.
- create folder.
- current path.
- remote stat.
- navigation.

---

# 97. المرحلة 4 — Multi-file Job

ندعم:

```text
files[]
```

داخل Job.

---

# 98. المرحلة 5 — Scheduler

نضيف:
- device workers.
- per-device queues.
- concurrent devices.
- sequential jobs per device.

---

# 99. المرحلة 6 — Progress

نضيف:
- accurate total bytes.
- current file.
- speed.
- ETA.
- overall progress.

---

# 100. المرحلة 7 — Retry

نضيف:
- retry policy.
- classified errors.
- exponential backoff.

---

# 101. المرحلة 8 — Resume

نضيف:
- checkpoint.
- remote size check.
- resume offset.
- reconnect.

---

# 102. المرحلة 9 — UX الجديدة

نبني:

```text
Selection
 ↓
Floating Copy Button
 ↓
Transfer Modal
 ↓
Device selection
 ↓
App/File Browser
 ↓
Path selection
 ↓
Copy
 ↓
Mini Transfer Center
```

---

# 103. المرحلة 10 — Performance Benchmark

نقيس:

```text
Current Engine
vs
Transfer Engine v2
```

وهدفنا إثبات التحسن بالأرقام.

---

# 104. الترتيب النهائي للأولوية

| الأولوية | الميزة |
|---|---|
| P0 | Persistent AFC session |
| P0 | Multi-file Job |
| P0 | Device Scheduler |
| P0 | Folder navigation |
| P0 | Real progress model |
| P1 | Retry |
| P1 | Resume |
| P1 | Create folder |
| P1 | Conflict handling |
| P1 | New Copy Modal |
| P2 | Performance tuning |
| P2 | Strict hash verification |
| P2 | Advanced queue priority |

---

# 105. النتيجة المعمارية المستهدفة

```text
                         NEXORA
                            │
                         React
                            │
                     Floating Copy
                            │
                      Transfer Modal
                            │
             ┌──────────────┴──────────────┐
             │                             │
        Device Browser              Selected Files
             │                             │
             └──────────────┬──────────────┘
                            │
                       REST API
                            │
                     Go Transfer Service
                            │
                    Transfer Engine v2
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
      Scheduler        Session Manager    Job Manager
          │                 │                 │
      ┌───┼───┐             │                 │
      │   │   │             │                 │
      ▼   ▼   ▼             ▼                 ▼
    iOS Android USB      AFC Session      Progress/Retry
      │     │      │         │
      │     │      │         ▼
      │     │      │      afcclient
      │     │      │       interactive
      │     │      │         │
      └─────┴──────┴─────────┘
                │
             Device
```

---

# 106. المبادئ التي لا يجب كسرها

## 1

**لا نضيف لغة جديدة.**

Go هو قلب Transfer Engine.

## 2

**لا نضيف Framework جديد.**

React الحالي يبقى.

## 3

**لا نبدأ بإعادة بناء كل شيء.**

التطوير تدريجي.

## 4

**الجهاز هو وحدة الجدولة.**

كل جهاز Worker مستقل.

## 5

**Session واحدة طويلة العمر لكل Device/App في iOS.**

## 6

**لا نستخدم Polling كحل أساسي للتقدم إذا استطعنا معرفته من طبقة النقل.**

## 7

**لا نقرأ الملف كاملًا إلى RAM.**

## 8

**الوجهة يحددها المسار الحالي الذي اختاره المستخدم.**

## 9

**Multi-file Job هو الأساس.**

## 10

**فصل UI عن Transfer Engine.**

---

# 107. تصور تجربة المستخدم من البداية للنهاية

```text
المستخدم يحدد:

☑ الحلقة 1
☑ الحلقة 2
☑ الحلقة 3
☑ الحلقة 4

الإجمالي:
4 ملفات • 5.2 GB

           ↓

يظهر زر:
[ نسخ 4 ملفات ]

           ↓

يفتح Transfer Modal

┌───────────────────────────────┐
│ الأجهزة │ التطبيقات/الملفات  │
│         │                      │
│ iPhone  │ VLC                  │
│ Android │ Infuse               │
│ USB     │ Documents            │
└───────────────────────────────┘

           ↓

المستخدم يضغط:
VLC → عرض

           ↓

يختار:

Documents
→ Series
→ Breaking Bad
→ Season 1

           ↓

يضغط:

[ نسخ 4 ملفات ]

           ↓

تغلق النافذة

           ↓

يظهر Mini Transfer Center

┌───────────────────────────────┐
│ Copying → iPhone 15           │
│ 63%                           │
│ 3.2 GB / 5.2 GB               │
│ 61 MB/s • 00:32               │
└───────────────────────────────┘

           ↓

إذا انقطع iPhone:

┌───────────────────────────────┐
│ الجهاز غير متصل               │
│ تم الاحتفاظ بالتقدم           │
│ سيتم الاستئناف عند الاتصال    │
└───────────────────────────────┘

           ↓

يعود الهاتف

           ↓

Reconnect → Resume

           ↓

✓ اكتمل النسخ
```

---

# 108. القرار النهائي

الخطة الأساسية لـ NEXORA هي:

```text
Go-only Transfer Engine
+
React UX الحالية المطورة
+
libimobiledevice tools الموجودة
+
Persistent interactive afcclient session
+
Per-device worker
+
Multi-file jobs
+
Folder browser
+
Create folder
+
Progress
+
Speed
+
ETA
+
Retry
+
Resume
+
Cancel
+
Verification
+
Benchmark-driven buffering
```

ولا نحتاج في هذه المرحلة لإدخال لغة أو Framework جديد.

الخطوة التقنية الأولى بعد اعتماد هذه الخطة هي **استخراج Transfer Engine من `service.go` إلى طبقات مستقلة مع الإبقاء على API الحالية، ثم بناء `AFCSession` طويل العمر داخل Go واختباره على iPhone حقيقي قبل تغيير واجهة المستخدم**.

---

# 109. مصادر العمل

## مصدر المشروع الحالي

الوثيقة الأساسية التي تم تحليلها:

`USB_TRANSFER_SYSTEM_AR.md`

وهي تصف البنية الحالية، ملفات Go، REST API، أدوات libimobiledevice، تدفق iOS، تدفق Android، واختبارات النقل الحالية.

## libimobiledevice / afcclient

الوثائق والمصدر الرسميان المستخدمان للتحقق من وضع `afcclient` الحالي:

- libimobiledevice
- `afcclient` manual
- `tools/afcclient.c`

يجب مراجعة النسخة الموجودة فعليًا داخل `.tools` في NEXORA قبل الدمج، لأن بعض سلوكيات CLI قد تختلف حسب build/version.

---

# 110. معيار النجاح

لا يعتبر Transfer Engine v2 مكتملًا لمجرد أن النسخ يعمل.

يجب أن يحقق:

### وظيفيًا

- اختيار عدة ملفات.
- اختيار عدة حلقات.
- اختيار المسار.
- تصفح الملفات والمجلدات.
- إنشاء مجلد.
- تنفيذ عدة Jobs.
- دعم عدة أجهزة.
- Cancel.
- Retry.
- Resume.

### أداءً

- عدم إنشاء process جديد لكل ملف في iOS.
- استخدام Session طويلة العمر.
- Buffer قابل للضبط.
- Streaming.
- Memory ثابتة تقريبًا بالنسبة لحجم الملفات.
- Progress سريع التحديث.
- Benchmark حقيقي.

### UX

- Copy FAB صغير وحديث.
- Transfer Modal Responsive.
- أجهزة على اليمين.
- العناصر المحددة أسفل الأجهزة.
- محتوى الوجهة في المنتصف.
- زر النسخ في أسفل المنتصف.
- Mini Transfer Center أسفل الشاشة.
- عرض واضح للسرعة والحجم والتقدم وحالة الاتصال.

### معماريًا

- Go هو قلب النظام.
- React مسؤولة عن UX.
- Transfer Engine مستقل.
- Backend لكل نوع جهاز.
- Scheduler مركزي.
- Session Manager.
- Device Worker.
- Job Manager.
- Retry/Resume/Progress كطبقات مستقلة.

---

## نهاية الوثيقة
