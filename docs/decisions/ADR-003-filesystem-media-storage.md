# ADR-003: Filesystem-Based Original Media Storage

## Status

Existing / Verified

## Context

المكتبة تتكون من ملفات فيديو موجودة على أقراص محلية/مرفقة، مع حاجة للفهرسة والبث والنقل دون نسخ bytes إلى database.

## Decision

تظل ملفات الفيديو الأصلية على filesystem؛ PostgreSQL يخزن path والبيانات الفنية والعلاقات فقط.

## Evidence from Current Code

- migration `0001_init_schema.sql` يعرف `video_files.file_path`, `file_size`, `duration`, codecs/tracks ولا يعرف BLOB للملف.
- `scanner` يمشي roots من `NEXORA_MEDIA_ROOTS` ويعيد `FileInfo` لمسارات فعلية.
- API streaming يفتح `file_path` بـ `os.Open`.

## Consequences

- يمكن إدارة مكتبة كبيرة دون تضخيم PostgreSQL بملفات الفيديو.
- صحة paths واتصال الأقراص تؤثر مباشرة على البث.
- authorization للوصول إلى paths مسؤولية Go backend، لا schema وحده.

## What This Does Not Decide

- لا يحدد topology الأقراص أو backup policy.
- لا يقر السلوك الأمني الحالي لـ `mediaPathAllowed` كسياسة نهائية.
- لا يقرر object storage أو CDN مستقبلًا.
