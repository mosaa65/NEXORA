# ملفات تصميم واجهات النسخ في NEXORA — بنية وتدفق البيانات

> شرح معمّق لملفات الواجهة (UI) الخاصة بنظام النسخ عبر USB:
> كل شاشة/مكوّن، مصدر بياناتها، وكيف تُرسل الأوامر إلى الخادم والعكس.
> يُقرأ مع `USB_COPY_SYSTEM_ANALYSIS.md` (التحليل الكامل) و `SOLVED_ISSUES.md` (المشاكل المحلولة).

---

## 1. الخريطة العامة (تدفق البيانات من البداية للنهاية)

```
┌────────────────────────── طبقة العرض (React) ──────────────────────────┐
│                                                                         │
│  MediaDetailsPage / AdminTransferPage / FloatingCopyButton              │
│        │  openTransferModal(file, title) — جمع ملفات مختارة            │
│        ▼                                                                │
│  TransferModal.jsx  ──► startTransfer({deviceId, targetApp, ...})      │
│        │                                          │                     │
│        │  يعرض بواسطة context                    │ POST /api/transfer/copy
│        ▼                                          ▼                     │
│  MiniTransferCenter.jsx ◄── TransferContext  ──► lib/api.js ──► HTTP    │
│        ▲                             │                 │                │
│        └── SSE events (jobs/job) ◄───┘                 ▼                │
│                                                       الخادم (API)      │
└─────────────────────────────────────────────────────────────────────────┘
```

**قاعدة أساسية:** كل «قراءة» حية (وظائف، أجهزة) تأتي عبر **بث SSE واحد** من الخادم، وكل
«أمر» (بدء نسخ، إلغاء، إنشاء مجلد) يُرسل عبر **HTTP** صريح. هذا هو تصميمنا بعد حل مشكلة
تجمد النسبة — لا يوجد polling للبيانات الحيوية (باستثناء صفحتين ثانويتين مذكورتين لاحقاً).

---

## 2. الملفات بالتفصيل

### 2.1 `client/src/context/TransferContext.jsx` — الدماغ المركزي

**الدور:** يحتوي كل حالة النسخ المشتركة بين الشاشات، ويحافظ على اتصال SSE الوحيد مع الخادم.

**الحالة (state) التي يحملها:**
| الحالة | المعنى |
|---|---|
| `selectedFiles` | الملفات المختارة للنسخ (معولمة: `id`, `filePath`, `title`, `size`, `resolution`, `mediaTitle`, `poster`) |
| `isTransferModalOpen` | هل نافذة النسخ مفتوحة |
| `activeJobs` | قائمة الوظائف الحالية (تتحدث بالبث) |
| `devices` | قائمة الأجهزة المتصلة (تتحدث بالبث) |
| `isCenterExpanded` | توسيع/طي مركز التقدم |
| `centerDismissed` | هل أُغلق المركز يدوياً (يُعاد إظهاره تلقائياً لنسخة جديدة) |

**الاتصال (من أين تأتي البيانات):**
- `EventSource` واحد على `/api/transfer/events` يُفتح عند تحميل التطبيق ويُغلق عند الإنهاء.
- ثلاثة أحداث مسماة:
  - `jobs` → لقطة كاملة للوظائف (تُعرض كما هي).
  - `job` → دمج وظيفة واحدة حسب `id` (إضافة/تحديث).
  - `devices` → لقطة كاملة للأجهزة المرتبة.
- عند انقطاع الاتصال يجري المتصفح إعادة توصيل تلقائية، والخادم يعيد بذر اللقطات.

**الدوال (كيف تُرسل/تُعرض):**
| الدالة | المهمة |
|---|---|
| `openTransferModal(file, mediaTitle, poster)` | فتح النافذة بملف واحد أو بالحالة الحالية |
| `toggleFileSelection` / `selectMultipleFiles` / `removeFileFromSelection` / `clearSelection` | إدارة قائمة الملفات المختارة |
| `startTransfer({deviceId, targetApp, targetFolder, subFolder, files})` | بناء `payload` واستدعاء `startDeviceTransfer` ثم `fetchJobs()` وتوسعة المركز |
| `cancelJob(jobId)` | استدعاء `cancelTransferJob` ثم تحديث القائمة |
| `refreshJobs` | جلب يدوي لقائمة الوظائف (HTTP مباشر) |

**قيمة الـ context المعرّضة:** كل الحالات السابقة + `hasRunningJobs` + `setIsCenterExpanded`
في `value` واحد يستهلكه أي مكوّن عبر `useTransfer()`.

---

### 2.2 `client/src/components/transfer/TransferModal.jsx` — النافذة الرئيسية للنسخ

**الدور:** نافذة كاملة (Full-screen modal) لاختيار جهاز ووجهة، ثم بدء النسخ.

**متى تُفتح (من يستدعي `openTransferModal`):**
- `MediaDetailsPage.jsx` — زر قص/نسخ لكل ملف أو حلقة (أسطر 599، 657) أو زر عام (424-426).
- `AdminTransferPage.jsx` — زر نسخ لأي عمل فهرسي.
- `FloatingCopyButton.jsx` — زر عائم عند وجود ملفات مختارة.

**مصادر البيانات داخل النافذة:**
| البيان | المصدر |
|---|---|
| قائمة الأجهزة | `devices` من الـ context (أحداث SSE `devices`) — تحديث لحظي (فصل/وصل) |
| زر «تحديث» | `getTransferDevices()` — HTTP حقيقي واحد لفحص فوري |
| تطبيقات iPhone الداعمة لـ File Sharing | `getTransferDeviceApps(deviceID)` (حينما يكون الجهاز iOS) |
| مجلدات التطبيق المفتوح على iOS | `browseTransferPath(deviceID, "/Documents", "ios", bundleID)` |
| تصفح مجلدات جهاز التخزين | `browseTransferPath(deviceID, path, "storage"|…)` |
| إنشاء مجلد على الجهاز | `createTransferFolder({device_id, path, device_type, bundle_id})` |

**كيف تُرسل النسخ:** `handleStartCopy` تبني الوجهة حسب نوع الجهاز ثم تستدعي `startTransfer`:

| نوع الجهاز | الوجهة المرسلة |
|---|---|
| iOS | `targetApp = bundleID` + `subFolder = currentPath` (المسار داخل Documents) |
| وحدات تخزين (flash) | `targetFolder = currentPath` |
| Android | `targetFolder ∈ {Movies, Download, DCIM}` + `subFolder` (اسم مجلد فرعي) |

**حالة الصفحة داخل النافذة** (ثلاث «وجهات نظر»):
1. **iOS + تطبيق محدد** → تصفح مجلدات التطبيق (شجرة مجلدات مع «مجلد جديد»).
2. **iOS بدون تصفح** → قائمة بطاقات التطبيقات + إدخال Bundle ID يدوي.
3. **تخزين USB** → عرض جاهز للنسخ المباشر للمسار.
4. **Android** → شبكة مجلدات وجهة + حقل المجلد الفرعي.

---

### 2.3 `client/src/components/transfer/FloatingCopyButton.jsx` — الزر العائم

**الدور:** زرار يظهر عند وجود ملفات مختارة (من صفحات التفاصيل) ويوفر «فتح نافذة النسخ»
و«مسح التحديد». يقرأ فقط من الـ context ولا يرسل شيئاً بنفسه.

---

### 2.4 `client/src/components/transfer/MiniTransferCenter.jsx` — مركز التقدم العائم

**الدور:** الشاشة التي تعرض التقدم أثناء النسخ (أسفل الشاشة، يمين/يسار).

**مصدر البيانات:** `activeJobs` **فقط** من الـ context (محدث بالبث). لا يجلب أي HTTP.

**كيف يعرض:**
- `relevantJobs` = وظائف الحالات: `processing/pending/queued/retrying/waiting_device/completed/failed`.
- `primaryJob` = أول وظيفة غير منتهية (أو أول وظيفة) → شريط التقدم العلوي المصغّر + نسبة + سرعة + ETA.
- القائمة الموسّعة: لكل وظيفة شريط + ملف `finished_files/total_files` + مسار الوجهة + الخطأ إن وُجد.
- زر «إلغاء» → `cancelJob(id)` من الـ context.
- زر ✕ → `setCenterDismissed(true)` (يُخفى؛ يُعاد تلقائياً عند أي وظيفة جديدة).

---

### 2.5 `client/src/components/CopyToPhoneModal.jsx` — النافذة المستقلة (مرجعية)

**الدور:** نافذة نسخ منفصلة تابعة لبطاقات الوسائط المباشرة (قبل دمجها في نافذة النسخ الموحدة).

> **وضعها الحالي:** غير مستوردة من أي ملف آخر (لا يوجد `import` لها في الشيفرة) — مكوّن
> مرجعي/جوهري محتفظ به. احتفظت به وتعمل كاملة لكنها لا تستخدم الـ context ولا البث.

**تتعامل مع البيانات بنفسها:**
- الأجهزة: `getTransferDevices()` + إعادة فحص كل 2.5 ثانية فقط إن كانت القائمة فارغة.
- التطبيقات/المجلدات: `getTransferDeviceApps` + `getTransferAppFolders(deviceID, bundleID)`.
- تتبع الوظيفة: **polling** عبر `getTransferJob(activeJob.id)` كل 500 ms حتى الوصول لحالة نهائية.
- الإرسال: `startDeviceTransfer(payload)` مع `file_id` / `source_path` / `target_app` /
  `target_folder` / `sub_folder`، والإلغاء عبر `cancelTransferJob`.

> بما أن إصلاح الكاش في `lib/api.js` استثنى كل `/api/transfer/`، فإن هذا الـ polling يقرأ دائماً
> بيانات حديثة — لا مشكلة تجمد، لكنها ليست لحظية كالبث.

---

### 2.6 `client/src/pages/admin/AdminTransferPage.jsx` — صفحة إدارة النسخ

**الدور:** لوحة تحكم (مسار `transfer` في المتصفح) لأجهزة الزبائن والوظائف.

**مصادر البيانات (كلها HTTP):**
```js
Promise.allSettled([getTransferDevices(), getTransferJobs(), getMediaList({ limit: 20 })])
```
- `getTransferDevices()` ← القائمة المعروضة (ممررة عبر `normalizeTransferDevices`).
- `getTransferJobs()` ← جدول الوظائف + أزرار إلغاء.
- `getMediaList()` ← كتالوج للأعمال، وكل عمل له زر يفتح `openTransferModal(file, title)`.
- **تحديث:** يدوي (زر) + تلقائي كل 3 ثوانٍ (`setInterval(refreshData, 3000)`).

---

### 2.7 `client/src/lib/api.js` — طبقة الاتصال بالخادم

**الدور:** كل الطلبات HTTP + الاشتراك/الكاش. (لا تعرض — تمثل وسطاء فقط).

**دوال قسم النسخ:**

| الدالة | الطريقة/المسار | الوصف |
|---|---|---|
| `getTransferDevices()` | `GET /api/transfer/devices` | قائمة الأجهزة الموصولة |
| `getTransferDeviceApps(id)` | `GET /api/transfer/device-apps?device_id=` | تطبيقات iOS الداعمة لـ File Sharing |
| `getTransferAppFolders(id, bundle)` | `GET /api/transfer/device-app-folders?...` | شجرة مجلدات التطبيق iOS |
| `startDeviceTransfer(payload)` | `POST /api/transfer/copy` | بدء وظيفة نسخ وإرجاع `{ok, job}` |
| `getTransferJobs()` | `GET /api/transfer/jobs` | كل الوظائف (الأحدث أولاً) |
| `getTransferJob(id)` | `GET /api/transfer/job/{id}` | وظيفة واحدة |
| `cancelTransferJob(id)` | `POST /api/transfer/cancel/{id}` | إلغاء وظيفة |
| `browseTransferPath(...)` | `GET /api/transfer/browse?...` | قائمة مجلدات/ملفات بعيدة (Phase 3) |
| `createTransferFolder(payload)` | `POST /api/transfer/mkdir` | إنشاء مجلد على الجهاز |
| `resolveAPIURL(path)` | — | تكوين مسار كامل (يُستخدم لبناء SSE URL) |

**ملاحظة كاش:** بعد إصلاح المشكلة 1، كل `/api/transfer/*` خارج الكاش، و`cache:"no-store"` محترم —
يعني كل قراءات النسخ طازجة دائماً.

---

### 2.8 `client/src/lib/transferDevices.js` — محوّل قوائم الأجهزة

**الدور:** تحويل مصفوفة أجهزة الخادم إلى قائمة عرض مرتبة.

- `isRealIOSDevice(d)` — iOS حقيقي (`type=ios` و id يبدأ بـ `ios_` وليس `ios_mtp_`).
- `isWindowsIOSPlaceholder(d)` — كشف أجهزة iOS الوهمية/الاحتياطية الناتجة عن MTP على Windows
  (id يبدأ بـ `ios_mtp_` أو باسم يحتوي iphone/ipad/apple).
- `normalizeTransferDevices(devices)` — يحذف الأجهزة الوهمية **فقط عندما** يوجد iOS حقيقي،
  ثم يرتب: iOS → Android → Storage → احتياطية.

**مستخدمة من:** `TransferModal` و `AdminTransferPage` و `CopyToPhoneModal`.

---

## 3. من أين تأتي البيانات (جدول القراءة الشامل)

| ما نعرضه | في أي واجهة | المصدر (الأنسب) |
|---|---|---|
| قائمة الأجهزة اللحظية | TransferModal / مركز | أحداث SSE `devices` (بذر عند الاتصال + عند التغير) |
| الزر «تحديث الأجهزة» | TransferModal | `getTransferDevices()` (HTTP فوري) |
| قائمة الوظائف | مركز التقدم / Admin | أحداث SSE `jobs` (لقطة عند الاتصال) |
| تقدم وظيفة واحدة | مركز التقدم | أحداث SSE `job` (أو `getTransferJob(id)` لـ CopyToPhoneModal) |
| تطبيقات iOS | TransferModal / CopyToPhoneModal | `getTransferDeviceApps(deviceID)` |
| مجلدات تطبيق iOS | TransferModal | `browseTransferPath(...)` أو `getTransferAppFolders` |
| محتوى مجلد بعيد | TransferModal (Browser) | `browseTransferPath(deviceID, path, type, bundleID)` |
| كتالوج الوسائط | AdminTransferPage | `getMediaList` + `getMediaDetail` |

---

## 4. كيف تُرسل البيانات (جدول الإرسال الشامل)

| الإجراء | الواجهة التي تنفذه | الطلب |
|---|---|---|
| بدء نسخ | TransferModal → `startTransfer` | `POST /api/transfer/copy` |
| إلغاء وظيفة | MiniTransferCenter / CopyToPhoneModal / Admin | `POST /api/transfer/cancel/{id}` |
| إنشاء مجلد | TransferModal (Browser) | `POST /api/transfer/mkdir` |
| اختيار/إلغاء اختيار ملف | TransferModal (محلي/context) | لا طلب — حالة محلية فقط |

**صيغة `POST /api/transfer/copy` (payload):**
```json
{
  "device_id": "ios_xxxx или 012345…",
  "source_path": "D:\\Media\\Movie.mkv",
  "source_paths": ["D:\\Media\\Movie.mkv", "..."],
  "file_id": 0,
  "target_app": "org.videolan.vlc-ios",
  "target_folder": "Movies",
  "sub_folder": "Game of Thrones S01"
}
```

**استجابة `POST /api/transfer/copy`:**
```json
{ "ok": true, "job": { "id": "job_…", "status": "queued", "progress": 0, … } }
```

**صيغة أحداث SSE (الخادم → العميل):**
```
event: job
data: {"type":"job","job":{"id":"job_…","status":"processing","phase":"copying","progress":42.5,"speed_bps":123456,"eta_seconds":18,"current_file":"Movie.mkv"}}

event: devices
data: {"type":"devices","devices":[{"id":"ios_…","name":"iPhone","type":"ios",…}]}
```

---

## 5. خريطة الشاشات → السياق → الواجهة

```
MediaDetailsPage ──┬── openTransferModal(file)
AdminTransferPage ─┤── openTransferModal(file, title)
FloatingCopyButton ┘        │
                            ▼
                 TransferContext (حداث SSE + startTransfer/cancelJob/jobs/devices)
                            │                    │
                            │ family             │ family
                            ▼                    ▼
                 TransferModal        MiniTransferCenter
                 (نافذة النسخ عمودياً)   (شريط التقدم العائم)
                            │
                            └── lib/api.js ──► الخادم
```

---

## 6. ملاحظات ونقاط مطابقة (أين تلتقي الأسماء)

1. حالة الوظيفة في الخادم مرتبطة بأسماء ثابتة تستخدمها الواجهة: `queued / pending /
   processing / retrying / waiting_device / connecting / checking_destination /
   copying / verifying / completed / failed / cancelled`.
2. أسماء حقول JSON: `Device` (id, name, type, model, status), `TransferJob`
   (id, status, phase, progress, transferred, transferred_bytes, total_bytes,
   speed_bps, speed_mbps, eta_seconds, current_file, file_name, file_size,
   file_count, total_files, finished_files, completed_files, destination_path, error).
3. تحويل «أجهزة الخادم → ما يعرضه المستخدم» يتم دائماً عبر `normalizeTransferDevices`
   في كل واجهة تعرض أجهزة.
4. عند تغيير نافذة/وحدة؟ كل المكوّنات مركّبة على مستوى `App.jsx` مرة واحدة داخل
   `<TransferProvider>` فيتقاسمون حالة واحدة دون إعادة تثبيت.