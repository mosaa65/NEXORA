import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import UnifiedMediaCard from "../UnifiedMediaCard.jsx";
import { getMediaRelated, getMediaList } from "../../lib/api.js";
import { relatedTitle } from "../../lib/watchContent.js";

/**
 * RelatedRail — "شاهد أيضًا" beneath the player.
 *
 * Source priority:
 *   1. `GET /api/media/{id}/related` — NEXORA's local relationship graph.
 *   2. Fallback: same-kind, top-rated titles from the catalogue.
 *
 * Renders catalogue `UnifiedMediaCard`s so suggestions share the app's card
 * identity. A suggestion with no local copy is shown, dimmed, and disabled.
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
        if (related.length > 0) {
          setItems(
            related
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
          );
          setLoading(false);
          return;
        }
        // Fallback: same-kind top rated from the catalogue.
        try {
          const res = await getMediaList({ type, sort: "rating", limit: 14 });
          if (!alive) return;
          const list = (res?.items || []).filter((m) => !excluded.has(String(m.id)));
          setItems(
            list.map((m) => ({
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
            }))
          );
        } catch {
          if (alive) setItems([]);
        } finally {
          if (alive) setLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [mediaId, type]); // eslint-disable-line react-hooks/exhaustive-deps

  const heading = useMemo(() => relatedTitle(type), [type]);

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
                  onOpen={item.canOpen ? () => navigate(`/media/${item.localId}`) : undefined}
                />
              {!item.local && <span className="nexora-related-flag">غير متوفر محليًا</span>}
            </div>
          ))}
      </div>
    </section>
  );
}