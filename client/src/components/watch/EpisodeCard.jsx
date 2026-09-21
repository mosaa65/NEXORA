import Icon from "../Icon.jsx";
import { resolveAPIURL } from "../../lib/api.js";
import { clock, episodeFacts, episodeLabel, readProgress } from "../../lib/watchContent.js";

/**
 * EpisodeCard — the "square" episode tile used by the watch screen and the
 * fullscreen queue. Built with the same vocabulary as UnifiedMediaCard so the
 * whole app feels like one family.
 *
 * Layers:
 *   1. The still (16:9) with a soft gradient.
 *   2. Overlays inside the image: episode number, play affordance, resume bar,
 *      "completed" and "N versions" badges.
 *   3. A compact meta column under the image (AR title, then one smart line).
 *
 * Never invents data: missing fields simply drop their chip.
 */
export default function EpisodeCard({
  episode,
  index = 0,
  type = "series",
  active = false,
  onPlay,
  onDetails,
  className = "",
}) {
  const title = episodeLabel(episode, index, type);
  const still =
    resolveAPIURL(episode?.still_path) || "/nexora-episode-placeholder.PNG";

  const available = episode?.has_local_file !== false;
  const versions = Number(episode?.file_count || 0);
  const progress = readProgress(episode?.id);
  const facts = episodeFacts(episode);

  return (
    <article
      className={`nexora-episode-card group ${active ? "is-active" : ""} ${available ? "" : "is-unavailable"} ${className}`}
    >
      <button
        type="button"
        className="nexora-episode-thumb"
        onClick={() => onPlay?.(episode)}
        disabled={!available}
        aria-label={available ? `تشغيل ${title}` : `${title} غير متوفرة`}
      >
        <img
          src={still}
          alt=""
          loading="lazy"
          onError={(event) => {
            if (event.currentTarget.src.indexOf("nexora-episode-placeholder") === -1) {
              event.currentTarget.src = "/nexora-episode-placeholder.PNG";
            }
          }}
        />
        <span className="nexora-episode-scrim" aria-hidden="true" />

        {/* Episode number */}
        <span className="nexora-episode-num">
          {episode?.episode_number ? `حلقة ${episode.episode_number}` : `${index + 1}`}
        </span>

        {/* Completed / versions badges */}
        <span className="nexora-episode-badges">
          {progress?.completed && (
            <span className="nexora-episode-badge nexora-episode-badge--done">
              <Icon name="checkbox" className="h-3 w-3" />
              مكتملة
            </span>
          )}
          {versions > 1 && (
            <span className="nexora-episode-badge nexora-episode-badge--versions">{versions} إصدارات</span>
          )}
          {!available && (
            <span className="nexora-episode-badge nexora-episode-badge--missing">غير متوفرة</span>
          )}
        </span>

        {/* Play affordance */}
        {available && (
          <span className="nexora-episode-play" aria-hidden="true">
            <Icon name="play" className="h-4 w-4" />
          </span>
        )}

        {/* Resume bar */}
        {available && progress && !progress.completed && (
          <span className="nexora-episode-progress" aria-hidden="true">
            <i style={{ width: `${progress.ratio * 100}%` }} />
          </span>
        )}

        {/* Runtime chip */}
        {episode?.duration > 0 && (
          <span className="nexora-episode-duration">{clock(episode.duration)}</span>
        )}
      </button>

      <div className="nexora-episode-meta">
        <p className="nexora-episode-title" title={title}>{title}</p>
        {facts.length > 0 && (
          <div className="nexora-episode-facts" dir="ltr">
            {facts.map((fact, i) => (
              <span key={`${fact}-${i}`}>{fact}</span>
            ))}
          </div>
        )}
        {progress && !progress.completed && (
          <p className="nexora-episode-resume">متابعة من {clock(progress.position)}</p>
        )}
        <div className="nexora-episode-actions">
          {onDetails && (
            <button type="button" className="nexora-episode-details" onClick={() => onDetails(episode)}>
              <Icon name="info" className="h-3 w-3" />
              تفاصيل
            </button>
          )}
          {available && (
            <button type="button" className="nexora-episode-watch" onClick={() => onPlay?.(episode)}>
              <Icon name="play" className="h-3 w-3" />
              مشاهدة
            </button>
          )}
        </div>
      </div>
    </article>
  );
}