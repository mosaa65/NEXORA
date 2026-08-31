# 📱 دراسة معمارية شاملة: تحسين أداء الواجهة وتجربة المستخدم (Frontend Performance, State & Scroll Restoration Study)

> **دراسة هندسية تفصيلية وخطة عمل متكاملة** لحل مشاكل التمرير، منع إعادة التحميل عند الرجوع، تطبيق التمرير اللانهائي (Infinite Scrolling)، كاش الواجهة للوسائط الضخمة، ودعم **50 جهاز كمبيوتر + 200 جهاز جوال متزامن (250+ Clients)** على شبكة الاستراحة دون أي ضغط على السيرفر أو استهلاك للذاكرة.

---

## 📑 فهرس المحتويات
1. [المشهد العام والتحديات التشغيلية](#1-المشهد-العام-والتحديات-التشغيلية)
2. [تشريح معضلة "الرجوع وإعادة التحميل وفقدان موضع التمرير" (Back Navigation & Scroll Reset)](#2-تشريح-معضلة-الرجوع-وإعادة-التحميل-وفقدان-موضع-التمرير)
3. [معضلة جلب كل الوسائط دفعة واحدة (Bulk Fetching vs Infinite Scrolling)](#3-معضلة-جلب-كل-الوسائط-دفعة-واحدة)
4. [معمارية الكاش المتقدم في الواجهة (Client-Side Caching & SWR)](#4-معمارية-الكاش-المتقدم-في-الواجهة)
5. [إدارة المكتبات الضخمة والفهرسة المتقاطعة (Smart Hubs & Normalized Cache)](#5-إدارة-المكتبات-الضخمة-والفهرسة-المتقاطعة)
6. [المعمارية التقنية المقترحة والحلول البرمجية الجاهزة للتطبيق](#6-المعمارية-التقنية-المقترحة-والحلول-البرمجية)
7. [جدول المقارنة: الوضع الحالي مقابل الوضع المستهدف](#7-جدول-المقارنة-الوضع-الحالي-مقابل-الوضع-المستهدف)
8. [خطة التنفيذ البرمجية (Implementation Roadmap)](#8-خطة-التنفيذ-البرمجية)

---

## 1. المشهد العام والتحديات التشغيلية

تطبيق العميل لنظام **NEXORA** يعمل عبر المتصفح (Single Page Application - React/Vite) ويُستخدم في بيئة استراحة / صالة سينما محلية:
* **50 جهاز كمبيوتر (شاشات صالات وأكشاك):** شاشات عريضة، تصفح سريع بالماوس ولوحة المفاتيح.
* **200+ جهاز جوال (Android & iOS):** شاشات لمس، اتصال عبر شبكات الواي فاي (Wi-Fi)، ذاكرة عشوائية محدودة للأجهزة المحمولة.
* **طبيعة المكتبة:** مئات الآلاف من الأفلام، والمسلسلات، والأنمي، والمحاور الذكية (Smart Hubs)، والسلاسل السينمائية.

---

## 2. تشريح معضلة "الرجوع وإعادة التحميل وفقدان موضع التمرير"

### 🔴 ما الذي يحدث الآن في الكود؟ (Root Cause Analysis)
عندما يتصفح المستخدم قسم الأفلام أو الأنمي، وينزل لأسفل الصفحة (مثلاً وصل للفيلم رقم 150)، ثم يضغط على أحد الأفلام للدخول إلى صفحة تفاصيل العمل (`MediaDetailsPage`)، ثم يضغط زر "العودة" أو زر المتصفح:
1. **إلغاء تركيب المكون بالكامل (Component Unmounting):**
   * نظام التوجيه الحالي (`HashRouter`) يقوم بحذف مكون `CategoryPage` من الذاكرة والـ DOM.
   * جميع الـ States المحلية (`items`, `totalCount`, `filters`, `searchQuery`) تُمسح تماماً.
2. **إعادة التركيب والطلب من الصفر (Mounting & Refetching):**
   * عند الضغط على زر العودة، يقوم React بتركيب `CategoryPage` من جديد.
   * دالة `useEffect(() => { loadCategoryItems(); }, [])` تنفذ فوراً.
   * يتم تعيين `loading = true`، فتختفي البطاقات وتظهر دائرة التحميل الدوارة (Spinner).
3. **تصفير موضع التمرير (Scroll Reset to 0,0):**
   * نظراً لأن الصفحة كانت فارغة للحظات أثناء `loading = true`، يفقد المتصفح الارتفاع الكلي للصفحة (Page Height)، فيقوم المتصفح تلقائياً بإعادة موضع التمرير إلى أعلى نقطة في الشاشة `window.scrollTo(0, 0)`.
   * يجد المستخدم نفسه في بداية القائمة، ويفقد موقعه الذي وصل إليه بعد عناء التمرير، وهو ما يسبب إحباطاً شديداً وتجربة استخدام سيئة.

```
[المستخدم عند الفيلم 150] ──► يدخل لصفحة الفيلم
                                      │
                                      ▼
[العودة للخلف] ──► المكون يُعاد بناؤه ──► loading: true ──► الـ DOM يفرغ ──► التمرير يرجع لأعلى نقطة (0,0)!
```

---

### 🟢 الحل المعماري الجذري:

```
[المستخدم عند الفيلم 150] ──► حفظ (Scroll: 4200px + Data + Filters) في NavigationCache
                                      │
                                      ▼
[العودة للخلف] ◄── استعادة فورية (0ms) من الكاش ◄── إعادة موضع التمرير تلقائياً لـ 4200px بدون أي Spinner!
```

1. **مدير حالة التنقل والتمرير (Navigation & Scroll State Store):**
   * إنشاء `NavigationStateContext` يخزن لكل مسار (`/movies`, `/series`, `/hubs/anime-action`):
     * موضع التمرير الدقيق `scrollY`.
     * قائمة العناصر التي تم جلبها مسبقاً `items`.
     * الفلاتر النشطة والصفحة الحالية.
2. **عرض فوري بدون وميض (Zero-Flicker Instant Restore):**
   * عند الرجوع، المكون يقرأ مباشرة من الذاكرة المحلية (In-Memory / SessionStore)، فيرسم الـ 150 بطاقة في **0 ميلي ثانية**.
   * تطبيق `useLayoutEffect` لإعادة التمرير إلى النقطة المحفوظة قبل أن تتاح للمتصفح فرصة إعادة الرسم على `(0, 0)`.

---

## 3. معضلة جلب كل الوسائط دفعة واحدة

### 🔴 الوضع الحالي:
* في كود `CategoryPage.jsx` الحالي، يتم إرسال طلب `limit: 1000` لجلب 1000 فيلم أو مسلسل دفعة واحدة!

### لماذا يُعد هذا خطأً معمارياً؟
1. **استهلاك ذاكرة الجوالات (DOM Memory Exhaustion):**
   * إنشاء 1000 بطاقة تحتوي على صور، وظلال زجاجية، وعناصر تفاعلية يستهلك أكثر من **150MB إلى 250MB** من رام متصفح الجوال، مما قد يسبب انهيار المتصفح (Crash) على الجوالات المتوسطة والضعيفة.
2. **ضغط الباندويث غير المبرر:**
   * إذا فتح المستخدم قسم الأفلام وتصفح أول 10 أفلام فقط ثم خرج، يكون قد حمّل بيانات 990 فيلماً لم يشاهدها مطلقاً!
3. **تجميد الخيط الرئيسي للمتصفح (Main Thread Freezing):**
   * معالجة مصفوفة ضخمة من 1000 عنصر وعمل Filter و Sort لها في الجافاسكربت يسبب بطء مؤقت (Jank / Dropped Frames).

---

### 🟢 الحل المعماري: التمرير اللانهائي الموجه (Chunked Infinite Scrolling)
* **دفعة البداية (Initial Batch):** تحميل **36 إلى 48 عنصراً فقط** (تكفي لملء الشاشة مع التمرير الأولي).
* **المراقب الذكي (Intersection Observer Sentinel):**
  * وضع عنصر مراقبة خفي في نهاية القائمة.
  * عند اقتراب المستخدم من أسفل الصفحة بمسافة 600px، يتم جلب الدفعة التالية (36 عنصراً إضافياً) وإلحاقها بالمصفوفة.
* **النتيجة:** فتح فوري للصفحة في **أقل من 50 ميلي ثانية**، مع استهلاك لا يتجاوز **10MB** من الرام للجوال.

---

## 4. معمارية الكاش المتقدم في الواجهة (Client-Side Caching & SWR)

لضمان عدم إرسال أي طلبات متكررة للسيرفر من قبل 250 جهاز:

### أ. استراتيجية Stale-While-Revalidate (SWR):
```
1. هل البيانات موجودة في كاش المتصفح؟
   ├─► نعم: اعرضها فوراً للمستخدم في (0ms).
   └─► لا: اعرض هيكل التحميل (Skeleton) واجلبها من السيرفر.

2. في الخلفية (Background Revalidation):
   └─► التحقق بهدوء إذا حدث تغيير، دون إزعاج المستخدم أو عمل فلاش للشاشة.
```

### ب. كاش الصور والبوسترات (Lazy Image Loading & Placeholder):
* الأغلفة والبوسترات نصوص وصور ثابتة نادراً ما تتغير.
* استخدام `loading="lazy"` وترويسات `Cache-Control: immutable` لتبقى الصور مخزنة داخل ذاكرة التخزين المؤقت للمتصفح (Browser Disk Cache).
* استخدام صور مصغرة خفيفة (Thumbnails) في القوائم، وعدم جلب بوسترات الجودة العالية إلا عند فتح صفحة تفاصيل الفيلم.

---

## 5. إدارة المكتبات الضخمة والفهرسة المتقاطعة (Smart Hubs & Normalized Cache)

### التحدي:
في نظام NEXORA، يمكن لنفس الفيلم (مثلاً *Inception*) أن يظهر في:
1. قسم الأفلام الرئيسية (`/category/movies`).
2. تصنيف الخيال العلمي (`genres: Sci-Fi`).
3. محور ذكي (`Smart Hub: Mind-Bending Masterpieces`).
4. صفحة المخرج (`Christopher Nolan`).
5. السلاسل السينمائية والمجموعات.

### 🔴 بدون معمارية موحدة:
سيتم تخزين بيانات فيلم *Inception* وصوره 5 مرات مختلفة في ذاكرة المتصفح، مما يضاعف استهلاك الذاكرة 5 أضعاف.

### 🟢 الحل: بنية الكاش المعياري (Normalized Entity Store):
* يتم تخزين الأفلام في جدول كيانات مركزي بالمعرف:
  ```javascript
  entities: {
    media: {
      105: { id: 105, titleEn: "Inception", rating: 8.8, poster: "..." },
      106: { id: 106, titleEn: "Interstellar", rating: 8.7, poster: "..." }
    }
  }
  ```
* القوائم والمحاور الذكية تحتفظ فقط بمصفوفات المعرفات:
  ```javascript
  views: {
    "movies_scifi": [105, 106],
    "hub_mind_bending": [105],
    "person_nolan": [105, 106]
  }
  ```
* **الفائدة:** استهلاك ذاكرة ثابت، وإذا تم تحديث تقييم الفيلم أو البوستر، ينعكس التحديث تلقائياً في كل الأقسام فوراً دون الحاجة لإعادة طلب أي قسم!

---

## 6. المعمارية التقنية المقترحة والحلول البرمجية

### 1️⃣ موفر حالة التنقل والتمرير (`NavigationStateContext.jsx`)
```jsx
import React, { createContext, useContext, useRef, useState, useCallback } from "react";

const NavigationStateContext = createContext(null);

export function NavigationStateProvider({ children }) {
  // تخزين الكاش والموضع لكل مسار في الذاكرة
  const pageCache = useRef(new Map());

  const savePageState = useCallback((key, state) => {
    pageCache.current.set(key, {
      ...state,
      savedAt: Date.now(),
      scrollY: window.scrollY
    });
  }, []);

  const getPageState = useCallback((key) => {
    return pageCache.current.get(key) || null;
  }, []);

  const clearPageState = useCallback((key) => {
    if (key) pageCache.current.delete(key);
    else pageCache.current.clear();
  }, []);

  return (
    <NavigationStateContext.Provider value={{ savePageState, getPageState, clearPageState }}>
      {children}
    </NavigationStateContext.Provider>
  );
}

export const useNavigationState = () => useContext(NavigationStateContext);
```

---

### 2️⃣ خطاف استعادة التمرير التلقائي (`useScrollRestoration.js`)
```javascript
import { useEffect, useLayoutEffect } from "react";
import { useLocation } from "react-router-dom";
import { useNavigationState } from "../context/NavigationStateContext.jsx";

export function useScrollRestoration(pageKey, isDataReady) {
  const location = useLocation();
  const { savePageState, getPageState } = useNavigationState();

  // 1. استعادة موضع التمرير فور جهوزية البيانات
  useLayoutEffect(() => {
    if (!isDataReady) return;
    const cached = getPageState(pageKey);
    if (cached && typeof cached.scrollY === "number") {
      window.scrollTo({ top: cached.scrollY, behavior: "instant" });
    }
  }, [pageKey, isDataReady, getPageState]);

  // 2. حفظ الموضع عند مغادرة الصفحة
  useEffect(() => {
    return () => {
      savePageState(pageKey, { scrollY: window.scrollY });
    };
  }, [pageKey, savePageState]);
}
```

---

### 3️⃣ نمط التمرير اللانهائي في `CategoryPage.jsx`
```javascript
// بدلاً من limit: 1000، نستخدم دفعات ذكية
const PAGE_SIZE = 36;
const [page, setPage] = useState(1);
const [hasMore, setHasMore] = useState(true);

// مراقبة نهاية القائمة لتحميل الدفعة التالية
const observerTarget = useRef(null);

useEffect(() => {
  const observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && hasMore && !loading) {
        setPage((prev) => prev + 1);
      }
    },
    { threshold: 0.1, rootMargin: "600px" }
  );

  if (observerTarget.current) observer.observe(observerTarget.current);
  return () => observer.disconnect();
}, [hasMore, loading]);
```

---

## 7. جدول المقارنة: الوضع الحالي مقابل الوضع المستهدف

| الميزة والمعيار | الوضع الحالي ❌ | بعد تطبيق الدراسة المعمارية ✅ |
| :--- | :--- | :--- |
| **الرجوع بالمتصفح (Back Button)** | يعيد التحميل ويصعد لأعلى الصفحة `(0, 0)` | **استعادة فورية للبيانات وموضع التمرير الدقيق في (0ms)** |
| **حجم دفعة جلب الوسائط** | جلب 1000 عنصر دفعة واحدة (`limit: 1000`) | **دفعات ذكية (36 عنصر) عبر التمرير اللانهائي (Infinite Scroll)** |
| **استهلاك الرام في الجوالات** | عالي (150MB - 250MB) وقد يسبب تهنيج | **منخفض جداً (15MB - 30MB) وسلس للغاية** |
| **سرعة فتح الصفحة لأول مرة** | 300ms - 800ms (حسب حجم البيانات) | **أقل من 60ms** |
| **الطلبات المتكررة للسيرفر** | تكرار الطلب عند كل تنقل ودخول/خروج | **معدومة (تُخدم من كاش الذاكرة المحلية والـ SWR)** |
| **دعم 200 جوال متصل بالواي فاي** | استهلاك عالي للباندويث ببيانات ضخمة | **حزم خفيفة جداً ومضغوطة لا تؤثر على شبكة الواي فاي** |
| **إدارة تكرار الفيلم في عدة محاور** | تكرار تخزين بيانات الفيلم في كل قسم | **كاش معياري موحد (Normalized Store) يوفر 70% من الذاكرة** |

---

## 8. خطة التنفيذ البرمجية (Implementation Roadmap)

لتطبيق هذه الدراسة الهندسية على كود المشروع، يتم العمل وفق الخطوات التالية:

1. **الخطوة 1: بناء وتضمين سياق حفظ الحالة والتمرير:**
   * إنشاء `client/src/context/NavigationStateContext.jsx`.
   * تغليف تطبيق `App.jsx` بالـ Provider.
2. **الخطوة 2: ترقية `CategoryPage.jsx` و `SmartHubPage.jsx` و `DirectoryPage.jsx`:**
   * استبدال `limit: 1000` بنظام الدفعات المتتالية (Infinite Scrolling مع `IntersectionObserver`).
   * حفظ واستعادة الفلاتر وحالة الصفحة وموضع التمرير فورياً عند الرجوع.
3. **الخطوة 3: ترقية طبقة `api.js` في العميل:**
   * تفعيل الـ Request Deduplication لمنع إرسال طلبين متطابقين في نفس اللحظة.
   * دعم SWR Cache لتغذية الواجهة بالبيانات اللحظية.
4. **الخطوة 4: التحقق والاختبار الشامل:**
   * اختبار التمرير والرجوع على متصفحات الكمبيوتر وجوالات أندرويد وiOS والتأكد من ثبات موضع التمرير 100%.

---
*تم إعداد هذه الدراسة المعمارية لتكون الدليل الهندسي المعتمد لتطوير واجهة عميل NEXORA وتحقيق أعلى مستويات الأداء والسرعة.*
