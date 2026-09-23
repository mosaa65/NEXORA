import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import UnifiedMediaCard from "../UnifiedMediaCard.jsx";
import { getMediaRelated, getMediaList } from "../../lib/api.js";
import { relatedTitle } from "../../lib/watchContent.js";

/**
 * RelatedRail — the "قد تعجبك" rail beneath the player.
 *
 * Source priority:
 *   1. `GET /api/media/{id}/related` — the provider's (TMDB) recommendation and
 *      similar lists, already persisted locally in `media_related_titles`.
 *   2. Top-up: same-kind, top-rated titles from the catalogue, used when the
 *      provider returned fewer than 8 entries so the rail is never near-empty.
 *
 * Renders catalogue `UnifiedMediaCard`s so suggestions share the app's card
 * identity. A locally matched suggestion opens playback directly; one with no
 * local copy is dimmed, flagged and not clickable.
 */
export default function RelatedRail({ mediaId, type, excludeIds = [] }) {
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    const excluded = new Set(excludeIds.map(String));

    getMediaRelated(mediaId, 18)
      .then((res) => {
        if (!alive) return [];
        const list = Array.isArray(res) ? res : res?.items || [];
        return list;
      })
      .catch(() => [])
      .then(async (related) => {
        if (!alive) return;

        // TMDB's recommendation/similar list is the primary source. It is mapped
        // first, then the catalogue tops the rail up when the provider returned
        // too few entries (or none) — a rail that is usually empty is worse than
        // one that mixes provider suggestions with local top-rated titles.
        const fromProvider = related
          .filter((r) => !r.local_media_id || !excluded.has(String(r.local_media_id)))
          .map((r) => ({
            id: r.local_media_id || `${r.provider}:${r.external_id}`,
            localId: r.local_media_id || null,
            local: Boolean(r.local && r.local_media_id),
            titleAr: r.title_ar || "",
            titleEn: r.title_en || r.original_title || "",
            type: r.local_media_type || r.kind || type,
            year: r.release_year,
            rating: r.rating,
            posterPath: r.poster_path,
            canOpen: Boolean(r.local && r.local_media_id),
          }))
          // Playable suggestions come first: the rail's job is to keep watching,
          // so the entries the viewer can actually open should not be buried
          // under posters that only lead to an "unavailable" notice.
          .sort((a, b) => Number(b.canOpen) - Number(a.canOpen));

        if (fromProvider.length >= 8) {
          setItems(fromProvider);
          setLoading(false);
          return;
        }

        // Fallback / top-up: same-kind top rated from the catalogue.
        try {
          const res = await getMediaList({ type, sort: "rating", limit: 14 });
          if (!alive) return;
          const known = new Set([
            ...fromProvider.map((item) => String(item.localId)).filter((id) => id !== "null"),
            ...excluded,
          ]);
          const extra = (res?.items || [])
            .filter((m) => !known.has(String(m.id)))
            .map((m) => ({
              id: m.id,
              localId: m.id,
              local: true,
              titleAr: m.title_ar || "",
              titleEn: m.title_en || "",
              type: m.type || type,
              year: m.release_year,
              rating: m.rating,
              posterPath: m.poster_path,
              canOpen: true,
            }));
          setItems([...fromProvider, ...extra].slice(0, 18));
        } catch {
          if (alive) setItems(fromProvider);
        } finally {
          if (alive) setLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [mediaId, type]); // eslint-disable-line react-hooks/exhaustive-deps

  const heading = relatedTitle();

  if (!loading && items.length === 0) return null;

  return (
    <section className="nexora-related" aria-label={heading}>
      <div className="nexora-related-head">
        <h2 className="nexora-related-title">{heading}</h2>
        <span className="nexora-related-count">{loading ? "…" : `${items.length}`}</span>
      </div>

      <div className="nexora-related-rail">
        {loading
          ? Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="nexora-related-skeleton" aria-hidden="true" />
          ))
          : items.map((item) => (
            <div key={item.id} className={`nexora-related-item ${item.canOpen ? "" : "is-muted"}`}>
              <UnifiedMediaCard
                media={{
                  id: item.localId,
                  titleAr: item.titleAr,
                  titleEn: item.titleEn,
                  type: item.type,
                  year: item.year,
                  rating: item.rating,
                  posterPath: item.posterPath,
                }}
                  variant="compact"
                  // A local match goes straight to playback: the rail exists to keep
                  // watching, and the work-details page is one more decision than the
                  // viewer asked for. A suggestion with no local copy has no player to
                  // open, so it stays closed and carries the "not available" flag.
                  onOpen={item.canOpen ? () => navigate(`/watch/${item.localId}`) : undefined}
                />
              {!item.local && <span className="nexora-related-flag">غير متوفر محليًا</span>}
            </div>
          ))}
      </div>
    </section>
  );
}