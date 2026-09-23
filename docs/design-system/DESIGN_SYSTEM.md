# 🎨 NEXORA Design System Documentation

> الفلسفة المعمارية: **"عدّل في مكان واحد، يتغير النظام كامل"** (Single Source of Truth).

---

## 🏛️ الهيكلية

```
client/src/
├── design-system/
│   ├── tokens.css       # المتغيرات الأساسية (Colors, Radius, Shadows, Transitions)
│   ├── index.css        # نقطة الدخول واستدعاء الثيمات
│   └── themes/
│       ├── dark.css     # الثيم الليلي الافتراضي
│       └── light.css    # الثيم النهاري
├── components/
│   └── ui/              # مكتبة المكونات الموحدة
│       ├── Button.jsx   # زر متعدد الأشكال والأحجام مع مؤشر التحميل
│       ├── Input.jsx    # حقول الإدخال، Textarea، و Select
│       ├── Badge.jsx    # وسوم الحالة والأنواع
│       ├── Card.jsx     # بطاقات GlassCard وبطاقات الإحصائيات MetricCard
│       ├── Modal.jsx    # النوافذ المنبثقة ونوافذ التأكيد ConfirmModal
│       ├── ProgressBar.jsx # أشرطة التقدم الذكية
│       ├── Spinner.jsx  # مؤشرات التحميل
│       └── index.js     # تصدير موحد (Barrel Export)
├── context/
│   └── ThemeContext.jsx # إدارة حالة الثيم وحفظها في localStorage
└── hooks/
    └── useTheme.js      # خطاف مخصص لاستخدام الثيم
```

---

## 🎯 متغيرات التصميم (Tokens)

جميع الألوان والقيم تستخدم كـ CSS Custom Properties:

| المتغير | القيمة الافتراضية (Dark) | الاستخدام |
|---------|-------------------------|-----------|
| `--color-primary` | `#2563EB` | اللون الأساسي (أزرق سينمائي) |
| `--color-accent` | `#F59E0B` | لون التمييز والتأكيد (ذهبي) |
| `--color-secondary` | `#0891B2` | اللون الثانوي (سماوي) |
| `--color-success` | `#10B981` | حالات النجاح والاتصال |
| `--color-warning` | `#F59E0B` | التنبيهات والتحذيرات |
| `--color-danger` | `#EF4444` | الأخطاء وعمليات الحذف |
| `--color-info` | `#22D3EE` | المعلومات والشارات التقنية |
| `--bg-base` | `#060A14` | الخلفية العامة للنظام (كحلي عميق) |
| `--bg-card` | `#0C1322` | خلفية البطاقات والقوائم |
| `--border-default` | `rgba(255, 255, 255, 0.20)` | الحدود الافتراضية |
| `--accent-rgb` | `245, 158, 11` | قنوات الـ accent لاستخدامها داخل `rgba()` |
| `--primary-rgb` | `37, 99, 235` | قنوات الـ primary لاستخدامها داخل `rgba()` |

> **ملاحظة:** لوحة الألوان الحالية (أزرق سينمائي + ذهبي) حلّت محلّ اللوحة السابقة
> (بنفسجي + فوشيا). كل القيم مُعرّفة في `design-system/tokens.css` فقط، ولا يجوز
> كتابة لون يدويًا في مكان آخر — استُبدلت 44 قيمة مكتوبة يدويًا في `styles.css`
> بالمتغيرات لهذا السبب.

---

## 🧩 استخدام المكونات الموحدة

```jsx
import { Button, Input, Card, Modal, Badge, ProgressBar } from "@/components/ui";

// زر مع حالة تحميل
<Button variant="primary" size="md" loading={isLoading} onClick={handleClick}>
  حفظ التعديلات
</Button>

// بطاقة زجاجية
<Card hover interactive className="p-6">
  <Badge variant="success">متصل</Badge>
  <h3>عنوان العمل</h3>
</Card>
```
