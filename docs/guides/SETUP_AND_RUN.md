# 🚀 دليل إعداد وتشغيل NEXORA

## المتطلبات الأساسية

| الأداة | الإصدار الأدنى | الغرض |
|--------|---------------|-------|
| **Docker Desktop** | 4.x | تشغيل PostgreSQL + Meilisearch |
| **Go** | 1.22+ | تشغيل خادم API |
| **Node.js** | 18+ | تشغيل واجهة React |
| **npm** | 9+ | إدارة حزم الواجهة |

---

## التشغيل خطوة بخطوة

### 1. تشغيل الخدمات (PostgreSQL + Meilisearch)

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose up -d
```

### 2. تشغيل الخادم (Backend API)

افتح نافذة PowerShell جديدة:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\server

# تحميل متغيرات البيئة
Get-Content ..\.env | ForEach-Object {
  if ($_ -match '^\s*([^#=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
  }
}

go run .\cmd\api
```

### 3. تشغيل الواجهة (Frontend)

افتح نافذة PowerShell ثانية:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\client
npm install   # المرة الأولى فقط
npm run dev
```

### 4. اسم NEXORA داخل الشبكة وفتح الموقع

الواجهة تستعمل مسارات نسبية (`/api` و`/assets`)؛ لذلك لا يوجد `localhost` محفوظ داخل بطاقات النظام أو بياناته. الاسم الذي تفتحه به هو الاسم الذي سيظهر في كل الروابط.

الاسم المقترح للشبكة المحلية هو: **`nexora.local`**. قبل استخدامه يجب أن يحلّه DNS محلي أو ملف `hosts` في كل جهاز عميل. لا يمكن للكود وحده تسجيل اسم شبكة؛ هذا قرار DNS على الشبكة.

للتجربة الموثوقة على أجهزة Windows، افتح PowerShell كمسؤول على كل جهاز عميل ثم نفّذ—مع استبدال العنوان بعنوان جهاز خادم NEXORA الفعلي:

```powershell
Add-Content -Path "$env:SystemRoot\System32\drivers\etc\hosts" -Value "192.168.1.20 nexora.local"
```

بعدها افتح: **http://nexora.local:5173**. يظل Vite يمرر `/api` و`/assets` داخلياً إلى API، ولا يرى المستخدم `localhost` أو `127.0.0.1`.

> في بيئة إنتاجية يفضّل إنشاء سجل DNS ثابت في الراوتر/خادم DNS ثم تقديم الواجهة وAPI خلف reverse proxy على `http://nexora.local` أو HTTPS، بدلاً من تكرار ملف hosts.

---

## إيقاف الخدمات

1. أوقف الواجهة والخادم بـ `Ctrl + C` في نوافذهما
2. أوقف Docker:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose down
```

> **ملاحظة**: البيانات في PostgreSQL محفوظة عبر Docker volumes ولن تُفقد عند الإيقاف.

---

## إعداد البيئة

انسخ `.env.example` إلى `.env` وعدّل القيم حسب بيئتك:

```powershell
copy .env.example .env
```

## بناء النسخة الإنتاجية

```powershell
# بناء الواجهة
cd client && npm run build

# بناء الخادم
cd server && go build -o api.exe ./cmd/api
```
