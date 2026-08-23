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
| `--color-primary` | `#7C3AED` | اللون الأساسي (بنفسجي ملكي) |
| `--color-accent` | `#C026D3` | لون التمييز والتأكيد (فوشيا نيون) |
| `--color-secondary` | `#2563EB` | اللون الثانوي (أزرق كهربائي) |
| `--color-success` | `#10B981` | حالات النجاح والاتصال |
| `--color-warning` | `#F59E0B` | التنبيهات والتحذيرات |
| `--color-danger` | `#EF4444` | الأخطاء وعمليات الحذف |
| `--bg-base` | `#08070E` | الخلفية العامة للنظام |
| `--bg-card` | `#0E0C1A` | خلفية البطاقات والقوائم |
| `--border-default` | `rgba(255, 255, 255, 0.10)` | الحدود الافتراضية |

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
