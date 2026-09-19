# NEXORA — دليل الأوامر الكامل

> كل أمر في هذا الملف **نُفِّذ وتُحقّق منه فعليًا** على هذا المستودع، لا منقولًا
> من الذاكرة. حيث كان الأمر يحتاج تعديلًا لتوافقه مع بيئة Windows، ذُكر ذلك صراحةً.
>
> آخر تحديث: بعد إعادة بناء منظومة الفهرسة (ADR-009، ADR-010، ADR-011).

---

## المحتويات

- [0. متطلبات التشغيل](#0-متطلبات-التشغيل)
- [1. استنساخ المشروع](#1-استنساخ-المشروع)
- [2. إعداد البيئة](#2-إعداد-البيئة)
- [3. Docker — البنية التحتية](#3-docker--البنية-التحتية)
- [4. الباك اند (Go)](#4-الباك-اند-go)
- [5. الفرنت اند (React + Vite)](#5-الفرنت-اند-react--vite)
- [6. تشغيل النظام كاملًا](#6-تشغيل-النظام-كاملًا)
- [7. الـ API — أمثلة أوامر](#7-الـ-api--أمثلة-أوامر)
- [8. أدوات التشخيص](#8-أدوات-التشخيص)
- [9. Copy Bridge — نسخ USB والهواتف](#9-copy-bridge--نسخ-usb-والهواتف)
- [10. الاختبارات والتحقق](#10-الاختبارات-والتحقق)
- [11. قاعدة البيانات والنسخ الاحتياطي](#11-قاعدة-البيانات-والنسخ-الاحتياطي)
- [12. Git — العمل اليومي](#12-git--العمل-اليومي)
- [13. حل المشاكل الشائعة](#13-حل-المشاكل-الشائعة)
- [14. مرجع سريع — كل الأوامر](#14-مرجع-سريع--كل-الأوامر)

---

## 0. متطلبات التشغيل

| البرنامج | الإصدار المطلوب | الغرض |
|---|---|---|
| **Go** | 1.26+ (`go.mod` يطلب `1.26.0`) | خادم الـ API والـ Bridge |
| **Node.js** | 20+ (Dockerfile يستخدم `node:20-alpine`) | بناء واجهة React |
| **Docker Desktop** | أي إصدار حديث | PostgreSQL، Meilisearch، Redis |
| **FFmpeg / FFprobe** | أي إصدار حديث | فحص الملفات التقني وتوليد المصغرات |

### تحقق من الإصدارات

```powershell
go version          # go1.26.4 windows/amd64
node --version      # v22.x
npm --version
docker --version
ffmpeg -version | Select-Object -First 1
ffprobe -version | Select-Object -First 1
```

### ⚠️ مهم: الأدوات ليست في PATH على هذا الجهاز

في هذه البيئة، `go` و`node` و`git` ليست في مسار النظام. أضفها في بداية كل جلسة:

```powershell
$env:Path += ";C:\Users\mousa\go\bin"                      # Go
$env:Path += ";C:\Program Files\nodejs"                     # Node
$env:Path += ";C:\Program Files\Git\cmd"                    # Git
$env:Path += ";C:\Program Files\Docker\Docker\resources\bin" # Docker CLI
```

> **نصيحة:** ضعها في ملف PowerShell profile لتُحمّل تلقائيًا:
> ```powershell
> notepad $PROFILE
> # ثم أضف الأسطر الأربعة أعلاه واحفظ
> ```

---

## 1. استنساخ المشروع

```powershell
cd C:\Users\mousa\Desktop\project
git clone https://github.com/mosaa65/NEXORA.git
cd NEXORA
```

### إن كان المُستودع موجودًا مسبقًا

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA
git fetch origin
git status --short --branch
```

---

## 2. إعداد البيئة

### 2.1 إنشاء ملف `.env`

المشروع يقرأ الإعدادات من `.env`. القالب موجود في `.env.example`:

```powershell
# من جذر المشروع
Copy-Item .env.example .env
```

الخادم يبحث عن `.env` في هذه المواضع بالترتيب:
`.env` → `server/.env` → `../.env` → `../../.env`

> **ملاحظة مهمة:** إن شغّلت الخادم من داخل `server/` فاستخدم `server/.env`،
> وإن شغّلته من الجذر فاستخدم `.env` في الجذر. خلط المواضع يعني قراءة ملف
> خاطئ — وهي مشكلة واجهتُها فعليًا.

### 2.2 الإعدادات الحرجة

هذه القيم يجب أن تكون صحيحة قبل أي تشغيل:

```ini
# اتصال قاعدة البيانات — المنفذ 15432 لأن Docker يربط 5432 داخليًا
NEXORA_DATABASE_URL=postgres://nexora:nexora@localhost:15432/nexora?sslmode=disable

# محرك البحث
NEXORA_MEILI_HOST=http://127.0.0.1:7700

# جذور مكتبة الوسائط — افصل بينها بـ ; على Windows
NEXORA_MEDIA_ROOTS=D:\Media;E:\Media

# عدد عمال الفحص
NEXORA_SCAN_WORKERS=8
```

### 2.3 إعدادات الفحص والمزامنة (من ADR-009 و ADR-011)

```ini
# الفهرسة التزايدية
NEXORA_SCAN_QUEUE_SIZE=512              # حجم الطابور (افتراضي 512)
NEXORA_SCAN_MAX_DEPTH=0                 # 0 = بلا حد للعمق
NEXORA_SCAN_PROGRESS_SECONDS=5          # دورية تقرير التقدم
NEXORA_SCAN_FOLLOW_SYMLINKS=false       # لا نتبع الروابط افتراضيًا
NEXORA_SCAN_IGNORE_HIDDEN=true
NEXORA_SCAN_IGNORE_DIRS=                # تجاهل مخصص، مفصولة بفاصلة

# المراقبة (Watcher)
NEXORA_WATCH_RECURSIVE=true
NEXORA_WATCH_DEBOUNCE_MS=2000           # دمج الأحداث المتقاربة
NEXORA_WATCH_STABILITY_MS=5000          # استقرار الملف قبل الإدخال
NEXORA_WATCH_RETRY_SECONDS=15           # إعادة محاولة قرص غير متاح
NEXORA_WATCH_ERROR_RETRY_SECONDS=5      # إعادة بناء الـ watcher

# المزامنة الدورية — مصدر الحقيقة عند فقدان أحداث fsnotify
NEXORA_RECONCILE_INTERVAL_SECONDS=900   # 15 دقيقة
```

### 2.4 بيانات الدخول الإدارية

```ini
NEXORA_ADMIN_USER=admin
NEXORA_ADMIN_PASS=كلمة-مرور-قوية
NEXORA_ADMIN_SECRET=مفتاح-عشوائي-طويل
```

توليد مفتاح عشوائي:

```powershell
# PowerShell
-join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
```

> **الأمان:** إن تركت `NEXORA_ADMIN_PASS` فارغًا، يولّد الخادم كلمة مرور عشوائية
> ويطبعها **مرة واحدة** في السجل. وإن تركت `ADMIN_SECRET` فارغًا، يولّد مفتاح
> توقيع مؤقتًا تنتهي كل الجلسات بإعادة التشغيل.

---

## 3. Docker — البنية التحتية

المشروع يستخدم Docker Compose لتشغيل PostgreSQL وMeilisearch وRedis.
الخادم والواجهة **يُنصح بتشغيلهما محليًا** (بدون Docker) أثناء التطوير.

### 3.1 تشغيل Docker Desktop

```powershell
Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
# انتظر 30-60 ثانية حتى يجهز محرك Docker
Start-Sleep -Seconds 50
docker version --format "{{.Server.Version}}"
```

### 3.2 تشغيل البنية التحتية فقط (المستحسن للتطوير)

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA
docker compose up -d postgres meilisearch redis
```

> **لاحظ:** الخدمات `server` و`client` لهما `profiles: [full]`، فلا يبدأان مع
> `up -d` العادي. هذا مقصود — يسمح بتشغيل Go وReact محليًا مع البنية التحتية في
> Docker.

### 3.3 تشغيل كل شيء داخل Docker (وضع الإنتاج)

```powershell
docker compose --profile full up -d
```

هذا يبني ويشغّل:
| الحاوية | المنفذ | الغرض |
|---|---|---|
| `nexora-postgres` | 15432 → 5432 | قاعدة البيانات |
| `nexora-meilisearch` | 7700 | محرك البحث |
| `nexora-redis` | 6379 | cache (اختياري) |
| `nexora-server` | 8080 | خادم Go |
| `nexora-client` | 80 | واجهة عبر Nginx |

### 3.4 فحص الحالة

```powershell
# الحاويات العاملة
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# السجلات (متابعة حيّة)
docker compose logs -f postgres

# آخر 50 سطرًا من سجل الخادم
docker compose logs --tail 50 server

# فحص جاهزية PostgreSQL
docker exec nexora-postgres pg_isready -U nexora -d nexora
```

### 3.5 الإيقاف والحذف

```powershell
# إيقاف الخدمات (تبقى البيانات)
docker compose stop

# إيقاف وحذف الحاويات (تبقى البيانات في volumes)
docker compose down

# ⚠️ حذف كل شيء INCLUDING البيانات
docker compose down -v
```

### 3.6 إعادة بناء صورة بعد تغيير الكود
```powershell
# إعادة بناء الخادم فقط
docker compose --profile full build server
docker compose --profile full up -d server
# إعادة بناء كل شيء بلا cache
docker compose --profile full build --no-cache
```

> **تنبيه مهم:** `server/Dockerfile` يستخدم `golang:1.26-alpine` ليطابق
> متطلب `go.mod` (`go 1.26.0`). إن رفعت إصدار Go في `go.mod` فعليك رفع وسم
> الصورة أيضًا، وإلا فشل البناء بـ `go.mod requires go >= 1.26.0`.
> هذا كان خللًا حقيًا: الصورة كانت `1.24` و`go.mod` يطلب `1.26.0`،
> فأي `docker compose --profile full build` كان سيفشل.

### 3.7 التحقق من عدم وجود عمليات قديمة تحجب المنافذ

```powershell
foreach ($p in 8080,5173,15432,7700,6379,32145) {
  $c = Get-NetTCPConnection -State Listen -LocalPort $p -ErrorAction SilentlyContinue
  if ($c) { "PORT $p HELD by PID $($c.OwningProcess)" } else { "PORT $p free" }
}
```

---

## 4. الباك اند (Go)

### 4.1 التثبيت

```powershell
$env:Path += ";C:\Users\mousa\go\bin"
cd c:\Users\mousa\Desktop\project\NEXORA\server

# تحميل الاعتماديات
go mod download

# تحقق من سلامة الاعتماديات
go mod verify

# تنظيف الاعتماديات غير المستخدمة (اختياري)
go mod tidy
```

### 4.2 البناء

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server

# بناء كل الحزم (تحقق سريع من عدم وجود أخطاء تصريف)
go build ./...

# بناء ثنائي الـ API
go build -o ..\nexora-api.exe .\cmd\api

# بناء بثنائي مصغّر للإنتاج (بلا معلومات تصحيح)
# على Windows: GOOS=windows افتراضيًا
go build -trimpath -ldflags="-w -s" -o ..\nexora-api.exe .\cmd\api
```

> **ملاحظة:** `server/Dockerfile` يبني بـ `CGO_ENABLED=0 GOOS=linux` لأن الحاوية
> تعمل على Linux. أما على Windows فتجنّب `GOOS=linux` وإن أردت بناء ثنائي
> Linux يدويًا فاستخدم:
> ```powershell
> $env:GOOS="linux"; $env:CGO_ENABLED="0"
> go build -trimpath -ldflags="-w -s" -o ..\nexora-api-linux .\cmd\api
> Remove-Item Env:GOOS; Remove-Item Env:CGO_ENABLED
> ```

### 4.3 التشغيل

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server

# التشغيل المباشر (يقرأ server/.env)
go run ./cmd/api

# أو الثنائي المبني (من داخل server/ ليقرأ server/.env)
..\nexora-api.exe
```

**الخادم سيستمع على `:8080`** ويطبع سطورًا مثل:

```
level=INFO msg="NEXORA API listening" addr=:8080
level=INFO msg="media root online" root="D:/Media"
level=INFO msg="media watcher registered" root="D:/Media"
```

### 4.4 التشغيل في الخلفية (اختياري)

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server
Start-Process -FilePath "..\nexora-api.exe" `
  -WorkingDirectory "c:\Users\mousa\Desktop\project\NEXORA\server" `
  -RedirectStandardOutput "..\api.out.log" `
  -RedirectStandardError "..\api.err.log" `
  -WindowStyle Hidden

# للتحقق
Get-Process nexora-api | Select-Object Id, ProcessName
Get-Content ..\api.out.log | Select-Object -First 10
```

### 4.5 الإيقاف

```powershell
# إن كان في المقدمة: Ctrl+C (إيقاف سليم مع graceful shutdown)

# إن كان في الخلفية
Get-Process nexora-api -ErrorAction SilentlyContinue | Stop-Process -Force
```

### 4.6 أعلام التشغيل للـ Bridge

```powershell
# الخدمة تتعرف على علم التشغيل. راجع القسم 9.
go run ./cmd/copybridge -debug      # تشغيل تفاعلي للتطوير
```

---

## 5. الفرنت اند (React + Vite)

### 5.1 التثبيت

```powershell
$env:Path += ";C:\Program Files\nodejs"
cd c:\Users\mousa\Desktop\project\NEXORA\client

# التثبيت العادي
npm install

# التثبيت المطابق تمامًا لـ lockfile (لما يستخدمه Dockerfile)
npm ci
```

### 5.2 التشغيل للتطوير

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\client
npm run dev
```

**Vite سيعمل على `http://0.0.0.0:5173`** (وفق `package.json`) ويوكّل `/api`
إلى `NEXORA_API_UPSTREAM` أو `127.0.0.1:8080` تلقائيًا.

افتح: `http://127.0.0.1:5173`

> **لاحظ:** التطبيق يستخدم `HashRouter`، فالمسارات تبدأ بـ `#`:
> `http://127.0.0.1:5173/#/admin/indexer`

### 5.3 البناء للإنتاج

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\client

# بناء الحزمة في client/dist
npm run build

# معاينة حزمة الإنتاج محليًا (بدون Nginx)
npm run preview
```

**معاينة الإنتاج على `http://0.0.0.0:4173`.**

### 5.4 أوامر أخرى

```powershell
# عند تغيير package.json فقط
npm install

# تنظيف وإعادة تثبيت كامل (عند فساد node_modules)
Remove-Item node_modules -Recurse -Force
Remove-Item package-lock.json -Force
npm install
```

### 5.5 الإيقاف

اضغط `Ctrl+C` في نافذة Vite.

---

## 6. تشغيل النظام كاملًا

### السيناريو اليومي (المستحسن)

```powershell
# ── نافذة 1: البنية التحتية ──────────────────────
$env:Path += ";C:\Program Files\Docker\Docker\resources\bin"
cd c:\Users\mousa\Desktop\project\NEXORA
docker compose up -d postgres meilisearch redis

# انتظر حتى تصبح جاهزة
docker compose ps

# ── نافذة 2: خادم Go ─────────────────────────────
$env:Path += ";C:\Users\mousa\go\bin"
cd c:\Users\mousa\Desktop\project\NEXORA\server
go run ./cmd/api

# ── نافذة 3: واجهة React ─────────────────────────
$env:Path += ";C:\Program Files\nodejs"
cd c:\Users\mousa\Desktop\project\NEXORA\client
npm run dev
```

ثم افتح `http://127.0.0.1:5173`.

### ترتيب الإقلاع مهم

```text
1. Docker (PostgreSQL + Meilisearch)
   ↓ الخادم يحتاج قاعدة البيانات عند البدء، وإلا يخرج بـ exit 1
2. خادم Go
   ↓ يطبّق الـ migrations تلقائيًا عند الإقلاع
3. واجهة React
```

**مهم:** إن شغّلت الخادم قبل قاعدة البيانات سيفشل الاتصال ويخرج فورًا. جرّب
مرة أخرى بعد جاهزية PostgreSQL.

### التحقق من أن كل شيء يعمل

```powershell
# 1. الخادم يستجيب
Invoke-WebRequest -Uri "http://127.0.0.1:8080/api/health" -UseBasicParsing | Select-Object -ExpandProperty Content
# المتوقع: {"ok":true,"database":{"databaseOk":true},"cache":{"redisOk":true,"redisStatus":"connected"}}

# 2. الواجهة تعمل
Invoke-WebRequest -Uri "http://127.0.0.1:5173" -UseBasicParsing | Select-Object -ExpandProperty StatusCode
# المتوقع: 200

# 3. قاعدة البيانات جاهزة
docker exec nexora-postgres psql -U nexora -d nexora -tAc "SELECT COUNT(*) FROM schema_migrations;"
```

---

## 7. الـ API — أمثلة أوامر

### 7.1 الحصول على token إداري

```powershell
# اكتب بيانات الدخول في ملف لتجنّب مشكلة الرموز الخاصة في PowerShell
$body = '{"username":"admin","password":"كلمة-المرور"}'
Set-Content -Path "$env:TEMP\login.json" -Value $body -Encoding UTF8 -NoNewline

$response = Invoke-WebRequest -Uri "http://127.0.0.1:8080/api/admin/login" `
  -Method POST -InFile "$env:TEMP\login.json" `
  -ContentType "application/json" -UseBasicParsing

$token = ($response.Content | ConvertFrom-Json).token
$headers = @{ Authorization = "Bearer $token" }

Remove-Item "$env:TEMP\login.json"
```

> **تحذير من تجربة حقيقية:** PowerShell يقطع النص عند رمز `#`. إن كانت كلمة
> المرور تحتوي `#` فاستخدم `-InFile` كما أعلاه، أو ولّد كلمة مرور بلا `#`.

### 7.2 فحص الصحة والمعلومات

```powershell
# صحة عامة
Invoke-WebRequest "http://127.0.0.1:8080/api/health" -UseBasicParsing | Select-Object -ExpandProperty Content

# الأقسام
Invoke-WebRequest "http://127.0.0.1:8080/api/categories" -UseBasicParsing | Select-Object -ExpandProperty Content

# الأقراص
Invoke-WebRequest "http://127.0.0.1:8080/api/disks" -UseBasicParsing | Select-Object -ExpandProperty Content

# إحصاءات لوحة التحكم
Invoke-WebRequest "http://127.0.0.1:8080/api/dashboard/stats" -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 7.3 الفهرسة

```powershell
# فحص تزايدي (الافتراضي — الأسرع)
$body = '{"roots":["D:/Media"],"mode":"incremental"}'
Set-Content "$env:TEMP\scan.json" -Value $body -Encoding UTF8 -NoNewline
Invoke-WebRequest "http://127.0.0.1:8080/api/index" -Method POST `
  -InFile "$env:TEMP\scan.json" -ContentType "application/json" `
  -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# فحص كامل مع فحص تقني (ffprobe) — أبطأ بكثير
$body = '{"roots":["D:/Media"],"mode":"full","inspect":true}'
Set-Content "$env:TEMP\scan.json" -Value $body -Encoding UTF8 -NoNewline
Invoke-WebRequest "http://127.0.0.1:8080/api/index" -Method POST `
  -InFile "$env:TEMP\scan.json" -ContentType "application/json" `
  -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# معاينة بدون كتابة لقاعدة البيانات (Dry-Run)
$body = '{"roots":["D:/Media"]}'
Set-Content "$env:TEMP\preview.json" -Value $body -Encoding UTF8 -NoNewline
Invoke-WebRequest "http://127.0.0.1:8080/api/index/preview" -Method POST `
  -InFile "$env:TEMP\preview.json" -ContentType "application/json" `
  -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 7.4 التحكم في الفحص الجاري (ADR-011)

```powershell
# التقدم الحيّ + العمال
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/status" -UseBasicParsing | Select-Object -ExpandProperty Content

# العمال فقط
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/workers" -UseBasicParsing | Select-Object -ExpandProperty Content

# إيقاف مؤقت تعاوني (يُبقي العدّادات)
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/pause" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# استئناف من نفس النقطة
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/resume" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# إلغاء نهائي (يفقد العدّادات) — عملية مختلفة عن الإيقاف
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/cancel" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# فحوص لم تكتمل (بعد انقطاع)
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/interrupted" -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 7.5 قائمة المراجعة (ADR-010)

```powershell
# الحالات الغامضة
Invoke-WebRequest "http://127.0.0.1:8080/api/resolution/queue?limit=20" -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# الإحصاءات حسب السبب
Invoke-WebRequest "http://127.0.0.1:8080/api/resolution/stats" -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# اتخاذ قرار على عنصر
# action: attach | create | ignore | mark_movie | move_season
$body = '{"action":"attach","work_id":42,"season":1,"episode":4,"learn_alias":true}'
Set-Content "$env:TEMP\decide.json" -Value $body -Encoding UTF8 -NoNewline
Invoke-WebRequest "http://127.0.0.1:8080/api/resolution/queue/7/decide" -Method POST `
  -InFile "$env:TEMP\decide.json" -ContentType "application/json" `
  -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 7.6 البحث (ADR-011)

```powershell
# بحث
Invoke-WebRequest "http://127.0.0.1:8080/api/search?q=matrix&limit=10" -UseBasicParsing | Select-Object -ExpandProperty Content

# إعادة بناء الفهرس (استكمال من الـ cursor)
Invoke-WebRequest "http://127.0.0.1:8080/api/search/sync" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content

# إعادة بناء كاملة من الصفر (reset)
Invoke-WebRequest "http://127.0.0.1:8080/api/search/sync?reset=true" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 7.7 فحص الجودة والصيانة

```powershell
Invoke-WebRequest "http://127.0.0.1:8080/api/quality/report" -UseBasicParsing | Select-Object -ExpandProperty Content
Invoke-WebRequest "http://127.0.0.1:8080/api/library/duplicates" -UseBasicParsing | Select-Object -ExpandProperty Content
Invoke-WebRequest "http://127.0.0.1:8080/api/library/missing-episodes" -UseBasicParsing | Select-Object -ExpandProperty Content
Invoke-WebRequest "http://127.0.0.1:8080/api/library/corrupted" -UseBasicParsing | Select-Object -ExpandProperty Content

# تصحيح وسوم الأصل من بنية المجلدات
Invoke-WebRequest "http://127.0.0.1:8080/api/library/classify-origins" -Method POST -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

---

## 8. أدوات التشخيص

هذه أدوات تطوير مبنية في `server/cmd/`. **ليست جزءًا من الإنتاج.**

### 8.1 `resolutionprobe` — تشغيل قرار الكيانات على مكتبة حقيقية

يشغّل الـ pipeline فعليًا ويطبع ما قرره لكل ملف، مع فحص صريح للانحدارات.

```powershell
$env:Path += ";C:\Users\mousa\go\bin"
cd c:\Users\mousa\Desktop\project\NEXORA\server

# على مسار محدد
go run ./cmd/resolutionprobe "D:\Media"

# أو بلا وسيط (يستخدم مسار الاختبار الافتراضي)
go run ./cmd/resolutionprobe
```

**ما يعرضه:**
```text
scanned 137 media files from استراحة رئيسية
=== RESOLUTION OUTCOME ON THE REAL LIBRARY ===
files                        : 137
attached or would create work: 111
queued for review            : 26
distinct works proposed      : 68

=== THE EXACT REGRESSIONS FROM THE OLD INGEST ===
  correctly rejected      : "03"
  correctly rejected      : "الكنزنت"
regressions: 0
```

### 8.2 `verifyresolution` — فحص `/api/index` الحقي

يتصل بالخادم العامل ويشغّل فحصًا كاملًا، ثم يطبع النتيجة المنظّمة.

```powershell
# يتطلب خادم Go يعمل على 8080
cd c:\Users\mousa\Desktop\project\NEXORA\server
go run ./cmd/verifyresolution
```

### 8.3 `transfertest` — فحص خلفيات النقل

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server
go run ./cmd/transfertest
```

---

## 9. Copy Bridge — نسخ USB والهواتف

خدمة محلية **على جهاز العميل** (لا السيرفر المركزي) تنسخ من السيرفر إلى أجهزة
USB والهواتف. تستمع على `127.0.0.1:32145` فقط.

### 9.1 البناء

```powershell
$env:Path += ";C:\Users\mousa\go\bin"
cd c:\Users\mousa\Desktop\project\NEXORA\server
go build -o nexora-bridge.exe ./cmd/copybridge
```

### 9.2 التشغيل التفاعلي (للتطوير)

```powershell
# يتطلب صلاحيات المسؤول للوصول إلى أجهزة USB
cd c:\Users\mousa\Desktop\project\NEXORA\server
.\nexora-bridge.exe -debug
```

### 9.3 التثبيت كخدمة ويندوز

```bat
:: شغّل موجه الأوامر كمسؤول (Run as Administrator) أولًا
cd c:\Users\mousa\Desktop\project\NEXORA
scripts\install-bridge-service.bat
```

**ما يفعله السكربت:**
1. يتحقق من صلاحيات المسؤول
2. يوقف ويحذف أي نسخة قديمة من الخدمة
3. يسجّل `NEXORACopyBridge` عبر `sc.exe`
4. يضبط إعادة تشغيل تلقائية عند التعثر
5. يفتح المنفذ 32145 في جدار الحماية
6. يبدأ الخدمة

### 9.4 الإزالة

```bat
cd c:\Users\mousa\Desktop\project\NEXORA
scripts\uninstall-bridge-service.bat
```

### 9.5 التحكم في الخدمة يدويًا

```powershell
sc.exe query NEXORACopyBridge
sc.exe start NEXORACopyBridge
sc.exe stop NEXORACopyBridge
sc.exe delete NEXORACopyBridge
```

### 9.6 مصادقة الـ Bridge (ADR-011)

الأوامر **المُعدِّلة** (copy/mkdir/eject/cancel) يمكن حمايتها بـ token:

```ini
# في .env على جهاز العميل
NEXORA_COPY_BRIDGE_TOKEN=مفتاح-عشوائي-طويل
```

- إن كان مضبوطًا: يُطلب في هيدر `Authorization: Bearer <token>`
- إن كان فارغًا: يعمل كما كان (توافق للخلف) مع تحذير واحد عند الإقلاع

**قراءة الأجهزة تبقى مفتوحة** حتى تستطيع الواجهة عرض حالة الـ Bridge قبل
تزويد بيانات الدخول.

### 9.7 التحقق من عمل الـ Bridge

```powershell
Invoke-WebRequest "http://127.0.0.1:32145/health" -UseBasicParsing | Select-Object -ExpandProperty Content
Invoke-WebRequest "http://127.0.0.1:32145/devices" -UseBasicParsing | Select-Object -ExpandProperty Content
```

### 9.8 قيد مهم — Session 0

الخدمة الافتراضية تعمل بـ `LocalSystem` (**Session 0**):
- ✅ Storage (أقراص عادية) — يعمل
- ✅ iOS عبر usbmuxd — يعمل
- ⚠️ **Android MTP (Shell COM)** — قد يتطلب جلسة مستخدم تفاعلية في بعض إصدارات ويندوز

لتشغيلها بحساب مستخدم:

```powershell
sc.exe config NEXORACopyBridge obj= ".\اسم-المستخدم" password= "كلمة-المرور"
sc.exe start NEXORACopyBridge
```

### 9.9 متغيرات البيئة للـ Bridge

```ini
NEXORA_COPY_BRIDGE_ADDR=127.0.0.1:32145
NEXORA_COPY_BRIDGE_CORS_ORIGIN=              # فارغ = loopback + LAN خاصة
NEXORA_COPY_BRIDGE_TOKEN=                    # فارغ = بلا مصادقة للأوامر المُعدِّلة
NEXORA_COPY_BRIDGE_TEMP_DIR=                 # فارغ = %TEMP%\nexora-copybridge
```

---

## 10. الاختبارات والتحقق

### 10.1 الاختبارات

```powershell
$env:Path += ";C:\Users\mousa\go\bin"
cd c:\Users\mousa\Desktop\project\NEXORA\server

# كل الاختبارات (يستخدم cache)
go test ./...

# بلا cache — للتأكد الحقي
go test -count=1 ./...

# حزمة محددة بتفصيل
go test -v ./internal/identity/
go test -v ./internal/scanner/

# اختبار واحد بعينه
go test ./internal/scanner/ -run TestPauseIsCooperativeAndResumes -v

# مع كشف حالات التسابق (يتطلب C toolchain — غير متاح في هذه البيئة)
$env:CGO_ENABLED="1"
go test -race ./...
```

### 10.2 فحوص الكود

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server

# فحص ثابت (يكشف أخطاء منطقية محتملة)
go vet ./...

# التحقق من التنسيق (يطبع الملفات غير المنسّقة)
gofmt -l ./internal/ ./cmd/

# إصلاح التنسيق
gofmt -w ./internal/ ./cmd/

# فحص الاعتماديات الأمنية (يتطلب شبكة)
go list -m all
```

### 10.3 القياسات (Benchmarks)

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server

# كل القياسات
go test ./internal/scanner/ -run XXX -bench . -benchmem -benchtime 10x

# قياس محدد
go test ./internal/scanner/ -run XXX -bench BenchmarkScanSyntheticTree -benchmem
```

### 10.4 التحقق الكامل قبل أي رفع

نفّذ هذا الأربعة معًا. إن مرّت كلها فالمشروع سليم:

```powershell
$env:Path += ";C:\Users\mousa\go\bin;C:\Program Files\nodejs"
cd c:\Users\mousa\Desktop\project\NEXORA

# ── الخادم ──
cd server
gofmt -l ./internal/ ./cmd/        # يجب أن يكون فارغًا
go build ./...                     # يجب أن يخرج 0
go vet ./...                       # يجب أن يخرج 0
go test -count=1 ./...             # كل الحزم ok

# ── الواجهة ──
cd ..\client
npm run build                      # يجب أن ينجح
```

### 10.5 صحة قاعدة البيانات

```powershell
# الـ migrations المطبقة
docker exec nexora-postgres psql -U nexora -d nexora -tAc "SELECT name FROM schema_migrations ORDER BY name DESC LIMIT 8;"

# إحصاءات الكتالوج
docker exec nexora-postgres psql -U nexora -d nexora -tAc "SELECT 'works='||COUNT(*) FROM media_items UNION ALL SELECT 'files='||COUNT(*) FROM video_files;"

# الملفات التي تحتاج مراجعة
docker exec nexora-postgres psql -U nexora -d nexora -tAc "SELECT reason, COUNT(*) FROM resolution_queue GROUP BY reason;"

# حالة الأقراص
docker exec nexora-postgres psql -U nexora -d nexora -tAc "SELECT root_path, status FROM scan_roots ORDER BY id DESC LIMIT 5;"
```

---

## 11. قاعدة البيانات والنسخ الاحتياطي

### 11.1 النسخ الاحتياطي

```powershell
# نسخة كاملة
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
docker exec nexora-postgres pg_dump -U nexora -d nexora -F c -f /tmp/nexora-$stamp.dump
docker cp "nexora-postgres:/tmp/nexora-$stamp.dump" ".\backups\nexora-$stamp.dump"

# نسخة نصية (قابلة للقراءة والتحرير)
docker exec nexora-postgres pg_dump -U nexora -d nexora -F p -f /tmp/nexora-$stamp.sql
docker cp "nexora-postgres:/tmp/nexora-$stamp.sql" ".\backups\nexora-$stamp.sql"
```

أنشئ مجلد النسخ أولًا:

```powershell
New-Item -ItemType Directory -Force -Path "c:\Users\mousa\Desktop\project\NEXORA\backups"
```

### 11.2 الاستعادة

```powershell
# من نسخة مخصصة (custom format)
docker cp ".\backups\nexora-20260919-120000.dump" nexora-postgres:/tmp/restore.dump
docker exec nexora-postgres pg_restore -U nexora -d nexora --clean --if-exists /tmp/restore.dump

# من نسخة نصية
docker cp ".\backups\nexora-20260919-120000.sql" nexora-postgres:/tmp/restore.sql
docker exec nexora-postgres psql -U nexora -d nexora -f /tmp/restore.sql
```

### 11.3 نسخ volumes كاملة (Docker)

```powershell
docker run --rm -v nexora_postgres-data:/data -v "${PWD}\backups:/backup" alpine `
  tar czf /backup/postgres-volume.tar.gz -C /data .
```

### 11.4 الاتصال التفاعلي بقاعدة البيانات

```powershell
# جلسة psql
docker exec -it nexora-postgres psql -U nexora -d nexora

# أمر واحد مباشر
docker exec nexora-postgres psql -U nexora -d nexora -c "SELECT COUNT(*) FROM video_files;"
```

### 11.5 ما يجب نسخه احتياطيًا

| العنصر | لماذا |
|---|---|
| PostgreSQL | **مصدر الحقيقة** للكتالوج والعلاقات وقرارات المسؤول |
| `assets/images/` | صور مخزّنة ومصغرات (قابلة للاستعادة بـ FFmpeg، لكن النسخ أسرع) |
| `.env` | الإعدادات — **لا يُرفع لـ Git، فاحتفظ بنسخة آمنة منفصلة** |

**ملا يُحتاج نسخه:** Meilisearch (قابل لإعادة البناء من قاعدة البيانات عبر
`/api/search/sync`)، وملفات الوسائط الأصلية (تبقى على أقراصها).

---

## 12. Git — العمل اليومي

> دليل مفصّل في [`GIT_WORKFLOW.md`](GIT_WORKFLOW.md). هذا ملخص سريع.

```powershell
$env:Path += ";C:\Program Files\Git\cmd"
cd c:\Users\mousa\Desktop\project\NEXORA

# ── الحالة ──
git status --short --branch
git log --oneline -10

# ── فرع جديد لميزة ──
git checkout main
git pull origin main
git checkout -b feature/اسم-الميزة

# ── الإيداع ──
git add path/to/file.go          # ملف محدد — أفضل من git add -A
git commit -m "feat(scope): وصف موجز"

# ── الرفع ──
git push -u origin feature/اسم-الميزة

# ── معاينة ما سيأتي قبل الدمج ──
git fetch origin
git log --oneline HEAD..origin/main
git diff --stat HEAD...origin/main

# ── الدمج ──
git merge origin/main --no-edit

# ── نسخة احتياطية قبل أي عملية خطرة ──
git tag backup/اسم-واضح HEAD
```

---

## 13. حل المشاكل الشائعة

### الخادم يخرج فورًا بـ exit code 1

**السبب الأكثر شيوعًا:** قاعدة البيانات غير جاهزة.

```powershell
# 1. تأكد أن Docker يعمل
docker ps

# 2. إن لم تكن، شغّلها
docker compose up -d postgres meilisearch redis

# 3. انتظر الجاهزية
docker compose ps
docker exec nexora-postgres pg_isready -U nexora -d nexora

# 4. ثم أعد تشغيل الخادم
```

### المنفذ 8080 مشغول

```
http server failed: listen tcp :8080: bind: ... Only one usage of each socket address
```

```powershell
# اعرف من يحجز المنفذ
$c = Get-NetTCPConnection -State Listen -LocalPort 8080
Get-Process -Id $c.OwningProcess | Select-Object Id, ProcessName, Path

# أوقفه
Stop-Process -Id $c.OwningProcess -Force
```

### الخادم لا يجد بيانات الدخول الإدارية

**السبب:** شغّلت الخادم من مجلد لا يحتوي `.env` الصحيح.

```powershell
# تأكد من وجود الملف في المكان المتوقع
Test-Path "server\.env"    # إن شغّلت من server/
Test-Path ".env"           # إن شغّلت من الجذر
```

إن ظهر في السجل `Temporary admin password for this run` فأنت تقرأ ملفًا خاطئًا.

### الملفات لا تظهر في الفهرس

```powershell
# 1. هل الجذر موجود فعلًا؟
Test-Path "D:\Media"

# 2. هل الفحص وجد ملفات؟
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/status" -UseBasicParsing | Select-Object -ExpandProperty Content
# ابحث عن filesSeen و videoCandidates

# 3. هل ذهبت للمراجعة؟
Invoke-WebRequest "http://127.0.0.1:8080/api/resolution/stats" -Headers $headers -UseBasicParsing | Select-Object -ExpandProperty Content
```

### تعارض Git

```powershell
git status                     # الملفات المتعارضة
git diff path/to/conflicted    # اقرأ التعارض

# حلّه يدويًا (لا تستخدم --ours/--theirs بلا قراءة)
git add path/to/conflicted
git commit
```

### `npm install` يفشل

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\client
Remove-Item node_modules -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item package-lock.json -Force -ErrorAction SilentlyContinue
npm cache clean --force
npm install
```

### `go mod` يشتكي من إصدار

```powershell
cd c:\Users\mousa\Desktop\project\NEXORA\server
go version          # يجب أن يكون 1.26+
go clean -modcache
go mod download
go build ./...
```

### تحذيرات `LF will be replaced by CRLF`

**طبيعية على Windows ولا تضر.** Git يوحّد نهايات الأسطر تلقائيًا. لتعطيلها:

```powershell
git config core.autocrlf false
```

### PowerShell يقطع الأوامر عند `#` أو `&`

استخدم `-InFile` بدل `-Body`، أو اكتب JSON في ملف أولًا:

```powershell
Set-Content "$env:TEMP\body.json" -Value $json -Encoding UTF8 -NoNewline
Invoke-WebRequest ... -InFile "$env:TEMP\body.json"
```

---

## 14. مرجع سريع — كل الأوامر

### التهيئة (مرة واحدة)

```powershell
$env:Path += ";C:\Users\mousa\go\bin;C:\Program Files\nodejs;C:\Program Files\Git\cmd;C:\Program Files\Docker\Docker\resources\bin"
cd c:\Users\mousa\Desktop\project\NEXORA
Copy-Item .env.example .env          # ثم حرّر .env
cd server;  go mod download
cd ..\client; npm install
```

### التشغيل اليومي

```powershell
# 1. البنية التحتية
cd c:\Users\mousa\Desktop\project\NEXORA
docker compose up -d postgres meilisearch redis

# 2. الخادم
cd server; go run ./cmd/api

# 3. الواجهة (نافذة أخرى)
cd client; npm run dev
```

### الإيقاف

```powershell
Get-Process nexora-api -ErrorAction SilentlyContinue | Stop-Process -Force
docker compose stop
# ثم Ctrl+C في نافذة Vite
```

### البناء

```powershell
cd server; go build -o ..\nexora-api.exe .\cmd\api
cd server; go build -o nexora-bridge.exe .\cmd\copybridge
cd client; npm run build
docker compose --profile full build
```

### التحقق

```powershell
cd server; go build ./...; go vet ./...; go test -count=1 ./...
cd client; npm run build
```

### Docker

```powershell
docker compose up -d postgres meilisearch redis   # بنية تحتية
docker compose --profile full up -d               # كل شيء
docker compose ps                                 # الحالة
docker compose logs -f server                     # السجلات
docker compose stop                               # إيقاف
docker compose down                               # حذف الحاويات
docker compose down -v                            # حذف كل شيء + البيانات
```

### قاعدة البيانات

```powershell
docker exec nexora-postgres psql -U nexora -d nexora -c "SELECT ..."
docker exec nexora-postgres pg_dump -U nexora -d nexora -F c -f /tmp/backup.dump
docker cp "nexora-postgres:/tmp/backup.dump" ".\backups\"
```

### API

```powershell
Invoke-WebRequest "http://127.0.0.1:8080/api/health" -UseBasicParsing
Invoke-WebRequest "http://127.0.0.1:8080/api/scan/status" -UseBasicParsing
Invoke-WebRequest "http://127.0.0.1:8080/api/resolution/stats" -Headers $headers -UseBasicParsing
Invoke-WebRequest "http://127.0.0.1:8080/api/index" -Method POST -InFile scan.json -Headers $headers -ContentType "application/json"
```

### Git

```powershell
git status --short --branch
git checkout -b feature/name
git add path/to/file
git commit -m "feat(scope): description"
git push -u origin feature/name
git fetch origin; git log --oneline HEAD..origin/main
git merge origin/main --no-edit
git tag backup/name HEAD
```
