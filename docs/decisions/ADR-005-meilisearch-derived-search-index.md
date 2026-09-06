# ADR-005: Meilisearch as Derived Search Index

## Status

Existing / Verified

## Context

الكتالوج يحتاج text search سريعًا على حقول عربية وإنجليزية وتصنيفات وفلاتر.

## Decision

يستخدم Go `search.Client` Meilisearch كمؤشر بحث مشتق من `MediaDocument` صادر من PostgreSQL.

## Evidence from Current Code

- `server/internal/search/client.go` ينشئ/configures index ويرسل documents ويبحث عبر REST.
- `handleIndex` يستدعي `repository.ListSearchDocuments` ثم `search.IndexDocuments`.
- `handleSearch` يمرر query إلى `search.SearchDocuments`.

## Consequences

- البحث السريع منفصل عن SQL catalogue reads.
- فشل sync يمكن أن يسبب search index متأخرًا، لكن لا يحذف بيانات PostgreSQL.
- index قابل لإعادة البناء من documents المصدرية.

## What This Does Not Decide

- لا يقرر أن كل catalogue screen يجب أن يستعمل Meilisearch.
- لا يثبت eventual-consistency SLA أو سياسة retry كاملة.
- لا يقرر Redis كطبقة cache للبحث.
