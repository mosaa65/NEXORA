# ADR-004: PostgreSQL as Catalogue Source of Truth

## Status

Existing / Verified

## Context

التطبيق يحتاج بيانات دائمة عن الأعمال، المواسم، الملفات، العلاقات، snapshots، settings، وعمليات TMDB queue.

## Decision

PostgreSQL هو المصدر الدائم/authoritative للكتالوج وmetadata التطبيقية، ويصل إليه Go عبر repository.

## Evidence from Current Code

- `internal/app/app.go` يفتح PostgreSQL ويشغل migrations عند startup.
- `internal/db/repository.go` يضم عمليات ingest/read/update للكتالوج والملفات والـ metadata.
- migrations تنشئ `media_items`, `seasons`, `video_files`, snapshots, settings, relations, queue.
- search sync يقرأ documents من repository قبل إرسالها إلى Meilisearch.

## Consequences

- data model والعلاقات والـ settings persistent ومُهاجرة عبر SQL migrations.
- Meilisearch وasset cache لا يجب أن يصبحا المصدر الوحيد لبيانات العمل.
- browser لا يتصل بقاعدة البيانات مباشرة.

## What This Does Not Decide

- لا يقرر multi-region replication أو backup/restore policy.
- لا يجعل PostgreSQL مخزنًا لbytes الفيديو أو preview images.
- لا يحدد schema مستقبلية جديدة.
