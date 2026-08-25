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

## التشغيل من الجوال عبر Wi-Fi

اللابتوب هو خادم NEXORA المحلي: يشغّل Docker وقاعدة البيانات وخادم Go وواجهة React. يستطيع أي جوال على **نفس الراوتر وشبكة Wi-Fi** فتح الموقع منه.

### الخطوات كل مرة

1. افتح PowerShell داخل المشروع وشغّل خدمات قاعدة البيانات والبحث:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose up -d
```

2. افتح نافذة PowerShell ثانية لتشغيل الخادم الخلفي. اتركها مفتوحة:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\server
$env:Path += ";C:\Users\mousa\Desktop\project\NEXORA\.tools\go\bin"
go run .\cmd\api
```

3. افتح نافذة PowerShell ثالثة لتشغيل واجهة الموقع. اتركها مفتوحة:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\client
npm run dev
```

### أي رابط أفتح؟

سيظهر Vite سطران مشابهـان لهذا:

```text
Local:   http://localhost:5173/
Network: http://192.168.1.35:5173/
```

- `localhost` طبيعي تماماً، لكنه يعمل **على اللابتوب نفسه فقط**.
- الرابط بعد `Network` هو الذي تفتحه في الجوال. في الشبكة الحالية استخدم:

```text
http://192.168.1.35:5173
```

إذا تغيّر الراوتر أو أعيد اتصال Wi-Fi فقد يتغير الرقم. لمعرفة الرقم الصحيح شغّل على اللابتوب:

```powershell
ipconfig
```

ثم ابحث تحت `Wireless LAN adapter Wi-Fi` عن `IPv4 Address`، واستخدمه في الجوال بهذه الصيغة:

```text
http://عنوان-IP-الجديد:5173
```

### الإيقاف

- أوقف API والواجهة بالضغط على `Ctrl + C` في نافذتيهما.
- لإيقاف Docker عند الانتهاء:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose down
```

> إذا لم يفتح الجوال الموقع، تأكد أن الجهازين على نفس Wi-Fi، ثم اسمح لـNode.js عبر Windows Firewall على شبكة **Private** عندما تظهر رسالة الحماية.

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
