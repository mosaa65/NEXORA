# 🤝 إرشادات المساهمة في مشروع NEXORA

> دليل شامل للمطورين الجدد والفريق للعمل على المشروع بانسجام.

---

## 🚀 كيف تبدأ

1. **استنسخ المشروع**:
   ```bash
   git clone <repo-url>
   cd NEXORA
   ```

2. **اقرأ دليل التشغيل**: [`docs/guides/SETUP_AND_RUN.md`](docs/guides/SETUP_AND_RUN.md)

3. **تعرف على هيكل المشروع**:
   ```
   NEXORA/
   ├── client/          → واجهة React + Vite + TailwindCSS
   ├── server/          → خادم Go (REST API)
   ├── docs/            → التوثيق المعماري والأدلة
   ├── .agents/         → قواعد للذكاء الاصطناعي
   └── _archive/        → ملفات تاريخية مرجعية
   ```

---

## 🌿 قواعد الفروع (Branching)

| النمط | الاستخدام | مثال |
|-------|----------|------|
| `feature/<name>` | ميزة جديدة | `feature/theme-switcher` |
| `fix/<name>` | إصلاح خلل | `fix/admin-login-redirect` |
| `docs/<name>` | تحديث توثيق | `docs/api-endpoints` |
| `refactor/<name>` | إعادة هيكلة | `refactor/split-admin-page` |
| `chore/<name>` | مهام صيانة | `chore/update-deps` |

**القاعدة**: لا تدفع مباشرة إلى `main`. افتح Pull Request دائماً.

---

## 💬 قواعد رسائل الـ Commit

اتبع نمط [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]
```

**الأنواع المتاحة**:

| النوع | الاستخدام |
|-------|----------|
| `feat` | ميزة جديدة |
| `fix` | إصلاح خلل |
| `docs` | تحديث توثيق |
| `style` | تنسيق كود (لا تغيير وظيفي) |
| `refactor` | إعادة هيكلة (لا ميزة جديدة ولا إصلاح) |
| `test` | إضافة أو تعديل اختبارات |
| `chore` | صيانة عامة |

**أمثلة**:
```
feat(admin): add disk usage monitoring page
fix(player): resolve subtitle loading on LAN stream
docs(tmdb): update integration guide with rate limiting
refactor(ui): extract Button component from AdminPage
```

---

## 🎨 قواعد التصميم والكود

### الواجهة (Frontend)
- **استخدم Design Tokens حصرياً**: لا تكتب ألواناً مباشرة مثل `#7C3AED` — استخدم `var(--color-primary)`
- **استخدم مكونات UI الموحدة**: استورد من `components/ui/` بدل كتابة أزرار وبطاقات من الصفر
- **لا ملفات عملاقة**: إذا تجاوز المكون 300 سطر، قسّمه إلى مكونات أصغر
- **التعليقات بالإنجليزية**: كتابة أسماء المتغيرات والتعليقات بالإنجليزية
- **محتوى الواجهة بالعربية**: النصوص المعروضة للمستخدم بالعربية (RTL)

### الخلفية (Backend)
- **اتبع Go conventions**: `gofmt`, `golint`, أسماء واضحة
- **خطأ = تسجيل**: كل خطأ يجب تسجيله (log) وإرجاعه بشكل مناسب

---

## 📝 مراجعة الكود (Code Review)

قبل دمج أي PR، تأكد من:
- [ ] الكود يبني بنجاح (`npm run build` للواجهة، `go build` للخادم)
- [ ] لا يوجد ألوان مباشرة أو أنماط مخالفة للـ Design System
- [ ] رسالة الـ commit تتبع النمط المحدد
- [ ] لا ملفات مؤقتة أو شخصية مرفوعة

---

## ❓ أسئلة؟

إذا واجهت أي مشكلة:
1. اقرأ التوثيق في مجلد `docs/`
2. تحقق من `.agents/rules/NEXORA.md` لقواعد التطوير
3. افتح Issue على GitHub مع وصف واضح
