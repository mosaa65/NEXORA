# NEXORA Agent — Device Model

> **الحالة:** النموذج الحالي مسطّح (`Device`). الطلب §9 يطلب نموذجاً فيه
> `storages[]`. هذا الملف يوثّق **ما هو موجود**، والفرق، وخطة التوسيع.

---

## 1. النموذج الحالي الفعلي

`server/internal/transfer/models.go`:

```go
type DeviceType string
const (
    DeviceAndroid DeviceType = "android"
    DeviceIOS     DeviceType = "ios"
    DeviceStorage DeviceType = "storage"
)

type Device struct {
    ID         string     `json:"id"`         // "disk_E" | "mtp_<name>" | "ios_<udid>"
    Name       string     `json:"name"`
    Model      string     `json:"model"`
    Type       DeviceType `json:"type"`
    Status     string     `json:"status"`
    FreeSpace  int64      `json:"free_space"`
    TotalSpace int64      `json:"total_space"`
    FileSystem string     `json:"file_system,omitempty"`
}
```

### المعرّفات المُنتَجة (Prefixes)

| النوع | الصيغة | المصدر |
|---|---|---|
| قرص قابل للإزالة | `disk_<letter>` | `removable_windows.go` — `"disk_" + letter` |
| جهاز MTP/Android | `mtp_<name>` | `service.go` — يُبنى أثناء الاكتشاف |
| iPhone | `ios_<udid>` | `go_ios_backend.go` |

**البادئة مهمة:** `Connect` في Android يفحص البادئة:

```go
if strings.HasPrefix(deviceName, "mtp_") || strings.HasPrefix(deviceName, "ios_") {
    parts := strings.SplitN(deviceName, "_", 3)
    if len(parts) == 3 {
        deviceName = strings.ReplaceAll(parts[2], "_", " ")
    }
}
```

---

## 2. الفرق عن النموذج المطلوب (الطلب §9)

### المطلوب

```json
{
  "id": "android-01",
  "type": "android_mtp",
  "name": "Samsung A55",
  "manufacturer": "...",
  "model": "...",
  "connection": "usb",
  "storages": [
    { "id": "internal", "name": "Internal Storage" },
    { "id": "sdcard",   "name": "SD Card" }
  ],
  "capabilities": [...]
}
```

### الموجود

```json
{
  "id": "mtp_Samsung A55",
  "name": "Samsung A55",
  "model": "",
  "type": "android",
  "status": "...",
  "free_space": 0,
  "total_space": 0,
  "file_system": ""
}
```

### الفجوات

| المطلوب | الموجود | الأثر |
|---|---|---|
| `storages[]` (متعدد) | مساحة واحدة ضمنياً (`Select-Object -First 1`) | **لا كارت SD** 🔴 |
| `manufacturer` | ❌ | لا تمييز مصنّع |
| `connection` | ❌ | لا تمييز USB/شبكة |
| `capabilities[]` | ❌ | لا إعلان للقدرات (Resume؟ Delete؟) |
| `free_space`/`total_space` للأجهزة المحمولة | **0** | لا عرض للمساحة |

> **⚠️ حرج:** `storages[]` فجوة حقيقية — جهاز له ذاكرة داخلية + SD يُكتب على
> الأولى فقط. مُدرَج في نطاق العمل.

---

## 3. طبقة التجريد العملية — `TransferBackend`

الطلب §10 يطلب `DeviceProvider` ثم providers منفصلة. **هذا موجود** — باسم آخر:

```
DeviceProvider  ≡  TransferBackend        (backend.go)
    ├── FilesystemProvider  ≡  StorageBackend   (storage_backend.go)
    ├── WPDProvider         ≡  (مُخطَّط — ROADMAP)
    ├── AndroidMTPProvider  ≡  AndroidBackend   (android_backend.go)
    ├── IOSProvider         ≡  IOSBackend       (go_ios_backend.go)
    └── (مستقبلاً)              ADB, Network, NAS
```

```go
type TransferBackend interface {
    Connect(ctx, Device, TransferDestination) error
    List(ctx, path) ([]RemoteEntry, error)
    Stat(ctx, path) (RemoteEntry, error)
    Mkdir(ctx, path) error
    Put(ctx, source, destination, PutOptions) error
    PutStream(ctx, reader, size, destination, PutOptions) error
    Delete(ctx, path) error
    Rename(ctx, oldPath, newPath) error
    Close() error
}
```

**الحقن عبر Factory:**

```go
type BackendFactory func(device Device) (TransferBackend, error)
```

> **قرار معمارى:** لا نُنشئ `DeviceProvider` جديدة. **نُوسّع `TransferBackend`**
> (إضافات اختيارية فقط)، لأن العقد موجود، مُختبَر، ويخدم 3 backends تعمل.
> إعادة التسمية بلا قيمة، وتكسر المحرك.

---

## 4. Capabilities (مقترح — الطلب §9, §15)

لمنع الواجهة من محاولة عملية لا يدعمها الـ backend:

```go
// مقترح — يُضاف عند الحاجة الفعلية
type Capabilities struct {
    List        bool
    Delete      bool
    Rename      bool
    Resume      bool
    StreamWrite bool   // كتابة من مجرى بلا ملف temp
    MultiStorage bool
}
```

**الواقع الحالي** (من الكود):

| Backend | List | Delete | Rename | StreamWrite | MultiStorage |
|---|---|
| Storage | ✅ | ✅ | ✅ | ✅ | — |
| iOS (AFC) | ✅ | — | — | ✅ | — |
| Android (MTP) | ❌ | ❌ | ❌ | ❌ | ❌ |

الواجهة اليوم **لا تعرف** هذه الفروق — تحاول وتفشل. Capabilities تُصلح هذا.

---

## 5. مثال الاستجابة للواجهة (الحالي)

```json
[
  { "id": "disk_E", "name": "Kingston (E:)", "type": "storage",
    "free_space": 28765432100, "total_space": 64424509440,
    "file_system": "exFAT" },

  { "id": "mtp_Samsung A55", "name": "Samsung A55", "type": "android",
    "free_space": 0, "total_space": 0 },

  { "id": "ios_00008110...", "name": "iPhone", "type": "ios",
    "free_space": ..., "total_space": ... }
]
```

الواجهة تُعرضها موحّدة — كما يطلب الطلب §20.

---

## 6. نموذج الجهاز المستهدف (خطة — لا تنفيذ)

```go
// مستقبلاً — توسيع لا استبدال
type Device struct {
    ID           string
    Name         string
    Model        string
    Manufacturer string        // جديد
    Type         DeviceType
    Connection   string        // جديد: "usb" | "network" | "local"
    Status       string
    Storages     []DeviceStorage  // جديد
    Capabilities Capabilities     // جديد
}

type DeviceStorage struct {
    ID         string
    Name       string
    FreeSpace  int64
    TotalSpace int64
    FileSystem string
}
```

**التوافق:** الحقول الحالية (`FreeSpace`, `TotalSpace`) **تبقى** (لا تُحذف)،
وتُضاف الجديدة. الواجهة القديمة تستمر في العمل.

---

## 7. مصادر

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [Shell.NameSpace](https://learn.microsoft.com/en-us/windows/win32/shell/shell-namespace)
