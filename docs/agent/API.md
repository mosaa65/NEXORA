# NEXORA Agent — API

> **الحالة:** موثّق من الكود الفعلي (`server/internal/copybridge/server.go`).
> **Base URL:** `http://127.0.0.1:32145` — **محلي فقط**.

---

## 1. نقاط النهاية الفعلية (16)

| # | المسار | الطريقة | الوظيفة |
|---|---|
| 1 | `/api/health` | GET | حالة الوكيل |
| 2 | `/api/transfer/devices` | GET | اكتشاف الأجهزة |
| 3 | `/api/transfer/device-apps` | GET | تطبيقات الجهاز (iOS) |
| 4 | `/api/transfer/device-app-folders` | GET | مجلدات التطبيق (iOS) |
| 5 | `/api/transfer/browse` | GET | تصفح مجلد على الجهاز |
| 6 | `/api/transfer/mkdir` | POST | إنشاء مجلد |
| 7 | `/api/transfer/eject` | POST | إخراج آمن لقرص |
| 8 | `/api/transfer/copy` | POST | بدء نسخ |
| 9 | `/api/transfer/jobs` | GET | قائمة المهام |
| 10 | `/api/transfer/job/{id}` | GET | مهمة واحدة |
| 11 | `/api/transfer/cancel/{id}` | POST | إلغاء مهمة |
| 12 | `/api/transfer/events` | GET | **SSE** — بث الأحداث |

> **لا يوجد `POST /execute` ولا أي Command Shell مفتوح.** الطلب §31 يمنع هذا
> صراحةً — والواقع **مطابق**: كل endpoint عملية NEXORA محددة.

---

## 2. المصادقة (الطلب §18)

### الرمز — اختياري ومُقارَن بأمان

```go
// من copybridge/server.go
NEXORA_COPY_BRIDGE_TOKEN   // متغير بيئة
```

`tokenMatches` يستخدم **مقارنة constant-time** (لمنع timing attacks)، ويغطي
4 حالات مُختبَرة: بلا رمز، رمز خاطئ، رمز صحيح، بادئة/طول مختلف.

> **⚠️ ملاحظة:** الرمز **اختياري**. إن لم يُضبط، لا مصادقة. هذا مُوثَّق كخطر في
> [SECURITY.md](SECURITY.md) والخطة المقترحة **جعل الرمز إلزامياً**.

### CORS

```go
corsAllow  // يرفض المناشئ العامة بـ 403
NEXORA_COPY_BRIDGE_CORS_ORIGIN   // `*` فقط عند الطلب الصريح
```

> **مطابق للطلب §18:** لا يُكشف API على الشبكة المحلية بلا داعٍ — الاستماع على
> `127.0.0.1` فقط.

---

## 3. نموذج الطلب/الاستجابة

### `GET /api/transfer/devices`

```json
[
  { "id": "disk_E", "name": "Kingston (E:)", "type": "storage",
    "free_space": 28765432100, "total_space": 64424509440, "file_system": "exFAT" },
  { "id": "mtp_Samsung A55", "name": "Samsung A55", "type": "android" },
  { "id": "ios_00008110...", "name": "iPhone", "type": "ios" }
]
```

### `POST /api/transfer/copy`

```json
{
  "device_id": "mtp_Samsung A55",
  "source_path": "D:\\Media\\Movie.mkv",
  "sub_folder": "Movies",
  "target_folder": "Movies"
}
```

> `source_paths` (متعدد) و`target_app` (iOS) مدعومان أيضاً — انظر `CopyRequest`
> في `models.go`.

### `GET /api/transfer/job/{id}`

```json
{
  "id": "job_123",
  "device_id": "mtp_Samsung A55",
  "device_type": "android",
  "file_name": "Movie.mkv",
  "file_size": 80000000,
  "total_bytes": 80000000,
  "transferred_bytes": 42000000,
  "progress": 52.5,
  "speed_bps": 94000000,
  "eta_seconds": 404,
  "phase": "copying",
  "status": "processing"
}
```

> **مطابق للطلب §11:** كل الحقول المطلوبة (`bytes_total`, `bytes_transferred`,
> `speed`) موجودة، بـ `int64`.

---

## 4. الأحداث (SSE) — الطلب §19

### `GET /api/transfer/events`

قناة **Server-Sent Events** — تدفق أحادي من الوكيل للواجهة.

```text
data: {"type":"transfer.progress","job_id":"job_123","transferred_bytes":...data: {"type":"device.connected","device_id":"mtp_Samsung A55"}
```

**الأحداث المُبثَّة فعلاً:**
- توصيل/فصل الأجهزة (Hotplug)
- تحديثات تقدم النقل

> **فائدة معمارية:** SSE لا يستهلك موارد الشبكة بـ HTTP Polling — الواجهة تتلقى
> فورياً. مُوثَّق في `plan.md` كقرار متعمد.

**الأحداث المطلوبة (الطلب §19) والفجوة:**

| الحدث المطلوب | الحالة |
|---|---|
| Device Connected / Disconnected | ✅ |
| Device Changed | ⚠️ جزئي |
| Storage Available / Removed | ⚠️ جزئي |
| Transfer Started / Progress / Completed / Failed | ✅ |
| Transfer Paused / Resumed | ❌ (Pause غير مُنفَّذ) |

---

## 5. أكواد الخطأ (الطلب §29)

### الحالة الفعلية

```go
const (
    CodeAFCIOError     = "AFC_IO_ERROR"
    CodeSourceNotFound = "SOURCE_NOT_FOUND"
    CodeVerifyFailed   = "VERIFY_FAILED"
    CodeUserCancelled  = "USER_CANCELLED"
)
```

### المطلوب — الفجوة

```text
DEVICE_NOT_FOUND          ← يُضاف
DEVICE_DISCONNECTED       ← يُضاف
STORAGE_NOT_AVAILABLE     ← يُضاف
PERMISSION_DENIED         ← يُضاف
INSUFFICIENT_SPACE        ← يُضاف
SOURCE_NOT_FOUND          ✅ موجود
DESTINATION_NOT_FOUND     ← يُضاف
TRANSFER_CANCELLED        ✅ (= USER_CANCELLED)
TRANSFER_FAILED           ← يُضاف
UNSUPPORTED_OPERATION     ← يُضاف
MTP_ERROR                 ← يُضاف
WPD_ERROR                 ← عند تنفيذ WPD
NETWORK_ERROR             ← عند تنفيذ NetworkProvider
AUTH_ERROR                ← يُضاف
```

**المطلوب:** ألا تُعرض أخطاء Go الخام للمستخدم — طبقة Mapping.

---

## 6. مصادر

- [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [Go net/http ServeMux](https://pkg.go.dev/net/http#ServeMux)
