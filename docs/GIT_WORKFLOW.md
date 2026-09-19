# Git Workflow Guide — NEXORA

> هذا الملف يشرح كيف أُعدّ هذا العمل في Git، وكيف تتعامل مع الفروع والدمج
> والرفع لاحقًا. مكتوب من واقع العملية التي نُفذت فعليًا، لا من نظرية عامة.

---

## 1. حالة المستودع بعد العمل

| العنصر | القيمة |
|---|---|
| الفرع الرئيسي | `main` |
| الفرع الجديد | `feature/indexer-rebuild-and-resolver` |
| الريموت | `https://github.com/mosaa65/NEXORA.git` |
| الـ commits المضافة | **9** |
| نسخة احتياطية (tag) | `backup/indexer-rebuild-pre-merge` |

### الـ commits التسعة (بالترتيب)

| # | الـ commit | المحتوى |
|---|---|---|
| 1 | `8b5a552` feat(scanner) | إعادة بناء الفهرسة + Pause/Resume + migration 0022 |
| 2 | `0426b29` feat(identity) | طبقة قرار الكيانات كاملة + migration 0023 |
| 3 | `40970e9` feat(indexer) | وصل الـ resolver بكل المسارات + إزالة الحد الصامت 10,000 |
| 4 | `9fb4bd2` feat(admin-ui) | مركز التحكم + مركز المراجعة (الواجهة) |
| 5 | `068f69b` feat(db,copybridge) | حالة الـ projection + مصادقة الـ bridge |
| 6 | `be27a4f` docs | ADR-009/010/011 + ARCHITECTURE + ENDPOINTS + التقرير |
| 7 | `63b25b0` refactor(db) | حذف الـ ingest لكل ملف |
| 8 | `04fe350` style | gofmt فقط |
| 9 | `0564f7a` Merge | دمج `origin/main` |

**لماذا هذا الترتيب؟** لأن كل commit قابل للمراجعة وحده. لو راجعت `8b5a552`
فهمت الفهرسة، ولو راجعت `0426b29` فهمت قرار الكيانات، دون أن تختلط
تغييرات التنسيق أو التوثيق مع المنطق.

---

## 2. كيف تتابع العمل من الآن

### الخطوة 1 — افتح Pull Request

GitHub أعطاك الرابط جاهزًا:

```
https://github.com/mosaa65/NEXORA/pull/new/feature/indexer-rebuild-and-resolver
```

افتحه، واكتب في وصف الـ PR ملخصًا من `docs/INDEXER_REBUILD_REPORT.md`، ثم
اطلب المراجعة.

### الخطوة 2 — بعد قبول الـ PR

من الطرفية:

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA
git checkout main
git pull origin main                      # main صار يحتوي عملك
git branch -d feature/indexer-rebuild-and-resolver   # احذف الفرع المحلي
```

> `-d` (وليس `-D`) يحذف فقط إذا كان الفرع مدموجًا فعلًا — وهذا حماية منك، لا من Git.

### الخطوة 3 — إذا أردت العودة لحالة ما قبل الدمج

الـ tag موجود، فالرجوع آمن:

```powershell
git checkout backup/indexer-rebuild-pre-merge
# أو لإنشاء فرع من تلك النقطة:
git branch recovery/from-backup backup/indexer-rebuild-pre-merge
```

---

## 3. الأوامر التي تحتاجها فعليًا

### كل يوم

```powershell
# ابدأ فرعًا جديدًا لأي ميزة
git checkout main
git pull origin main
git checkout -b feature/اسم-الميزة

# اعرض ما تغيّر قبل الإيداع
git status
git diff                       # غير المُودَع
git diff --cached              # المُودَع

# أودع تغييرات محددة (أفضل من git add -A)
git add path/to/file.go path/to/other.go
git commit
```

### فحص الحالة بأمان

```powershell
git status --short --branch    # فرع + حالة مختصرة
git log --oneline -10          # آخر 10 commits
git log --oneline origin/main..HEAD   # ما عندك وليس على الريموت
```

### رفع التحديثات

```powershell
git push                       # للفرع الحالي المتتبّع
git push -u origin feature/x   # أول مرة لفرع جديد
```

---

## 4. قواعد مهمة تعلّمتها من هذه الجلسة

### 4.1 تحقّق قبل أن تودع، لا بعده

شغّل هذا **قبل كل commit** لتتأكد أن لا شيء حساس أو مؤقت سيُرفع:

```powershell
git status --porcelain | ForEach-Object { $_.Substring(3) }
```

ابحث يدويًا عن: `.env`, `*.log`, `*.exe`.

### 4.2 `git add -A` خطر

`git add -A` يضيف **كل** شيء بلا تمييز — بما فيه ملفات مؤقتة أو مفاتيح.
الأفضل:

```powershell
git add server/internal/scanner/         # مجلد محدد
git add path/to/specific.go              # ملف محدد
```

> في هذه الجلسة، `api.out.log` و`nexora-api.exe` كانا سيُرفعان لولا الفحص.
> أُضيفت قواعد في `.gitignore` تمنعهما مستقبلًا.

### 4.3 لا تلمس الأفرع المشتركة مباشرة

لا تعمل على `main` مباشرة. اعمل على فرع، ثم PR. السبب: لو أخطأت، الحذف
سهل — والفرع الرئيسي يبقى نظيفًا.

### 4.4 احتفظ بنسخة احتياطية قبل أي دمج

قبل أي دمج معقد، ثبّت نقطة رجوع:

```powershell
git tag backup/اسم-واضح HEAD
```

الـ tag لا يُحذف مع الأفرع، فهو شبكة أمان حقيقية.

### 4.5 افحص ما سيجلبه الدمج **قبل** أن تدمج

```powershell
git fetch origin
git log --oneline HEAD..origin/main     # ما سيأتي
git diff --stat HEAD...origin/main      # ما هي الملفات المتأثرة؟
```

هذا ما جعل الدمج في هذه الجلسة آمنًا: رأيت مسبقًا أن `origin/main` يغيّر
`transfer/` و`copybridge/` فقط، وهو **نفس العمل** الموجود محليًا، فالدمج
مرّ بلا تعارض.

### 4.6 عند التعارض: توقّف وافهم

إذا ظهر تعارض:

```powershell
git status                    # الملفات المتعارضة
git diff path/to/conflicted   # ما التعارض؟
```

علامات التعارض:

```
<<<<< HEAD           (نسختك)
=======
>>>>>>> origin/main    (النسخة الواردة)
```

**حلّه يدويًا** باختيار الصحيح (قد يكون الاثنين معًا)، ثم:

```powershell
git add path/to/conflicted
git commit
```

**لا تستخدم `git checkout --ours` أو `--theirs` بلا قراءة** — يحذف أحد
الطرفين بلا تفكير، وهذا كيف يُفقد العمل.

### 4.7 للتراجع عن آخر commit دون فقدان التغييرات

```powershell
git reset --soft HEAD~1       # يلغي الـ commit ويُبقي التغييرات مُودَعة
git reset HEAD~1              # يلغي الـ commit ويُبقي التغييرات غير مُودَعة
```

> تجنّب `git reset --hard` إلا وأنت متأكد تمامًا — يحذف التغييرات نهائيًا.

---

## 5. كيف تُنشئ commit جيد

النمط المتّبع في هذا العمل:

```
<النوع>(<المجال>): <وصف موجز بالحاضر>

لماذا التغيير مطلوب؟
ما الخطأ الذي كان موجودًا؟

ماذا نُفذ؟

- نقطة أولى
- نقطة ثانية

ما الاختبارات؟

Refs: ADR-XXX
```

**الأنواع المستخدمة:** `feat` (ميزة) · `fix` (إصلاح) · `refactor` (إعادة هيكلة
بلا تغيير سلوك) · `docs` · `style` (تنسيق فقط) · `test` · `chore`.

**القاعدة الأهم:** اكتب **لماذا**، لا **ماذا**. الكود يوضّح "ماذا" بنفسه؛
أما لماذا كان الخطأ موجودًا ولماذا اخترت هذا الحل فذلك ما يضيع بعد أسبوعين.

---

## 6. التحقق بعد كل دمج

بعد أي دمج، شغّل هذه الأوامر للتأكد أن المشروع سليم:

```powershell
# الخادم
cd c:\Users\mousa\Desktop\project\NEXORA\server
go build ./...
go vet ./...
go test -count=1 ./...

# الواجهة
cd c:\Users\mousa\Desktop\project\NEXORA\client
npm run build
```

إن مرّت الأربعة، الدمج سليم. وإن فشل واحد، **لا ترفع** حتى تصلحه.

---

## 7. ملاحظات خاصة بهذا المستودع

- **Git ليس في PATH** على هذا الجهاز. استخدم:
  ```powershell
  $env:Path += ";C:\Program Files\Git\cmd"
  ```
- **Go ليس في PATH**. استخدم:
  ```powershell
  $env:Path += ";C:\Users\mousa\go\bin"
  ```
- **Node ليس في PATH**. استخدم:
  ```powershell
  $env:Path += ";C:\Program Files\nodejs"
  ```
- **`.env` محمي**: مسجّل في `.gitignore` بـ `.env`, `server/.env`, `client/.env`.
  لا يظهر في `git status` أبدًا، فلا يمكن رفعه بالخطأ.
- **تحذيرات `LF will be replaced by CRLF`** عند الإضافة: طبيعية على Windows،
  ولا تؤثر على المحتوى. Git يوحّد نهايات الأسطر تلقائيًا.

---

## 8. خلاصة العملية التي نُفذت

```text
1. فحص الحالة        →  كان main متأخرًا + ملفات مؤقتة غير مُودَعة
2. تنظيف            →  حذف nexora-api.exe و *.log وملف شارد فارغ
3. حماية            →  إضافة قواعد .gitignore
4. فرع جديد         →  feature/indexer-rebuild-and-resolver
5. نسخة احتياطية     →  tag backup/indexer-rebuild-pre-merge
6. 8 commits منطقية  →  كل واحدة قابلة للمراجعة وحدها
7. معاينة الدمج      →  git log HEAD..origin/main (رأينا ما سيأتي)
8. الدمج            →  بلا تعارضات
9. تحقق كامل        →  build + vet + 12 حزمة اختبار + بناء الواجهة
10. فحص نهائي        →  لا .env ولا ملفات مؤقتة في الـ commits
11. رفع             →  الفرع على الريموت، جز لـ PR
```

**لم يُفقد أي شيء في أي خطوة.** الشجرة نظيفة، والنسخة الاحتياطية موجودة،
وكل تغيير موثّق برسالته الخاصة.
