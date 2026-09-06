# ADR-002: Direct HTTP Range Streaming

## Status

Existing / Verified

## Context

المشغل الأصلي يحتاج الوصول إلى ملف فيديو محلي مع seek دون تحميله كله مسبقًا.

## Decision

الخادم يقدم الملف مباشرة عبر HTTP، ويضع `Accept-Ranges: bytes` ثم يستخدم Go `http.ServeContent` على `*os.File`.

## Evidence from Current Code

- `server/internal/api/server.go`: `handleStream`, `handleStreamByID`, و`serveMediaPath`.
- `serveMediaPath` يستدعي `os.Open`, `file.Stat`, ثم `http.ServeContent`.
- لا يظهر HLS manifest أو transcoding pipeline في stream path.

## Consequences

- لا يحمل handler bytes الفيلم كاملة في memory قبل الاستجابة.
- `http.ServeContent` يتولى Range requests وpartial responses.
- browser يجب أن يدعم الملف الأصلي؛ لا يوجد adaptation/transcode في المسار الحالي.

## What This Does Not Decide

- لا يثبت قدرة محددة لعدد concurrent streams.
- لا يعتمد HLS/DASH أو ABR أو GPU transcoding.
- لا يحل codec incompatibility على أجهزة العملاء.
