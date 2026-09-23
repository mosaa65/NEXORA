# ADR-016: Video.js as the Player Engine Behind a NEXORA Control Shell

## Status

Accepted / Implemented

## Context

المشغل السابق كان عنصر HTML5 `<video>` أصلًا مع controls مبنية بالكامل في React
داخل `client/src/components/VideoPlayer.jsx` (انظر ADR-001). هذا المسار كان يعمل،
لكن كل سلوك player (تقدّم، hotkeys، captions، timeline، fullscreen) كان مكتوبًا
يدويًا في المشروع، ما يُصعّب دعمًا مستقبليًا لمسارات صوت متعددة أو مصادر streaming
مختلفة.

تم اختيار Video.js كمحرّك playback لأنه يوفر: captions menu، playbackRates،
fullscreen/PiP APIs، وحلقة أحداث موحّدة (`play`/`pause`/`timeupdate`/…) فوق عنصر
فيديو HTML5 عادي، دون فرض transcoding أو HLS/DASH على المسار الحالي.

## Decision

- `client/src/components/NexoraPlayer.jsx` هو المشغل الوحيد المستخدم في التطبيق.
- Video.js يشغّل عنصر الفيديو وأحداث playback، وNEXORA تحتفظ بواجهة تحكم مخصّصة
  فوقه (مركز التشغيل ±10 ثوانٍ، شريط NEXORA، الشريط الزمني بمعاينة الصورة، تلميح
  المتابعة، درج الحلقات في fullscreen).
- شريط تحكم Video.js المدمج وbig-play معطّلان (`controls: false`, `bigPlayButton: false`, `controlBar: false`)، وHotkeys معطّلة أيضًا (`userActions.hotkeys = false`)، لأن NEXORA ترسم واجهة تحكم واحدة فقط وتعرّف خريطتها الخاصة (Space/K، ←/J، →/L، M، F، C). ترك الشريطين معًا كان يكرّر كل زر ويراكب شريطي تقدم.
- Fullscreen يُطلب على جذر NEXORA (`.nexora-vjs`)، لا على عنصر Video.js الداخلي، حتى تبقى طبقات NEXORA (الشريط، المركز، درج الحلقات) ظاهرة في fullscreen.
- عقود الـ props بقيت كما هي (`src`, `title`, `poster`, `tracks`, `fileId`,
  `onNext`, `playlist`, `currentFileId`, `onSelectFile`)، فالنافذة
  `RealVideoPlayerModal` في `App.jsx` لم تحتج أي تغيير في منطق البيانات.
- المشغل القديم `VideoPlayer.jsx` وأثر `plyr` غير المستخدم أُزيلا.

## Evidence from Current Code
- `client/src/App.jsx` يعرّف مسار `watch/:id` **داخل `CustomerCinemaLayout`** (فيحتفظ بشريط البحث والقائمة الجانبية)، و`handleQuickPlay` ينقل إليه بدل فتح نافذة عائمة.
- `client/src/pages/WatchPage.jsx` هو شاشة التشغيل: الحلقات على اليمين والفيديو على اليسار، وملخص واسم الحلقة أسفل الفيديو. زر «تصغير» يحوّل الفيديو إلى نافذة عائمة قابلة للتحريك (`nexora-mini`) دون مغادرة الصفحة.
- `client/src/components/PlayableFilesExplorer.jsx` (صفحة التفاصيل) يعرض لكل حلقة زرّي «تفاصيل» (`onShowDetails`) و«مشاهدة» (`onWatch`). «مشاهدة» ينتقل إلى `/watch/:id?file=…&play=fs` للتشغيل فورًا ملء الشاشة.
- `client/src/context/PlaybackContext.jsx` + `client/src/components/MiniPlayerDock.jsx` يحفظان الفيديو المصغّر فوق الـ Router، فيبقى شغّالًا عند التنقل بين الصفحات (`autoResume` في `NexoraPlayer` يُكمل من نفس الموضع).
- أزرار تصغير/خروج المسرح أثناء fullscreen تظهر أعلى يمين الفيديو عبر `nexora-fs-chrome` داخل `NexoraPlayer`.
- `client/src/components/NexoraPlayer.jsx` يقبل `fullscreenTarget` (عنصر الـ fullscreen) و`onMinimize`/`onExit` اختياريين.
- `client/src/components/NexoraPlayer.jsx` ينشئ `videojs(...)` بـ `controls:false`،
  يترجم `tracks` إلى `addRemoteTextTrack`، ويرسم الشريط/الشريط الزمني/المركز في
  React، ويحفظ التقدّم في `localStorage` بمفتاح `nexora:playback:{fileId|src}`
  (نفس صيغة المشغل السابق).
- `client/src/pages/DevPlayerPage.jsx` هو harness تطوير على المسار `/dev/player`
  يشغّل `NexoraPlayer` مقابل ملف حقي من الكتالوج.
- `client/package.json` يحتوي `video.js` ولم يعد يحتوي `plyr`.

## Consequences

- captions تُعرض عبر Video.js text tracks، وزر CC في شريط NEXORA يدوّر بين
  `showing`/`disabled` فوق نفس القائمة.
- التقدّم ما زال محليًا في المتصفح (localStorage) وليس في PostgreSQL.
- decoding/buffering ما زال على المتصفح؛ لا transcoding ولا HLS/DASH.
- حزمة العميل تتضمن `video.js` (chunk `player`).

## What This Does Not Decide

- لا يقرر HLS/DASH أو transcoding أو معالجة صوت/جودة متعددة.
- لا ينقل حفظ التقدّم إلى الخادم.
- لا يستبدل direct HTTP Range streaming (انظر ADR-002).
