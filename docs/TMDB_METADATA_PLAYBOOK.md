# دليل TMDB والبيانات الوصفية في NEXORA

> وثيقة تشغيلية للمطورين والـ agents. آخر مراجعة: 2026-08-16.
>
> الهدف: تحويل ملفات الوسائط المحلية المفهرسة إلى كتالوج غني يعمل دون اتصال في الاستخدام اليومي، مع مزامنة متحكم بها من TMDB عند الحاجة.

## 1. القرار المعماري

NEXORA هو **مالك بيانات الملفات المحلية**: المسار، الحجم، البصمة، جودة الملف، مسارات الصوت والترجمة وحالة التلف تأتي من الخادم المحلي فقط.

TMDB هو **مزود بيانات وصفية مرخّص ومؤقت**: العناوين، الملخصات، التصنيفات، الصور، الأشخاص، بيانات المواسم والحلقات، التقييمات، الاعتمادات، الفيديوهات الدعائية واللغات. لا يعيد TMDB ملف الفيديو نفسه أو ترجمة/مسار صوت الملف المحلي.

لا تتصل واجهة React بـ TMDB مباشرة. كل الاتصال يمر من Go (`internal/metadata`) ثم يخزّن في PostgreSQL والقرص. النتيجة: لا تُسرّب المفاتيح إلى المتصفحات، ويبقى الاستخدام اليومي محلياً حتى عند انقطاع الإنترنت.

```text
file scanner → parser → media_item/video_file
                          │
                          ▼
                    metadata job queue
                          │
                          ▼
                   TMDB client (Go فقط)
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
       PostgreSQL metadata        assets/images/tmdb
              │                       │
              └───────────┬───────────┘
                          ▼
                  local NEXORA API → UI/LAN clients
```

## 2. الامتثال قبل التشغيل

هذه ليست خطوة اختيارية:

- استخدم مفتاح القراءة من متغيرات البيئة فقط: `NEXORA_TMDB_BEARER_TOKEN` هو الخيار المفضل؛ `NEXORA_TMDB_API_KEY` مدعوم للتوافق. لا يُكتب أي مفتاح في Git أو المتصفح أو ملفات السجل.
- يجب أن تعرض صفحة **About / Credits** شعار TMDB المعتمد والنص التالي: `This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.`
- يُخزن كل سجل وملف صورة مصدره TMDB مع `fetched_at` و`expires_at`. الحد الأعلى للكاش هو **6 أشهر**؛ لا يعد NEXORA نسخة دائمة من TMDB.
- يُجلب فقط ما يتصل بعناصر مكتبة NEXORA أو ما يطلبه المسؤول صراحةً. لا تنفذ crawl شامل لقاعدة TMDB.
- إذا كان النظام يستخدم داخل مقهى/استراحة كمشروع ربحي أو يحقق دخلاً من تكامل TMDB، يلزم تأكيد الترخيص التجاري مع TMDB قبل الإنتاج.
- عند ظهور `429` يوقف العامل الطلبات مؤقتاً ويعيد المحاولة بـ exponential backoff مع jitter. لا يُفترض أن حد السرعة ثابت؛ الوثائق تشير إلى حدود تقريبية قرب 40 طلب/ثانية وتطلب احترام `429`.
- بيانات مزودي المشاهدة (JustWatch) اختيارية. لا تعرض إلا مع نسبة JustWatch المطلوبة وباستخدام رابط TMDB المعاد؛ لا تستخدمها كبَديل للبث المحلي.

المصادر الرسمية: [FAQ والنسبة المطلوبة](https://developer.themoviedb.org/docs/faq)، [شروط API](https://www.themoviedb.org/api-terms-of-use)، [حدود الطلبات](https://developer.themoviedb.org/docs/rate-limiting).

## 3. ما الذي يوفره TMDB فعلاً

| المجال | أمثلة البيانات | مكان العرض في NEXORA | أولوية الجلب |
| --- | --- | --- | --- |
| بحث ومطابقة | TMDB ID، العنوان، العنوان الأصلي، السنة، اللغة، الشعبية، نتيجة المطابقة | واجهة اختيار المطابقة للمسؤول | فوري |
| تفاصيل الفيلم/المسلسل | الملخص، الحالة، المدة، الأنواع، البلدان، شركات الإنتاج، الشبكات، التقييم وعدد الأصوات | صفحة التفاصيل | فوري |
| الترجمات | عناوين وملخصات TMDB بالعربية والإنجليزية واللغة الأصلية | صفحة التفاصيل والبحث | فوري للعربية/الإنجليزية؛ لغات أخرى عند الطلب |
| الصور | poster، backdrop، logo، profile، still؛ مع اللغة والأبعاد والتصويت | الواجهة السينمائية ولوحة الإدارة | فوري للصور الأساسية؛ البقية عند الطلب |
| الأشخاص والاعتمادات | ممثلون، شخصيات، مخرجون، كتّاب، صور الأشخاص | تبويب طاقم العمل | فوري لأعلى 20–30؛ الباقي عند الطلب |
| مواسم وحلقات TV | أرقام، أسماء، ملخصات، تاريخ بث، مدة، stills، ضيوف | شاشة المواسم والحلقات ومقارنة الملفات المحلية | فوري للمواسم الموجودة محلياً |
| فيديوهات TMDB | trailers/teasers/clips ومفتاح YouTube أو مزود آخر | تبويب مقاطع دعائية فقط | عند فتح التفاصيل |
| توصيات/مشابه | recommendations وsimilar | صفوف الاقتراحات، ولا تعني أن الملف موجود محلياً | عند فتح التفاصيل |
| اعتماد/تصنيف | certification، content ratings، release dates | بطاقات المحتوى والرقابة العائلية | فوري |
| معرّفات وكلمات مفتاحية | IMDb/TVDB/Wikidata/social IDs وkeywords | المطابقة والتحسين الإداري | فوري |
| توفر منصات خارجية | watch providers حسب البلد | اختياري، مع نسبة JustWatch | لا يدخل في MVP |

**حدود مهمة:** مفاتيح `videos` تخص المقاطع الدعائية، وليست ملفات الفيلم. ترجمات TMDB النصية هي ترجمة بيانات وصفية وليست ملفات `.srt/.ass` أو مسارات صوتية؛ هذه يكتشفها `internal/media` من الملف المحلي.

## 4. خريطة endpoints المعتمدة

الجذر `https://api.themoviedb.org/3`. جميع طلبات القراءة تستخدم `Authorization: Bearer <token>`؛ يمكن لـ v3 استخدام `api_key` أيضاً، لكن Bearer يوحد المصادقة بين v3 وv4.

| الهدف | endpoint |
| --- | --- |
| إعداد الصور واللغات والبلدان | `GET /configuration` و`/configuration/languages` و`/configuration/primary_translations` |
| بحث فيلم | `GET /search/movie?query=&year=&language=` |
| بحث مسلسل/أنمي | `GET /search/tv?query=&first_air_date_year=&language=` |
| مطابقة بمعرف خارجي | `GET /find/{external_id}?external_source=imdb_id` (أو المصدر المناسب) |
| تفاصيل فيلم | `GET /movie/{id}` |
| تفاصيل مسلسل | `GET /tv/{id}` |
| تفاصيل موسم | `GET /tv/{id}/season/{season_number}` |
| تفاصيل حلقة | `GET /tv/{id}/season/{season_number}/episode/{episode_number}` |
| نصوص مترجمة | `/movie/{id}/translations`، `/tv/{id}/translations`، وبالمثل للموسم والحلقة |
| الأصول | `/movie/{id}/images`، `/tv/{id}/images`، `/person/{id}/images`، والموسم/الحلقة |
| الاعتمادات | `/movie/{id}/credits`؛ للمسلسلات استخدم `/tv/{id}/aggregate_credits`؛ وللموسم `aggregate_credits` |
| بيانات مساعدة | `videos`, `external_ids`, `keywords`, `recommendations`, `similar`, `release_dates` (فيلم)، `content_ratings` (TV)، `watch/providers` |

تدعم تفاصيل الفيلم والمسلسل والموسم والحلقة والشخص `append_to_response`، حتى 20 طلباً فرعياً ضمن request واحد. الخطة الأساسية:

```text
movie/{id}?language=ar-SA&include_image_language=ar,en,null&
append_to_response=alternative_titles,credits,external_ids,images,keywords,recommendations,release_dates,reviews,similar,translations,videos

tv/{id}?language=ar-SA&include_image_language=ar,en,null&
append_to_response=aggregate_credits,images,videos,keywords,external_ids,recommendations,similar,content_ratings,translations
```

ثم نطلب `language=en-US` للتفاصيل نفسها إذا لم توجد العربية أو إذا أردنا حفظ اللغتين. يفضّل حفظ استجابة `translations` أيضاً لإتاحة لغة بديلة من دون طلب جديد.

### المواسم والحلقات المنفّذة

عند تحديث عنصر نوعه `series` أو `anime`، يستخرج NEXORA أرقام المواسم من تفاصيل الـ TV الإنجليزية ثم يجلب `GET /tv/{series_id}/season/{season_number}` لكل موسم باللغتين `en-US` و`ar-SA`، مع `aggregate_credits,credits,external_ids,images,translations,videos` في `append_to_response`. استجابة الموسم نفسها تضم `episodes`، لذلك تحفظ أسماء الحلقات وملخصاتها وتاريخ البث والمدد و`still_path` محلياً داخل `season_metadata_snapshots` من دون خلطها بملفات الفيديو المحلية.

- `GET /api/media/{id}/metadata/seasons?locale=ar-SA` يعيد النسخ المحلية للمواسم.
- صفحة التفاصيل تعطي العربية أولوية، ثم تستخدم الحقل الإنجليزي المقابل عند عدم توفرها.
- لا يعني وجود حلقة TMDB أن ملفاً قابلاً للتشغيل موجود في المكتبة؛ يظل التشغيل مقصوراً على `video_files` المحلية.

`watch/providers` لا يضم داخل الطلب أعلاه: يجلب NEXORA المورد المنفصل `GET /movie/{id}/watch/providers` ويحفظه تحت `watch_providers` داخل snapshot المحلي. لا تعرضه الواجهة إلا مع نسبة JustWatch ورابطه المعاد. الموارد التالية **ليست** بيانات كتالوج عامة يمكن تشغيلها بمفتاح التطبيق وحده: `account_states`، `add/delete rating` (تحتاج جلسة مستخدم)، و`latest` (فيلم عالمي وليس خاصاً بعنصر المكتبة). `changes` سجل تحديثات وليس بيانات عرض.

المراجع الرسمية: [تفاصيل الفيلم](https://developer.themoviedb.org/reference/movie-details)، [Append to response](https://developer.themoviedb.org/docs/append-to-response)، [صور اللغات](https://developer.themoviedb.org/docs/image-languages)، [تفاصيل الحلقة](https://developer.themoviedb.org/reference/tv-episode-details)، [مطابقة معرف خارجي](https://developer.themoviedb.org/reference/find-by-id).

## 5. استراتيجية المطابقة واللغات

1. يخرج `parser` عنواناً، سنة، نوعاً، رقم موسم/حلقة، وعنواناً عربياً/إنجليزياً إن أمكن.
2. نبحث بالنوع الصحيح أولاً (`search/movie` أو `search/tv`) وبالسنة إن كانت موثوقة.
3. نقيّم المرشحين: تطابق الاسم المطبّع، تقارب السنة، تطابق النوع، تطابق العنوان الأصلي/العربي، والشعبية كعامل أخير فقط.
4. إذا تجاوز المرشح درجة الثقة، يحفظ `tmdb_id` و`match_confidence` و`match_method=automatic`. إن كان غير حاسم، ينشأ `needs_review` ولا يكتب فوق بيانات المسؤول.
5. نطلب `ar-SA` أولاً؛ ثم `ar` أو `en-US`؛ ثم `original_language`. لا تستبدل قيمة غير فارغة من مسؤول النظام بترجمة TMDB تلقائياً.
6. في الصور: نطلب `include_image_language=ar,en,null`، ونرتب العربية ثم الإنجليزية ثم غير المعلّمة حسب `vote_average`/`vote_count`/الدقة.

## 6. تصميم التخزين المحلي المقترح

لا يكفي توسيع `media_items` بعشرات الأعمدة. تحفظ البيانات المتغيرة أو المتعددة في جداول منفصلة قابلة لإعادة المزامنة.

| جدول/مورد مقترح | مفاتيح مهمة | الغرض |
| --- | --- | --- |
| `metadata_sources` | `media_item_id`, `provider`, `external_id`, `raw_json`, `fetched_at`, `expires_at`, `etag`, `status` | المصدر، النسخة الخام، عمر الكاش وحالة المزامنة |
| `media_translations` | `media_item_id`, `locale`, `title`, `overview`, `tagline`, `is_fallback` | العربية/الإنجليزية/الأصلية |
| `media_assets` | `media_item_id`, `kind`, `locale`, `tmdb_path`, `local_path`, `width`, `height`, `sha256`, `fetched_at`, `expires_at` | poster/backdrop/logo/profile/still المحلية |
| `people` و`media_credits` | `tmdb_person_id`, الاسم/الصورة؛ `media_item_id`, الدور والترتيب | طاقم العمل وطاقم الإنتاج |
| `season_metadata` | `media_item_id`, `season_number`, `tmdb_id`, تواريخ/ملخص/صورة/حالة المزامنة | البيانات الوصفية للموسم |
| `episode_metadata` | `season_id`, `episode_number`, `tmdb_id`, اسم/ملخص/تاريخ/مدة/still | يربط TMDB بالحلقة المحلية |
| `metadata_relations` | `media_item_id`, `related_tmdb_id`, `relation_kind`, `sort_order` | recommendations/similar، وهي عناصر كتالوج وليست ملفات محلية |
| `metadata_jobs` | نوع المهمة، الهدف، المحاولة، `run_after`, الخطأ | صف مزامنة مستأنف ومحترم للحدود |

يبقى في `media_items` ملخص العرض السريع: `tmdb_id`، `metadata_provider`، العنوان/الملخص المعروض، السنة، التقييم، التصنيفات، `poster_path` و`banner_path` المحلية. لا يحفظ Bearer token أو `api_key` في قاعدة البيانات.

## 7. تخزين الصور ومدة الكاش

احفظ الصور تحت مسار حتمي، مثلاً:

```text
assets/images/tmdb/movie/550/poster/ar-SA/<sha256>.jpg
assets/images/tmdb/tv/1396/backdrop/null/<sha256>.jpg
assets/images/tmdb/person/287/profile/en/<sha256>.jpg
```

- احفظ `Content-Type`، الحجم، SHA-256، المسار الأصلي ووقت الجلب.
- حمّل الصورة إلى ملف مؤقت ثم `fsync` وrename ذري؛ لا تكتب مباشرة فوق أصل صالح.
- لا تحذف الأصل قبل نجاح النسخة الجديدة. عامل تنظيف يمسح الأصول المنتهية أو غير المرتبطة بعد 6 أشهر كحد أقصى.
- استخدام `GET /configuration` هو مصدر أحجام الصور وbase URL؛ لا تفترض `w500` أو `original` ثابتين.

## 8. سير العمل التشغيلي

### إثر فهرسة ملف جديد

1. تُنشئ الفهرسة `media_item`/`video_file` محلياً.
2. ينشئ النظام `metadata_jobs(kind=match)`، ولا يحظر مسح القرص أو المشاهدة.
3. العامل يطابق العمل، ثم يضع مهمة `hydrate` عند نجاحه.
4. `hydrate` يجلب التفاصيل والملحقات وصور العربية/الإنجليزية، يحفظ DB والأصول محلياً ضمن transaction منطقي.
5. يعيد فهرسة وثيقة Meilisearch من النسخة المحلية فقط.
6. عند الفشل: تحفظ حالة واضحة ومعلومات إعادة المحاولة؛ لا تمسح أي metadata سليمة سابقة.

### عند فتح صفحة التفاصيل

- تعرض قاعدة البيانات/القرص مباشرة.
- إن كان `expires_at` منتهياً، يرسل API مهمة refresh في الخلفية ثم يعيد النسخة المحفوظة حالاً (stale-while-revalidate).
- لا تنتظر TMDB في مسار واجهة العميل.

### جدول التجديد

| نوع البيانات | تحديث مقترح | أقصى احتفاظ |
| --- | --- | --- |
| configuration/languages | 30 يوماً | 6 أشهر |
| تفاصيل العمل والترجمات | 30 يوماً أو عند طلب المسؤول | 6 أشهر |
| الصور الأساسية | عند تغير `tmdb_path` أو 30–90 يوماً | 6 أشهر |
| مواسم وحلقات مستمرة | 24 ساعة أثناء الموسم، ثم 30 يوماً | 6 أشهر |
| recommendations/providers | 7 أيام إذا فُعّلت | 6 أشهر |

## 9. حدود التنفيذ الحالية وما يليها

الموجود الآن في `server/internal/metadata/tmdb.go` وواجهة التفاصيل:

- بحث إنجليزي مرجعي ثم جلب العربية بالـ TMDB ID نفسه، لذلك لا يختار البحث العربي عملاً مختلفاً بالاسم نفسه.
- فصل الكتابة إلى `title_en`/`plot_en` و`title_ar`/`plot_ar` بحسب locale؛ الواجهة لا تعرض نصاً صينياً أو لغة بديلة على أنه عربي.
- snapshot خام محلي لكل من `en-US` و`ar-SA` في `metadata_snapshots` مع TTL، ويُقرأ عبر `GET /api/media/{id}/metadata/raw?locale=...`.
- إسقاط إنجليزي مفهرس في `media_items.metadata_facets` (GIN JSONB): الأنواع، الكلمات المفتاحية، الشركات والبلدان، اللغات، المدة، الشعبية، العمر/الحالة والتقييمات. يستعمل هذا في الفلترة والفرز لاحقاً؛ أما المستند الإنجليزي الكامل فيبقى محفوظاً في `metadata_snapshots`.
- للفيلم: `alternative_titles`, `credits`, `external_ids`, `images`, `keywords`, `recommendations`, `release_dates`, `reviews`, `similar`, `translations`, `videos`، بالإضافة إلى `watch_providers` المنفصل.
- تنزيل poster وbackdrop محلياً، ومع snapshot الإنجليزية: أول 20 صورة profile للطاقم وأول 12 poster لكل من recommendations وsimilar. تحفظ الحقول المحلية (`local_profile_path` و`local_poster_path`) داخل JSON snapshot وتُعرض من `/assets/images/tmdb/...`; بيئة Vite تمرر `/assets` إلى خادم Go أيضاً.
- صفحة التفاصيل تعرض العنوان الإنجليزي أولاً وتحته العربية، ثم البطاقات الموسعة: طاقم العمل بصوره، المقاطع داخل iframe آمن من `youtube-nocookie.com`، العناوين البديلة، الترجمات، الإصدارات، شركات/بلدان الإنتاج، المعرّفات، المراجعات، والتوفر مع نسبة JustWatch. تُدمج فيديوهات العربية والإنجليزية مع إزالة التكرار لأن كثيراً من المقاطع لا تكون موجودة في locale العربية.
- اختبار `httptest` يحرس قائمة الموارد الموسعة وحفظ مزودي المشاهدة ضمن snapshot.

لا يزال مطلوباً للوصول إلى مزامنة إنتاجية كاملة:

1. مطابقة مرشحين بدرجة ثقة وواجهة اعتماد إداري بدلاً من اختيار نتيجة البحث الأولى.
2. عامل refresh مع backoff صريح لـ `429` وqueue قابلة للاستئناف.
3. جدول `media_assets` ومزامنة اختيارية لكل صور gallery/logo/profile، بدلاً من poster وbackdrop الأساسيين فقط.
4. ربط حلقات TMDB تلقائياً بملفات `video_files` المحلية بدرجة ثقة، بدلاً من إبقاء القائمتين منفصلتين كما هو الحال الآن.
5. شاشة Credits/About وشعار TMDB المعتمد قبل إطلاق نظام يستخدم محتوى TMDB.

## 10. قواعد غير قابلة للتنازل عنها للمطور التالي

- لا تكشف أي سر من `.env` في commit أو log أو استجابة HTTP.
- لا تجعل المتصفح يستدعي `api.themoviedb.org`.
- لا تخلط بين بيانات TMDB وبيانات ملفات الفيديو المحلية.
- لا تستبدل تعديلات المسؤول تلقائياً عند refresh.
- لا تنفذ bulk scrape، ولا تحتفظ بكاش TMDB أكثر من 6 أشهر.
- لا تعرض watch providers من دون نسبة JustWatch، ولا تعد المستخدم بأن TMDB يوفر ملفات فيديو أو ترجمات تشغيل.
- كل تغيير في schema أو endpoint يجب أن يحدّث هذه الوثيقة واختبارات الحزمة ذات الصلة.
