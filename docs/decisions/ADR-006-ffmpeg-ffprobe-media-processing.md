# ADR-006: FFmpeg and FFprobe as External Media Processing Tools

## Status

Existing / Verified

## Context

الفهرسة والجودة والـ timeline previews تحتاج information وعمليات media لا ينفذها React أو stream handler مباشرة.

## Decision

يستدعي Go FFprobe وFFmpeg كبرامج خارجية قابلة للضبط بالبيئة لinspection، verification، وتوليد thumbnails.

## Evidence from Current Code

- `server/internal/media/processor.go` يشغّل FFprobe JSON inspection وFFmpeg verification/thumbnail generation.
- `handleIndex` و`handleMediaInspect` يحفظان نتائج inspection.
- `handleFilePreview` يولد JPEG عند غياب cached bucket.
- streaming يستخدم `http.ServeContent` ولا يستدعي FFmpeg.

## Consequences

- media processing يعتمد على توفر executables على host.
- preview generation قد يستهلك CPU/disk في أول request لكل bucket.
- stream hot path يبقى direct file serving.

## What This Does Not Decide

- لا يعتمد FFmpeg كـ transcoder في current playback.
- لا يقرر HLS/DASH أو queue جديدة أو GPU acceleration.
- لا يثبت أن FFmpeg/FFprobe متوفران على كل deployment.
