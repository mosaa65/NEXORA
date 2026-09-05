# NEXORA Development Guide for Agents and Contributors

هذه التعليمات تحفظ المعمارية الحالية. لا تعتبر المقترحات المستقبلية تنفيذًا قائمًا.

## 1. Project Understanding

قبل تعديل أي جزء:

1. اقرأ [`docs/PROJECT_ANALYSIS.md`](docs/PROJECT_ANALYSIS.md).
2. اقرأ [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
3. اقرأ ADRs المرتبطة في [`docs/decisions/`](docs/decisions/README.md).
4. افحص الكود الفعلي المتأثر كاملًا، مع callers وcallees وtests ذات الصلة.

الـ Repository والكود الحالي هما مصدر الحقيقة للتنفيذ. لا تعامل `[PLANNED]` أو `[INFERRED]` كميزة أو قرار معتمدين.

## 2. Development Workflow

لكل تغيير متوسط أو كبير اتبع:

```text
Understand
↓
Inspect Existing Code
↓
Identify Boundaries
↓
Identify Affected Systems
↓
Compare Alternatives
↓
Recommend Approach
↓
Plan
↓
Implement
↓
Verify
```

لا تتبع نمط:

```text
Prompt
↓
Immediate Code
```

## 3. Existing Architecture First

قبل إنشاء service أو database table أو cache أو worker أو API endpoint أو dependency جديدة، تحقق أولًا من قابلية توسيع النظام الموجود.

لا تنشئ duplicate systems مثل:

- Duplicate Cache
- Duplicate Metadata System
- Duplicate Streaming Layer
- Duplicate File Scanner
- Duplicate API Client

المسارات الحالية التي يجب فحصها غالبًا: `client/src/lib/api.js`، `server/internal/api`، `server/internal/db/repository.go`، `server/internal/scanner`، `server/internal/media`، `server/internal/metadata`، و`server/internal/search`.

## 4. Video and Streaming Rules

المسار الحالي القابل للتحقق:

- Native HTML5 `<video>` هو playback engine الحالي.
- Custom React controls فوقه في `VideoPlayer.jsx`.
- Direct HTTP Range streaming عبر Go `http.ServeContent` هو المسار الحالي.
- لا يتم تحميل ملف الفيديو كاملًا في ذاكرة الخادم في stream handler.
- FFmpeg ليس جزءًا من playback hot path الحالي.
- لا تضف transcoding أو HLS/DASH دون حاجة مثبتة وتحليل وقرار معماري.

أي تغيير في streaming يجب أن يحلل: Memory، Disk I/O، concurrent clients، Range compatibility، وbrowser/container/codec compatibility.

## 5. Change Classification

### Small

أمثلة: UI text، CSS، أو bug محدود. يمكن التنفيذ بعد فهم الملفات المتأثرة.

### Medium

أمثلة: feature داخل subsystem موجود، endpoint جديد، database query، أو player behavior.

المطلوب:

```text
Inspect → Plan → Implement → Verify
```

### Architectural

أمثلة: database redesign، authentication، streaming architecture، HLS/DASH، transcoding، queue/cache جديد، dependency رئيسية، أو تغيير media storage model.

المطلوب:

```text
Inspect
↓
Analysis
↓
Alternatives
↓
Recommendation
↓
Owner Approval
↓
Implementation
```

لا تبدأ التنفيذ قبل موافقة المالك على تغييرات صعبة العكس أو واسعة الأثر.

## 6. Source of Truth Rules

- PostgreSQL: authoritative catalogue, file metadata, relationships, settings, and persistent application data.
- Meilisearch: derived/rebuildable search data.
- Asset previews/artwork: derived/cache files where practical, وليست catalogue source of truth.
- Browser localStorage/sessionStorage: per-browser state/cache، وليس shared persistent system data.

أي cache جديد يجب أن تكون له ownership وinvalidation/rebuild behavior واضحة.

## 7. Current Known Risks

اقرأ تفاصيلها في `docs/PROJECT_ANALYSIS.md` قبل العمل في المناطق المرتبطة:

- Path authorization behavior في `mediaPathAllowed`.
- Admin authentication gaps وdefault credentials/token behavior.
- Preview generation race عند أول طلب متزامن.
- External subtitle format mismatch.
- Watch progress محلي فقط.
- File watcher deletion behavior.
- Documentation drift بين docs القديمة والكود.

لا تصلح هذه المخاطر تلقائيًا لمجرد المرور عليها؛ صنّف التغيير واطلب موافقة المالك إن كان Architectural.

## 8. Code Modification Rules

- اقرأ الملفات المتأثرة كاملة وافهم callers وcallees.
- لا تغير ملفات غير مرتبطة ولا تعمل refactor واسعًا بلا سبب مثبت.
- لا تستبدل نظامًا يعمل دون إثبات مشكلة.
- لا تضف abstraction فقط لأن pattern مشهور، ولا تكرر logic موجود.
- حافظ على naming conventions والحدود الحالية.
- أضف/حدّث tests عندما يكون التغيير سلوكيًا أو backend قابلًا للاختبار.
- تحقق من behavior، وليس compilation فقط.

## 9. Documentation Rules

بعد أي تغيير مهم، راجع إن كان يلزم تحديث:

- `docs/PROJECT_ANALYSIS.md`
- `docs/ARCHITECTURE.md`
- ADR مرتبط أو ADR جديد للقرار المنفذ
- `docs/api/ENDPOINTS.md`
- `README.md`

لا تترك documentation تدعي وجود feature غير موجودة، ولا ترفع planned idea إلى current architecture بلا تنفيذ قابل للتحقق.

## 10. Response Rules for AI Agents

عند تنفيذ تغيير مهم، اجعل الرد يشمل:

```text
Understanding:
...

Affected Systems:
...

Existing Architecture:
...

Options:
...

Recommendation:
...

Implementation Plan:
...

Verification:
...

Documentation Updated:
...
```

للتغييرات الصغيرة، اختصر الأقسام بما يناسب حجم التغيير، لكن لا تتجاوز الفهم والتحقق.
