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

### 4. فتح الموقع

افتح المتصفح على: **http://localhost:5173**

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
