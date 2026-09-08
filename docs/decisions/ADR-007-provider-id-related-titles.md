# ADR-007: Provider-ID Related Titles Graph

## Status

Accepted / Implemented

## Context

TMDB returns recommendations and similar titles as part of the metadata detail document. Titles are translated, can be duplicated, and must not be used to decide whether a related title exists in NEXORA's local library.

## Decision

NEXORA persists related-title references in `media_related_titles`, keyed by provider, TMDB target ID, target kind, source media item, and relation type. The read endpoint matches candidates to local `media_items` by `metadata_provider`, `metadata_external_id`, and media kind. The browser receives an explicit `local` state and `local_media_id` when available.

## Evidence from Current Code

- `server/migrations/0021_add_media_related_titles.sql` defines the persistent provider-ID relation table and indexes.
- `syncRelatedFromMetadata` projects `recommendations` and `similar` from cached TMDB snapshots.
- `ListRelatedMedia` resolves local availability without a title comparison.
- `GET /api/media/{id}/related` and `RelatedMediaRail.jsx` render available and pending states.

## Consequences

- Browsing related titles is database-only and remains useful offline after enrichment.
- A related title can be presented before it is stored locally, with an explicit pending state.
- Re-running catalog relation sync safely backfills relationships from existing snapshots.

## What This Does Not Decide

- It does not add a download/request workflow for pending titles.
- It does not make TMDB live requests during customer browsing.
- It does not use name similarity as a fallback identity mechanism.
