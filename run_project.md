أوقفت خادم الواجهة وخادم الخلفية.

لتشغيل النظام، افتح نافذتين PowerShell من مجلد المشروع:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose up -d
```

ثم شغّل الخلفية:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\server
$env:Path += ";C:\Users\mousa\Desktop\project\NEXORA\.tools\go\bin"

Get-Content ..\.env | ForEach-Object {
  if ($_ -match '^\s*([^#=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2], 'Process')
  }
}

go run .\cmd\api
```

وفي نافذة PowerShell ثانية شغّل الواجهة:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA\client
npm run dev
```

ثم افتح: http://localhost:5173

للإيقاف:

- أوقف الواجهة والخلفية داخل نافذتيهما بـ `Ctrl + C`.
- أوقف Docker مع الاحتفاظ بالبيانات:

```powershell
cd C:\Users\mousa\Desktop\project\NEXORA
docker compose down
```

تعذر علي إيقاف Docker من جلستي لأن Docker Engine عندك يطلب صلاحيات أعلى، لكن الأمر الأخير سيعمل من PowerShell الخاص بك.

هل يموفر المواسم او الحقات ايضا ابحث في الموقع وضيف ومن ناحيه الموقع حقنا اريد تمرير في عجله  ماوس الفاره يكون يشتغل في التمرير الافقي واريد تضيف في الرائيسيه