# ADR-001: Native HTML5 Video Player with Custom React Controls

## Status
Superseded by [ADR-016](ADR-016-videojs-player-shell.md)

> ملاحظة: هذا الـ ADR يوصف المسار التاريخي (HTML5 `<video>` + React controls في
> `VideoPlayer.jsx`). استُبدل هذا المشغل بـ Video.js داخل شل NEXORA في ADR-012،
> وأُزيل `VideoPlayer.jsx`. يبقى هذا السجل لأن ADR-002 ومسارات streaming ما زالت
> تبني عليه.

## Context

يحتاج العميل تشغيل ملفات وسائط من Go API داخل متصفح، مع واجهة تحكم خاصة بالمشروع.

## Decision

المشغل الحالي يستخدم عنصر HTML5 `<video>` الأصلي، وتبني React فوقه controls مخصصة بدل اعتماد مكتبة player runtime في المسار الحالي.

## Evidence from Current Code

- `client/src/components/VideoPlayer.jsx` ينشئ `<video>` و`<source>` و`<track>` مباشرة.
- المكوّن يستدعي browser APIs: `play`, `pause`, `requestFullscreen`, `requestPictureInPicture`, `textTracks`, و`playbackRate`.
- `client/package.json` يحتوي `plyr`، لكن لا يوجد import أو إنشاء له في `client/src` وقت التحقق.

## Consequences

- decoding and buffering المتوافقين تقع على browser.
- التحكم في التصميم والسلوك موجود في React/CSS داخل المشروع.
- توافق containers/codecs يعتمد على client browser/device.

## What This Does Not Decide

- لا يقرر إضافة أو إزالة Plyr dependency.
- لا يقرر HLS/DASH أو transcoding.
- لا يضمن دعم كل codecs أو الأجهزة.
