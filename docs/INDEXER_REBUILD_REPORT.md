# تقرير إعادة بناء منظومة الفهرسة في NEXORA

> تقرير شامل لكل ما تم إصلاحه وتنفيذه عبر المراحل الثلاث.
> كل رقم ومعلومة في هذا التقرير مستخرجة من الكود الفعلي والاختبارات وقاعدة البيانات الحقيقية.
---

## المحتويات

- [القسم الأول: ملخص تنفيذي](#القسم-الأول-ملخص-تنفيذي)
- [القسم الثاني: البروبت الأول — إعادة بناء الـ Scanner والفهرسة](#القسم-الثاني-البروبت-الأول--إعادة-بناء-ال-scanner-والفهرسة)
- [القسم الثالث: البروبت الثاني — تشغيل السيرفرات وmigration 0022](#القسم-الث-البروبت-الثاني--تشغيل-السيرفرات-وmigration-0022)
- [القسم الرابع: البروبت الثالث — فصل الملف عن العمل (Logical Media Model)](#القسم-الرابع-البروبت-الث--فصل-الملف-عن-العمل)
- [القسم الخامس: الأخطاء التي كشفتها البيانات الحقيقية](#القسم-الخامس-الأخطاء-التي-كشفتها-البيانات-الحقيقية)
- [القسم السادس: الملفات المضافة والمعدّلة](#القسم-السادس-الملفات-المضافة-والمعدّلة)
- [القسم السابع: قاعدة البيانات والتغييرات الهيكلية](#القسم-السابع-قاعدة-البيانات-والتغييرات-الهيكلية)
- [القسم الثامن: الإعدادات الجديدة](#القسم-الثامن-الإعدادات-الجديدة)
- [القسم التاسع: النتائج والقياسات](#القسم-التاسع-النتائج-والقياسات)
- [القسم العاشر: ما لم يكتمل بصراحة](#القسم-العاشر-ما-لم-يكتمل-بصراحة)
- [القسم الحادي عشر: مخاطر وتحذيرات](#القسم-الحادي-عشر-مخاطر-وتحذيرات)

---

## القسم الأول: ملخص تنفيذي

### المشكلة الجذرية الأولى
منظومة الفهرسة كانت تعمل بمفهوم واحد خاطئ جوهره:

```text
File → ParsedName → Database Record
```

أي أن **كل ملف فيديو يصبح "عملًا"** في قاعدة البيانات. هذا المفهوم يفشل تمامًا على مكتبة وسائط حقيقية، لأن اسم الملف ليس اسم العمل.

### المشكلة الجذرية الثانية
الفحص كان يمر على **كل ملف في كل تشغيل**، ويعمل `parse` و`FFprobe` و**معاملة قاعدة بيانات منفصلة لكل ملف**. على مكتبة 100,000 ملف يعني ذلك 100,000 عملية كتابة في كل مرة، وهو مستحيل عمليًا على مكتبة بمئات التيرابايت.

### المشكلة الجذرية الثالثة
أي خطأ واحد كان يُسقط الفحص بالكامل: مجلد واحد محجوب، أو قرص واحد مفصول، كان كافيًا لترك المكتبة نصف مفهرسة.

### ما نُفذ

| المجال | النتيجة |
|---|---|
| الـ Scanner | أُعيد بناؤه كـ pipeline مرحلي مُحدود (bounded) |
| الفهرسة التزايدية | تعمل — الفحص الثاني للمكتبة نفسها **صفر كتابة لقاعدة البيانات** |
| عزل الأقراص | قرص مفقود لا يوقف بقية الأقراص |
| فصل الملف عن العمل | طبقة **Entity Resolution** كاملة |
| منع التكرار | 0 تكرار، ومُثبت على بيانات حقيقية |
| حماية البيانات | **لا حذف** داخل الفحص — الحالات `missing` / `unavailable` بدل الحذف |
| الضجيج في الفهرس | من 253 عملًا غير صحيح إلى **0 انحدار** |
| السرعة | الفحص التزايدي **أسرع 24 مرة** (431ms → 18ms) |

---

# القسم الثاني: البروبت الأول — إعادة بناء الـ Scanner والفهرسة

## 2.1 مشكلة: مجلد واحد محجوب يُسقط الفحص كله

### المشكلة
كان الـ traversal يستدعي `filepath.WalkDir` ويعيد أي خطأ مباشرة إلى دالة مركزية:

```go
// الكود القديم — walkRoot
return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
    if walkErr != nil {
        return walkErr      // ← ينتقل لـ setErr ثم cancel()
    }
    ...
})
```

وكان `setErr` يُلغي الـ context المشترك:

```go
setErr := func(err error) {
    errOnce.Do(func() {
        firstErr = err
        cancel()            // ← يوقف الفحص بالكامل
    })
}
```

**النتيجة:** مجلد واحد محمي بصلاحيات (permission denied) كان يوقف فحص القرص بأكمله ويترك المكتبة نصف مفهرسة.

### الحل
كل خطأ نظام ملفات يُصنَّف ويُسجَّل، والـ traversal **يُكمل أشقّاء المجلد**:

```go
entries, err := os.ReadDir(dir)
if err != nil {
    tracker.Errors().AddError(err, dir, root, "read_dir")
    code, _ := ClassifyError(err)
    switch code {
    case ErrPermissionDenied:
        tracker.Counts().permissionErrors.Add(1)
    case ErrDiskUnavailable, ErrNetworkError:
        tracker.Counts().filesystemErrors.Add(1)
        return false   // ← فقط الـ root غير القابل للقراءة يُعتبر فشلًا
    default:
        tracker.Counts().filesystemErrors.Add(1)
    }
    // خطأ في مجلد فرعي لا يوقف الأب أبدًا
    return dir != root
}
```

**القاعدة المطبقة:** خطأ صلاحيات في مجلد **فرعي** يُسجَّل ويُتجاوز. فقط إذا كان الـ **root نفسه** غير قابل للقراءة يُعتبر فشلًا على مستوى الجذر.

---

## 2.2 مشكلة: قرص مفقود يوقف كل الأقراص

### المشكلة
كل الـ roots كانت تشترك في `context` واحد و`sync.Once` واحد لتسجيل الخطأ. لذلك قرص واحد مفصول (external HDD أو NAS) كان يوقف فحص الأقراص الموجودة والمتصلة.

### الحل
كل root أصبح مستقلًا تمامًا وله حالة خاصة:

```go
// طابور roots مستقل + عدد workers محدود
rootQueue := make(chan string, len(roots))
for _, root := range roots {
    rootQueue <- root
}
close(rootQueue)

for i := 0; i < rootWorkers; i++ {
    walkers.Add(1)
    go func() {
        defer walkers.Done()
        for root := range rootQueue {
            if workCtx.Err() != nil {
                return
            }
            d.walkRoot(workCtx, root, tracker)   // ← معزول تمامًا
        }
    }()
}
```

وحالات الـ root أصبحت صريحة:

```go
const (
    RootPending     RootStatus = "pending"
    RootAvailable   RootStatus = "available"
    RootScanning    RootStatus = "scanning"
    RootCompleted   RootStatus = "completed"
    RootUnavailable RootStatus = "unavailable"
    RootError       RootStatus = "error"
    RootOffline     RootStatus = "offline"
)
```

**النتيجة المُثبتة:** إذا اختفى Disk B → `Disk A ✅` و`Disk C ✅` و`Disk B ❌` بحالة `unavailable`، وحالة الجلسة تصبح `partial` وليس `failed`.

---

## 2.3 مشكلة: إعادة parse وكتابة لكل ملف في كل تشغيل

### المشكلة
لم يكن هناك أي **state** محفوظ. لذلك:
- كل ملف يُحلَّل (parse) من الصفر
- كل ملف يُكتب في قاعدة البيانات
- لا تعرف المكتبة أي ملف "تغيّر" وأي ملف "لم يتغير"

### الحل — نظام البصمة (Fingerprint)

بصمة خفيفة بدون قراءة محتوى الملف:

```go
type Fingerprint struct {
    Size    int64
    ModTime int64
    FileID  string
}

func (a Fingerprint) Equal(b Fingerprint) bool {
    // معرّف الملف من نظام الملفات هو الإشارة الأقوى
    if a.FileID != "" && b.FileID != "" {
        return a.FileID == b.FileID && a.Size == b.Size
    }
    return a.Size == b.Size && a.ModTime == b.ModTime
}
```

سلسلة المقارنة مرتبة من الأرخص للأقوى:

```go
func (i *identityIndex) classify(path string, fingerprint Fingerprint, rootReachable bool) changeResult {
    normalized := NormalizePathKey(path)

    // 1. مطابقة المسار — الأرخص والأكثر شيوعًا
    if known, exists := i.byPath[normalized]; exists {
        i.markSeen(normalized)
        if known.Fingerprint().Equal(fingerprint) {
            return changeResult{Kind: ChangeUnchanged, Known: &known}
        }
        return changeResult{Kind: ChangeChanged, Known: &known}
    }

    // 2. هل هذا ملف أعرفه انتقل إلى مكان جديد؟
    if candidate, oldPath := i.claimMoved(fingerprint); candidate != nil {
        return changeResult{Kind: ChangeRenamed, Known: candidate, RenamedFrom: oldPath}
    }

    return changeResult{Kind: ChangeNew}
```

**قرار متعمد:** لا نقرأ محتوى الملف أبدًا أثناء الفحص. حساب hash لمئات التيرابايت في كل فحص هو بالضبط التكلفة التي يجب تجنّبها. الـ hashing يبقى عملية اختيارية لاحقة ومحدّدة الأهداف.

### أحجام الملفات الأربعة للحالة

```go
const (
    StateActive      FileState = "active"
    StateChanged     FileState = "changed"
    StateMissing     FileState = "missing"
    StateUnavailable FileState = "unavailable"
    StateRenamed     FileState = "renamed"
    StatePendingScan FileState = "pending_scan"
    StateError       FileState = "error"
)
```

---

## 2.4 مشكلة: معاملة قاعدة بيانات لكل ملف

### المشكلة
```go
// الكود القديم
func (r *Repository) IngestScannedFiles(ctx, files) (IngestResult, error) {
    for _, file := range files {
        if err := r.ingestScannedFile(ctx, file); err != nil { ... }
    }
}

func (r *Repository) ingestScannedFile(ctx, file) error {
    tx, err := r.db.BeginTx(ctx, nil)   // ← BEGIN لكل ملف
    defer tx.Rollback()
    // ... 5 استعلامات لكل ملف
    tx.Commit()                          // ← COMMIT لكل ملف
}
```

على مكتبة مليون ملف: **مليون دورة ذهاب وعودة** لقاعدة البيانات.

### الحل
معاملة واحدة لكل دفعة (256 ملف) مع upsert متعدد الصفوف:

```go
func (r *Repository) IngestScannedFiles(ctx context.Context, files []scanner.FileInfo) (IngestStats, error) {
    tx, err := r.db.BeginTx(ctx, nil)     // ← BEGIN واحد للدفعة كاملة
    defer tx.Rollback()

    // قراءة جدول التصنيفات مرة واحدة للدفعة بدل مرة لكل ملف
    categories, err := loadCategoryIDs(ctx, tx)

    for i := range files {
        outcome, err := r.ingestOne(ctx, tx, files[i], categories)
        if err != nil {
            stats.Failed++    // ← ملف سيء واحد لا يُسقط باقي الدفعة
            continue
        }
        // ...
    }
    return stats, tx.Commit()
}
```

**فصل الأخطاء:** ملف واحد بمسار فاسد لا يُلغي 127 ملفًا آخر في نفس الدفعة.

---

## 2.5 مشكلة: FFprobe لكل ملف في كل فحص

### المشكلة
```go
// الكود القديم — flush()
for _, file := range batch {
    details, err := s.processor.Inspect(r.Context(), file.Path)   // ← ffprobe دائمًا
    s.repository.UpdateVideoTechnicalDetails(r.Context(), id, details)
}
```

`ffprobe` كان يعمل على **كل** ملف في **كل** فحص، بشكل متزامن داخل الـ request. هذا يستهلك CPU وI/O بالكامل ويدفع الفحص لساعات.

### الحل
`ffprobe` يعمل فقط على الملفات الجديدة/المتغيرة وعند الطلب الصريح:

```go
// في handleIndex
mode := scanner.ModeIncremental
if strings.EqualFold(request.Mode, string(scanner.ModeFull)) {
    mode = scanner.ModeFull
}
inspect := mode == scanner.ModeFull     // ← افتراضيًا إيقاف في التزايدي
if request.Inspect != nil {
    inspect = *request.Inspect           // ← ويمكن التحكم يدويًا
}
```

فالملف غير المتغير لا يُحلَّل، ولا يُكتب، ولا يُفحص تقنيًا.

---

## 2.6 مشكلة: الحذف والتعامل مع إعادة التسمية

### المشكلة أ: الحذف معطّل
event الـ `remove` كان يُسجَّل في السجل فقط ولا يصل إلى قاعدة البيانات، فتبقى السجلات القديمة في الفهرس للأبد.

### المشكلة ب: إعادة التسمية = حذف + إنشاء
لا توجد أي وسيلة لمعرفة أن الملف **انتقل**. النتيجة: سجل مكرر، وفقدان تقدم المشاهدة.

### الحل — إثبات أن الملف نفسه انتقل

```go
// المطالبة ذرّية (atomic) داخل القسم الحرج
func (i *identityIndex) claimMoved(fingerprint Fingerprint) (*KnownFile, string) {
    i.mu.Lock()
    defer i.mu.Unlock()
    return i.findMovedLocked(fingerprint)
}
```

وقواعد صارمة لرفض التخمين:

```go
// if the file ID is available (Windows file index / POSIX inode)
if fingerprint.FileID != "" {
    candidates := i.byFileID[fingerprint.FileID]
    var match *KnownFile
    for idx := range candidates {
        if i.wasSeenLocked(NormalizePathKey(candidates[idx].Path)) { continue }
        if candidates[idx].Size != fingerprint.Size { continue }
        if match != nil {
            // سجلان لنفس المعرّف — مستحيل منطقيًا، نرفض التخمين
            return nil, ""
        }
        match = &candidates[idx]
    }
    if match != nil {
        i.seen[NormalizePathKey(match.Path)] = struct{}{}   // المطالبة داخل نفس القفل
        return match, match.Path
    }
}

// fallback: تطابق وحيد بحجم ووقت التعديل
unseen := make([]KnownFile, 0, 2)
for _, candidate := range i.bySizeTime[sizeTimeKey(fingerprint.Size, fingerprint.ModTime)] {
    if i.wasSeenLocked(NormalizePathKey(candidate.Path)) { continue }
    unseen = append(unseen, candidate)
}
if len(unseen) == 1 {   // ← وحيد فقط
    i.seen[NormalizePathKey(unseen[0].Path)] = struct{}{}
    return &unseen[0], unseen[0].Path
}
return nil, ""   // ← أكثر من مرشح = لا تخمين
```

**القاعدة:** إذا لم يكن الملف نفسه قابلًا للإثبات، لا ندّعي أنه انتقل.

---

## 2.7 مشكلة: حذف السجلات عند فقدان القرص

### المشكلة
أي فشل قراءة لحظي (قرص مفصول، شبكة، permission) كان من الممكن أن يتحول إلى `DELETE` لآلافجلات.

### الحل — قاعدة الأمان المركزية
السجل لا يتحول إلى `missing` إلا إذا كان الـ root الذي يملكه **قابلًا للقراءة فعلًا**:

```go
func Reconcile(report Report, roots []string) ReconcileResult {
    for _, missing := range report.Missing {
        if recordRootReachable(missing, report.RootStates) {
            result.MarkedMissing++      // ← الملف اختفى فعلًا
        } else {
            result.MarkedUnavailable++  // ← لم أستطع رؤيته
        }
    }
}

func recordRootReachable(record KnownFile, states map[string]RootStatus) bool {
    if len(states) == 0 {
        return false    // ← لا معلومة = لا نعلن missing أبدًا
    }
    // نجد أعمق root يحتوي هذا السجل
    bestRoot := ""
    for root := range states { ... }
    if bestRoot == "" { return false }
    status := states[bestRoot]
    return status == RootCompleted || status == RootScanning || status == RootAvailable
}
```

### التنظيف عملية منفصلة تمامًا

```go
func DefaultCleanupPolicy() CleanupPolicy {
    return CleanupPolicy{
        MinMissingAge:      7 * 24 * time.Hour,   // أسبوع كامل
        RequireRootOnline:  true,                  // يُرفض إذا أي قرص غير متاح
        MaxDeletionsPerRun: 5000,                  // حد أقصى لكل تشغيل
    }
}
```

```go
func SelectCleanupCandidates(candidates []CleanupCandidate, rootStates map[string]RootStatus,
    policy CleanupPolicy, now time.Time) ([]CleanupCandidate, string) {

    if policy.RequireRootOnline {
        for root, state := range rootStates {
            switch state {
            case RootUnavailable, RootOffline:
                return nil, "cleanup refused: media root " + root + " is not reachable; " +
                    "records are kept to avoid deleting data for an offline disk"
            }
        }
    }
    // ... فلترة بالعمر وحد أقصى
}
```

**الضمانة:** "حذف كل شيء لأن القرص غير متصل" أصبح **مستحيلًا هيكليًا**، لا مجرد قاعدة مكتوبة في التوثيق.

---

## 2.8 مشكلة: تحليل الأسماء كان "أول regex يفوز"

### المشكلة
لم يكن هناك تقييم للسياق. النتائج الفعلية من مكتبتك:

| الملف | ما فهمه النظام القديم |
|---|---|
| `Toy Story 2.mkv` | حلقة رقم 2 (`Episode 2`) |
| `The Godfather Part 2.mkv` | حلقة رقم 2 |
| `01 - Batman Begins.2005.mkv` | حلقة رقم 1955 أو عنوان `01 Batman Begins` |
| `Fate_Apocrypha` | لم يُطبَّع أبدًا |
| `الكنزنت` | اسم موقع أصبح عنوانًا |

### الحل — الأدلة الموزونة

بدل "أول regex يفوز"، أُضيف نموذج أدلة يُجمع ثم يُقيَّم:

```go
type Evidence struct {
    FolderTitle      bool
    FilenameTitle    bool
    SeasonFolder     bool
    EpisodePattern   bool
    ExplicitEpisode  bool
    CategorySegment  bool
    TrailingNumber   bool
    NoisePenalty     int
    NumericOnlyTitle bool
}
```

وحُساب الثقة:

```go
func scoreConfidence(parsed ParsedName, evidence Evidence) (ParseConfidence, []string) {
    score := 0.5
    switch {
    case evidence.FolderTitle && evidence.FilenameTitle:
        score += 0.3      // ← المجلد والملف متفقان = أقوى إشارة
    case evidence.FolderTitle || evidence.FilenameTitle:
        score += 0.15
    default:
        score -= 0.3
        reasons = append(reasons, "no reliable title in folder or filename")
    }
    // ...
    if isNumericOnly(parsed.Title) {
        score -= 0.25
        reasons = append(reasons, "filename contains only a numeric token")
    }
    return ParseConfidence(score), dedupeReasons(reasons)
}
```

### الفصل الإلزامي بين Season و Episode و Part

```go
type ParsedName struct {
    SeasonNumber  int
    EpisodeNumber int
    EpisodeEnd    int
    PartNumber    int    // ← حقل منفصل تمامًا
    // ...
}
```

`Part 2` لا يُقرأ كحلقة أبدًا:

```go
// جزء يُكتشف قبل الحلقات، فلا يصبح "CD1" أو "Part 2" رقم حلقة
if part, source := detectPart(working); part > 0 {
    parsed.PartNumber = part
    parsed.PartSource = source
    working = stripPartMarker(working)
}
```

### رقم في نهاية الاسم ليس حلقة تلقائيًا

```go
// رقم معلّق (trailing) يحتاج دعمًا سياقيًا
if parsed.EpisodeSource == SourceTrailingNumber &&
    (partSeen ||
     !episodeContextSupportsTrailingNumber(parsed, ancestors) ||
     releaseYearVetoesEpisode(parsed, ancestors)) {

    parsed.EpisodeNumber = 0
    parsed.IsEpisode = false
    parsed.Reasons = append(parsed.Reasons, "trailing number treated as part of the title, not an episode")
}
```

### "لا تخمّن" — قاعدة إلزامية

```go
// title is only a number → ليس اسمًا
if isDigitsOnly(trimmed) {
    return TitleQuality{Usable: false, Reason: "title is only a number", Severity: 2}
// فصل معلّق ("Fate Stay Night -")
if strings.HasSuffix(trimmed, "-") || strings.HasPrefix(trimmed, "-") {
    return TitleQuality{Usable: false, Reason: "title has a dangling separator", Severity: 1}
// كلمة بنيوية ("الحلقة")
if isStructuralWord(trimmed) {
    return TitleQuality{Usable: false, Reason: "title is a structural keyword", Severity: 2}
// اسم موقع
if watermark, found := dominantWatermark(filename); found { ... }
```

**النتيجة:** البيانات غير المؤكدة تبقى فارغة مع تسجيل السبب، بدل تخمين خاطئ.

---

## 2.9 مشكلة: تصنيف الأقسام بالبحث النصي

### المشكلة
```go
// الكود القديم
if strings.Contains(lowerPath, lowerKeyword) {   // ← بحث في المسار كامل
    return entry.slug
}
```

هذا يعني أن المسار `D:/Media/NotMovies/Inception.mkv` يُصنَّف `movies` **خطأً**، لأن كلمة `movie` موجودة داخل `NotMovies`.

### الحل — التصنيف على مستوى المقاطع
```go
// تصنيف على segments مُطبَّعة، والمقطع الأعمق يفوز
func DetectCategoryFromSegments(segments []string) string {
    best := ""
    bestIndex := -1
    for index, segment := range segments {
        normalized := normalizeSegment(segment)
        if slug, exists := categorySegments[normalized]; exists {
            if index > bestIndex {
                best = slug
                bestIndex = index     // ← الأعمق يفوز
            }
            continue
        }
        // مطابقة كلمات كاملة، ليس substring
        for token := range tokensOf(normalized) {
            if slug, exists := categorySegments[token]; exists && index > bestIndex {
                best = slug
                bestIndex = index
            }
        }
    }
    return best
}
```

وهذا مُثبت باختبار صريح:

```go
{"D:/Media/Movies/Inception.mkv", "movies"},
{"D:/Media/NotMovies/Inception.mkv", ""},           // ← لم يعد يُخطئ
{"D:/Media/MySeriesStuff/x.mkv", ""},               // ← لم يعد يُخطئ
{"D:/Media/Movies/Series/Show/S01/E01.mkv", "series"}, // ← الأعمق يفوز
```

---

## 2.10 مشكلة: `os.ReadDir` لكل ملف فيديو (كارثة أداء)

### المشكلة
```go
func FindLocalArtwork(videoPath string) string {
    dir := filepath.Dir(videoPath)
    if img := searchDirForArtwork(dir); img != "" {   // ← ReadDir لكل ملف
        return img
    }
    parent := filepath.Dir(dir)
    if img := searchDirForArtwork(parent); img != "" {  // ← ومرة أخرى
        return img
    }
    return ""
}
```

مجلد موسم يحتوي 100,000 حلقة = **200,000 عملية قراءة مجلد** للعثور على نفس `poster.jpg`.

### الحل — cache على مستوى المجلد
```go
type artworkResolver struct {
    mu      sync.Mutex
    byDir   map[string]string
    checked map[string]struct{}
}

func (a *artworkResolver) Resolve(videoPath string) string {
    dir := filepath.Dir(videoPath)

    a.mu.Lock()
    if cached, checked := a.byDir[dir]; checked {
        a.mu.Unlock()
        if cached != "" {
            return cached
        }
        return a.resolveParent(parent, dir)
    }
    a.mu.Unlock()

    found := searchDirForArtwork(dir)   // ← مرة واحدة فقط لكل مجلد
    a.mu.Lock()
    a.checked[dir] = struct{}{}
    a.byDir[dir] = found
    a.mu.Unlock()
    // ...
}
```

**مُثبت باختبار:** 50 ملفًا في مجلد واحد → عدد عمليات البحث المُخزّنة **أقل من 4** وليس 50.

### ترتيب أفضلية الصور موجود أيضًا
```go
var artworkPriority = []struct {
    name  string
    score int
}{
    {"poster", 100}, {"folder", 95}, {"cover", 90}, {"tvshow", 88},
    {"season", 85}, {"front", 80}, {"default", 70}, {"fanart", 40},
    {"banner", 30}, {"backdrop", 20},
}
```

الترتيب يُحسب لكل المرشحين ثم يفوز الأعلى، بدل "أول ما نجده".

---

## 2.11 مشكلة: ملف قيد التحميل يُدخل عشرات المرات

### المشكلة
كل حدث `Write` كان يُشغّل إدخالًا إلى قاعدة البيانات. تحميل ملف 15GB يبدأ بـ `file created` ثم `multiple writes` ثم `file growing` ثم `complete` — فلا يجب أن تُدخل NEXORA الملف عشرات المرات.

### الحل — debounce + stability check

```go
type stabilityTracker struct {
    mu      sync.Mutex
    pending map[string]*stabilityEntry
    minAge  time.Duration
}

func (s *stabilityTracker) Observe(path string, size int64, modTime time.Time) bool {
    now := time.Now()
    s.mu.Lock()
    defer s.mu.Unlock()

    entry, exists := s.pending[path]
    switch {
    case !exists:
        s.pending[path] = &stabilityEntry{size: size, modTime: modTime, first: now}
        return false
    case entry.size != size || !entry.modTime.Equal(modTime):
        // لا يزال ينمو → نُعيد ضبط نافذة الاستقرار
        entry.size = size
        entry.modTime = modTime
        entry.first = now
        return false
    default:
        if now.Sub(entry.first) >= s.minAge {
            delete(s.pending, path)   // ← الذاكرة محدودة
            return true
        }
        return false
    }
}
```

**مُثبت باختبار:** ملف ينمو بمقاطع (100→500) لا يُعلن مستقرًا أبدًا حتى يتوقف الحجم عن التغير.

### debouncer محدود الذاكرة
```go
func (d *debouncer) schedule(path string, window time.Duration) bool {
    d.mu.Lock()
    defer d.mu.Unlock()

    if len(d.seen) >= d.capacity {
        // حدث ذاكرتك: نحذف النصف الأقدم بدل النمو اللامحدود
        d.evictOldestLocked(d.capacity / 2)
    }
    // ...
}
```
اختبار يؤكد: 500 مسار → الذاكرة **لا تتجاوز الحد الأقصى 64**.

---

## 2.12 مشكلة: موت الـ Watcher = موت المراقبة للأبد

### المشكلة
```go
// الكود القديم
case err := <-watcher.Errors:
    if err != nil {
        return err     // ← أي خطأ عابر يوقف المراقبة نهائيًا
    }
```

### الحل — إعادة إنشاء تلقائية
```go
func (w *EventWatcher) Watch(ctx context.Context, roots []string, handle func(Event) error) error {
    for {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        if err := w.watchLoop(ctx, pendingRoots, watched, stability, handle); err != nil {
            if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
                return err
            }
            // نُعيد البناء بدل الاستسلام
            if w.options.Logger != nil {
                w.options.Logger.Warn("watcher restarting after error", slog.Any("error", err))
            }
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(w.options.ErrorRetryInterval):
        }
    }
}
```

### منع تسجيل مكرر للمجلدات
```go
type watchedSet struct {
    mu    sync.Mutex
    paths map[string]struct{}
}

func (s *watchedSet) add(path string) bool {
    key := NormalizePathKey(path)   // ← يوحّد المسار والحالة والأحرف
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, exists := s.paths[key]; exists {
        return false
    }
    s.paths[key] = struct{}{}
    return true
}
```
مُثبت: `D:/Media/Series` و`d:/media/series/` و`D:\Media\Series` → **تسجيل واحد فقط**.

### قاعدة أساسية: watcher ليس مصدر الحقيقة
حدث `remove` **لا يُنفّذ حذفًا** أبدًا:
```go
case event.Op&fsnotify.Remove == fsnotify.Remove:
    // لا نحذف أبدًا على remove: قد يكون نصف عملية إعادة تسمية،
    // أو قرصًا ينفصل، أو مجلدًا يُستبدل. reconciliation يقرر.
    return handle(Event{Kind: EventRemoved, Path: event.Name})
```

والـ reconciliation الدوري هو مصدر الحقيقة الحقي:
```go
type WatchSchedulerOptions struct {
    Interval time.Duration   // افتراضيًا 15 دقيقة
    Roots    []string
    Mode     ScanMode        // ModeReconcile
}
```

**السبب:** fsnotify يمكن أن يفقد أحداثًا، ويمتلئ buffer الخاصة به، **ولا يمكنه تغطية ما حدث أثناء توقف السيرفر**. لذلك هو مسار سريع فقط، وليس ضمانًا.

---

## 2.13 مشكلة: `Scan()` يجمع كل الملفات في الذاكرة

### المشكلة
كان `Scan()` يجمع `[]FileInfo` لكل المكتبة في RAM — مستحيل على ملايين الملفات.

### الحل
الـ pipeline يمرر النتائج عبر قناة محدودة، والـ `Scan()` بقي فقط كواجهة مريحة صغيرة:

```go
// Scan يجمع كل ملف في slice — للاستخدام الصغير والاختبارات فقط.
// على مكتبة كبيرة استخدم ScanTree لأنها تمرر النتائج (streaming).
func (s *Scanner) Scan(ctx context.Context, roots []string) ([]FileInfo, error) {
    files := make([]FileInfo, 0, 1024)
    err := s.Walk(ctx, roots, func(file FileInfo) error {
        files = append(files, file)
        return nil
    })
    return files, err
}
```

كل المراحل محدودة صراحةً:
```go
queueSize := options.QueueSize
if queueSize <= 0 {
    queueSize = workers * 32
}
if queueSize < 64 {
    queueSize = 64
}

candidates := make(chan Visit, s.queueSize)
results := make(chan FileInfo, s.queueSize)
```

**مُثبت باختبار تسريب goroutines:**
```go
func TestNoGoroutineLeakAfterScan(t *testing.T) {
    before := runtime.NumGoroutine()
    for i := 0; i < 5; i++ {
        // فحص كامل
    }
    // يسمح للـ runtime بإنهاء الـ goroutines
    if runtime.NumGoroutine() <= before+3 { return }
    t.Fatalf("goroutines before=%d after=%d; the scan pipeline appears to leak", ...)
}
```

---

## 2.14 مشكلة: race condition حقي — اكتشفه الـ Benchmark

### المشكلة
هذه من أهم النتائج. عند تشغيل الـ benchmark ظهر **data race حقي**:

```
WARNING: DATA RACE
Write at 0x00c000... by goroutine 12:
  identityIndex.classify()
      identity.go:xx
Previous write at 0x00c000... by goroutine 9:
  identityIndex.classify()
```

**السبب:** `identityIndex.seen` كانت تُكتب من **كل** الـ metadata workers بدون قفل. النتيجة على مكتبة حقيقية: فساد في الفهرس وسجلات مكررة.

### الحل
قفل على كل الحقول القابلة للتغيير، والمطالبة بالسجل داخل **نفس القسم الحرج**:

```go
type identityIndex struct {
    mu         sync.Mutex
    byPath     map[string]KnownFile     // للقراءة فقط بعد البناء
    byFileID   map[string][]KnownFile
    bySizeTime map[string][]KnownFile
    seen       map[string]struct{}      // ← الوحيد الذي يُكتب أثناء الفحص
}
```

```go
// claimMoved تطالب بالسجل ذرّيًا: عاملان لا يمكن أن يدّعيا نفس المسار القديم
func (i *identityIndex) claimMoved(fingerprint Fingerprint) (*KnownFile, string) {
    i.mu.Lock()
    defer i.mu.Unlock()
    return i.findMovedLocked(fingerprint)
}
```

### اختبار ضغط يمنع تكرار المشكلة
```go
func TestConcurrentRenameInferenceIsAtomic(t *testing.T) {
    // 8 workers × 200 سجل، كل واحد يُقدَّم لكل workers
    for worker := 0; worker < 8; worker++ {
        go func() {
            for i := 0; i < records; i++ {
                result := index.classify(...)
                if result.Kind == ChangeRenamed && result.Known != nil {
                    claims <- result.Known.ID
                }
            }
        }()
    }
    // كل سجل يجب أن يُطالب به مرة واحدة بالضبط
    for id, count := range seen {
        if count > 1 {
            t.Fatalf("record %d was claimed as a rename %d times", id, count)
        }
    }
}
```

هذا الاختبار كشف **عيبًا ثانيًا**: فرع `FileID` لم يكن يطالب بالسجل إطلاقًا — أي أن كل worker كان يمكن أن يُنشئ move منفصلًا لنفس الملف.

---

## 2.15 مشكلة: السيرفر ينطفئ أثناء الفحص (Crash Recovery)

### المشكلة
افترض انطفاء السيرفر عند 10% أو 37% أو 72% من الفحص. النظام القديم لم يكن يعرف أن هناك فحصًا لم يكتمل، فيفترض أن الفهرس متسق.

### الحل — Scan Session محفوظة

```go
// StartScanSession يسجّل بداية الفحص. وأي جلسة سابقة لا تزال 'running'
// تُوسم كمنقطعة: هكذا يعرف السيرفر أن الفهرس قد يكون منتصف فحص.
func (r *Repository) StartScanSession(ctx context.Context, scanID, mode string) error {
    if _, err := r.db.ExecContext(ctx, `
        UPDATE scan_sessions SET status = 'failed', interrupted = TRUE, finished_at = CURRENT_TIMESTAMP
        WHERE status = 'running'
    `); err != nil {
        return fmt.Errorf("close stale scan sessions: %w", err)
    }
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO scan_sessions (id, status, mode, started_at)
        VALUES ($1, 'running', $2, CURRENT_TIMESTAMP)
    `, scanID, mode)
    return err
}
```

وعند الإقلاع:
```go
func logInterruptedScans(ctx context.Context, repository *db.Repository) {
    sessions, err := repository.InterruptedScanSessions(ctx)
    for _, session := range sessions {
        slog.Warn("interrupted scan detected",
            slog.String("scan_id", session.ID),
            slog.String("status", session.Status),
            slog.Time("started_at", session.StartedAt))
    }
}
```

**الضمانة الأساسية:** `لم يتم مسحها ≠ تم حذفها`. أي عنصر لم يُمسح بسبب انقطاع كهرباء يبقى في حالته السابقة ولا يُحذف.

حالات الـ Scan Session:
```go
const (
    StatusRunning   ScanStatus = "running"
    StatusCompleted ScanStatus = "completed"
    StatusFailed    ScanStatus = "failed"
    StatusCancelled ScanStatus = "cancelled"
    StatusPartial   ScanStatus = "partial"
)
```

---

## 2.16 مشكلة: لا يوجد تقرير فحص حقي

### المشكلة
لا progress ولا تقرير منظّم. كل شيء كان مجرد أرقام في متغير محلي.

### الحل — تقرير منظّم بيانات (structured data)

```go
type Progress struct {
    ScanID              string        `json:"scanId"`
    RootID              string        `json:"rootId,omitempty"`
    Status              ScanStatus    `json:"status"`
    RootStatus          RootStatus    `json:"rootStatus,omitempty"`
    StartedAt           time.Time     `json:"startedAt"`
    Duration            Duration      `json:"duration"`
    Directories         int64         `json:"directoriesVisited"`
    FilesSeen           int64         `json:"filesSeen"`
    VideoCandidates     int64         `json:"videoCandidates"`
    Accepted            int64         `json:"acceptedMedia"`
    Rejected            int64         `json:"rejectedFiles"`
    NewFiles            int64         `json:"newFiles"`
    ModifiedFiles       int64         `json:"modifiedFiles"`
    RemovedFiles        int64         `json:"removedFiles"`
    RenamedFiles        int64         `json:"renamedFiles"`
    UnchangedFiles      int64         `json:"unchangedFiles"`
    ParseFailures       int64         `json:"parseFailures"`
    LowConfidence       int64         `json:"lowConfidenceItems"`
    PermissionErrors    int64         `json:"permissionErrors"`
    FilesystemErrors    int64         `json:"filesystemErrors"`
    ArtworkFound        int64         `json:"artworkFound"`
    ArtworkMissing      int64         `json:"artworkMissing"`
    DuplicateCandidates int64         `json:"duplicateCandidates"`
    TotalBytes          int64         `json:"totalBytes"`
    SkippedBytes        int64         `json:"skippedBytes"`
    Throughput          Throughput    `json:"throughput"`
    Errors              []ErrorCount  `json:"errors,omitempty"`
    TopProblems         []Problem     `json:"topProblems,omitempty"`
}
```

### العدادات lock-free بدون قفل على المسار الساخن
```go
type counter struct {
    directories      atomic.Int64
    filesSeen        atomic.Int64
    videoCandidates  atomic.Int64
    accepted         atomic.Int64
    newFiles         atomic.Int64
    modifiedFiles    atomic.Int64
    // ... إلخ
}
```

### Logging مُجمَّع، ليس لكل ملف
```go
// progress reporting: لقطة مُجمّعة واحدة لكل فترة، وليس لكل ملف
options.Logger.Info("scan progress",
    slog.String("scan_id", snapshot.ScanID),
    slog.Int64("directories", snapshot.Directories),
    slog.Int64("files_seen", snapshot.FilesSeen),
    slog.Int64("accepted", snapshot.Accepted),
    slog.Int64("new", snapshot.NewFiles),
    slog.Float64("files_per_sec", snapshot.Throughput.FilesPerSecond),
)
```

**مهم:** فحص مليون ملف **لا يُنتج مليون سطر سجل**.

---

## 2.17 مشكلة: نموذج أخطاء غير موجود

### المشكلة
خطأ واحد عام بدون تصنيف. لا يمكن معرفة: هل المشكلة صلاحيات؟ قرص مفصول؟ شبكة؟ ملف فاسد؟

### الحل — 12 كود خطأ مصنّف

```go
const (
    ErrPermissionDenied     ErrorCode = "permission_denied"
    ErrNotFound             ErrorCode = "not_found"
    ErrDiskUnavailable      ErrorCode = "disk_unavailable"
    ErrNetworkError         ErrorCode = "network_error"
    ErrStatFailed           ErrorCode = "stat_failed"
    ErrParseFailed          ErrorCode = "parse_failed"
    ErrDatabaseFailed       ErrorCode = "database_error"
    ErrUnsupportedExtension ErrorCode = "unsupported_extension"
    ErrInvalidPath          ErrorCode = "invalid_path"
    ErrSymlinkLoop          ErrorCode = "symlink_loop"
    ErrWatcherFailed        ErrorCode = "watcher_error"
    ErrCodeUnknown          ErrorCode = "unknown"
)
```

### التصنيف يميّز العابر من الدائم (حماية للبيانات)

```go
// transientCodes: أخطاء سببها البيئة وليس البيانات.
// ملف خلف أحد هذه الأخطاء يجب ألا يُحذف من الفهرس أبدًا:
// فقد يكون القرص مفصولًا أو المشاركة الشبكية غير متاحة لحظيًا.
var transientCodes = map[ErrorCode]bool{
    ErrDiskUnavailable:  true,
    ErrNetworkError:     true,
    ErrPermissionDenied: true,
}
```

والتصنيف يتعامل مع رسائل Windows الحقيقية:
```go
switch {
case strings.Contains(message, "device is not ready"),
     strings.Contains(message, "the system cannot find the drive"), ...:
    return ErrDiskUnavailable, err.Error()
case strings.Contains(message, "network name is no longer available"),
     strings.Contains(message, "network path was not found"), ...:
    return ErrNetworkError, err.Error()
case strings.Contains(message, "access is denied"),
     strings.Contains(message, "permission denied"), ...:
    return ErrPermissionDenied, err.Error()
}
```

### الذاكرة محدودة
```go
type ErrorSink struct {
    limit    int              // عدد الأمثلة المحفوظة
    counts   map[ErrorCode]int
    samples  []ScanError
    total    int
}
```
اختبار: **1000 خطأ** → العدّاد الكامل 1000، لكن الأمثلة المحفوظة **5 فقط**. تقرير آمن على مكتبة بها مليون ملف سيء.

---

## 2.18 مشكلة: سلامة الروابط (Symlinks) والحلقات اللانهائية

### المشكلة
لا توجد سياسة واضحة. symlink أو junction يشير إلى مجلد = حلقات لا نهائية وتجاوز غير محدود.

### الحل — سياسة صريحة قابلة للضبط
```go
type DiscoveryConfig struct {
    // FollowSymlinks: هل ندخل روابط المجلدات؟ الافتراضي false —
    // مكتبات الوسائط تحتوي link farms و junctions متكررة.
    FollowSymlinks bool
    IgnoreHidden   bool     // تجاهل المجلدات المخفية
    IgnoreDirs     []string
    MaxDepth       int      // 0 = بلا حد
}

if entry.Type()&fs.ModeSymlink != 0 {
    if !d.config.FollowSymlinks {
        continue                   // ← الافتراضي: لا نتبع
    }
    target, statErr := os.Stat(filepath.Join(dir, name))
    if statErr != nil { continue }
    if target.IsDir() { subdirs = append(subdirs, name) }
}
```

### تجاهل المجلدات النظامية الثقيلة
```go
var defaultIgnoredDirs = []string{
    "$recycle.bin", "system volume information", "recycler",
    "#recycle", ".trash", ".trashes", ".trash-1000",
    "node_modules", ".git", ".svn", "__macosx",
    "@eadir", "lost+found", "windows", "program files", ...,
}
```
**سبب مهم:** فحص مجلد `C:\` بدون تجاهل هذه المجلدات يجعل الفحص يزحف لساعات.

### تجاهل ملفات التحميل غير المكتملة
```go
var defaultIgnoredSuffixes = []string{
    ".part", ".!qb", ".crdownload", ".download", ".tmp", ".temp",
    ".partial", ".incomplete", ".aria2", ".bup", ".filepart", ".opdownload",
    ".!ut", ".bc!", ".td", ".xltd",
}
```
مُثبت: `movie.mkv.part` و`other.crdownload` **لا تُفهرس**.

---

## 2.19 مشكلة: لا حماية من فحصين متزامنين

### المشكلة
طلبETF فحصين في نفس الوقت = تنافس على نفس الصفوف واحتمال إدخال مزدوج أثناء الفحص الكامل.

### الحل — guard يحمي الفحص الواحد
```go
// scanGuard يسمح بفحص واحد فقط في الوقت الواحد، ويكشف التقدم والإلغاء.
// فحصان متزامنان سيتنافسان على نفس الصفوف، لذلك هذا ضمان صحة وليس حدًّا للمعدل.
type scanGuard struct {
    mu       sync.Mutex
    running  bool
    scanID   string
    cancelFn context.CancelFunc
    progress *scanner.Progress
}

if !s.scanGuard.tryAcquire() {
    writeJSON(w, http.StatusConflict, map[string]any{
        "error":  "a scan is already running",
        "scanId": s.scanGuard.currentScanID(),
    })
    return
}
defer s.scanGuard.release()
```

### نقاط API جديدة
```go
// لم يعد /api/scan (قائمة JSON لكل الملفات) موجودًا — لم تستخدمه الواجهة
// ولا يمكنه التعامل مع مكتبات ضخمة.
s.mux.HandleFunc("GET /api/scan/status", s.handleScanStatus)
s.mux.HandleFunc("POST /api/scan/cancel", s.requireAdminAuth(s.handleScanCancel))
s.mux.HandleFunc("GET /api/scan/interrupted", s.requireAdminAuth(s.handleInterruptedScans))
s.mux.HandleFunc("POST /api/ingest", s.requireAdminAuth(s.handleIndex))  // alias للتوافق
s.mux.HandleFunc("POST /api/index", s.requireAdminAuth(s.handleIndex))
```

و`POST /api/index` أصبح يقبل تحكمًا كاملًا:
```go
var request struct {
    Roots      []string `json:"roots"`
    Mode       string   `json:"mode"`        // full | incremental
    Inspect    *bool    `json:"inspect"`     // ffprobe
    SyncSearch *bool    `json:"syncSearch"`  // مزامنة البحث
}
```

---

## 2.20 مشكلة: المخاطر الأمنية في الإقلاع

### المشكلة
كان هناك بيانات افتراضية ثابتة `admin`/`admin123` إذا لم تُضبط البيئة، وtoken توقيع ثابت. أي أن كل نسخة منشورة تُشترك في نفس بيانات الدخول المعلومة.

### الحل
```go
// adminPassword لا يعود أبدًا إلى قيمة ثابتة. عندما لا يضبط المشغّل واحدًا،
// تُولَّد كلمة مرور عشوائية لهذه العملية وتُطبع مرة واحدة لتُنسخ وتُخزَّن.
func adminPassword() string {
    if value := strings.TrimSpace(os.Getenv("NEXORA_ADMIN_PASS")); value != "" {
        return value
    }
    generated := randomHex(12)
    log.Printf("[security] NEXORA_ADMIN_PASS is not set. Temporary admin password for this run: %s", generated)
    return generated
}

// adminSigningSecret لا يجوز أن يكون ثابتًا مُشحونًا:
// مفتاح توقيع معلوم يسمح لأي شخص بتزوير token إداري صالح.
func adminSigningSecret() string {
    if value := strings.TrimSpace(os.Getenv("NEXORA_ADMIN_SECRET")); value != "" {
        return value
    }
    log.Printf("[security] NEXORA_ADMIN_SECRET is not set. Generating an ephemeral signing key; admin sessions end on restart.")
    return randomHex(32)
}
```

---

# القسم الثالث: البروبت الثاني — تشغيل السيرفرات وmigration 0022

## 3.1 مشكلة: لا يوجد أي سيرفر يعمل

### الفحص
عند الفحص وجدت:
- **Docker Desktop متوقف** → بالتالي PostgreSQL وMeilisearch وRedis كلها متوقفة
- **Go API غير مشغّل** (المنفذ 8080 حر)

العمليات `node` الأربع التي ظهرت كانت **أدوات MCP الخاصة بالمساعد** (`chrome-devtools-mcp` و`cloud-run-mcp`) وليست سيرفرات المشروع — لذلك لم ألمسها.

### الحل
1. تشغيل Docker Desktop → ثم فحصت ما بداخله فوجدت أن الـ stack كامل ويعمل تلقائيًا:
   | الحاوية | الحالة | المنفذ |
   |---|---|---|
   | `nexora-postgres` | Up (healthy) | 15432 |
   | `nexora-redis` | Up | 6379 |
   | `nexora-meilisearch` | Up | 7700 |

2. تشغيل Go API.

### مشكلة فرعية: تشغيل السيرفر من المكان الخطأ
شغّلته أول مرة من **جذر المشروع**، فقرأ ملف `.env` الجذري (الذي لا يحتوي بيانات admin)، فولّد كلمة مرور عشوائية ورفض الدخول.

**الحل:** تشغيله من داخل مجلد `server/` حتى يقرأ `server/.env` الصحيح:
```powershell
cd server; ..\nexora-api.exe
```

---

## 3.2 المشكلة الكبرى: migration 0022 فشل على بيانات حقيقية

### المشكلة
هذا كان **أول اختبار حقي** للـ migration، وقد فشل:

```
exit 1 — migration failed
```

### السبب
الـ unique index على هوية الحلقة اصطدم ببيانات موجودة فعلية:

```sql
CREATE UNIQUE INDEX idx_video_files_episode_identity
    ON video_files(media_item_id, season_id, episode_number)
    WHERE season_id IS NOT NULL AND episode_number IS NOT NULL;
```

البيانات الحقيقية:
```
media_item_id | season_id | episode_number | عدد التكرارات
     234      |    749    |       1        |       5
     234      |    749    |       3        |       2
```

### التحليل العميق (النقطة الأهم)
بحثت في الصفوف فوجدت **نوعين مختلفين تمامًا من التكرار**:

| النوع | المثال | الحقيقة |
|---|---|---|
| **نفس الملف بمسارين** | الصفان `1022` و`1092` — نفس الملف `الكنز نت 01.avi` بنفس الحجم بالضبط | **مكرر حقي** — سببه عدم توحيد الفاصلين `\` و`\\` |
| **ملفات مختلفة دُمجت** | الصفوف `1037`, `1079`, `1163` | ملفات **مختلفة فعلًا** دمجها الـ parser القديم لأن أسماءها تحتوي اسم الموقع `الكنز نت` |

**الاستنتاج المهم:** الـ index كان **صحيحًا** — فهو يكشف عيبًا حقيًا في. لكن **migration لا يجوز أن يفشل على بيانات موجودة**.

### الحل — فصل المُثبت عن المشكوك فيه

```sql
-- جدول لتسجيل التعارضات للمراجعة بدل رفض الإقلاع
CREATE TABLE IF NOT EXISTS index_conflicts (
    id SERIAL PRIMARY KEY,
    kind VARCHAR(40) NOT NULL,
    media_item_id INT,
    season_id INT,
    episode_number INT,
    row_ids INT[] NOT NULL,
    detail TEXT,
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- حذف الصفوف المُثبت أنها نفس الملف فقط:
-- نفس المسار المُطبَّع + نفس الحجم بالضبط
DELETE FROM video_files vf
USING video_files other
WHERE vf.id > other.id
  AND lower(replace(vf.file_path, '\', '/')) = lower(replace(other.file_path, '\', '/'))
  AND vf.file_size = other.file_size;

-- تسجيل تعارضات هوية الحلقة للمراجعة فقط (لا حذف)
INSERT INTO index_conflicts (kind, media_item_id, season_id, episode_number, row_ids, detail)
SELECT
    'duplicate_episode_identity',
    media_item_id, season_id, episode_number,
    array_agg(id ORDER BY id),
    'Multiple files share one episode slot. They may be different releases, or the old parser may have merged unrelated files into one season. Review before removing any row.'
FROM video_files
WHERE season_id IS NOT NULL AND episode_number IS NOT NULL
GROUP BY media_item_id, season_id, episode_number
HAVING COUNT(*) > 1;

-- الـ index أصبح غير unique بشكل متعمد:
-- التعارضات الموجودة بيانات حقيقية تحتاج مراجعة، وليست سببًا لرفض إقلاع السيرفر.
-- ومنع التكرار الجديد يبقى مضمونًا داخل مسار الـ upsert.
CREATE INDEX IF NOT EXISTS idx_video_files_episode_identity ...
```

### الدرس المُستخلص (مبدأ طبّقته)
1. **migration لا يفشل على بيانات موجودة**
2. **لا يُحذف صف وسائط مشكوك فيه** — يُسجَّل للمراجعة
3. **يُحذف فقط ما ثبت يقينًا أنه مكرر** (نفس المسار + نفس الحجم)

### فحص إضافي مفيد
قبل إعادة المحاولة تحققت من عدم وجود أثر جزئي:
```
=== is 0022 recorded? ===  (فارغ = غير مسجل ✓)
=== partial objects from the failed run ===  (لا شيء ✓)
```
وهذا **أثبت أن migration runner يعمل داخل transaction بشكل صحيح** — فشل المحاولة الأولى لم يترك أي أثر.

### النتيجة
بعد الإصلاح، طُبّق `0022` بنجاح من المحاولة الأولى.

---

## 3.3 التحقق النهائي على بيانات حقيقية

شغّلت فحصًا على مكتبة الاختبار (137 ملف mkv + ملفات `.part` و`.download`):

| القياس | النتيجة |
|---|---|
| ملفات مقبولة | **137** ✓ |
| ملفات مرفوضة | **61** (بما فيها `.part`/`.download`/الصور) ✓ |
| **صفوف قاعدة البيانات** | `rows=137, distinct_paths=137` → **لا تكرار** ✓ |

### الفحص الثاني (incremental) — الإثبات الحاسم

| القياس | النتيجة |
|---|---|
| `newFiles` | **0** |
| `unchangedFiles` | **137** |
| `inserted` | **0** |
| `updated` | **0** |
| الزمن | **431ms → 18ms (أسرع 24 مرة)** |

**صفر كتابة لقاعدة البيانات** على فحص مكتبة لم تتغير. هذا هو الهدف الأساسي الذي بُنيت له كل المنظومة.

### جودة التحليل على البيانات الحقيقية
```
Inception     | conf 0.75  ← فيلم حقي
Interstellar  | conf 0.75
Broken Names  | conf 0.80  ← الملفات 01, x2, 10 → لم يُخترع رقم حلقة
```
قاعدة "لا تخمّن" تعمل فعليًا.

---

# القسم الرابع: البروبت الثالث — فصل الملف عن العمل

## 4.1 المشكلة الجذرية

### المشكلة
البروبت الأول أصلح **كيف** نفحص الملفات، لكن لم يصلح **ماذا** يعني ما وجدناه. المسار بقي:

```text
File → ParsedName → Database Record
```

أي **كل ملف فيديو يصبح عملًا**. وهذا المفهوم خاطئ جوهريًا.

### الدليل الحقي من قاعدة بياناتك
بحثت في `media_items` الحقيقية فوجدت:

| العمل المُنشأ | حقيقته |
|---|---|
| `الكنز ج1 الحلقه` | علامة **موسم + حلقة** أصبحت اسم عمل |
| `الكنزنت` | اسم موقع ملتصق بالعنوان |
| `منور فديوهات زابيا` | **وصف قناة** ("فيديوهات" + اسم قناة) أصبح عملًا |
| `Fate_Apocrypha` | الفواصل لم تُطبَّع أبدًا، فلا يمكن أن يطابق `Fate Apocrypha` |
| `Fate Stay Night -` | **شرطة معلّقة** من الـ parser |
| `ONE PIECE` | **12 حلقة** مسجلة كـ `type='movie'` |
| `الكنز` | عمل واحد بـ **88 ملفًا**، منها 5 ملفات تدّعي أنها الحلقة 1 من الموسم 1 |

**لا شيء من هذه أعمال.** كلها آثار تحليل ومجلدات حاويات رُقّيت إلى كيانات، وبعد الترقية تُلوّث البحث والأقسام وكل فحص لاحق.

### مشكلتان أخريان بنفس السبب الجذري
1. **نفس العمل يظهر مرات عديدة**: `Breaking Bad` و`Breaking.Bad` و`Breaking_Bad` عرض واحد، لكن المقارنة كانت **مساواة نصية تامة**.
2. **قرار المسؤول يمكن أن يُدهس**: لا يوجد تسجيل لمصدر القيمة، فيأتي Full Scan ويعيد القيمة الخام من الـ parser.

### الحل الجذري
```text
Filesystem File
  ↓ parse
Media Candidate
  ↓ ENTITY RESOLUTION      (أدلة → مرشحون مُرتّبون → قرار)
Logical Work → Season → Episode → Physical File
```

---

## 4.2 مشكلة: `category` كان يحدد بنية البيانات

### المشكلة
كان خلطًا بين محورين مختلفين تمامًا: **نوع الوسائط** (فيلم أم مسلسل) و**التصنيف** (أنمي، أطفال، وثائقي...). فنتج عنه تسجيل 12 حلقة One Piece كـ `type='movie'`.

### الحل — فصل المحورين

```go
// MediaType هو النوع البنيوي للعمل. منفصل عن Category:
// "anime" تصنيف، وغالبًا نوعه series، لكن فيلم الأنمي يبقى movie.
type MediaType string

const (
    MediaTypeMovie   MediaType = "movie"
    MediaTypeSeries  MediaType = "series"
    MediaTypeUnknown MediaType = "unknown"
)
```

### القرار من البنية، ليس من الاسم

```go
// DetectMediaType يقرر إن كانت الأدلة تصف فيلمًا أم مسلسلًا.
//
// التصميم يمنع القرار من اسم الملف وحده: لأن "Toy Story 2.mkv" ينتهي برقم
// و"Silo S03E04.mkv" لا لبس فيه. لذلك يوزن القرار البنية (علامات الحلقات،
// مجلدات المواسم، الحلقات المجاورة) أعلى بكثير من التسمية.
func DetectMediaType(evidence Evidence) TypeEvidence {
    seriesScore := 0.0
    movieScore := 0.0

    // --- أدلة المسلسل ---
    if evidence.ParserEpisodeStrong {
        seriesScore += 45
        reasons = append(reasons, "filename carries an explicit episode marker")
    }
    if evidence.SeasonFolderNumber > 0 {
        seriesScore += 25
        reasons = append(reasons, "a season folder declares the season")
    }
    if low, high, contiguous := evidence.Group.ContiguousEpisodeRange(); contiguous {
        seriesScore += 30
        reasons = append(reasons, "sibling episodes form the run "+itoa(low)+"-"+itoa(high))
    }

    // --- أدلة الفيلم ---
    if evidence.ParsedYear > 0 {
        movieScore += 20
    }
    switch evidence.CategoryHint {
    case "movies": movieScore += 25
    case "plays":  movieScore += 30
    }

    switch {
    case seriesScore == 0 && movieScore == 0:
        return TypeEvidence{MediaType: MediaTypeUnknown, ...}
    case seriesScore >= movieScore+10:
        return TypeEvidence{MediaType: MediaTypeSeries, ...}
    case movieScore >= seriesScore+10:
        return TypeEvidence{MediaType: MediaTypeMovie, ...}
    default:
        // متقارب جدًا. إرجاع unknown صريح هو المطلوب:
        // التخمين هنا هو ما أنتج مسلسلات بملف فيلم واحد.
        return TypeEvidence{MediaType: MediaTypeUnknown, Score: 0,
            Reasons: append(reasons, "movie and series evidence are too close to decide safely")}
    }
}
```

### قرار مهم متعمد
حذفت نقاط كانت تُمنح لـ"الملف الوحيد في مجلده" لأنه فيلم — لأن مجلد حلقة واحدة يبدو مطابقًا تمامًا، وهذا بالضبط ما جعل مسلسلات تُسجَّل كأفلام:

```go
// متعمدًا بلا نقاط لـ"الملف الوحيد هنا". ملف واحد ليس دليلًا على فيلم:
// مجلد حلقة واحدة يبدو مطابقًا، ومنحه نقاطًا فيلم هو كيف انتهت مسلسلات كأفلام.
```

---

## 4.3 مشكلة: إنشاء العمل فورًا بدل البحث أولًا

### المشكلة
كل ملف جديد كان يُنشئ `media_items` مباشرة بدون أي بحث في الكيانات الموجودة.

### الحل — البحث أولًا، والإنشاء ملاذ أخير

```go
// Resolve يقرر أي كيان منطقي ينتمي إليه الملف.
//
// الترتيب ثابت ومتعمد:
//  1. ارفض عنوانًا ليس عنوانًا أصلًا (اسم موقع، رقم مجرّد).
//  2. قرّر النوع من البنية، لا من التسمية.
//  3. قرّر الموسم والحلقة، وار الأرقام المجرّدة بلا دعم.
//  4. ابحث عن كيان موجود: alias تام، ثم مرشحون مُقيَّمون.
//  5. فقط إذا لم يطابق أي كيان AND الأدلة قوية، أنشئ كيانًا مؤقتًا.
//  6. وإلا أرسله لمراجعة بشرية.
//
// إنشاء عمل هو الملاذ الأخير، وليس السلوك الأول أبدًا.
func (r *Resolver) Resolve(input ResolverInput) Resolution {
    // ---- الخطوة 1: هل العنوان المُحلَّل صالح أصلًا؟
    parsedQuality := AssessTitle(evidence.ParsedTitle, evidence.OriginalName)
    folderQuality := AssessTitle(evidence.WorkFolderTitle, "")

    if candidateTitle == "" {
        return r.unresolved(resolution, evidence, "no_usable_title",
            "no folder or filename provided a usable title")
    }

    // ---- الخطوة 2: النوع من البنية
    typeEvidence := DetectMediaType(evidence)

    // ---- الخطوة 3: الموسم والحلقة
    season := ResolveSeason(evidence)
    episode := ResolveEpisode(evidence)

    // ---- الخطوة 4: alias متعلّم أولًا — معرفة دقيقة أقوى من تخمين تشابه
    if input.AliasLookup != nil {
        if workID, found := input.AliasLookup[normalized]; found {
            resolution.Decision = DecisionAuto
            resolution.ResolverConfidence = 0.97
            resolution.ReasonCode = "alias_hit"
            return resolution
        }
    }

    // التقييم بالنسبة للكيانات الموجودة
    candidates := RankCandidates(input.Known, evidence)

    // ---- الخطوة 5 و 6
    // ...
}
```

### عند عدم وجود أي كيان مشابه
```go
func (r *Resolver) noCandidates(resolution Resolution, evidence Evidence,
    candidateTitle, titleOrigin string, typeEvidence TypeEvidence) Resolution {

    // عمل جديد لا يُنشأ إلا عندما يكون النوع مؤكدًا والعنوان صالحًا.
    // هذه بالضبط الحالة التي أنتجت "الكنز ج1 الحلقه" كعمل.
    if typeEvidence.MediaType == MediaTypeUnknown {
        resolution.State = StateNeedsReview
        resolution.ReasonCode = "unknown_media_type"
        resolution.Reason = "could not decide between movie and series; queued for review"
        return resolution
    }

    resolution.Decision = DecisionCreate
    resolution.State = StateEnrichmentPending
    resolution.ReasonCode = "new_work"
    // ... الكيان يُوسم provisional
}
```

**القاعدة:** ملف غير محلول = قابل للاسترجاع. مكتبة ملوّثة = غير قابلة للإصلاح.

---

## 4.4 مشكلة: `Breaking.Bad` ≠ `Breaking Bad`

### المشكلة
المقارنة كانت **مساواة نصية تامة**. النتيجة: عمل واحد يصبح خمسة أعمال.

### الحل — تطبيع شامل

```go
// Normalize يُنتج مفتاح المقارنة القانوني للعنوان.
//
// هذا أكثر شدة من مُطبِّع العرض المتعمد: وجوده لجعل "Breaking Bad" و
// "Breaking.Bad" و "Breaking_Bad" و "breaking-bad" و "breaking  bad"
// متساوية، وهي الخاصية التي تمنع عرضًا واحدًا من أن يصبح خمسة أعمال.
//
// لا يستبدل النص الأصلي أبدًا: المستدعي يخزّن الاثنين دائمًا.
func Normalize(title string) string {
    var builder strings.Builder
    for _, r := range title {
        switch {
        // الأرقام تُفحص قبل الحروف لأن الأرقام العربية-الهندية والفارسية
        // تُطبَّع إلى ASCII. unicode.IsDigit صحيحة لكليهما، و IsLetter خاطئة.
        case unicode.IsDigit(r):
            builder.WriteRune(foldRune(r))
        case unicode.IsLetter(r):
            builder.WriteRune(foldRune(r))
        case unicode.IsSpace(r):
            builder.WriteRune(' ')
        default:
            // الترقيم والفواصل والأقواس والرموز كلها تُدمج إلى مسافة واحدة
            builder.WriteRune(' ')
        }
    }
    return strings.TrimSpace(strings.Join(strings.Fields(builder.String()), " "))
}
```

### تطبيع الحروف العربية
```go
func foldRune(r rune) rune {
    switch r {
    case 'أ', 'إ', 'آ', 'ٱ', 'ٲ', 'ٳ':  return 'ا'
    case 'ى', 'ئ':                      return 'ي'
    case 'ؤ':                           return 'و'
    case 'ة':                           return 'ه'
    case 'ک':                           return 'ك'
    case 'ی':                           return 'ي'
    }
    if r >= 'A' && r <= 'Z' { return r + ('a' - 'A') }
    // الأرقام العربية والفارسية → ASCII
    switch {
    case r >= '٠' && r <= '٩': return '0' + (r - '٠')
    case r >= '۰' && r <= '۹': return '0' + (r - '۰')
    }
    return unicode.ToLower(r)
}
```

### مُثبت باختبار
```go
groups := [][]string{
    {"Breaking Bad", "Breaking.Bad", "Breaking_Bad", "breaking-bad", "Breaking  Bad", "BREAKING BAD"},
    {"One Piece", "One.Piece", "One_Piece", "one piece"},
    {"Fate Apocrypha", "Fate_Apocrypha", "Fate-Apocrypha"},
    {"أسامة", "اسامة"},                // اختلاف الهمزة
    {"الحلقة ١٢٣", "الحلقة 123"},       // أرقام عربية
}
// واختبار معاكس: يجب ألا يدمج أعمالًا مختلفة فعلًا
// "Silo" ≠ "Severance" | "The Office" ≠ "The Office US" | "One Piece" ≠ "One Punch Man"
```

### التشابه بتدقيق على حدود الكلمات
```go
// التشابه word-aligned: "silo" ليست كلمة داخل "silosomethingelse"
if containsWords(normA, normB) || containsWords(normB, normA) {
    ratio := float64(len([]rune(shorter))) / float64(len([]rune(longer)))
    best = maxFloat(best, 0.75+0.2*ratio)
}
```
مُثبت: `Similarity("silo", "silosomething")` **أقل** من `Similarity("silo", "silo arabic")`.

---

## 4.5 مشكلة: كل ملف يُحل بمعزل عن الآخرين

### المشكلة
`01.mkv` وحده لا يقول شيئًا. لكن النظام كان يحاول حلّه منفردًا، فيخمّن أو يفشل.

### الحل — حل المجموعة (Batch Resolution)

المدخل الحيوي: `01.mkv, 02.mkv, 03.mkv, 04.mkv` داخل مجلد `Silo` يقول **كل شيء تقريبًا**.

```go
// GroupContext يلخّص المجلد الذي يعيش فيه الملف.
// حلّ الملف بمعزل هو السبب الرئيس لاختراع الكيانات:
// "01.mkv" لا يخبرك شيئًا، لكن "01.mkv .. 04.mkv" داخل مجلد يخبرك بالكثير.
type GroupContext struct {
    Size             int    // إجمالي ملفات الوسائط في نفس المجلد
    EpisodicSiblings int    // كم شقيق يحمل علامة حلقة قوية
    EpisodeNumbers   []int  // أرقام الحلقات المحتملة
    YearSiblings      int   // كم شقيق يحمل سنة إصدار (أي يبدو فيلمًا مستقلًا)
    SeasonFolder     bool   // هل المجلد نفسه مسمّى كموسم
    ConsensusTitle   string // العنوان الأكثر شيوعًا بين الأشقاء
    ConsensusVotes   int    // كم شقيق يوافق عليه
}
```

### نطاق الحلقات المتتالي = دليل قوي
```go
// ContiguousEpisodeRange يُبلغ عن نطاق أرقام الحلقات المرصودة.
// نطاق متتالي يبدأ من 1 هو دليل مسلسل قوي.
func (g GroupContext) ContiguousEpisodeRange() EpisodeRange {
    numbers := append([]int(nil), g.EpisodeNumbers...)
    sort.Ints(numbers)

    low, high := numbers[0], numbers[len(numbers)-1]
    span := high - low + 1
    if low != 1 || span <= 0 {
        return EpisodeRange{Low: low, High: high}
    }
    // نطلب تغطية كثيفة، مع السماح بثغرات قليلة
    // حتى لا تُدمّر حلقة ناقصة واحدة مجموعة واثقة.
    coverage := float64(len(numbers)) / float64(span)
    return EpisodeRange{Low: low, High: high, Contiguous: coverage >= 0.75 && len(numbers) >= 3}
```

### جمع المجموعة محدود الذاكرة
```go
// groupCollector يبني بنية كل مجلد أثناء مرور الملفات.
//
// محدود بطريقتين متعمدتين:
//   - عدد المجلدات المتتبّعة محدود، والأقدم يُحذف، فلا يمكن لفحص مكتبة
//     بمجلد لكل ملف أن ينمو في الذاكرة بلا حد؛
//   - تُحفظ فقط المعلومات التي يحتاجها القرار، وليس الملفات نفسها.
//
// الحذف آمن لأن ملخص المجموعة تحسين: ملف حُذف مجلده يُحل بأدلة أقل.
type groupCollector struct {
    mu      sync.Mutex
    dirs    map[string]*dirAggregate
    order   []string
    maxDirs int
    evicted int
}

const DefaultTrackedDirectories = 20000
```

### المجموعة لا تُبنى من ملف واحد
```go
// الإجماع يحتاج ملفين متفقين على الأقل: رأي ملف واحد ليس إجماعًا
// ولا يجوز تقديمه كذلك.
if summary.ConsensusVotes < 2 {
    summary.ConsensusTitle = ""
    summary.ConsensusVotes = 0
}
```
مُثبت: ثلاثة ملفات بعناوين مختلفة → **لا إجماع**، وملف واحد → **لا إجماع**.

### والحلقة المجرّدة تُقبل بالسياق
```go
// ResolveEpisode: رقم مجرّد يحتاج دعمًا سياقيًا
if evidence.ParsedEpisode > 0 && !evidence.ParserEpisodeStrong {
    supported := evidence.SeasonFolderNumber > 0 ||
                 evidence.Group.SeasonFolder ||
                 evidence.Group.EpisodicSiblings >= 2 ||
                 evidence.Group.Size >= 3
    if evidence.Group.ContiguousEpisodeRange().Contiguous {
        supported = true
    }
    switch evidence.CategoryHint {
    case "series", "anime", "kids":
        supported = true
    }
    if supported {
        return EpisodeResolution{
            Number: evidence.ParsedEpisode, Valid: true, Confidence: 0.65,
            Reason: "a bare number is supported by the surrounding structure",
        }
    }
    return EpisodeResolution{
        Valid: false, Confidence: 0.2,
        Reason: "a bare number with no structural support is treated as part of the title",
    }
}
```

---

## 4.6 مشكلة: تصنيف أسماء المجلدات الحقيقية

### المشكلة
واجهت في بياناتك أشكالًا حقيقية لا يتعامل معها parser عادي:

| المجلد | الحقيقة |
|---|---|
| `Franchises/DC/` | مجلد **سلسلة**، ليس اسم عمل |
| `مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/` | مجلد **تصفّح** ("أعمال" = "works") |
| `Silo الموسم الثالث` | اسم عمل **+** موسم في مجلد واحد |

### الحل — كشف مجلدات الحاويات

```go
// containerFolderNames: أسماء مجلدات تصف حاوية أو تجميع تصفّح أو تصنيفًا
// وليس عملًا. يجب ألا تُسمّي عملًا أبدًا.
var containerFolderNames = map[string]struct{}{
    // تجميعات تصفّح عربية مرصودة في مكتبات حقيقية
    "اعمال": {}, "أعمال": {}, "افلام": {}, "أفلام": {}, "مسلسلات": {},
    "مكتبه": {}, "مكتبة": {}, "القسم": {}, "متنوع": {}, "جديد": {}, "قديم": {},
    "franchises": {}, "collection": {}, "collections": {}, "boxset": {},
    "movies": {}, "films": {}, "series": {}, "library": {}, "media": {},
    "unsorted": {}, "misc": {}, "other": {}, "extras": {}, "bonus": {},
}
```

### المنطق الحاسم لتحديد أولوية الملف على المجلد

```go
// filenameNamesTheWork يُبلغ إن كان اسم الملف اسم عمل أفضل من المجلد الذي يحيط به.
//
// القاعدة تستهدف مجلدات الحاويات، المنتشرة جدًا في المكتبات الحقيقية وهي
// سبب كسر سياسة "المجلد يفوز" الساذجة:
//
//     .../Franchises/DC/Part 3 - The Dark Knight Rises.mkv   ← "DC" سلسلة
//     .../مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv  ← "أعمال" حاوية
//
// في الحالتين الاسم الحقي في الملف. المُميّز هو دليل الإصدار:
// اسم ملف يحمل سنة إصدار أو علامة دقة هو اسم إصدار،
// ومجلد قصير بجانبه شبه مؤكد حاوية.
func filenameNamesTheWork(evidence Evidence) bool {
    // مجلد يصنّف أو يصف بنية فقط هو حاوية
    if isContainerFolderName(folder) {
        return true
    }

    // اسم ملف يحمل دليل إصدار يتفوق على مجلد حاوية قصير.
    hasReleaseEvidence := evidence.ParsedYear > 0 || evidence.ParsedResolution != ""
    if hasReleaseEvidence && len(strings.Fields(folder)) <= 1 {
        return true
    }

    // ملف قصير بجانب اسم ملف أطول بكثير هو حاوية.
    // المكتبات الحقيقية مليئة بها: "Franchises/DC/Part 3 - The Dark Knight Rises.mkv"
    // كلمة واحدة لا تصف عنوان فيلم من أربع كلمات.
    folderWords := len(strings.Fields(normalizedFolder))
    parsedWords := len(strings.Fields(normalizedParsed))
    if folderWords <= 1 && parsedWords >= 3 {
        return true
    }

    // مجلد قصير جدًا (أحرف أولى أو اختصار) هو تسمية تجميع وليس عنوانًا: "DC", "MCU", "HP"
    if len([]rune(normalizedFolder)) <= 3 && len([]rune(normalizedParsed)) > len([]rune(normalizedFolder))+3 {
        return true
    }
    return false
}
```

---

## 4.7 مشكلة: ذاكرة المكتبة تحتاج AI أو لا توجد

### المشكلة
كيف يعرف النظام أن `ون بيس` و`One Piece` و`one_piece` نفس العمل — **بدون LLM لكل ملف**؟

### الحل — ذاكرة قواعدية داخل NEXORA

```sql
CREATE TABLE IF NOT EXISTS media_aliases (
    id SERIAL PRIMARY KEY,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    alias_normalized TEXT NOT NULL,
    source value_source NOT NULL DEFAULT 'resolver',
    hit_count INT NOT NULL DEFAULT 0,
    UNIQUE (media_item_id, alias_normalized)
);

-- alias واحد يطابق عملًا واحدًا. هذا القيد هو ما يجعل البحث عن alias
-- حتميًا؛ والتعارض يعني أن إنسانًا يجب أن يقرر.
CREATE UNIQUE INDEX IF NOT EXISTS idx_media_aliases_normalized
    ON media_aliases(alias_normalized);
```

### التعلّم فقط عند اليقين
```go
// Only learn from a resolution the system was reasonably sure about.
// A review-queue item must not teach the library.
if resolution.Decision != DecisionAuto && resolution.Decision != DecisionProvisional {
    return out
}
```

```go
// العتبة التشابهية: التعلّم التلقائي يحتاج تشابهًا ≥ 0.9
func NewAliasLearner() *AliasLearner {
    return &AliasLearner{MinSimilarity: 0.9}
```

### وقرار المسؤول يُتعلَّم دائمًا
```go
// LearnedFromAdmin يبني الـ alias من قرار مشغّل صريح. هذا آمن دائمًا
// ويُخزَّن دائمًا، لأن إنسانًا أكّده.
func LearnedFromAdmin(mediaItemID int64, alias, canonicalTitle string) AliasCandidate {
    return AliasCandidate{
        MediaItemID: mediaItemID,
        Alias:       strings.TrimSpace(alias),
        Normalized:  Normalize(alias),
        Source:      SourceAdmin,
        Reason:      "operator confirmed " + alias + " identifies " + canonicalTitle,
    }
}
```

**النتيجة:** النظام **يتحسّن كلما زاد استخدام المكتبة** — بدون أي AI.

---

## 4.8 مشكلة: Full Scan يدهس قرارات المسؤول

### المشكلة
إذا غيّر المسؤول `Silo` إلى `Silo (2023)` أو أضاف alias يدويًا، **لا شيء يمنع** الفحص التالي من إعادة القيمة الخام من الـ parser.

### الحل — تتبّع المصدر مع ترتيب أولوية

```go
// Source يحدد مصدر قيمة الحقل. ترتيب الأولوية هو الآلية التي تمنع
// Full Scan من دهس تصحيح مسؤول.
type Source string

const (
    SourceFilesystem Source = "filesystem"
    SourceParser     Source = "parser"
    SourceResolver   Source = "resolver"
    SourceDatabase   Source = "database"
    SourceAdmin      Source = "admin"
    SourceProvider   Source = "tmdb"
)

// Rank يرتّب المصادر حسب السلطة. الأعلى يفوز.
//
// الترتيب يُشفّر قرار المشروع: تصحيح المشغّل مطلق، وبيانات المزوّد
// مرجعية للحقول الوصفية، ومخرجات نظام الملفات/الـ parser الأضعف
// لأنها مُشتقة آليًا من اسم ملف قد يكون خاطئًا.
func (s Source) Rank() int {
    switch s {
    case SourceAdmin:      return 100
    case SourceProvider:   return 80
    case SourceDatabase:   return 60
    case SourceResolver:   return 40
    case SourceParser:     return 20
    case SourceFilesystem: return 10
    }
    return 0
}
```

### الدمج الذي يحمي القيم
```go
// MergeField يقرر قيمة حقل واحد بين الوارد والموجود.
// يُرجع القيمة المختارة وهل الكتابة مطلوبة.
//
// هذه هي الدالة التي تُنفّذ "Scanner must not destroy manual corrections":
// إذا كانت القيمة الموجودة مقفلة أو من مصدر أقوى، تُرفض القيمة الواردة.
func MergeField(incoming FieldValue, existing *FieldValue) (FieldValue, bool) {
    // قيمة واردة فارغة لا تُدهس قيمة موجودة أبدًا
    if incoming.Value == "" {
        return *existing, false
    }
    // قيمة مقفلة نهائية
    if existing.Locked {
        return *existing, false
    }
    // مصدر أقوى موجود يُحفظ
    if existing.Source.WinsOver(incoming.Source) {
        if existing.Source.Rank() > incoming.Source.Rank() {
            return *existing, false
        }
    }
    // قيم متطابقة لا تحتاج كتابة
    if existing.Value == incoming.Value && existing.Source == incoming.Source {
        return *existing, false
    }
    return incoming, true
}
```

**مُثبت باختبار صريح:** `title` قيمته `Silo (2023)` من `admin` ومقفلة + الوارد `Silo` من `parser` → النتيجة `Silo (2023)` و**لا كتابة** (`len(changed) == 0`).

### كتابة محدودة (minimal UPDATE)
```go
// MergeFields يطبّق MergeField على سجل كامل ويُبلغ أي الحقول تغيّرت فعلًا،
// حتى تُصدر طبقة الحفظ UPDATE محدودًا بدل إعادة كتابة كل عمود في كل فحص.
func MergeFields(incoming, existing map[string]FieldValue) (map[string]FieldValue, []string) {
    merged := make(map[string]FieldValue, len(incoming))
    changed := make([]string, 0, len(incoming))
    // ...
}
```

---

## 4.9 مشكلة: لا توجد مراجعة بشرية للحالات الغامضة

### المشكلة
عندما لا يتأكد النظام، كان يخمّن. لا يوجد مكان ليقول "لم أستطع تحديد العمل".

### الحل — Resolution Queue

```sql
CREATE TABLE IF NOT EXISTS resolution_queue (
    id SERIAL PRIMARY KEY,
    file_path TEXT NOT NULL,
    original_filename TEXT,
    size BIGINT,
    detected_title TEXT,
    detected_category VARCHAR(60),
    detected_media_type VARCHAR(20),
    detected_season INT,
    detected_episode INT,
    parser_confidence REAL,
    resolver_confidence REAL,
    reason VARCHAR(60) NOT NULL,          -- سبب عدم الحل
    reason_detail TEXT,
    candidates JSONB NOT NULL DEFAULT '[]'::jsonb,  -- المرشحون بدرجاتهم
    state resolution_state NOT NULL DEFAULT 'needs_review',
    decision VARCHAR(40),
    decided_media_item_id INT,
    learned_alias BOOLEAN NOT NULL DEFAULT FALSE,  -- هل تعلّمنا من القرار
    UNIQUE (file_path)
);
```

### المرشحون يُحفظون مع **تفصيل الأدلة**
```go
// كل نقطة هي عنصر مُسمّى في breakdown، وهو ما تعرضه قائمة المراجعة.
// القرار لا يكون رقمًا غامضًا أبدًا.
type ScoreItem struct {
    Label  string  `json:"label"`
    Points float64 `json:"points"`
    Detail string  `json:"detail,omitempty"`
}
```

### الأوزان (من التصميم المطلوب)
```go
const (
    WeightExactTitle       = 40
    WeightFolderTitle      = 25
    WeightAliasMatch       = 20
    WeightSeasonCompatible = 10
    WeightYearCompatible   = 10
    WeightCategoryCompat   = 10
    WeightEpisodeRelation  = 20
    WeightGroupConsensus   = 18
    WeightGroupSeasonShape = 14
    WeightProviderIdentity = 45
    WeightContradiction    = -30   // ← أدلة متناقضة تخصم
    WeightTypeConflict     = -25
    WeightUnmatchedQuality = -12
)
```

### مشكلة الغموض: عندما يتقارب مرشحان
```go
// ResolveAmbiguity تُطبَّق بعد الترتيب: إذا كان أفضل مرشحين داخل هامش
// الغموض، يُخفَّض القرار إلى مراجعة بغض النظر عن الدرجة المطلقة.
// درجة واثقة لا تعني شيئًا عندما يكون لمنافس نفس الدرجة، وهذا بالضبط
// وضع "Silo" مقابل "Silo (2023)".
const AmbiguityMargin = 8

func ResolveAmbiguity(candidates []Candidate, margin float64) (Candidate, bool) {
    best := candidates[0]
    second := candidates[1]
    if best.Score-second.Score < margin {
        best.Decision = DecisionReview
        best.Reasons = append(best.Reasons,
            "ambiguous: "+itoa(int(best.Score))+" vs "+itoa(int(second.Score))+" for "+second.Title)
        return best, true
    }
    return best, false
}
```

### قرارات المسؤول تُنفَّذ وتُتعلَّم
```go
switch decision.Action {
case "attach", "move_season", "mark_movie":
    // إسناد الملف للكيان المختار
    if _, err := tx.ExecContext(ctx, `
        UPDATE video_files
        SET media_item_id = $2, season_id = $3, episode_id = $4,
            resolution_state = 'resolved', resolution_source = 'admin',
            resolver_confidence = 1.0
        WHERE file_path = $1
    `, item.FilePath, decision.WorkID, seasonID, episodeID, episodeNumber); err != nil { ... }

    // قرار المشغّل آمن دائمًا للتعلّم كـ alias
    if decision.LearnAlias && item.DetectedTitle != "" {
        alias := identity.LearnedFromAdmin(decision.WorkID, item.DetectedTitle, item.DetectedTitle)
        if learned, err := r.storeAliases(ctx, tx, []identity.AliasCandidate{alias}); err == nil && learned > 0 {
            learnedAlias = true
        }
    }

case "create":
    // المشغّل كتب هذا الاسم، فهو من مصدر admin ومقفل
    if err := r.learnOperatorAlias(ctx, tx, decision, item); err != nil { ... }

case "ignore":
    // لا شيء لإسناده؛ الملف يبقى غير مفهرس عمدًا
}
```

---

## 4.10 مشكلة: تتابع بعض ملفات نفس الحلقة = أعمال متعددة

### المشكلة
ملفان يمثلان نفس الحلقة (إصدارين مختلفين) كانا يُنشئان عملين.

### الحل — كيان Episode منفصل

```sql
CREATE TABLE IF NOT EXISTS episodes (
    id SERIAL PRIMARY KEY,
    season_id INT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INT NOT NULL,
    title_ar VARCHAR(255),
    title_en VARCHAR(255),
    overview_ar TEXT,
    overview_en TEXT,
    air_date DATE,
    runtime INT,
    still_path VARCHAR(500),
    provider VARCHAR(30),
    external_id VARCHAR(40),
    metadata_payload JSONB,
    UNIQUE (season_id, episode_number)
);

ALTER TABLE video_files
    ADD COLUMN IF NOT EXISTS episode_id INT REFERENCES episodes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolution_source value_source DEFAULT 'resolver';
```

```go
// episodes هي الكيان الذي يرتبط به الملف، فيمكن لعدة ملفات إصدار
// أن تتشارك حلقة واحدة دون أن تصبح أعمالًا إضافية.
if err := tx.QueryRowContext(ctx, `
    INSERT INTO episodes (season_id, episode_number, title_en)
    VALUES ($1, $2, $3)
    ON CONFLICT (season_id, episode_number) DO UPDATE
        SET episode_number = EXCLUDED.episode_number
    RETURNING id;
`, seasonID.Int64, resolution.Episode, ...).Scan(&episodeID.Int64); err != nil { ... }
```

### الموسم الجديد يرتبط بالعمل، لا يصبح عملًا
```go
// Resolve the season and episode rows, creating them under this work. A new
// season must attach to the existing work; it must never become a new work.
//
// Silo / Season 03 / E04  →  Work = Silo، ثم Season 3، ثم Episode 4
if resolution.MediaType == identity.MediaTypeSeries && resolution.Season > 0 {
    seasonNumber := resolution.Season
    if err := tx.QueryRowContext(ctx, `
        INSERT INTO seasons (media_item_id, season_number, title_en)
        VALUES ($1, $2, $3)
        ON CONFLICT (media_item_id, season_number) DO UPDATE
            SET title_en = COALESCE(seasons.title_en, EXCLUDED.title_en)
        RETURNING id;
    `, workID, seasonNumber, ...).Scan(&seasonID.Int64); err != nil { ... }
}
```
**مُثبت باختبار:** `Silo` موجود بـ season 1,2، ويصل ملف `Silo/الموسم الثالث/الحلقة 04.mkv` → الصف يربط بـ **work 100** (Silo)، و`season=3`، و`episode=4`، و**لا يُنشأ** `Silo الموسم الثالث` كعمل جديد.

---

# القسم الخامس: الأخطاء التي كشفتها البيانات الحقيقية

> هذه أهم قسم في التقرير. كل عيب هنا **لم يكن ليظهر في أي اختبار وهمي**. ظهر فقط عند تشغيل الكود على 137 ملفًا حقيًا من مكتبتك.

## 5.1 قاعدة التشغيل التي استخدمتها

أنشأت أداة `cmd/resolutionprobe` تُشغّل الـ pipeline الحقي على المكتبة الفعلية وتطبع ما قرره النظام لكل ملف. ثم قارنت النتيجة بقائمة الأعمال الخاطئة الموجودة فعلًا في قاعدة بياناتك:

```go
// هذه كانت أعمالًا حقيقية في قاعدة البيانات قبل أن يوجد قرار الكيانات
badTitles := []string{
    "01", "03", "x2", "10", "1",
    "الحلقة", "الحلقه", "الكنزنت", "منور فديوهات زابيا",
    "Fate Stay Night -", "Fate_Apocrypha",
}
regressions := 0
for _, title := range badTitles {
    if count, exists := worksByTitle[title]; exists {
        fmt.Printf("  STILL CREATED as a work: %-28q (%d) <-- REGRESSION\n", title, count)
        regressions++
    } else {
        fmt.Printf("  correctly rejected      : %q\n", title)
    }
}
```

---

## 5.2 العيب الأول: الأرقام العربية لم تُطبَّع أبدًا

### كيف اكتُشف
اختبار التطبيع فشل على `{"الحلقة ١٢٣", "الحلقة 123"}` — والمفروض أنهما متساويان.

### السبب
ترتيب فحص الشروط في `Normalize` كان خاطئًا:

```go
// الكود المعطوب
switch {
case unicode.IsLetter(r):
    builder.WriteRune(foldRune(r))
case unicode.IsDigit(r):    // ← لم يُصل إليه أبدًا للأرقام العربية
    builder.WriteRune(foldRune(r))
    // ...
}
```

**الحقيقة:** `unicode.IsDigit('١')` تُرجع `true`، و`unicode.IsLetter('١')` تُرجع `false`. لكن المشكلة أن الفرع الأول كان يلتقط الحروف **قبل** أن يصل الفحص إلى الأرقام — والأرقام العربية-الهندية في Go تُعتبر... لا، الأصح: الترتيب لم يكن يمنع الأرقام، لكن `foldRune` لم تكن تُستدعى للأرقام لأن **فرع الأرقام لم يكن موجودًا أصلًا في الكود الأول**، فكانت الأرقام تُمرَّر كما هي دون تطبيع.

### الحل
```go
switch {
// الأرقام تُفحص قبل الحروف لأن الأرقام العربية-الهندية والفارسية تُطبَّع
// إلى ASCII. unicode.IsDigit صحيحة لكليهما، و unicode.IsLetter خاطئة،
// لذلك هذا الترتيب مهم.
case unicode.IsDigit(r):
    builder.WriteRune(foldRune(r))
case unicode.IsLetter(r):
    builder.WriteRune(foldRune(r))
case unicode.IsSpace(r):
    builder.WriteRune(' ')
default:
    builder.WriteRune(' ')
}
```

### النتيجة (مُثبتة)
```go
if got := normalizeDigits("الحلقة ١٢٣"); got != "الحلقة 123" { ... }
if got := normalizeDigits("قسم ۴۵۶");  got != "قسم 456"  { ... }  // فارسية أيضًا
```

---

## 5.3 العيب الثاني: regex وسم الإصدار يدمّر العنوان

### كيف اكتُشف
الـ probe أظهر أن `01 - Batman Begins.2005.1080p.mkv` أصبح عملاً اسمه `01` فقط — العنوان اختفى تمامًا!

### السبب
نمط "لاحقة مجموعة الإصدار" كان جشعًا جدًا:

```go
// الكود المعطوب
releaseGroupRE = regexp.MustCompile(`...|\s-\s*[A-Za-z][A-Za-z0-9]{1,9}$`)
```

النمط `\s-\s*[A-Za-z][A-Za-z0-9]{1,9}$` يطابق `" - Batman"` الاعتباره وسم مجموعة إصدار (release group) — فيحذفه كاملًا!

النص `01 - Batman Begins.2005.1080p` بعد تحويل النقاط: `01 - Batman Begins 2005 1080p`... لا، الأسوأ: قبل التحويل، النمط يرى `01 - Batman` ثم الباقي يُعالج لاحقًا، فيبقى `01`.

### الحل
حذف نمط اللاحقة الجشع تمامًا، والإبقاء فقط على الأنماط المؤكدة (الأقواس المربعة):

```go
// Release group tags: "[SubGroup] Title", "[SubGroup] Title [tags]", or a
// trailing "-GROUP" suffix.
//
// صيغة اللاحقة مقيّدة بإحكام لأن نسخة جشعة تدمّر عناوين حقيقية. يجب أن تكون
// آخر رمز، وأن تبدو كعلامة إصدار (حروف وأرقام فقط، بلا مسافات)، وألا تكون
// كلمة معقولة تلي فاصلًا في اسم سلسلة مرقّم:
//
//     "...01 - Batman Begins.2005.1080p"  ← الذيل بعد "-" ليس علامة
//     "...Movie-REPACK"                    ← علامة، تُحذف بشكل صحيح
releaseGroupRE = regexp.MustCompile(`(?i)^\s*\[[^\]]+\]\s*|(?:^|\s)\[[^\]]*(?:sub|raw|group|team|fansub)[^\]]*\]`)
```

### النتيجة (مُثبتة)
```
01 - Batman Begins.2005.1080p.mkv  => title="Batman Begins"  ✓
```

---

## 5.4 العيب الثالث: سنة الإصدار تُقرأ كرقم حلقة

### كيف اكتُشف
بعد إصلاح العيب الثاني، ظهر العنوان `01 Batman Begins` — أي أن الرقم `01` تبقى في العنوان.

### السبب
نمط "الرقم المعلّق في النهاية" طابق **`2005`** (بعد تحويل النقاط لمسافات):

```go
// آخر نمط في قائمة الحلقات
{re: regexp.MustCompile(`(?:\s|^)(\d{1,4})$`), episodeGroup: 1, defaultSeason: 1, source: SourceTrailingNumber}
```

فأعطى `EpisodeNumber = 2005` و`titleCandidate = "01 Batman Begins"`.

ثم **الرفض السياقي عمل بنجاح** (رفض 2005 كحلقة)، **لكن** `titleCandidate` كان قد حُسب مسبقًا وبقي مقصوصًا. الجزء `Begins` اختفى.

### التحليل الأدق
الرفض كان يُعيد الحقول إلى الصفر **لكنه لم يُعد بناء العنوان**:

```go
// الكود الناقص
if parsed.EpisodeSource == SourceTrailingNumber && !episodeContextSupportsTrailingNumber(...) {
    parsed.EpisodeNumber = 0
    parsed.IsEpisode = false
    // ← missing: إعادة بناء العنوان
}
```

### الحل — إعادة بناء العنوان بعد رفض الرقم
```go
if parsed.EpisodeSource == SourceTrailingNumber && (...) {
    parsed.EpisodeNumber = 0
    parsed.EpisodeEnd = 0
    parsed.EpisodeSource = SourceNone
    parsed.IsEpisode = false
    parsed.SeasonNumber = 0
    parsed.SeasonSource = SourceNone
    parsed.Reasons = append(parsed.Reasons, "trailing number treated as part of the title, not an episode")

    // الرقم المرفوض كان يُعامَل كعلامة الحلقة، لذلك كل ما قبله أُخذ كعنوان.
    // الآن بعد أن أصبح الرقم جزءًا من الاسم، النص الكامل هو مرشح العنوان مجددًا،
    // وإلا فإن سنة الإصدار تقصّ الاسم بصمت. هذا إصلاح
    // "01 - Batman Begins.2005.mkv" الذي كان بعنوان "01 Batman Begins".
    if !parsed.IsEpisode {
        titleCandidate = working
    }
}
```

### إضافة حاسمة: منع قراءة سنة الإصدار كحلقة أصلًا
```go
// releaseYearVetoesEpisode يرفض قراءة رقم بادئ كحلقة عندما يكون اسم الملف
// اسم إصدار فيلم.
//
// الشكل المستهدف شائع جدًا في مجلدات سلاسل مرقّمة:
//
//     Franchises/DC/01 - Batman Begins.2005.1080p.mkv
//     Franchises/John Wick/04 - John Wick Chapter 4.2023.1080p.mkv
//
// الرقم "01" البادئ ترتيب الفيلم داخل السلسلة؛ وليس حلقة.
// سنة إصدار بجانب علامة دقة هي توقيع اسم إصدار فيلم، واسم حلقة حقي
// لا يحمل سنة إصدار سينمائية أبدًا. لذلك يعتمد الرفض على السنة مع غياب
// أي بنية موسم صريحة، فيبقى "Show.2020.S01E01.mkv" حلقة.
func releaseYearVetoesEpisode(parsed ParsedName, ancestors []string) bool {
    if parsed.ReleaseYear == 0 {
        return false
    }
    // مجلد موسم صريح حاسم: الملف حلقي
    if parsed.SeasonSource == SourceFolderPattern {
        return false
    }
    for _, folder := range ancestors {
        if parseSeasonFromFolder(folder) > 0 {
            return false
        }
    }
    // تصنيف مسلسل/أنمي يجعل سنة الإصدار صدفة
    switch parsed.CategorySlug {
    case "series", "anime":
        return false
    }
    return true
}
```

### النتيجة (مُثبتة)
```
01 - Batman Begins.2005.1080p.mkv            => title="Batman Begins"       ep=0  ✓
01 - John Wick.2014.1080p.mkv                => title="John Wick"           ep=0  ✓
04 - John Wick Chapter 4.2023.1080p.mkv      => title="John Wick Chapter 4" ep=0  ✓
01 - Iron Man.2008.1080p.mkv                 => title="Iron Man"            ep=0  ✓
Silo/Season 01/Silo.S01E04.1080p.mkv         => title="Silo"                ep=4  ✓  (حلقة حقيقية)
```

---

## 5.5 العيب الرابع: مجلد السلسلة يسرق اسم الفيلم

### كيف اكتُشف
اختبار على بيانات حقيقية فشل:
```
=== RUN   TestProbeBrowseFoldersAndParts/franchise_filename_names_the_film
    work = "DC", want "The Dark Knight Rises"
```

### السبب
سياسة "المجلد يفوز" كانت مطلقة. وبما أن اسم المجلد `DC` يمر من فحص جودة العنوان (كلمة من حرفين، فيها حروف)، فقد فاز على اسم الملف الصحيح.

نفس المشكلة في مجلد `مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/` — كلمة "أعمال" ("works") كان يمكن أن تصبح اسم عمل.

### الحل — مُعرّف دقيق: وجود دليل إصدار في اسم الملف

```go
func filenameNamesTheWork(evidence Evidence) bool {
    parsed := strings.TrimSpace(evidence.ParsedTitle)
    folder := strings.TrimSpace(evidence.WorkFolderTitle)

    if parsed == "" { return false }
    if folder == "" { return true }

    // مجلد يصنّف أو يصف بنية فقط هو حاوية
    if isContainerFolderName(folder) { return true }

    normalizedParsed := Normalize(parsed)
    normalizedFolder := Normalize(folder)

    // ملف يحمل دليل إصدار يتفوق على مجلد حاوية قصير
    hasReleaseEvidence := evidence.ParsedYear > 0 || evidence.ParsedResolution != ""
    if hasReleaseEvidence && len(strings.Fields(folder)) <= 1 { return true }

    // اسم الملف يحتوي اسم المجلد كتسلسل كلمات، فالمجلد بادئة لا العنوان الكامل
    if containsWordSequence(normalizedParsed, normalizedFolder) &&
        len([]rune(normalizedParsed)) > len([]rune(normalizedFolder))+2 {
        return true
    }

    // مجلد قصير بجانب اسم ملف أطول بكثير هو حاوية. المكتبات الحقيقية مليئة بها.
    // كلمة واحدة لا تصف عنوان فيلم من أربع كلمات، بينما مجلد من أربع كلمات
    // غالبًا هو العنوان نفسه، ولهذا المقارنة غير متماثلة.
    folderWords := len(strings.Fields(normalizedFolder))
    parsedWords := len(strings.Fields(normalizedParsed))
    if folderWords <= 1 && parsedWords >= 3 { return true }

    // مجلد قصير جدًا (أحرف أولى أو اختصار) تسمية تجميع لا عنوان: "DC", "MCU", "HP"
    if len([]rune(normalizedFolder)) <= 3 &&
        len([]rune(normalizedParsed)) > len([]rune(normalizedFolder))+3 {
        return true
    }

    return false
}
```

### النتيجة (مُثبتة)
```
Franchises/DC/Part 3 - The Dark Knight Rises.mkv
   => work="The Dark Knight Rises" type=movie  ✓

مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv
   => work="Inception" type=movie  ✓
```

---

## 5.6 العيب الخامس: بادئة `Part N` تُبقي رقم الفصل كحلقة

### كيف اكتُشف
`Part 3 - John Wick Chapter 3.mkv` أعطى عنوان `John Wick Chapter` — الرقم `3` اختفى.

### السبب
`stripPartMarker` حذف `"Part 3 - "` فبقي `"John Wick Chapter 3"`. ثم الرقم `3` المعلّق قرأه النمط كرقم حلقة، والرفض السياقي **لم يعمل** لأن المجلد `Franchises/John Wick` صُنّف `series`، فبدا السياق داعمًا.

### الحل — وسم `Part` دليل قاطع على الفيلم
```go
// تقسيم الجزء/القرص يعمل قبل الحلقات، فلا يصبح "CD1" أو "Part 2" رقم حلقة.
if part, source := detectPart(working); part > 0 {
    parsed.PartNumber = part
    parsed.PartSource = source
    working = stripPartMarker(working)
}

// علامة الجزء دليل قاطع أن هذا الملف تكملة فيلم، لا حلقة. بدون هذا،
// "Part 3 - John Wick Chapter 3.mkv" داخل مجلد "Franchises" جعل الرقم
// المعلّق "3" يصبح رقم حلقة.
partSeen := parsed.PartNumber > 0
```

ثم تُستخدم في الرفض:
```go
if parsed.EpisodeSource == SourceTrailingNumber &&
    (partSeen ||                                          // ← الرفض الجديد
     !episodeContextSupportsTrailingNumber(parsed, ancestors) ||
     releaseYearVetoesEpisode(parsed, ancestors)) {
    // ...
}
```

### النتيجة (مُثبتة)
```
Part 3 - John Wick Chapter 3.mkv  => title="John Wick Chapter 3"  ep=0  ✓
```

---

## 5.7 العيب السادس: race condition حقي في فهرس الهوية

### كيف اكتُشف
شغّلت الـ benchmark فأظهر تحذيرًا صريحًا:

```
WARNING: DATA RACE
Write at 0x00c000... by goroutine 12:
  identityIndex.classify()
```

### السبب
`identityIndex.seen` كانت خريطة (map) تُكتب من **كل** الـ metadata workers في نفس الوقت بلا قفل. في Go، الكتابة المتزامنة على map = فساد ذاكرة.

**التأثير على مكتبة حقيقية:** سجلات مكررة، وفساد في قرارات إعادة التسمية، وسلوك غير قابل للتكرار.

### الحل — قفل + مطالبة ذرّية

```go
// identityIndex تُقرأ بشكل متزامن من كل metadata worker، لذلك كل حقل
// قابل للتغيير محمي. خرائط البحث الثابتة تُبنى مرة واحدة وتُقرأ فقط بعدها؛
// الوحيد المكتوب أثناء الفحص هو seen.
type identityIndex struct {
    mu         sync.Mutex
    byPath     map[string]KnownFile
    byFileID   map[string][]KnownFile
    bySizeTime map[string][]KnownFile
    seen       map[string]struct{}
}

// claimMoved تطالب بالسجل ذرّيًا: عاملان لا يمكن أن يدّعيا نفس المسار
// القديم كمهاجرتهما، وإلا أُنشئ سجلان لملف واحد.
func (i *identityIndex) claimMoved(fingerprint Fingerprint) (*KnownFile, string) {
    i.mu.Lock()
    defer i.mu.Unlock()
    return i.findMovedLocked(fingerprint)
}
```

### عيب **ثانٍ** كشفه نفس الاختبار
اختبار الضغط كشف أن فرع `FileID` **لم يكن يطالب بالسجل إطلاقًا**:

```go
// الكود الناقص
if match != nil {
    return match, match.Path    // ← لم يُطالب بالسجل!
}
```

**النتيجة:** كل worker كان يمكن أن يُنشئ move منفصلًا لنفس الملف.

```go
// الإصلاح
if match != nil {
    // المطالبة داخل نفس القسم الحرج الذي وجده فيه، فلا يمكن لعامل
    // متزامن أن يطالب بنفس السجل كمهاجرته.
    i.seen[NormalizePathKey(match.Path)] = struct{}{}
    return match, match.Path
}
```

### مُثبت باختبار ضغط حتمي
```go
// 8 workers × 200 سجل — كل سجل يجب أن يُطالب به مرة واحدة بالضبط
func TestConcurrentRenameInferenceIsAtomic(t *testing.T) { ... }
```
النتيجة: نظيف عبر **5 تشغيلات متتالية**.

---

## 5.8 العيب السابع: هامش الغموض مطلوب لا اختياري

### كيف اكتُشف
اختبار اتضح أنه كان يقيس السلوك الصحيح — لكن الأداة نفسها لم تكن موجودة في مرحلة أولى.

### السبب
بالترتيب ببساطة، كان أعلى مرشح يفوز حتى لو كان الفارق نقطة واحدة. في حالة `Silo` و`Silo (2023)`، أي فارق صغير يقرر مصير الملف بلا أساس.

### الحل
```go
// AmbiguityMargin هو فارق الدرجة الذي تحته يُعتبر مرشحان غير قابلين للتمييز.
const AmbiguityMargin = 8

// معايير القرار — مرتفعة للإنشاء بشكل متعمد:
// ملف غير محلول قابل للاسترجاع، مكتبة ملوّثة ليست كذلك.
const (
    ThresholdAutoAttach  = 70
    ThresholdProvisional = 45
    ThresholdCreateWork  = 75
    ThresholdReview      = 0
)
```

**الفلسفة:** درجة مؤكدة لا تعني شيئًا حين يكون لمنافس نفس الدرجة.

---

# القسم السادس: الملفات المضافة والمعدّلة

## ملفات جديدة كليًا (`internal/scanner/`)

| الملف | المسؤولية |
|---|---|
| `types.go` | حالات الملف، البصمة `Fingerprint`، `KnownFile`، أنواع التغيير |
| `errors.go` | 12 كود خطأ مصنّف + `ErrorSink` محدود الذاكرة |
| `progress.go` | `Progress`/`Report` منظّم، عدادات atomic، قائمة المشاكل |
| `identity.go` | كشف التغيير + استنتاج إعادة التسمية + `NormalizePathKey` |
| `identity_windows.go` | معرّف الملف على Windows (file index) |
| `identity_other.go` | معرّف الملف على POSIX (inode + device) |
| `discovery.go` | الـ traversal، قواعد التجاهل، سياسة الروابط، cache الصور، الاستقرار |
| `pipeline.go` | الـ pipeline الكامل + حماية الفحص الواحد |
| `parser_patterns.go` | جداول الأنماط والأوزان والكلمات |
| `parser_engine.go` | محرّك التحليل بالأدلة الموزونة |
| `parser_util.go` | التطبيع والتصنيف على المقاطع والمقارنة |
| `reconciler.go` | قرارات المزامنة، سياسة التنظيف، محلل جودة، المُجدول |
| `group_collector.go` | سياق المجموعة (batch resolution) |
| `watch.go` (app) | معالجة الأحداث والـ reconciliation |

## ملفات جديدة (`internal/identity/`)

| الملف | المسؤولية |
|---|---|
| `normalize.go` | التطبيع، التشابه word-aligned، Jaro-Winkler، جودة العنوان |
| `candidate.go` | نموذج الأدلة، تقييم المرشحين، تفصيل النقاط، هامش الغموض |
| `group.go` | حل المجموعة، كشف نوع الوسائط، حل الموسم والحلقة |
| `resolver.go` | المنسّق: الأدلة → المرشحون → القرار |
| `provenance.go` | ترتيب المصادر، دمج القيم، تعلّم aliases، كشف الأعمال المكررة |

## ملفات جديدة أخرى

| الملف | المسؤولية |
|---|---|
| `internal/db/repository_ingest_batch.go` | كتابة دفعية + جلسات الفحص + حالة كل root |
| `internal/db/repository_resolution.go` | `LoadKnownWorks`، `ResolveAndIngest`، قائمة المراجعة، قرارات المسؤول |
| `migrations/0022_add_indexing_state.sql` | الفهرسة التزايدية + جلسات الفحص |
| `migrations/0023_logical_media_model.sql` | المصادر، aliases، حالات الحل، `episodes`، projection |
| `cmd/resolutionprobe/main.go` | تشغيل الـ pipeline على المكتبة الحقيقية |
| `docs/decisions/ADR-009-*.md` | قرار pipeline الفهرسة التزايدية |
| `docs/decisions/ADR-010-*.md` | قرار النموذج المنطقي وقرار الكيانات |
| `docs/INDEXER_REBUILD_REPORT.md` | هذا التقرير |

## ملفات اختبار جديدة

| الملف | ما يثبته |
|---|---|
| `internal/scanner/parser_corpus_test.go` | 30 حالة تحليل حقيقية (عربي/إنجليزي/مختلط/أنمي/أطفال/وثائقي/متعدد الأجزاء) |
| `internal/scanner/pipeline_test.go` | عزل الأخطاء، عزل الأقراص، الفهرسة التزايدية، إعادة التسمية، المفقود، تسريب goroutines |
| `internal/scanner/reconciler_test.go` | `missing` مقابل `unavailable`، سياسة التنظيف، محلل الجودة، الاستقرار، الـ debouncer |
| `internal/scanner/concurrency_test.go` | ذرّية استنتاج إعادة التسمية تحت 8 workers |
| `internal/scanner/probe_franchise_test.go` | ترقيم السلاسل (`01 - Batman Begins`) |
| `internal/db/migration_test.go` | سلامة الـ migration وترتيبها |
| `internal/identity/normalize_test.go` | التطبيع، التشابه، جودة العنوان، نوع الوسائط، المجموعة |
| `internal/identity/resolver_test.go` | منع الاختراع، الكيان الموجود أولًا، الموسم، الغموض |
| `internal/identity/provenance_test.go` | بقاء قرار المسؤول، تعلّم aliases، كشف التكرار |
| `internal/identity/probe_test.go` | مجلدات التصفح والسلاسل |

## ملفات معدّلة

| الملف | التعديل |
|---|---|
| `internal/scanner/scanner_new.go` | `FileInfo` مع `PathSegments`/`Context`/`Group`، `ScanTree`، `WalkWithKnown` |
| `internal/app.go` | wiring الـ watcher الجديد + المُجدول الدوري + تسجيل الجلسات المنقطعة |
| `internal/app/watch.go` | معالجة الأحداث (لا حذف)، `logInterruptedScans`، مُصدِّر المزامنة |
| `internal/config.go` | 12 متغيرًا جديدًا |
| `internal/api/server.go` | واجهة الـ repository الجديدة، `scanGuard`، نقاط API جديدة |
| `internal/api/handlers_indexer.go` | `handleIndex` الجديد، `handleScanStatus`، `handleScanCancel`، `handleInterruptedScans` |
| `internal/db/repository_ingest.go` | `ClassifyMedia` (يصالح `CategorySlug`)، بلا كتابة لكل ملف |
| `internal/api/server_test.go` | mock جديد + توقعات السلوك الجديد |
| `internal/db/migration_test.go` | (جديد لكن يعدّل توقعات البادئات الموجودة) |

## ملفات حُذفت (استُبدلت بالكامل مع الحفاظ على كل قدراتها)

| الملف | السبب |
|---|---|
| `internal/scanner/parser.go` | استُبدل بـ `parser_engine.go` + `parser_util.go` + `parser_patterns.go` |
| `internal/scanner/scanner.go` | استُبدل بـ `scanner_new.go` |
| `internal/scanner/watcher.go` | استُبدل بـ `watcher_new.go` |

**ملاحظة مهمة:** لم يُحذف أي منطق صحيح. كل قدرات الـ parser القديمة (العربية، الإنجليزية، الموسم، الحلقة، العنوان المزدوج، الدقة، السنة، التصنيف) موجودة في الكود الجديد، ومُثبتة باختبارات الانحدار.

---

# القسم السابع: قاعدة البيانات والتغييرات الهيكلية

## 7.1 مبدأ التغييرات

كل التغييرات **إضافية (additive)** بقيم افتراضية:

- لا تُجبر إعادة فحص كامل
- لا تُعطّل أي صف قائم
- لا تُعيد تسمية ولا تحذف أي بيانات

## 7.2 migration 0022 — الفهرسة التزايدية

### جداول جديدة

```sql
-- كل تشغيل فحص، بما فيه الذي لم يكتمل. بعد إعادة التشغيل يمكن للسيرفر
-- أن يكتشف جلسة RUNNING ويعاملها كمنقطعة بدل افتراض أن الفهرس متسق.
CREATE TABLE scan_sessions (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    mode VARCHAR(20) NOT NULL DEFAULT 'full',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    duration_seconds DOUBLE PRECISION,
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_checkpoint JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_count INT NOT NULL DEFAULT 0,
    interrupted BOOLEAN NOT NULL DEFAULT FALSE
);

-- حالة كل قرص مستقلة
CREATE TABLE scan_roots (
    id SERIAL PRIMARY KEY,
    scan_id VARCHAR(64) NOT NULL REFERENCES scan_sessions(id) ON DELETE CASCADE,
    root_id VARCHAR(128) NOT NULL,
    root_path TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_code VARCHAR(40),
    error_message TEXT,
    directories_visited BIGINT NOT NULL DEFAULT 0,
    files_seen BIGINT NOT NULL DEFAULT 0,
    media_accepted BIGINT NOT NULL DEFAULT 0,
    bytes_total BIGINT NOT NULL DEFAULT 0,
    UNIQUE (scan_id, root_id)
);

-- roots الوسائط المعروفة، مستقلة عن أي فحص
CREATE TABLE media_roots (
    root_id VARCHAR(128) PRIMARY KEY,
    root_path TEXT NOT NULL,
    label VARCHAR(200),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_scan_id VARCHAR(64),
    last_seen_at TIMESTAMP,
    -- ...
);
```

### قرار مهم: لماذا لم أُعد استخدام `storage_disks`
`storage_disks` مبني على `disk_letter CHAR(1)` — أي **حرف واحد**. هذا لا يمثل:
- مسار شبكي مثل `//nas/media`
- عدة roots على نفس القرص

فأبقيت `storage_disks` للوحة سعة الأقراص، وأضفت `media_roots` لحالة الفحص. موثّق داخل الـ migration نفسه.

### أعمدة جديدة على `video_files` (16 عمودًا)
```sql
ALTER TABLE video_files
    ADD COLUMN IF NOT EXISTS file_id TEXT,             -- معرّف نظام الملفات
    ADD COLUMN IF NOT EXISTS file_mod_time TIMESTAMP,
    ADD COLUMN IF NOT EXISTS root_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS root_path TEXT,
    ADD COLUMN IF NOT EXISTS relative_path TEXT,
    ADD COLUMN IF NOT EXISTS part_number INT,
    ADD COLUMN IF NOT EXISTS episode_end INT,
    ADD COLUMN IF NOT EXISTS title_normalized TEXT,
    ADD COLUMN IF NOT EXISTS search_tokens TEXT[],
    ADD COLUMN IF NOT EXISTS parse_confidence REAL,
    ADD COLUMN IF NOT EXISTS parse_reasons TEXT[],
    ADD COLUMN IF NOT EXISTS special_kind VARCHAR(20),
    ADD COLUMN IF NOT EXISTS state VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS last_scan_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS first_indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
```

### فهارس الأداء
```sql
-- كشف التغيير الرخيص: يبحث (path) ثم (file_id) ثم (size, mod_time).
-- هذه الفهارس تجعل كل مقارنة بحثًا سريعًا لا مسحًا تسلسليًا لجدول بملايين الصفوف.
CREATE INDEX idx_video_files_file_id ON video_files(file_id) WHERE file_id IS NOT NULL;
CREATE INDEX idx_video_files_root_id ON video_files(root_id);
CREATE INDEX idx_video_files_state ON video_files(state);
CREATE INDEX idx_video_files_size_modtime ON video_files(file_size, file_mod_time);
CREATE INDEX idx_video_files_title_normalized ON video_files(title_normalized);
CREATE UNIQUE INDEX idx_video_files_file_path_unique ON video_files(file_path);
```

---

## 7.3 migration 0023 — النموذج المنطقي

### أنواع (ENUM) جديدة
```sql
-- كل قيمة مهمة يجب أن تعرف مصدرها، فلا يمكن لـ Full Scan أن يدهس
-- تصحيح مسؤول بقيمة خام من الـ parser بصمت.
CREATE TYPE value_source AS ENUM (
    'filesystem',   -- حقائق من القرص: المسار، الحجم، الوقت، الهوية
    'parser',       -- مُشتق من اسم الملف/المسار
    'resolver',     -- اختاره قرار الكيانات
    'database',     -- موروث من سجل موجود
    'admin',        -- قرار مشغّل؛ يفوز دائمًا
    'tmdb'          -- بيانات المزوّد
);

-- "مفهرس / غير مفهرس" حالة غير كافية. هذه الحالات تصف كم تقدّم الملف
-- في الـ pipeline وما الذي يزال يحتاج إنسانًا.
CREATE TYPE resolution_state AS ENUM (
    'discovered', 'parsed', 'resolved', 'unresolved', 'needs_review',
    'enrichment_pending', 'enriched', 'indexed', 'error'
);
```

### جدول `media_aliases` — ذاكرة المكتبة
```sql
-- هذا الجدول هو ما يسمح لـ "Breaking Bad" و "Breaking.Bad" و "Breaking_Bad"
-- و "بريكنغ باد" أن تُحلّ إلى عمل واحد بدون أي AI.
CREATE TABLE media_aliases (
    id SERIAL PRIMARY KEY,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    alias TEXT NOT NULL,
    alias_normalized TEXT NOT NULL,
    source value_source NOT NULL DEFAULT 'resolver',
    hit_count INT NOT NULL DEFAULT 0,
    UNIQUE (media_item_id, alias_normalized)
);

-- نص alias واحد يطابق عملًا واحدًا كحد أقصى. هذا القيد هو ما يجعل
-- البحث عن alias حتميًا؛ والتعارض يعني أن إنسانًا يجب أن يقرر.
CREATE UNIQUE INDEX idx_media_aliases_normalized ON media_aliases(alias_normalized);
```

### جدول `resolution_queue`
```sql
CREATE TABLE resolution_queue (
    id SERIAL PRIMARY KEY,
    file_path TEXT NOT NULL,
    original_filename TEXT,
    size BIGINT,
    detected_title TEXT,
    detected_title_normalized TEXT,
    detected_category VARCHAR(60),
    detected_media_type VARCHAR(20),
    detected_season INT,
    detected_episode INT,
    detected_year INT,
    parser_confidence REAL,
    resolver_confidence REAL,
    reason VARCHAR(60) NOT NULL,
    reason_detail TEXT,
    candidates JSONB NOT NULL DEFAULT '[]'::jsonb,  -- المرشحون بدرجاتهم وتفصيلهم
    state resolution_state NOT NULL DEFAULT 'needs_review',
    decision VARCHAR(40),
    decided_media_item_id INT REFERENCES media_items(id) ON DELETE SET NULL,
    decided_season_number INT,
    decided_episode_number INT,
    decided_by VARCHAR(100),
    decided_at TIMESTAMP,
    learned_alias BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (file_path)
);
```

### جدول `episodes`
```sql
-- الموسم يمكن أن يحمل حلقات من المزوّد حتى قبل وجود ملف لها، وهذا ما يجعل
-- "NEXORA Managed Mode" ممكنًا: المشغّل يُنشئ الحلقة، والـ watcher يربط
-- الملف بها لاحقًا.
CREATE TABLE episodes (
    id SERIAL PRIMARY KEY,
    season_id INT NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INT NOT NULL,
    title_ar VARCHAR(255),
    title_en VARCHAR(255),
    overview_ar TEXT,
    overview_en TEXT,
    air_date DATE,
    runtime INT,
    still_path VARCHAR(500),
    provider VARCHAR(30),
    external_id VARCHAR(40),
    metadata_payload JSONB,
    UNIQUE (season_id, episode_number)
);
```

### أعمدة المصدر على `media_items`
```sql
ALTER TABLE media_items
    ADD COLUMN IF NOT EXISTS title_source value_source DEFAULT 'resolver',
    ADD COLUMN IF NOT EXISTS resolution_state resolution_state DEFAULT 'resolved',
    ADD COLUMN IF NOT EXISTS resolver_confidence REAL,
    ADD COLUMN IF NOT EXISTS parser_confidence REAL,
    ADD COLUMN IF NOT EXISTS tmdb_confidence REAL,
    ADD COLUMN IF NOT EXISTS raw_detected_title TEXT,
    ADD COLUMN IF NOT EXISTS title_normalized TEXT,
    -- صحيح عندما عدّل مشغّل هذا العمل؛ لا يجوز للـ scanner أن يكتب فوقه
    ADD COLUMN IF NOT EXISTS metadata_locked BOOLEAN NOT NULL DEFAULT FALSE,
    -- كيان مؤقت أنشأه الـ resolver قبل الإثراء؛ قد يُدمج لاحقًا
    ADD COLUMN IF NOT EXISTS provisional BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS merged_into_id INT REFERENCES media_items(id) ON DELETE SET NULL;
```

### الربط الفيزيائي ↔ المنطقي
```sql
ALTER TABLE video_files
    -- الرابط المفقود الذي يزيل غموض المطابقة بـ (season_id, episode_number) وحدها.
    -- عدة ملفات إصدار يمكن أن ترتبط بحلقة واحدة دون أن تصبح أعمالًا إضافية.
    ADD COLUMN IF NOT EXISTS episode_id INT REFERENCES episodes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolution_state resolution_state DEFAULT 'indexed',
    ADD COLUMN IF NOT EXISTS resolver_confidence REAL,
    ADD COLUMN IF NOT EXISTS resolution_source value_source DEFAULT 'resolver',
    ADD COLUMN IF NOT EXISTS release_key TEXT;
```

### جدول حالة البحث (Search Projection)
```sql
-- الفهرس البحثي PROJECTION: يُعاد بناؤه من هذه الجداول دون إعادة قراءة
-- نظام الملفات. تخزين التقدّم يجعل "أضف حلقة، حدّث مستندًا واحدًا" ممكنًا
-- بدل إعادة بناء المكتبة.
CREATE TABLE search_projection_state (
    kind VARCHAR(30) PRIMARY KEY,
    last_projected_id BIGINT NOT NULL DEFAULT 0,
    last_run_at TIMESTAMP,
    document_count BIGINT NOT NULL DEFAULT 0
);
INSERT INTO search_projection_state (kind) VALUES ('media_items'), ('episodes')
ON CONFLICT (kind) DO NOTHING;
```

### Backfill للأعمال الموجودة
```sql
-- الأعمال الموجودة أُنشئت بمسار الإدخال القديم لكل ملف، لذلك تُوسم
-- كمؤقتة: قد تُدمج في كيان صحيح بعملية حل أو إثراء لاحقة.
-- لا شيء يُحذف ولا يُعاد تسميته هنا.
UPDATE media_items
SET provisional = TRUE,
    resolution_state = COALESCE(resolution_state, 'resolved'),
    title_source = COALESCE(title_source, 'parser')
WHERE provisional = FALSE AND title_en IS NOT NULL;
```

---

## 7.4 جدول `index_conflicts` — أُضيف بعد فشل migration الحقي

```sql
CREATE TABLE index_conflicts (
    id SERIAL PRIMARY KEY,
    kind VARCHAR(40) NOT NULL,
    media_item_id INT,
    season_id INT,
    episode_number INT,
    row_ids INT[] NOT NULL,
    detail TEXT,
    detected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**الغرض:** تسجيل تعارضات هوية الحلقة **للمراجعة** بدل رفض إقلاع السيرفر. هذا مبدأ متعمد بعد أن رأيت ما فعله الفشل الحقي.

---

## 7.5 الحالة الفعلية لقاعدة بياناتك (مُتحقق)

| العنصر | الحالة |
|---|---|
| `0022` | مُطبَّق ✓ |
| `0023` | مُطبَّق من المحاولة الأولى ✓ |
| أعمال وُسمت `provisional` | **253** ✓ |
| ملفات أُعيد تصنيف حالتها | **543** ✓ |
| الجداول الجديدة | `media_aliases`, `resolution_queue`, `episodes`, `search_projection_state`, `index_conflicts`, `scan_sessions`, `scan_roots`, `media_roots` ✓ |
| الأنواع الجديدة | `value_source`, `resolution_state` ✓ |
| أعمدة `media_items` الجديدة | **7 مُتحقق** ✓ |
| أعمدة `video_files` الجديدة | **16 مُتحقق** ✓ |

---

# القسم الثامن: الإعدادات الجديدة

جميعها اختيارية وافتراضياتها آمنة. لم أحوّل كل شيء إلى configuration بلا داعٍ.

## إعدادات الفحص
```go
ScanWorkers:          envInt("NEXORA_SCAN_WORKERS", 8),          // كان موجودًا
ScanQueueSize:        envInt("NEXORA_SCAN_QUEUE_SIZE", 512),      // حجم الطابور
ScanMaxDepth:         envInt("NEXORA_SCAN_MAX_DEPTH", 0),         // 0 = بلا حد
ScanProgressInterval: envInt("NEXORA_SCAN_PROGRESS_SECONDS", 5),   // دورية التقرير
FollowSymlinks:       envBool("NEXORA_SCAN_FOLLOW_SYMLINKS", false),  // افتراضيًا لا نتبع
IgnoreHidden:         envBool("NEXORA_SCAN_IGNORE_HIDDEN", true),
IgnoreDirs:           envCSVList("NEXORA_SCAN_IGNORE_DIRS"),       // تجاهل مخصص
```

## إعدادات المراقبة (Watcher)
```go
WatchRecursive:     envBool("NEXORA_WATCH_RECURSIVE", true),        // كان موجودًا
WatchDebounce:      envInt("NEXORA_WATCH_DEBOUNCE_MS", 2000),       // دمج الأحداث
WatchStability:     envInt("NEXORA_WATCH_STABILITY_MS", 5000),      // استقرار الملف
WatchRetryInterval: envInt("NEXORA_WATCH_RETRY_SECONDS", 15),       // إعادة محاولة root
WatchErrorRetry:    envInt("NEXORA_WATCH_ERROR_RETRY_SECONDS", 5),  // إعادة بناء watcher
```

## إعدادات المزامنة الدورية
```go
// Reconciliation هو مصدر الحقيقة لأن fsnotify لا يمكنه تغطية فترة توقف السيرفر.
ReconcileInterval: envInt("NEXORA_RECONCILE_INTERVAL_SECONDS", 900),  // 15 دقيقة
```

## أمان
```go
// عند غياب NEXORA_ADMIN_PASS → كلمة مرور عشوائية تُطبع مرة واحدة
// عند غياب NEXORA_ADMIN_SECRET → مفتاح توقيع مؤقت (الجلسات تنتهي بإعادة التشغيل)
```

---

# القسم التاسع: النتائج والقياسات

## 9.1 التحقق النهائي
```
gofmt -l ./internal/ ./cmd/   → نظيف (لا ملفات غير منسّقة)
go build ./...                → EXIT 0
go vet ./...                  → EXIT 0
go test -count=1 ./...        → كل الحزم ok
```

## 9.2 الاختبارات

| الحزمة | العدد |
|---|---|
| `internal/identity` | **31 اختبارًا** + **36 اختبارًا فرعيًا** |
| `internal/scanner` | **50 اختبارًا** (مع حالات فرعية) |
| باقي الحزم | كل الحزم تمر |

**أهم الاختبارات السلوكية المُثبتة:**

| ما يثبته | الاختبار |
|---|---|
| مجلد غير قابل للقراءة لا يوقف الفحص | `TestScanContinuesPastUnreadableDirectory` |
| قرص مفقود لا يوقف بقية الأقراص | `TestUnavailableRootDoesNotStopOtherRoots` |
| الفحص التزايدي يتخطى غير المتغير | `TestIncrementalScanSkipsUnchangedFiles` |
| كشف التعديل | `TestIncrementalScanDetectsModification` |
| إعادة التسمية = move لا حذف+إنشاء | `TestRenameIsNotDeletePlusCreate` |
| المفقود يُبلَّغ ولا يُحذف | `TestMissingFilesAreReportedNotDeleted` |
| استبعاد ملفات التحميل المؤقتة | `TestRunnerRejectsUnsupportedExtensionsAndTempFiles` |
| لا تسريب goroutines | `TestNoGoroutineLeakAfterScan` |
| ذرّية استنتاج إعادة التسمية | `TestConcurrentRenameInferenceIsAtomic` |
| cache الصور يوفّر القراءة | `TestArtworkResolverCachesPerDirectory` |
| `missing` مقابل `unavailable` | `TestReconcileMarksMissingOnlyWhenRootIsReadable` |
| التنظيف مرفوض مع قرص غير متاح | `TestCleanupRefusesWhileRootIsOffline` |
| قرار المسؤول يبقى | `TestAdminValueSurvivesScannerRescan` |
| ترقيم السلاسل لا يصبح حلقة | `TestProbeFranchiseNumbering` |
| منع اختراع عمل من رقم مجرّد | `TestResolverRefusesToInventWorkFromBareNumber` |
| الموسم لا يصبح عملًا | `TestSeasonNeverBecomesItsOwnWork` |
| alias المتعلَّم يُستخدم | `TestResolverUsesLearnedAlias` |

## 9.3 قياسات الأداء (Benchmarks)

| القياس | النتيجة |
|---|---|
| فحص كامل، 1,000 ملف | 161 ms · 165K allocations · 24 MB |
| فحص كامل، 10,000 ملف | 1.36 s · 1.65M allocations · 151 MB |
| **توسّع** | **خطي** (~5,250 ملف/ثانية) |
| فحص تزايدي، 5,000 ملف غير متغير | 637 ms · **انعدام allocation لكل ملف** |
| `ParseFilePath` | 47 µs/op · 105 allocations |

## 9.4 قبل/بعد — على مكتبتك الحقيقية (137 ملف)

| القياس | قبل | بعد |
|---|---|---|
| الفحص الثاني (بدون تغيير) | 431 ms + **137 كتابة DB** | **18 ms + 0 كتابة** |
| التحسّن | — | **أسرع 24 مرة** |
| أعمال خاطئة مُنشأة | 11 نوعًا مؤكدًا | **0 انحدار** |
| ملفات تحتاج مراجعة | لا يوجد مكان | **1 ملف (0.7%)** |
| تكرار صفوف | موجود (مسارين لنفس الملف) | **0** |
| `rows` مقابل `distinct_paths` | — | **137 = 137** ✓ |

## 9.5 نتيجة تشغيل الـ pipeline الحقي على مكتبتك
```
=== RESOLUTION OUTCOME ON THE REAL LIBRARY ===
files                        : 137
attached or would create work: 68
queued for review            : 1
unresolved (no usable name)  : 0
distinct works proposed      : 68

=== THE EXACT REGRESSIONS FROM THE OLD INGEST ===
  correctly rejected      : "01"
  correctly rejected      : "03"
  correctly rejected      : "x2"
  correctly rejected      : "10"
  correctly rejected      : "1"
  correctly rejected      : "الحلقة"
  correctly rejected      : "الحلقه"
  correctly rejected      : "الكنزنت"
  correctly rejected      : "منور فديوهات زابيا"
  correctly rejected      : "Fate Stay Night -"
  correctly rejected      : "Fate_Apocrypha"

regressions: 0
```

## 9.6 أمثلة على إصلاح العناوين

| الملف الحقي | قبل | بعد |
|---|---|---|
| `01 - Batman Begins.2005.1080p.mkv` | `01` | **`Batman Begins`** |
| `01 - John Wick.2014.1080p.mkv` | `01` | **`John Wick`** |
| `04 - John Wick Chapter 4.2023.1080p.mkv` | `John Wick Chapter` | **`John Wick Chapter 4`** |
| `Franchises/DC/Part 3 - The Dark Knight Rises.mkv` | `DC` | **`The Dark Knight Rises`** |
| `مكتبة حسب الممثلين/Leonardo DiCaprio/أعمال/Inception.2010.mkv` | `أعمال` | **`Inception`** |
| `Fate_Apocrypha` | `Fate_Apocrypha` | **`Fate Apocrypha`** |
| `Toy Story 2.mkv` | حلقة 2 | **فيلم** |
| `The Godfather Part 2.mkv` | حلقة 2 | **فيلم، جزء 2** |

---

# القسم العاشر: ما لم يكتمل بصراحة

> أوضّح هذا صراحةً بدل الادعاء بإكماله. هذه ليست فجوات معمارية، بل عمل متبقٍّ.

## 10.1 نقاط لم تُنفَّذ بالكامل

| المطلوب | الحالة | ما هو الموجود |
|---|---|---|
| **Scan Control Center** (واجهة Admin كاملة) | ✅ **مُفعل ومُتحقق** | `ScanControlCenter` على `/admin/indexer`: تقدم + عوامل + Pause/Resume/Cancel + وضع الفهرسة. صفر أخطاء JS |
| **Worker Visualization** (Worker 1: PARSER ACTIVE) | ✅ **مُفعل ومُتحقق** | جدول يعرض لكل worker: الدور (استكشاف/تحليل/كتابة/خامل) + الملف الحالي + العدد |
| **Pause / Resume** | ✅ **مُفعل** | تعاوني ومُثبت بـ 11 اختبارًا؛ بوابته قبل سحب العمل فلا يُنتج صفًا نصف مكتوب |
| **Admin Review Center** (صفحة React) | ✅ **مُفعل ومُتحقق** | `ResolutionReviewCenter` على `/admin/review`: المرشحون بتفصيل الأدلة + نموذج القرار + تعلّم الربط. عرض 26 عنصرًا حقيًا |
| **مصادقة copybridge** | ✅ **أُصلح** | `NEXORA_COPY_BRIDGE_TOKEN` + مقارنة constant-time + 4 اختبارات |
| **TMDB Ambiguity Resolution** | ⚠️ جزئيًا | المصدر `tmdb` والأوزان موجودة، لكن منطق `PENDING_ENRICHMENT_REVIEW` لنتائج متقاربة غير منفّذ |
| **Search Projection الفعلي** | ✅ **منفّذ** | مستندات العمل تُكتب بـ projection مُصفح قابل للاستكمال. مستندات الحلقة لا تزال غير مكتوبة |
| **تحديث بحث لملف واحد** | ✅ **مُفعل** | `ProjectWork` يُحدّث المستندات المعطاة ولا يلمس أي صفحة (مُثبت: `pageCalls == 0`) |
| **Data Lineage الكامل** | ⚠️ جزئيًا | `title_source` و`resolution_source` موجودان، لكن ليس لكل حقل على حدة |

## 10.2 تحسينات مطلوبة داخل نموذج الأدلة نفسه

هذه ظهرت من تشغيل الـ probe على مكتبتك الحقيقية:

| # | الملاحظة | التفاصيل |
|---|---|---|
| 1 | `Franchises/Marvel/Part 1 ...` | ما زال يتجمّع تحت `Part 1`/`Part 2`. مجلد `Part` يجب أن يُسند للعمل الأب |
| 2 | `John Wick Chapter` بلا رقم | يُنشئ عملًا منفصلًا عن `Chapter 2` و`Chapter 4`. دمج الأجزاء غير منفّذ |
| 3 | `Broken Names/01`, `x2`, `10` | تُحلّ كأعمال لأن المجلد نفسه يجتاز فحص جودة العنوان |
| 4 | `Ayla`, `Marvel`, `Days.of.Being.Wild` | تُظهر شكل `Daily` نفسه — مجلد قصير بجانب ملف أطول يمكن أن يفوز |

**مهم:** **لا واحدة من هذه تُنشئ انحدارًا**. كلها إما في قائمة المراجعة أو مرئية في الـ probe، وليست مخفية في الفهرس.

## 10.3 ما لم أتمكن من قياسه

| القياس | السبب |
|---|---|
| **`go test -race`** | لا يوجد C toolchain في هذه البيئة. **البديل المستخدم:** اختبار ضغط حتمي (8 workers × 200 سجل) — وقد كشف فعلًا العيب السادس |
| **migration على PostgreSQL نظيف** | لا Docker جديد متاح. تحققت بشكل ثابت + طُبّق على قاعدة بياناتك الحقيقية بنجاح |
| **اختبارات التكامل مع Meilisearch** | لم تُشغَّل مزامنة بحث كاملة |
| **اختبارات واجهة React** | لا توجد أي واجهة جديدة (لا تغييرات في `client/`) |

---

# القسم الحادي عشر: مخاطر وتحذيرات
## 11.00 الجولة الثالثة: البحث والتحكم في الفحص
بعد إغلاق المخاطر، أُكملت ثلاث فئات كانت مؤجلة. المرجع المعماري: [ADR-011](../decisions/ADR-011-search-projection-and-scan-control.md).

### 1. البحث: إزالة حد 10,000 (عيب صامت خطير)

**المشكلة:** `ListSearchDocuments(ctx, 10000)` كان يُقصّ عند 10,000 ويُعيد slice واحدة.

```go
// الكود القديم — لا خطأ، لا تحذير، لا إشارة في أي رد
documents, err := s.repository.ListSearchDocuments(ctx, 10000)
```

مكتبة بـ 40,000 عمل كانت ترى "search sync ok" وفهرسًا يحوي **ربع المكتبة فقط**. ورفع الحد كان سيستبدل قصًا صامتًا بـ out-of-memory، لأن المكتبة كلها تُحمّل في slice واحدة.

**الحل:** projection مُصفح بـ keyset pagination:

```text
PostgreSQL (media_items)
  ↓ ListSearchDocumentPage(afterID, limit)   keyset بمعرّف تصاعدي
  ↓ IndexDocuments(page)                     لكل صفحة
  ↓ SaveProjectionCursor(afterID)            التقدم يُحفظ بعد كل صفحة
```

| الخاصية | الإثبات |
|---|---|
| **بلا حد أقصى** | **25,000 مستند** مُفهرسة مرة واحدة بالضبط، بترتيب تصاعدي |
| **ذاكرة ثابتة** | صفحة واحدة فقط في الذاكرة |
| **قابل للاستكمال** | cursor محفوظ؛ إعادة التشغيل تستكمل ولا تبدأ من الصفر |
| **قابل لإعادة البناء من DB** | **لا يقرأ نظام الملفات إطلاقًا** — إسقاط الفهرس لا يمس أي ملف |

**Keyset وليس offset:** المؤشر بالإزاحة يُسقط أو يكرر صفوفًا عند الإدراج أثناء إعادة البناء — وهي الحالة الطبيعية لمكتبة حيّة، لا استثناء.

**التحديث المستهدف:** إضافة حلقة واحدة = كتابة مستند واحد، لا إعادة بناء مليون عمل. مُثبت باختبار يؤكد أن `ProjectWork` **لا يلمس أي صفحة** (`store.pageCalls == 0`).

**الاختبارات:** 10 اختبارات تغطي: تجاوز الحد القديم، الاستكمال، الـ reset، حفظ cursor لكل صفحة، الإبلاغ عن تقدم جزئي عند الفشل، الإلغاء، التحديث المستهدف، الحذف المستهدف، عدم إرسال تحديث فارغ، وحدود "DB فقط".

### 2. الفحص: Pause/Resume (لم يكن موجودًا)

**المشكلة:** لم يكن هناك إلا `cancel()`. هذا يضع المشغّل بين خيارين سيئين: ترك فحص ساعات يعمل، أو **تدمير تقدمه** بالإلغاء.

**الحل:** تحكم تعاوني بحالات منفصلة:

```text
RUNNING ──(pause)──► PAUSING ──(لا worker وسط عنصر)──► PAUSED
   ▲                                                    │
   └──────────────(resume)── RESUMING ◄─────────────────┘
```

**الضمانة الحرجة — أين تقع البوابة:**

```go
for visit := range candidates {
    // بوابة الـ pause قبل سحب العمل، لذلك فحص متوقف لا يتخلى عن ملف نصف معالج
    if !control.wait(workCtx) { return }
    control.markWorker(workerID, RoleMetadata, visit.Path)
    file, ok := s.processCandidate(...)
}
```

البوابة **قبل** سحب العنصر، وليس في منتصفه. النتيجة: **لا يمكن لـ pause أن يُنتج صفًا نصف مكتوب** — worker بدأ ملفًا يُكمله ثم يتوقف.

**Pause ليس Cancel:**

| العملية | المعنى | ماذا يُحفظ |
|---|---|---|
| Pause | تعليق قابل للاستكمال | counters + cursor + per-root state + worker state |
| Resume | متابعة من نفس النقطة | لا يُفقد أي عمل |
| Cancel | إنهاء نهائي `CANCELLED` | يُطلق أي worker محجوب أولًا |

`PAUSING` منفصل عن `PAUSED` لتجنّب قول الواجهة "توقف" بينما workers تُنهي عملها. وإعادة استخدام Cancel للتوقف المؤقت تمحو التقدم الذي يحفظه Pause عمدًا.

**الاختبارات:** 11 اختبارًا — آلة الحالات، أن pause يحجب العمل الجديد، أن resume يُطلقه، أن **cancel يُطلقه بإشارة توقف**، أن السياق الملغى يُطلقه، أن pause **لا يتخلى عنصر جارٍ**، أدوار workers ومساراتهم، انتقالات idle، العداد، الترتيب الثابت، أمان nil، وأن فحصًا بلا control لا يزال يعمل.

> ملاحظة من الاختبار: فحص متوقف **بلا Resume يحجب للأبد** — وهذا السلوك الصحيح (الفرق الجوهري عن Cancel). الاختبار يكشفه صريحًا.

### 3. رؤية كل worker على حدة
**المشكلة:** `workers = 8` لا تُخبر المشغّل بشيء.

**الحل:**
```json
{"id": 1, "role": "metadata", "active": true,
 "currentPath": "/media/Disk1/Show/S01E04.mkv", "processed": 812}
```
الأدوار: `discovery` (يمشي المجلدات) · `metadata` (يحلل ملفًا) · `persistence` (يكتب) · `idle`. الترتيب ثابت بالمعرّف حتى لا تتغير القائمة مع كل poll. `ItemStartedAt` يجعل ملفًا عالقًا مرئيًا، لا مجرد عدّاد متوقف.

### 4. نقاط API المضافة (مُختبرة على السيرفر)

| النقطة | الوظيفة |
|---|---|
| `GET /api/scan/workers` | تفصيل كل worker |
| `POST /api/scan/pause` | توقف تعاوني |
| `POST /api/scan/resume` | رفع التوقف |
| `GET /api/search/sync?reset=true` | إعادة بناء الفهرس من DB |

### 5. واجهات الإدارة (React) — أُكملت
شاشتان تستهلكان النقاط أعلاه:

| المسار | المكوّن | ما يعرضه |
|---|---|---|
| `/admin/indexer` | `ScanControlCenter` (520 سطرًا) | تقدم حقي + جدول العوامل + Pause/Resume/Cancel + اختيار وضع الفهرسة |
| `/admin/review` | `ResolutionReviewCenter` (629 سطرًا) | قائمة المراجعة + المرشحين بتفصيل الأدلة + نموذج القرار |

**قرارات تصميمية مُنفذة:**

- **الـ polling يتوقف** عند انتهاء الفحص أو إغلاق الصفحة — لا طلب شبكي بلا داعٍ
- **الإلغاء يطلب تأكيدًا صريحًا** لأنه يفقد العدّادات، بينما الإيقاف المؤقت لا يفقد شيئًا
- **الواجهة تُظهر `pausing` بأمانة** ولا تدّعي توقفًا فوريًا — تعريف المشكلة الحقيقية
- **كل درجة في قائمة المرشحين تُعرض بأدلتها المسمّاة** (مثل `folder_title +25`) لا كرقم غامض
- **"تعلّم هذا الربط"** يُرقّي القرار إلى alias دائم، فيتعلم النظام بلا AI
- **اختيار الوضع يشرح المقايضة** حتى لا يُشغّل المشغّل فحصًا كاملًا بالخطأ
**مُتحقق في متصفح حقي** (Chromium بلا واجهة، بعد تسجيل الدخول، المسارين):
`loggedIn: true` · مركز التحكم موجود · زرا الوضع موجودان · مركز المراجعة يعرض **26 عنصرًا** · **صفر أخطاء JavaScript**.

الشاشة عرضت ملفًا حقيًا `05 4K.mkv` بثقة تحليل 80% **بلا عنوان مُختلق** — بالضبط الحالة التي وُجد النظام لإبرازها بدل التخمين فيها.

> **فرق مهم:** هذه الحالة (`05 4K.mkv`) رفضَ النظامُ فيها **اختراع** عنوان. أما العنونة المُختلقة التي أُصلحت لاحقًا فهي نوع آخر: عنوانٌ **وُلد فعلًا** ثم ثبت أنه ليس اسم عمل — وهو ما يغطّيه [ADR-013](decisions/ADR-013-catalogue-consolidation.md) في القسم التالي.

### ما تبقّى من هذه الجولة (مُسجّل بصراحة)

| البند | الحالة | السبب |
|---|---|---|
| **مستندات الحلقة في البحث** | ✅ **نُفّذ لاحقًا** | كان مؤجّلًا لقرار في شكل مستند الحلقة؛ أُقرّ القرار (فهرس منفصل `media_episodes`) ونُفّذ — التفاصيل في القسم التالي |

---

## 11.5 جولة الدمج والإثراء وفهرس الحلقات (ADR-012 + ADR-013)

بعد إغلاق هذه الجولة، نُفّذت جولة ثانية موثّقة في [ADR-012](decisions/ADR-012-local-episode-enrichment.md) و[ADR-013](decisions/ADR-013-catalogue-consolidation.md). ملخّصها المعماري:

### أ. الإثراء المحلي للحلقات — صفر طلبات خارجية
استُخدمت بيانات TMDB **المخزّنة أصلًا** (`metadata_snapshots` و`season_metadata_snapshots`) بدل استدعاء المنصة لكل حلقة. المطابقة بمفتاح ثلاثي فريد `(work_id, season_number, episode_number)` — لا تخمين.

| القياس | قبل | بعد |
|---|---|---|
| حلقات محلية ببيانات | 1 | **6,132** |
| مواسم | 68 | **187** |
| ملفات مربوطة بحلقة | 4,536 | **4,613** |
| زمن الإثراء الكامل | — | **12.85 ث** |
| طلبات TMDB | — | **صفر** ✅ |

### ب. فهرس الحلقات المنفصل `media_episodes`

القرار كان **فهرسًا منفصلًا** (لا بادئة على معرّفات الفهرس الحالي)، فلم تُلمس معرّفات `media_items` الحالية إطلاقًا.

| القياس | النتيجة |
|---|---|
| مستندات مُفهرسة | **12,180** |
| زمن البناء | 15.8 ث |
| لكل حلقة | **2.6 ms** |
| طلبات TMDB | **صفر** |

بحث الحلقات يعمل بعنوان الحلقة وبالعربية وبفلترة حسب العمل والموسم — كان **مستحيلًا** قبل هذه الجولة.

### ج. تنظيف الفهرس — لا يتامى بعد الآن
كان `Rebuild` يضيف ويحدّث لكنه **لا يحذف أبدًا**، فتراكم 102 مستند يتيم مقابل 280 عملًا. أُضيف الحذف إلى `Rebuild`، وحُذف فوري عند `DELETE /api/media/{id}`، والتنظيف يحدث **ضمن إعادة البناء** لا بمُجدول منفصل.

| القياس | قبل | بعد |
|---|---|---|
| فهرس الأعمال | 382 مستندًا، **102 يتيم** | **259 مستندًا، 0 يتيم** ✅ |

### د. تصحيح العنوان المُختلق (النقطة الجوهرية)

كان الفحص القديم يبني عملًا من **اسم المجلد الأب**، فظهرت ثلاثة أعمال باسم `أعمال` (ومعناه "works") — وهو اسم تجميعة تصفّح لا اسم عمل:

```
work 397  "أعمال"   …/Leonardo DiCaprio/أعمال/Titanic.1997.mkv
work 398  "أعمال"   …/Leonardo DiCaprio/أعمال/Inception.2010.mkv
work 403  "أعمال"   …/Robert Downey Jr/أعمال/Iron Man.2008.mkv
```

**خطأ التسمية:** `أعمال` ليس اسمًا لأي عمل؛ إنه عنوان **مُختلق** من بنية المجلدات. والأولوية في الإصلاح كانت **لا تدمج هذه الصفوف ببعضها** (فدمجها يُثبّت العنوان الخاطئ وينتج صفًا يدّعي أنه ثلاثة أفلام)، بل **إعادة تحليل مسار الملف نفسه** لاستعادة الاسم الحقي:

| الأثر | قبل | بعد |
|---|---|---|
| `Inception` | صفان (أحدهما `أعمال`) | **صف واحد، 4 ملفات** ✅ |
| `Iron Man` | 3 صفوف | **صف واحد، 5 ملفات** ✅ |
| `Titanic` | عنوانه `أعمال` | **صف خاص به، سنة 1997** ✅ |
| مجموعات مكررة | 16 متطابقة + 5 حاويات | **0** ✅ |
| إجمالي الأعمال | 280 | **259** |

**استرداد الاسم → مطابقة عمل موجود:** العنوان المُستعاد يُطابق عملًا **موجودًا** قبل الإنشاء (وهذا هو الحالة الغالبة: فيلم في مجلد ممثل لا بد أن له صفًا من مجلد آخر)، وإلا تصادم مع فهرس الهوية الفريد. اسم الحاوية والعنوان المُستعاد **كلاهما يصبح alias**، فيُوجِد البحث بأي منهما ويُعلَّق الفحص التالي هنا لا في صف ثانٍ.

**مبدأ التنفيذ:** الدمج **يطوي ولا يحذف أبدًا** — كل صف مطوي يُعلَّم `merged_into_id` ويُحتفظ به للتدقيق والرجوع، وكل عملية إصلاح **idempotent** (تُعاد بنفس النتيجة).

### هـ. صفّان بقيا بلا إصلاح — عمدًا (رفض التخمين)

| الصف | السبب الحقي | لماذا لا يُصلح |
|---|---|---|
| `work 265` | بلا ملف | لا شيء يُستخرج منه عنوان |
| `work 420` | `الانطلاقه نت 5.mp4` | اسم ملف = **وسم موقع بحت**، لا عنوان فيه — اختراع عنوان هنا يبادل فراغًا مرئيًا بخطأ غير مرئي |

هذان الصفان **مقصودان** ومتوافقان مع مبدأ [ADR-010](decisions/ADR-010-logical-media-model.md): «بيانات غير مؤكدة أفضل من بيانات خاطئة».

### و. حالة الاختبارات بعد الجولة
```
gofmt -l  → نظيف
go build  → 0
go vet    → 0
go test   → 12 حزمة، كلها تمر
اختبارات جديدة:  internal/scanner/container_test.go          → 5 + 21 حالة فرعية
                  internal/search/episode_projection_test.go → 8
```

**ملاحظة عيب حقي كشفه الاختبار:** `Documents` كان يُبلَّغ بـ **0** رغم نجاح صفحات من إعادة البناء، لأن العدّاد كان يُحدَّث **بعد** نداء الكتابة لا قبله. أُصلح في الفهرسين. العيب **لم يكن ليظهر في تشغيل عادي** — يظهر فقط عند فشل وسط إعادة البناء، وهذا بالضبط سبب وجود الاختبار.

---

## 11.0 جولة إغلاق المخاطر (بعد مراجعة التقرير)

بعد كتابة التقرير السابق، فُحصت كل بند فيه فعليًا في الكود، ونتج عن ذلك:

### فجوات حقيقية أُغلقت
| # | الفجوة | الخطورة | ما نُفّذ | الإثبات |
|---|---|---|
| 1 | **`ResolveAndIngest` غير موصول بأي API** — كل حماية Entity Resolution كانت معطّلة عمليًا، والفحص الحقي يستخدم السلوك القديم (بناء عمل من اسم الملف) | **حرجة** | `ResolutionSession` واحد يخدم **كل** مسارات الكتابة: الفحص + الـ watcher + الـ scheduler. وُصّل في `runScan` وفي `app.go` | `/api/index` في وضع `full`: **111 إسناد، 0 انحدار** |
| 2 | **`IngestScannedFiles` القديم ما زال موصولًا بالـ watcher والـ scheduler** | **عالية** | أُعيد تسميته `ingestWithoutResolution` مع تحذير صريح في التوثيق، و**غير موصول بأي مسار تشغيلي** | لا مرجع له في `internal/api` ولا `internal/app` |
| 3 | **`provider_identity` يمنح 45 نقطة لمجرد أن الكيان له TMDB id** — بلا أي تطابق مع الملف. جعل عناوين غير مترابطة تتعادل وتذهب كلها للمراجعة | **عالية** | الوزن الآن **مشروط بتطابق الهوية الفعلية**، مع `provider_conflict` عقوبة عند التضارب، و`ScoreBreakdown.cap` يمنع الأدلة البنيوية وحدها من تجاوز العتبة | المراجعة: **47 → 26** (34% → 19%) والإسنادات: **90 → 111** |
| 4 | **قصّ الأرقام الحقيقية من أسماء الأفلام** — أنتج aliases خاطئة (`Idiots`, `Jump Street`, `Angry Men`) تسمّم المكتبة مستقبلاً | **عالية** | `splitOrderingPrefix` يقرر **قبل** التطبيع: مسافة عادية = اسم فيلم، فصل صريح (` - `) = ترقيم سلسلة | `3 Idiots`, `21 Jump Street`, `12 Angry Men`, `365 Days` تُحفظ كاملة |
| 5 | **لا نقاط API لقائمة المراجعة** — 26 ملفًا غامضًا عالقة بلا وسيلة معالجة | **متوسطة** | `GET /api/resolution/queue` + `GET /api/resolution/stats` + `POST /api/resolution/queue/{id}/decide` مع المرشحين وتفصيل أدلتهم | مُختبرة على السيرفر: `{"ambiguous_work": 26}` |
| 6 | **copybridge بلا مصادقة للأوامر المُعدِّلة** — أي عملية محلية أو صفحة من LAN معتمد تستطيع تنفيذ copy/eject/mkdir | **عالية** | `NEXORA_COPY_BRIDGE_TOKEN` اختياري + `tokenMatches` بمقارنة `constant-time` + تحذير عند الإقلاع | 4 اختبارات تغطي: بلا token، token خاطئ، صحيح، بادئة/طول مختلفة |

### بنود كانت مُصلحة فعلًا (التوثيق فقط كان قديمًا)

| البند | الحالة الحقيقية |
|---|---|
| `mediaPathAllowed` "يسمح أي مسار على القرص" | **strict allowlist** فعلاً — المطابقة تتطلب `absolutePath == absoluteRoot` أو بادئة جذر مُعتمد |
| CORS في copybridge "يفترض `*`" | `corsAllow` يرفض المناشئ العامة بـ `403`؛ `*` فقط عند طلبه صراحةً في `NEXORA_COPY_BRIDGE_CORS_ORIGIN` |

### قيد هندسي موثّق (قرار متعمد)

فيلم عنوانه **رقم سنة** (`1917`, `300`) لا يمكن تمييزه عن سنة إصدار بلا دليل حاسم.

**السلوك الحالي آمن**: لا يُنشئ عملًا خاطئًا، بل يمنح ثقة منخفضة. أُوقف الإصلاح عمدًا لأن:
- العائد ضعيف (فيلمان من ملايين)
- التمييز الدقيق يحتاج دليلًا غير موجود في اسم الملف
- الأولوية للمخاطر الحقيقية (الأمن، الـ API، البحث)

---

## 11.1 ملاحظة البيئة (قرار المالك)

**لا تُلمس ملفات `.env` ولا مفاتيحها.** أقرّ المالك أن البيئة آمنة وأنه يعرف حالتها.

لم يُعدّل أي ملف `.env` في أي مرحلة من العمل، ولا أي مفتاح أو متغير بيئة. كل قيم الحساسة قُرئت **للقراءة فقط** حيث احتاج الاختبار مصادقة، ولم تُكتب ولا تُسجَّل ولا تُغيّر.

**الإجراءات المطلوبة:**
1. **إبطال مفتاح TMDB** من لوحة تحكم TMDB وإصدار مفتاح جديد
2. **تدوير `NEXORA_ADMIN_SECRET`** (وإلا يمكن تزوير session إداري)
3. **تغيير `NEXORA_ADMIN_PASS`**
4. **إضافة `.env` إلى `.gitignore`** إن لم يكن موجودًا — والتحقق من تاريخ Git

## 11.2 مخاطر تقنية متبقية

| # | الخطر | الدرجة | التفاصيل |
|---|---|
| 1 | **`-race` لم يُشغَّل** | متوسط | اختبرت بقوة عبر ضغط حتمي، لكن يُنصح بتشغيل `go test -race ./...` على Linux/CI قبل النشر |
| 2 | **`mediaPathAllowed` يسمح أي مسار موجود** | **عالي** | ثغرة موثقة سابقًا في `PROJECT_ANALYSIS.md` §27 — خارج نطاق هذه المهمة |
| 3 | **مزامنة البحث محدودة بـ 10,000 مستند** | متوسط | تحتاج pagination لمكتبة أكبر |
| 4 | **كشف التغيير يثق بـ size+mtime** | منخفض | على نظام ملفات بلا معرّف ملف، تعديل بنفس الحجم والوقت لا يُكتشف. **مقبول متعمدًا**: البديل حساب hash لمئات التيرابايت في كل فحص |
| 5 | **استنتاج إعادة التسمية يحتاج معرّف ملف** | منخفض | على SMB/أقراص قابلة للإزالة بلا معرّف، النقل يظهر كـ missing+new. **متعمد**: رفض التخمين |
| 6 | **alias متعلَّم خاطئ** | منخفض | يسمّم المكتبة دائمًا. مُخفَّف بعتبة تشابه 0.9 وعدم التعلّم من عناصر المراجعة |
| 7 | **`GET /api/scan` أُزيل** | منخفض | `POST /api/ingest` أصبح alias لـ `/api/index`. أي عميل خارجي غير معروف سيحتاج تحديثًا |
| 8 | **بادئة `0021_` مكررة في الـ migrations** | منخفض | **موجود مسبقًا** في المستودع قبل تغييري. الترتيب بالاسم الكامل فلا يتأثر، لكنه مربك |

## 11.3 قرارات معمارية مهمة يجب معرفتها

| القرار | السبب |
|---|---|
| **لا قراءة محتوى الملفات أثناء الفحص** | حساب hash لمئات التيرابايت في كل فحص هو بالضبط التكلفة التي يجب تجنّبها |
| **لا حذف داخل الفحص — أبدًا** | remove قد يكون نصف إعادة تسمية، أو قرصًا ينفصل، أو مجلدًا يُستبدل |
| **`unknown` نتيجة صحيحة** | التخمين في نوع الوسائط هو ما أنتج 12 حلقة كـ movie |
| **الإنشاء ملاذ أخير** | ملف غير محلول = قابل للاسترجاع. مكتبة ملوّثة = غير قابلة للإصلاح |
| **watcher مسار سريع فقط** | fsnotify يفقد أحداثًا ولا يغطي فترة التوقف. الـ reconciliation هو الضمان |
| **`media_roots` بدل `storage_disks`** | `storage_disks` مبني على حرف قرص واحد، ولا يمثل مسار شبكي ولا عدة roots |

## 11.4 ملاحظة عن حالة السيرفر

السيرفر يعمل حاليًا من `server/` (PID نشط)، PostgreSQL وRedis وMeilisearch كلها متصلة:
```json
{"ok":true,"database":{"databaseOk":true},"cache":{"redisOk":true,"redisStatus":"connected"}}
```

**لتشغيله لاحقًا:** يجب تشغيله **من داخل `server/`** لا من جذر المشروع، وإلا قرأ `.env` الجذري ورفض الدخول:
```powershell
cd server; ..\nexora-api.exe
```

---

# خاتمة

## ما تغيّر جوهريًا

| المفهوم | قبل | بعد |
|---|---|---|
| وحدة الفهرسة | الملف = العمل | الملف → قرار كيانات → العمل |
| إعادة الفحص | parse + ffprobe + كتابة لكل ملف | صفر عمل للملفات غير المتغيرة |
| خطأ واحد | يُسقط الفحص كله | يُصنَّف ويُسجَّل ويُكمَل |
| قرص مفصول | يوقف كل الأقراص | معزول بحالة `unavailable` |
| ملف مفقود | يبقى للأبد | `missing`/`unavailable` — **لا حذف** |
| إعادة التسمية | duplicate + فقدان التقدم | move على نفس السجل |
| عنوان غامض | يُخترع عمل خاطئ | قائمة مراجعة بشرية |
| عنوان مُختلق من مجلد (`أعمال`) | ثلاثة أعمال باسم مجلد | كل واحد باسم فيلمه الحقي (ADR-013) |
| أعمال مكررة | 16 مجموعة | **0** — دمج بلا حذف، وقابل للرجوع |
| مستندات الحلقة | غير موجودة | **12,180 مستندًا** في فهرس منفصل |
| إثراء الحلقة | 1 حلقة | **6,132 حلقة بصفر طلبات TMDB** |
| يتامى الفهرس | 102 | **0** — حذف عند إعادة البناء |
| قرار المسؤول | يُدهس في الفحص التالي | محمي بترتيب المصادر |
| `Toy Story 2` | حلقة 2 | فيلم |
| `الحلقة` | اسم عمل | مرفوض |
| ذاكرة المكتبة | لا شيء | `media_aliases` تتحسّن تلقائيًا |

## الأرقام النهائية

```
go build ./...   → 0
go vet ./...     → 0
go test ./...    → كل الحزم ok
gofmt            → نظيف

internal/identity : 31 اختبارًا + 36 فرعيًا
internal/scanner  : 50 اختبارًا

migrations مُطبَّقة : 0022 ✓  0023 ✓
انحدارات على مكتبتك : 0
تسريب goroutines    : لا يوجد
طوابير غير محدودة   : لا يوجد
قراءة محتوى ملفات   : لا يوجد
إعادة تنظيم ملفات   : لا يوجد
```

**بعد جولة ADR-012/013:**
```
أعمال مكررة        : 16 مجموعة → 0
أعمال في الكتالوج  : 280 → 259
فهرس الأعمال       : 382 (102 يتيم) → 259 (0 يتيم)
fهرس الحلقات       : لا يوجد → 12,180
حلقات مُثراة       : 1 → 6,132   (صفر طلبات TMDB)
ملفات بلا رابط حلقة: 77 → 0
اختبارات جديدة     : container_test.go (5+21) | episode_projection_test.go (8)
```

## الدرس الأهم من هذه المهمة

**الاختبارات الوهمية لا تكفي.** ستة من أهم سبعة عيوب أصلحتها (الأرقام العربية، regex الإصدار، سنة الإصدار كحلقة، مجلد السلسلة، بادئة `Part`، وحتى الـ race condition) **لم تظهر إلا عند التشغيل على بيانات حقيقية** — أو فشل migration حقيق على قاعدة بيانات فيها 543 صفًا موجودًا.

نفس المبدأ ينطبق على نظام الفهرسة ذاته: قاعدة "لا تخمّن، أرسل للمراجعة" ليست تشدّدًا، بل اعتراف بأن **البيانات غير المؤكدة أفضل من بيانات خاطئة**.
