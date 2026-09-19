# NEXORA Architecture Decision Records

هذا المجلد يسجل قرارات موجودة ويمكن إثباتها من الكود الحالي. لا يعني `Existing / Verified` أن مالك المشروع أصدر قرارًا رسميًا جديدًا؛ بل يعني أن التنفيذ الحالي يثبت وجوده.

| ADR | العنوان | الحالة |
|---|---|---|
| [ADR-001](ADR-001-native-html5-video-player.md) | Native HTML5 Video Player with Custom React Controls | Existing / Verified |
| [ADR-002](ADR-002-direct-http-range-streaming.md) | Direct HTTP Range Streaming | Existing / Verified |
| [ADR-003](ADR-003-filesystem-media-storage.md) | Filesystem-Based Media Storage | Existing / Verified |
| [ADR-004](ADR-004-postgresql-catalogue-source-of-truth.md) | PostgreSQL as Catalogue Source of Truth | Existing / Verified |
| [ADR-005](ADR-005-meilisearch-derived-search-index.md) | Meilisearch as Derived Search Index | Existing / Verified |
| [ADR-006](ADR-006-ffmpeg-ffprobe-media-processing.md) | FFmpeg and FFprobe as External Media Processing Tools | Existing / Verified |
| [ADR-007](ADR-007-provider-id-related-titles.md) | Provider-ID Related Titles Graph | Accepted / Implemented |
| [ADR-008](ADR-008-local-copy-bridge-usb-transfer.md) | Local Copy Bridge for USB Device Transfer | Accepted / Implemented |
| [ADR-009](ADR-009-incremental-indexing-pipeline.md) | Incremental, Fault-Tolerant Media Indexing Pipeline | Accepted / Implemented |
| [ADR-010](ADR-010-logical-media-model.md) | Logical Media Model and Entity Resolution | Accepted / Implemented |
| [ADR-011](ADR-011-search-projection-and-scan-control.md) | Rebuildable Search Projection and Cooperative Scan Control | Accepted / Implemented |

أي اقتراح غير منفذ (مثل HLS/DASH أو transcoding أو Redis integration) لا يوثق هنا كـ ADR قائم حتى توجد موافقة وتنفيذ قابل للتحقق.
