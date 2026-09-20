# NEXORA Agent — Security

> **المبدأ (الطلب §16-17, §31):** الوكيل قدرة محلية عالية، لكن API = وظائف
> محددة. لا Command Shell مفتوح. لا تجاوز للأمان.

---

## 1. نموذج الثقة المحلي

```text
Agent = Local Trusted Component
    │
    ├── قدرة محلية عالية: قرص، USB، MTP، Win32
    │
    └── لكن مكشوف فقط عبر:
        ├── API = وظائف NEXORA محددة (16 endpoint)
        ├── Authorization = رمز + CORS + origin
        └── Audit = تسجيل العمليات
```

**لا تُنشأ واجهة تنفيذ أوامر عشوائية.** الطلب §31 يمنع صراحةً:

```json
POST /execute  { "command": "whatever" }     ❌ ممنوع
```

**والواقع مطابق:** لا يوجد مثل هذا Endpoint. كل عملية محددة بالاسم.

---

## 2. سطح الهجوم (Attack Surface) — التحليل الفعلي

الطلب §31 يسأل 10 أسئلة. هذه الإجابات من الكود:

| # | السؤال | الجواب الفعلي |
|---|---|---|
| 1 | هل يمكن لموقع آخر استخدام Agent؟ | **CORS يمنع** — `corsAllow` يرفض المناشئ العامة بـ 403 |
| 2 | هل يمكن لجهاز على الشبكة؟ | **لا** — الاستماع على `127.0.0.1` فقط |
| 3 | Copy بدون Authorization؟ | ⚠️ **نعم إن لم يُضبط الرمز** — الرمز اختياري |
| 4 | الوصول لملف خارج العملية؟ | ⚠️ يعتمد على `mediaPathAllowed` في السيرفر (خطر موثَّق) |
| 5 | Path Traversal؟ | ⚠️ يحتاج تحققاً صريحاً (فجوة) |
| 6 | Destination خبيث؟ | ⚠️ يحتاج تحققاً (فجوة) |
| 7 | إجبار Agent على حذف؟ | ⚠️ `Delete` غير موصول بـ endpoint مستقل؛ `CopyHere` يحذف الهدف أولاً |
| 8 | عمليات غير مسموحة؟ | **لا** — لا endpoint عام |
| 9 | تنفيذ أوامر نظام عشوائية؟ | **لا** — لا `/execute` |
| 10 | الوصول لكل WPD؟ | N/A — WPD غير مُنفَّذ |

---

## 3. المصادقة والتصريح

### الحالة الفعلية

```go
NEXORA_COPY_BRIDGE_TOKEN              // اختياري
tokenMatches(a, b) → constant-time    // ✅ آمن ضد timing
NEXORA_COPY_BRIDGE_CORS_ORIGIN        // تقييد المناشئ
```

### المطلوب (الطلب §18)

| العنصر | الحالة |
|---|---|
| Session/token | ✅ (اختياري) |
| Origin validation | ✅ |
| Authentication | ⚠️ اختياري |
| Authorization | ⚠️ بلا niveles |
| Request ID | ❌ |
| Device ID | ✅ |
| Job ID | ✅ |

**التوصية:** جعل الرمز **إلزامياً**، وإضافة `Request ID` للتتبع.

---

## 4. الصلاحيات (الطلب §16)

### مبدأ: لا صلاحية عمياء

```text
❌ "Agent يمكنه كل شيء"
✅ Agent له صلاحيات Win32 طبيعية، وAPI مقيّد، وAudit يسجّل
```

### ماذا يحتاج الوكيل فعلاً؟

| القدرة | مطلوبة؟ | الحالة |
|---|---|---|
| قراءة الملفات | ✅ | `os.Open` |
| كتابة الملفات | ✅ | `io.CopyBuffer` |
| إنشاء مجلدات | ✅ | `os.MkdirAll`, `NewFolder` |
| قراءة الأقراص | ✅ | `kernel32` |
| WPD | ⏭️ مُخطَّط | — |
| Registry | ⏭️ للتكامل فقط | غير مستخدم حالياً |
| Explorer integration | ⏭️ مستقبلاً | غير مستخدم |

> **مطابق للطلب:** "Agent قوي وموثوق، وليس Agent يتحايل على النظام".

---

## 5. Elevation — لا تجاوز

### الممنوع تماماً (الطلب §17)

```text
❌ bypass            ❌ exploit
❌ privilege escalation tricks
❌ تعطيل Windows Security / Defender
❌ تجاوز UAC بطرق ملتوية
❌ registry hacks لصلاحية غير مشروعة
```

### المسموح

```text
✅ Windows Service بـ svc.Handler  (موجود — cmd/copybridge)
✅ تشغيل تحت حساب مستخدم/Administrator عبر sc.exe
✅ تشغيل كـ Console (-debug) للتطوير
```

**الحقائق من ADR-008:** الخدمة تُثبَّت بـ `sc.exe` بصلاحيات طبيعية. لا تجاوز.

---

## 6. Session 0 Isolation (قيد موثّق)

خدمات Windows تحت `LocalSystem` تعمل في Session 0 المعزولة:

| التقنية | تعمل في Session 0؟ |
|---|---|
| Win32 (kernel32) | ✅ 100% |
| iOS (usbmuxd/AFC) | ✅ 100% |
| **Android (`Shell.Application` COM)** | ⚠️ **قد يحتاج سياق تفاعلي** |

**الحل المُوثَّق (ADR-008):** تشغيل الخدمة تحت حساب المستخدم عبر
`sc.exe config ... obj= ".\Administrator" password= "..."`، أو Tray Agent في
`shell:startup`.

> **ملاحظة مهمة:** هذا ضروري لأن Android يستخدم Shell COM — وهذا **سبب إضافي**
> لصالح WPD مستقبلاً (WPD لا يحتاج سياق تفاعلي).

---

## 7. مسح أمني إلزامي بعد التنفيذ (الطلب §31)

```text
[ ] هل يمكن لموقع آخر استخدام Agent؟
[ ] هل يمكن لجهاز على الشبكة استخدام Agent؟
[ ] هل يمكن إرسال Copy request بدون Authorization؟
[ ] هل يمكن الوصول لملف خارج العملية المطلوبة؟
[ ] هل يمكن استعمال Path Traversal؟
[ ] هل يمكن إرسال destination خبيث؟
[ ] هل يمكن إجبار Agent على حذف ملف؟
[ ] هل يمكن الوصول إلى عمليات غير مسموحة؟
[ ] هل يمكن تنفيذ أوامر نظام عشوائية؟
[ ] لا يوجد Command Execution API مفتوح
[ ] لا توجد أسرار hardcoded
[ ] لا توجد صلاحيات غير ضرورية
```

---

## 8. Logging (الطلب §30)

**مطلوب — structured:**

```text
timestamp · level · component · device_id · job_id
operation · message · error_code · duration · bytes_transferred
```

**ممنوع تسجيله:**
```text
❌ كلمات مرور    ❌ Tokens    ❌ أسرار    ❌ بيانات اعتماد
```

---

## 9. مصادر (Microsoft الرسمية)

- [Windows Portable Devices](https://learn.microsoft.com/en-us/windows/win32/wpd_sdk/windows-portable-devices)
- [Session 0 Isolation](https://learn.microsoft.com/en-us/windows/win32/services/interactive-services)
- [Windows Services](https://learn.microsoft.com/en-us/windows/win32/services)
- [UAC](https://learn.microsoft.com/en-us/windows/security/application-security/application-control/user-account-control/how-it-works)
