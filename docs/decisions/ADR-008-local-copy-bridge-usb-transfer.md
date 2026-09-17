# ADR-008: Local Copy Bridge for USB Device Transfer

## Status

Accepted / Implemented

## Context

المستخدمون يحتاجون نسخ الأفلام مباشرة إلى أجهزة USB وهواتف Android (MTP) وآيفون (AFC) من واجهة الويب. المتصفح لا يستطيع الوصول إلى أجهزة USB، والسيرفر المركزي لا يمكنه رؤية الأجهزة الموصولة بجهاز العميل إلا إن كان كل عميل يشغّل نسخة سيرفر كاملة مع صلاحيات أجهزة — وهو ما يخالف فصل الأدوار الحالي (سيرفر مركزي = catalogue + streaming فقط).

كذلك يجب ألا يتحول السيرفر المركزي إلى نقطة عبور (proxy) تنقل ملف الفيديو كاملًا من التخزين إلى السيرفر ثم إلى العميل، لأن ذلك يضاعف I/O ويستهلك ذاكرة ويضيف زمن انتقال.

## Decision

تشغيل خدمة محلية مستقلة على جهاز العميل باسم **NEXORA Copy Bridge**:

- ثنائي Go منفصل `server/cmd/copybridge`، يُثبّت كخدمة Windows باسم `NEXORACopyBridge` عبر `scripts/install-bridge-service.bat`، ويستمع افتراضيًا على `127.0.0.1:32145` فقط.
- واجهة HTTP محلية: أجهزة، تصفح، إنشاء مجلد، إخراج (eject)، نسخ، مهام، و SSE للتحديثات الحية.
- النسخ **Zero-Spool Direct Stream** من `/api/stream/file/{id}` (HTTP Range) على السيرفر المركزي إلى الجهاز:
  - USB Storage و iOS (AFC): دفق مباشر من `io.Reader` إلى الوجهة دون تنزيل كامل.
  - Android (MTP Shell COM): قيد تقني — Shell API يتطلب ملفًا محليًا، فيُخزَّن الملف مؤقتًا ثم يُنسخ ثم يُحذف.
- ربط CORS مضيّق: loopback دائمًا مسموح، وعناوين LAN الخاصة افتراضيًا، ورفض مناشئ الإنترنت العامة بـ 403 ما لم تُضَف عبر `NEXORA_COPY_BRIDGE_CORS_ORIGIN`.
- `/api/transfer/*` في السيرفر المركزي أصبحت اختيارية/معطّلة افتراضيًا عبر `NEXORA_SERVER_USB_TRANSFER=true`، لأن الأجهزة محلية للعميل.
- البث عبر معرّف قاعدة البيانات (`/api/stream/file/{id}`) يستخدم مسار الكتالوج مباشرة (`serveCataloguePath`) دون إعادة تحقق `mediaPathAllowed`، لأن الكتالوج هو مصدر الحقيقة والملفات قد تكون على أقراص/مشاركات خارج `NEXORA_MEDIA_ROOTS`. أما البث بمسار من العميل (`/api/stream?path=`) فما زال يخضع للتحقق الصارم.

## Evidence from Current Code

- `server/cmd/copybridge/main.go` + `service_windows.go` / `service_other.go` — تشغيل كـ SCM service أو `-debug`.
- `server/internal/copybridge/config.go` — `LoadConfig`, `CORSOrigins`, `NEXORA_COPY_BRIDGE_*`.
- `server/internal/copybridge/server.go` — routes, `withMiddleware`, `corsAllow`, SSE.
- `server/internal/copybridge/service.go` — `StartCopy`, `runBridgeJob`, `streamRemoteToDevice`, `remoteTargetPath`.
- `server/internal/transfer/backend.go` — `PutStream` في واجهة `TransferBackend`.
- `server/internal/transfer/{storage_backend.go, android_backend.go, go_ios_backend.go}` — تنفيذ `PutStream` لكل خلفية.
- `server/internal/api/handlers_stream.go` — `serveCataloguePath` مقابل `serveMediaPath`.
- `client/src/lib/api.js` — `COPY_BRIDGE_BASE`, `requestBridgeJSON`, `getBridgeHealth`.
- `client/src/context/TransferContext.jsx` — SSE من الـ bridge و`bridgeOnline`.

## Consequences

- جهاز العميل يحتفظ بالوصول الحصري لأجهزته؛ لا يتسرب أي جهاز USB إلى السيرفر أو إلى أجهزة أخرى.
- ملف الفيديو يُقرأ مرة من المصدر ويُكتب مرة على الوجهة في مسار Storage/iOS.
- Android يستهلك مساحة مؤقتة بحجم الملف (قيد Shell COM موثّق).
- الخدمة تعمل افتراضيًا كـ `LocalSystem` (Session 0)؛ Android MTP قد يتطلب جلسة مستخدم تفاعلية في بعض إصدارات ويندوز.
- يجب أن يعرف المستخدم تشغيل الخدمة؛ الواجهة تعرض شريطًا توضيحيًا عند تعطّلها.

## What This Does Not Decide

- لا يعتمد token مشتركًا للأوامر الحساسة (`NEXORA_COPY_BRIDGE_TOKEN`) — مُؤجَّل بقرار المالك.
- لا يعتمد Tray Agent بدل الخدمة.
- لا يعتمد transcoding أو HLS/DASH في مسار النسخ؛ البث لا يزال direct HTTP Range (ADR-002).
- لا يغيّر model تخزين الوسائط (ADR-003) ولا مصدر الحقيقة (ADR-004).
