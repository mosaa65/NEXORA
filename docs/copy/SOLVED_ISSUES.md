# المشاكل المحلولة في قسم النسخ عبر USB — توثيق مفصل

> هذا الملف يوثق كل مشكلة واجهتنا أثناء بناء/إصلاح قسم النسخ عبر USB (USB Copy)
> في NEXORA، مع سببها الجذري وخطوات الحل والملفات/الأسطر المعنية والتحقق.
> يقرأ مع `USB_COPY_SYSTEM_ANALYSIS.md` للفهم العام للنظام.

---

## ملخص سريع (جدول المشاكل)

| # | المشكلة / العرض | السبب الجذري | الحل |
|---|---|---|---|
| 1 | نسبة النسخ تتجمد (تبقى 1% رغم اكتمال النسخ) | كاش العميل يخزّن GETs للنقل 120 ثانية ويتجاهل `no-store` | استثناء `/api/transfer/` + احترام `no-store` في `api.js` |
| 2 | بعد النسخة الأولى، النسخة الثانية تُظهر «جاري إرسال الطلب…» وتمنع النسخ | `isSubmitting` لا يُعاد إلى `false` في مسار النجاح | إعادة تهيئته عند فتح النافذة + بعد النجاح |
| 3 | النسخة الثانية لا يُظهر شريط التقدم إطلاقاً | حالة إغلاق المركز `dismissed` محلية ودائمة | نقلها للـ context وإعادة إظهارها تلقائياً |
| 4 | بعد التحول إلى SSE، النسبة ثابتة (لا تتقدم) | تطابق معرّف `v2.ID` ≠ `job.ID` يمنع نشر أحداث التقدم + حالة updater غير نقية في React | توحيد `v2.ID = job.ID` + نقل reset إلى `useEffect` نقي |
| 5 | الـ polling السابق مع الكاش يعطي تأخيراً/تجمداً دورياً | جلب متكرر كل 0.5–4 ثانية مع تخزين مؤقت | استبدال الـ polling بنظام بث SSE لحظي (ميزة جديدة) |

---

## المشكلة 1 — نسبة النسخ تتجمد عند 1% ولا تتقدم

### العرض
- يبدأ النسخ بالفعل، الملف يُنقل كاملاً على الجهاز، لكن نسبة التقدم في الواجهة تبقى ثابتة
  (غالباً 1%) ثم «تقفز» إلى 100% عند الاكتمال أو لا تتحدث أبداً.

### السبب الجذري (عميل — وليس الخادم)
في `client/src/lib/api.js` كانت دالة `isCacheableRequest` تعتبر **أي** طلب GET قابلاً للتخزين
لمدة 120 ثانية (في ذاكرة + `sessionStorage`)، مع **تجاهلها تاماً** لخيار `cache: "no-store"`:

```js
// قبل الإصلاح
function isCacheableRequest(path, options) {
  return (!options.method || options.method.toUpperCase() === "GET")
    && !path.startsWith("/api/admin/")
    && !path.startsWith("/api/stream/");
}
```

- دوال النقل كلها تمرر `{ cache: "no-store" }` (مثل `getTransferJobs`, `getTransferJob`,
  `getTransferDevices`)، لكن هذا الخيار لم يكن يُقرأ إطلاقاً.
- النتيجة: طلب `GET /api/transfer/jobs` يُخزَّن، وكل محاولة جلب جديدة خلال 120 ثانية تعيد
  النسخة القديمة (النسبة الأولى 1%) بدلاً من البيانات الطازجة.
- ملاحظة مطمئنة: كاش الخادم (`server/internal/api/cache.go`) كان أصلاً `ttl=0` لكل مسارات
  النقل، أي أن الخادم لم يكن يخزّنها — المشكلة عميل بحت.

### الحل
`client/src/lib/api.js` — سطرا `isCacheableRequest`:

```js
function isCacheableRequest(path, options = {}) {
  return (!options.method || options.method.toUpperCase() === "GET")
    && options.cache !== "no-store"
    && !path.startsWith("/api/admin/")
    && !path.startsWith("/api/stream/")
    && !path.startsWith("/api/transfer/");   // كل قسم النسخ خارج الكاش نهائياً
}
```

- تُستثنى كل مسارات `/api/transfer/*` من التخزين.
- `options.cache === "no-store"` يُحترم الآن.
- التحقق: أي جلب لاحق داخل النقل يعود دائماً ببيانات حديثة من الخادم.

---

## المشكلة 2 — النسخة الثانية تُظهر «جاري إرسال الطلب…» وتمنع النسخ

### العرض
- بعد إتمام نسخة بنجاح، وعند فتح نافذة النسخ مجدداً واختيار ملف والضغط على زر النسخ:
  يظهر نص «جاري إرسال الطلب…» ولا يحدث شيء — الزرار يبقى معطلاً.

### السبب الجذري
في `client/src/components/transfer/TransferModal.jsx`:

```js
setIsSubmitting(true);   // عند الضغط على الزر
try {
  await startTransfer({ ... });
  // مسار النجاح لم يكن يعيد الوضع
} catch (err) {
  setErrorMsg(err.message);
  setIsSubmitting(false);   // فقط هنا كان يُعاد
}
```

- عند النجاح، النافذة تُغلق (داخل `startTransfer`)، لكن الحالة `isSubmitting` تبقى `true`.
- النافذة لا تُفكّك عند الإغلاق (تعيد `null` فقط)، لذلك عند إعادة فتحها تبقى `isSubmitting=true`
  فيبقى الزرار معطّلاً بنص «جاري إرسال الطلب…».

### الحل
1. **Effect عند الفتح** — إعادة تهيئة حالة النافذة المؤقتة في كل مرة تُفتح:

```js
useEffect(() => {
  if (isTransferModalOpen) {
    setErrorMsg("");
    setIsSubmitting(false);
  }
}, [isTransferModalOpen]);
```

2. **بعد النجاح مباشرة** — إعادة الوضع حتى لو استمرت النافذة:

```js
await startTransfer({ ... });
setIsSubmitting(false);   // أُضيفت في مسار النجاح
```

---

## المشكلة 3 — في النسخة الثانية لا يظهر شريط التقدم إطلاقاً

### العرض
- النسخة الأولى تظهر وتعمل. عند النسخ لمرة ثانية، يتنفذ النسخ فعلياً لكن لا يظهر
  شريط/مركز التقدم في أسفل الشاشة.

### السبب الجذري
في `client/src/components/transfer/MiniTransferCenter.jsx` كانت الحالة `dismissed` **محلية ودائمة**:

```js
const [dismissed, setDismissed] = useState(false);
...
if (relevantJobs.length === 0 || dismissed) return null;
```

- الزر ✕ يظهر فقط عندما تكون كل الوظائف **مكتملة** (`isAllComplete`).
- عند إغلاق المركز بعد النسخة الأولى يُصبح `dismissed=true` **للأبد** (المكوّن مركّب مرة واحدة
  على مستوى `App.jsx` ولا يُعاد تحميله)، فأي نسخة لاحقة لا يظهر شريطها.

### الحل
نُقلت الحالة إلى `TransferContext` وأصبحت تُعاد تلقائياً:

1. `client/src/context/TransferContext.jsx`:
   - حالة جديدة `centerDismissed` + `setCenterDismissed` معرّضة في الـ context.
   - `startTransfer` عند بدء نسخة جديدة: `setCenterDismissed(false)`.
   - `useEffect` نقي يتابع معرفات الوظائف (`knownJobIdsRef`) ويعيد الظهور عند وصول وظيفة جديدة:
     ```js
     if (hasNewJob) setCenterDismissed(false);
     ```
2. `client/src/components/transfer/MiniTransferCenter.jsx`:
   - يحذف `useState` المحلي ويقرأ `centerDismissed`/`setCenterDismissed` من الـ context.

---

## المشكلة 4 — بعد التحول إلى SSE أصبحت النسبة ثابتة لا تتقدم

### العرض
- بعد استبدال الـ polling بنظام SSE، الشريط يظهر لكن النسبة لا تتحرك إطلاقاً.

### السبب الجذري (سببان)

#### 4.1 على الخادم: عدم تطابق معرّف الوظيفة (الجذر)
`server/internal/transfer/service.go`:
- `StartCopy` يولّد الوظيفة القديمة بمعرّف: `job_<unixnano>_<base>` → `job.ID`.
- `buildV2Job` كان يولّد معرّفاً **مستقلاً** للبحث في المحرك: `v2.ID = jobID(sourcePath)`
  (معرّف جديد مختلف عن `job.ID`).
- الـ notifier الجديد (Inside `NewService`) يبحث عن الوظيفة القديمة عبر معرّف v2:
  ```go
  s.engine.SetNotifier(func(v2 *TransferJobV2) {
      s.mu.RLock()
      job, ok := s.jobs[v2.ID]   // ← لا يوجد! لأن المفتاح هو job.ID القديم
      ...
  })
  ```
- عندما أزلنا حلقات الـ polling القديمة (التي كانت تعوّض ذلك بتمرير `job` مباشرة إلى
  `mirrorV2Progress`) واستبدلناها بالانتظار على الأحداث، انكشف الكسر: أحداث تقدم المحرك
  لا تجد الوظيفة فلا تُنشر إطلاقاً، فتبقى النسبة ثابتة.

#### 4.2 على العميل: updater غير نقي (ثانوي)
- أضفنا `setCenterDismissed(false)` **داخل** دالة تحديث `setActiveJobs`. دالة الـ updater في
  React يجب أن تكون نقية، والاستدعاء الجانبي بداخلها قد يعبث بجدولة التحديثات.

### الحل
1. `server/internal/transfer/service.go` — في `runTransferV2` قبل الإرسال للمحرك:
   ```go
   v2 := s.buildV2Job(req, files)
   if v2 == nil { return s.executeTransferLegacy(...) }
   v2.ID = job.ID   // توحيد المعرّف مع الوظيفة القديمة
   if err := s.engine.Submit(v2); err != nil { ... }
   ```
   فيصبح الـ notifier يجد الوظيفة في `s.jobs[v2.ID]` وينشر كل انتقال تقدم (كل ~500 ms).
2. `client/src/context/TransferContext.jsx` — عاد `applyJobEvent` نقياً (دمج فقط)، وفُصل
   إعادة الظهور إلى `useEffect` نقي مع `knownJobIdsRef`.

### التحقق من هذه المشكلة
- `go build ./...` و `go vet` و `go test ./internal/transfer/...` كلها تمر.
- `npm run build` في العميل يمر.

---

## المشكلة 5 (حل شامل) — نظام البث الحي SSE بدل الـ polling

> ليست «مشكلة» بل الحل الجذري الذي تنفّذ لضمان وصول تقدم اللحظة، لأن المزج السابق
> (polling + كاش) هو أصل التجمد المتكرر.

### البنية المضافة

**الخادم:**
1. `server/internal/transfer/events.go` (جديد):
   - `TransferEvent` بأنواع `jobs` / `job` / `devices`.
   - `eventBroker` (fan-out) بقنوات مخزّنة، يسقط الأحداث للبطيء بدل الحجب.
2. `server/internal/transfer/engine.go` + `worker.go`:
   - الخطاف `NotifyUpdate` يُستدعى عند كل انتقال: بدء، تقدّم (كل 500 ms)، إعادة محاولة، نهاية، فشل.
3. `server/internal/transfer/service.go`:
   - `SetNotifier` في `NewService` يعكس تقدم v2 على الوظيفة القديمة وينشرها.
   - `SubscribeEvents(buffer)` و `SnapshotDevices()`.
   - `updateJob` و `CancelJob` و `mirrorV2Progress` تنشر `EventJob`.
   - `runTransferV2` تستبدل حلقة الـ polling بـ: انتظار الحدث الطرفي لذات الوظيفة.
4. `server/internal/api/server.go`:
   - `GET /api/transfer/events` — SSE مع:
     - بذر فوري (seed): لقطة الأجهزة والوظائف عند كل اتصال.
     - `event:` مسمّاة + `data:` JSON.
     - keep-alive كل 20 ثانية.
     - رؤوس تمنع التخزين/X-Accel-Buffering.
   - الـ middleware يمرّرها مباشرة لأن `catalogueCacheTTL` لا تطابق مسارات النقل.

**العميل:**
5. `client/src/context/TransferContext.jsx`:
   - `EventSource` وحيد على `/api/transfer/events` (يُفتح عند التحميل ويُغلق عند الإنهاء).
   - معالجة `jobs` (لقطة كاملة)، `job` (دمج بوظيفة حسب id)، `devices` (لقطة الأجهزة).
   - يستبدل interval الـ polling للوظائف (كان 800/4000 ms).
6. `client/src/components/transfer/TransferModal.jsx`:
   - يستقبل `devices` من الـ context (أحداث `devices` اللحظية) بدل polling كل 1.5 ثانية.
   - أبقينا زر «تحديث» يدوي يعمل HTTP حقيقي واحد.
7. `client/src/components/transfer/MiniTransferCenter.jsx`: بقي يعتمد على `activeJobs`
   من الـ context، فيتحدث تلقائياً مع كل حدث.

> ملاحظة: `CopyToPhoneModal.jsx` و `AdminTransferPage.jsx` لا تزال تستخدم polling داخلي
> (500 ms/3000 ms resp.) لكن بعد إصلاح كاش `api.js` لن تتجمد أرقامها، وتعتمد على
> `getTransferJob`/`getTransferJobs` المباشرة.

---

## فهرس تغييرات المشاكل (file:line حالياً)

| الملف | ما الذي تغيّر بسببه |
|---|---|
| `client/src/lib/api.js:16-22` | إصلاح الكاش (المشكلة 1) |
| `client/src/components/transfer/TransferModal.jsx:75-83, 281` | إعادة تهيئة `isSubmitting` (المشكلة 2) |
| `client/src/context/TransferContext.jsx` + `MiniTransferCenter.jsx` | رفع `centerDismissed` (المشكلة 3) |
| `server/internal/transfer/service.go:660-666` | توحيد `v2.ID = job.ID` (المشكلة 4) |
| `client/src/context/TransferContext.jsx` | نقاء الـ updater (المشكلة 4) |
| `events.go`, `engine.go`, `worker.go`, `service.go`, `api/server.go` | نظام SSE (المشكلة 5) |

---

## كيف تختبر أن كل شيء يعمل

1. ابدأ النسخة الأولى: يجب أن ترى النسبة تتحرك بشكل شبه لحظي (حدث كل ~500 ms).
2. عند اكتمالها أظهر المركز «✓ مكتمل».
3. ابدأ النسخة الثانية مباشرة: يجب أن يظهر المركز مجدداً (حتى لو أُغلق سابقاً) مع نسبة متحركة.
4. أغلق المركز (✕) ثم ابدأ نسخة ثالثة: يجب أن يعود تلقائياً.
5. افصل/صل جهازاً أثناء نشاط: قائمة الأجهزة تتحدث فوراً عبر أحداث `devices`.