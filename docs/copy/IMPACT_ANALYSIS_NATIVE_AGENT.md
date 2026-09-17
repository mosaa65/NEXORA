# تحليل أثر البنية الجديدة (Extension → Native Messaging → Go Agent) على النظام الحالي

> المستند: تحليل أثـر معماري قبل أي تنفيذ — بتكليف المالك («تحليل أثر على النظام الحالي»).
> التاريخ: 2026-09-09
> الحالة: **غير معتمد للتنفيذ** — وثيقة تحليل فقط.

---

## 1. خلاصة تنفيذية

البنية المستهدفة تمرر التحكم في **أقراص/USB/ملفات جهاز العميل (المتصفح)** عبر:
`Website (React) → Browser Extension → Native Messaging → Go Agent → Win32 API`.

النظام الحالي يملك بالفعل نظامًا كاملًا ومُختبَرًا لنسخ الملفات عبر USB **على جهاز الخادم** (`server/internal/transfer/*`). البنية الجديدة لا تكرر هذا النظام نظريًا بل تعالج سيناريو مختلفًا (أقراص **العميل**)، لكنها **تتقاطع معه** في الواجهة، وطبقة API، ونموذج الأمان. بدون فصل واضح سننشئ نظامين متوازيين ينتهكان `AGENTS.md §3`.

**استنتاج الأثر**: التأثير **واسع** لكنه **لا يلغي** النظام الحالي. التوافق الأمثل هو إبقاء server-side كما هو، وإضافة **قدرة client-side** قابلة للتفعيل/التعطيل، دون لمس طبقات النسخ القائمة إلا لإعادة استخدام نماذجها وأدواتها.

---

## 2. الصورة الحالية (المثبتة في الكود وليس الوثائق فقط)

### 2.1 أين يعمل النقل الحالي (Server-side)

| الطبقة | الملف | الدور |
|---|---|---|
| التركيب | `server/internal/app/app.go:75-103` | `transfer.NewService(transfer.Options{AndroidTargetFolder, IOSBundleID})` → يمرر إلى `api.NewServer` |
| الخدمة | `server/internal/transfer/service.go:21-51` | `Service{options, jobs, engine, eventBroker}`؛ محرك v2 + backends + مراقب أجهزة كل 1.5 ثانية + SSE bridge |
| البنية | `models.go:105-108` | `Options` يملك **حقلين فقط** (AndroidTargetFolder, IOSBundleID) — لا مفاهيم إضافية |
| واجهة API | `server/internal/api/server.go:303-315` | 11 مسارًا: devices/apps/folders/copy/jobs/job/cancel/events/browse/mkdir/eject |
| أحداث حيـة | `handlers_transfer.go:301-315` | `SubscribeEvents(256)` + بذر `SnapshotDevices()` + `ListJobs()` عبر SSE |
| الأقراص | `server/internal/disks/*` | `disks.Manager` (Windows/Other/مدير عام) مخصص لعرض/system browse |
| الواجهة | `client/src/context/TransferContext.jsx` + `components/transfer/*` | نافذة، مركز تقدم، زر عائم، عبر `api.js:441-527` |

**النموذج**: المتصفح → Go server (نفس جهاز الوسائط) → go-ios/MTP PowerShell/`os` copy على **أقراص موصولة بالخادم**. `discoverRemovableDrives` يعرض محركات الخادم als fallback. أي «الوصول» هنا يحدث داخل معالجة الخادم نفسها.

### 2.2 الفجوة الحقيقية التي تعالجها البنية الجديدة

- الحالية: لا يمكن للمتصفح قراءة/نسخ من أقراص **جهاز العميل** نفسه (محرك E: على حاسوب المستخدم) — لأنه sandbox المتصفح يمنع ذلك.
- الجديدة: تجعل أقراص **جهاز العميل** قابلة للوصول عبر موافقة صريحة من المستخدم داخل امتداد.

⇒ **ليست نسخة مكرره من النقل الحالي، بل قدرة جديدة.** الازدواجية المحتملة تكون فقط لو حاولنا إعادة بناء أدوات discovery/backends/SFE التي يملكها `server/internal/transfer` على جهاز العميل.

---

## 3. الأثر على كل منطقة (Affected Systems)

### 3.1 `server/internal/transfer/*` — الأثر: **صفر/منخفض**
- لا حاجة لتعديل backends أو المحرك أثناء إضافة client-side Agent.
- الأثر المحتمل الوحيد: مشاركة نماذج (models/errors/progress) أو إعادة استخدام `TransferBackend` إذا كانت العمليات مشتركة — إعادة استخدام اختيارية، ليست إلزامية.

### 3.2 `server/internal/api/server.go` + `handlers_transfer.go` — الأثر: **مرتفع إذا لم نفصل**
- إذا انتقلنا «كل نسخ من أي جهاز عبر API واحد» سنضطر لتوسيع CopyRequest/Naming/إجهاز → تعقيد عالي وخطر كسر الواجهة الحالية.
- **مقترح**: لا تلمس هذه endpoints؛ الـ Agent يتصل مستقلًا بقناة Native Messaging ولا يمر عبر Go API الحالي.

### 3.3 `server/internal/config` + `app.go` — الأثر: **منخفض**
- لا تغيير مطلوب لخادم NEXORA لوجود الـ Agent؛ الـ Agent تطبيق مستقل خارج سيرفر الوسائط.

### 3.4 الواجهة `client/src/context/TransferContext.jsx` + `api.js` + `components/transfer/*` — الأثر: **مرتفع (الأعلى)**
- المصدر الحالي للأجهزة/الوظائف هو SSE من الخادم. البنية الجديدة تضيف **مصدر أجهزة منفصل** (أقراص العميل).
- ستظهر قائمتان: «أجهزة الخادم» (الموجودة) و«أجهزة العميل» (الجديدة) — يجب فصل السياقات أو دمجها بحذر مع معرّفات IDs مختلطة (حاليًا `ios_*`,`mtp_*`,`disk_E`) لتجنب تعارض أسماء `disk_E` بين الخادم والعميل.

### 3.5 الأمان / الطريق السطحي الجديد — الأثر: **جديد وعالي**
- موقع خارجي يجب ألا يتحكم بالـ Agent (Origin/Token/Session/Signing/RateLimit/Permissions) حسب اشتراطات المالك.
- Native Host بالتسجيل في registry يضع Agent على جهاز العميل — عملية تثبيت وتحديث خارج نطاق git/NPM الحالي.

### 3.6 التوثيق — الأثر: **قائم ومطلوب**
- `USB_COPY_SYSTEM_ANALYSIS.md`, `COPY_UI_ARCHITECTURE.md`, `SOLVED_ISSUES.md`, `ARCHITECTURE.md`, `PROJECT_ANALYSIS.md`, `ENDPOINTS.md` ستُتحدّث بعد أي قرار/تنفيذ.

---

## 4. الخيارات (Options)

| الخيار | الوصف | الأثر | التوصية |
|---|---|---|---|
| **أ** — Agent مستقل تمامًا (قناة منفصلة) | Extension ↔ Native Messaging ↔ Agent؛ لا يمر بالـ Go API؛ الواجهة مضاف إليها «أقراص العميل» | منخفض على الخادم، مرتفع على الواجهة؛ لا ازدواجية مع نقل USB الحالي | ✅ **مفضل** |
| **ب** — Agent وكيل للخادم (يخدم أقراص الخادم عن بعد) | يتكرر به عمل `server/internal/transfer` على جهاز إضافي | ازدواجية عالية، صيانة مزدوجة، تعارض مع §3 | ❌ رفض |
| **ج** — محو النظام الحالي واستبداله بالكامل بالـ Agent | إزالة server-side transfer | خطر كسر ميزات مختبرة وSSE وAndroid/iOS | ❌ رفض |

---

## 5. التوصية

1. **الاحتفاظ** بنظام النقل الحالي (server-side) كاملًا كما هو.
2. **بناء Agent كتطبيق مستقل** (Go، قناة Native Messaging، أمان الملكية المذكورة) يخدم **أقراص العميل فقط**.
3. **في الواجهة**: إضافة سياق/مصدر «أقراص العميل» بشكل معزول، مع تفادي تعارض IDs عبر بادئة معرّف مميزة (مثل `nm_`/`agent_` قبل `disk_E`).
4. **عدم توسيع** `CopyRequest`/routes الحالية لكل العمليات الجديدة قبل قرار صريح.
5. **قبل أي PoC**: وثيقة أمان صريحة (Origin/Token/Session/Signing/RateLimit/Permissions) تمنع أي موقع خارجي من الاتصال بالـ Agent.

---

## 6. حدود لا يُتعدى (لا تنفيذ الآن)

- لا تعديل `server/internal/transfer/*` ولا `handlers_transfer.go` ولا `server.go:303-315`.
- لا إنشاء نظام ملفات/agent موازٍ داخل `server/` قبل موافقة المالك.
- لا كتابة كود تنفيذ (PoC) — هذه الوثيقة تحليلية فقط.

---

## 7. توصية للخطوة التالية (تحتاج موافقة المالك)

كتابة `docs/decisions/ADR-008-native-messaging-agent.md` أو خطة تطوير تفصيلية بمراحل (شبيهة بخطة v2) إذا قرر المالك اعتماد الخيار (أ).

---

*نهاية التحليل.*