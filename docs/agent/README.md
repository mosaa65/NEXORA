# NEXORA Agent — Documentation Index

> **اقرأ هذا أولاً.** ثم ابدأ من [MIGRATION_BASELINE](MIGRATION_BASELINE.md).

---

## ما هذا؟

`NEXORA Agent` هو المكوّن المحلي الذي يصل إلى أجهزة المستخدم (الأقراص، USB،
الهواتف) وينسخ إليها. **وهو موجود بالفعل** باسم `copybridge` وموثّق في ADR-008.

هذه المهمة **لا تبني Agent جديداً** — بل:
1. توثّق ما هو موجود بدقة.
2. **تعيد بناء طبقة Android فقط.**
3. تحافظ على iPhone / USB / الأقراص **كما هي** بلا مساس.

---

## الخريطة

| الوثيقة | اقرأها إذا أردت أن تعرف |
|---|---|
| **[MIGRATION_BASELINE](MIGRATION_BASELINE.md)** | **حالة المشروع قبل التغيير، وما هو محمي، وكيف ترجع** |
| [ARCHITECTURE](ARCHITECTURE.md) | المكونات، تدفق الأجهزة، تدفق النقل، حدود الطبقات |
| [ANDROID](ANDROID.md) | كيف يعمل Android الآن (Shell COM)، الفجوات، WPD، ADB، قيود Android الحديث |
| [WINDOWS_STORAGE](WINDOWS_STORAGE.md) | Win32، الأقراص، USB، External، أمان الملفات |
| [TRANSFER_ENGINE](TRANSFER_ENGINE.md) | Streaming، المخازن، الملفات الكبيرة، Progress، Retry، Verify |
| [DEVICE_MODEL](DEVICE_MODEL.md) | نموذج الجهاز، `TransferBackend` كـ `DeviceProvider`، Capabilities |
| [API](API.md) | 12 endpoint + SSE، المصادقة، أكواد الخطأ |
| [SECURITY](SECURITY.md) | نموذج الثقة، سطح الهجوم، Elevation، Audit |
| [ROADMAP](ROADMAP.md) | المرحلة الحالية، WPD، ADB، Providers مستقبلية |
| [TEST_PLAN](TEST_PLAN.md) | اختبارات عدم التراجع، الأندرويد، الأداء، الأمان |

---

## القاعدة الحاكمة

```text
PRESERVE WHAT WORKS
ISOLATE WHAT CHANGES
DOCUMENT EVERYTHING
TEST BEFORE DECLARING SUCCESS
```

---

## ملخص الحالة في جدول واحد

| الطبقة | الملف | الحالة | يُلمس في هذه المهمة؟ |
|---|---|
| Agent (خدمة + API) | `cmd/copybridge`, `internal/copybridge` | ✅ يعمل | ❌ لا |
| تجريد النقل | `backend.go` | ✅ يعمل | ⚠️ إضافات اختيارية فقط |
| المحرك v2 | `engine.go`, `worker.go`, `scheduler.go` | ✅ يعمل | ⚠️ إضافات فقط |
| **Storage (USB/HDD/أقراص)** | `storage_backend.go`, `removable_windows.go` | ✅ مكتمل | ❌ **محمي** |
| **iPhone (iOS/AFC)** | `go_ios_backend.go` | ✅ **مكتمل ومُختبر** | ❌ **محمي — قاعدة غير قابلة للتفاوض** |
| **Android (Shell COM)** | `android_backend.go` | ⚠️ **ناقص** | ✅ **هذا هو نطاق العمل** |
| WPD | — | ⏭️ مُخطَّط | ❌ لا (قرار المالك: خيار أ) |
| ADB | — | ⏭️ مُخطَّط | ❌ لا |

---

## قيود يجب معرفتها قبل البدء

1. **`service.go` ليس كوداً ميتاً** — موصول من 3 أماكن. **لا يُحذف.**
2. **لا WPD في المشروع** — Android يستخدم PowerShell Shell COM.
3. **`PutStream` في Android يمرّ بملف temp** — قاعدة الملفات الكبيرة. **غير
   قابل للإصلاح عبر Shell COM.** يحتاج WPD.
4. **لا جهاز Android في بيئة التطوير** — الاختبار الحي مُعلَّق، ويُوثَّق بصدق.
5. **`go fmt ./...` على ويندوز يلوّث 25 ملفاً** — لا تُشغّله.

---

## مصادر خارجية (Microsoft الرسمية — الطلب §34)

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [WPD API](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/wpd-application-programming-interface)
- [Win32 File Management](https://learn.microsoft.com/en-us/windows/win32/fileio/file-management-functions)
- [Windows Shell Extensions](https://learn.microsoft.com/en-us/windows/win32/shell/shell-exts)
- [Folder.CopyHere](https://learn.microsoft.com/en-us/windows/win32/shell/folder-copyhere)
