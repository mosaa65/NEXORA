import Icon from "../Icon.jsx";
import { resolveAPIURL } from "../../lib/api.js";
import { episodeFacts, episodeLabel, readProgress } from "../../lib/watchContent.js";

/**
 * EpisodeCard — the episode tile used by the watch screen's side list and the
 * fullscreen queue.
 *
 * The still fills the card: only the plain episode number sits on the image
 * (top-right), so the artwork stays readable, and everything else lives in a
 * slim data strip underneath. Facts are only shown when they exist — nothing is
 * invented.
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
  const still = resolveAPIURL(episode?.still_path) || "/nexora-episode-placeholder.PNG";

  const available = episode?.has_local_file !== false;
  const versions = Number(episode?.file_count || 0);
  const progress = readProgress(episode?.streamId || episode?.file_id || episode?.id);
  const facts = episodeFacts(episode);
  const number = episode?.episode_number || index + 1;

  return (
    <article className={`nexora-episode-card ${active ? "is-active" : ""} ${available ? "" : "is-unavailable"} ${className}`}>
      <div className="nexora-episode-thumb">
        <img
          src={still}
          alt=""
          loading="lazy"
          onError={(event) => {
            if (!event.currentTarget.src.includes("nexora-episode-placeholder")) {
              event.currentTarget.src = "/nexora-episode-placeholder.PNG";
            }
          }}
        />

        {/* Episode number — a bare numeral, top right of the still. */}
        <span className="nexora-episode-number">{number}</span>

        {/* Versions badge — bottom left. */}
        {versions > 1 && <span className="nexora-episode-versions">{versions} إصدارات</span>}

        {/* Unavailable badge. */}
        {!available && <span className="nexora-episode-missing">غير متوفرة</span>}

        {/* Completed check — bottom right. Marked on the still only when the data
            strip below does not already spell it out. */}
        {progress?.completed && !(facts.length > 0) && (
          <span className="nexora-episode-done" title="مكتملة">
            <Icon name="checkbox" className="h-3.5 w-3.5" />
          </span>
        )}

        {/* Play overlay (hover / active), and the cell's click target. */}
        {available && (
          <button type="button" className="nexora-episode-play" onClick={() => onPlay?.(episode)} aria-label={`تشغيل ${title}`}>
            <Icon name="play" className="h-4 w-4" />
          </button>
        )}

        {/* Resume bar across the bottom of the image. */}
        {available && progress && !progress.completed && (
          <span className="nexora-episode-progress" aria-hidden="true">
            <i style={{ width: `${progress.ratio * 100}%` }} />
          </span>
        )}
      </div>

      {/* Data strip under the image: only what the catalogue actually knows. */}
      {(facts.length > 0 || !available || progress?.completed) && (
        <div className="nexora-episode-body">
          {facts.length > 0 && (
            <div className="nexora-episode-facts" dir="ltr">
              {facts.map((fact, i) => (
                <span key={`${fact}-${i}`}>{fact}</span>
              ))}
            </div>
          )}
          <div className="nexora-episode-marks">
            {!available && <span className="nexora-episode-soon">قيد الإضافة</span>}
            {progress?.completed && (
              <span className="nexora-episode-done-text">
                <Icon name="checkbox" className="h-3 w-3" />
                مكتملة
              </span>
            )}
          </div>
        </div>
      )}
    </article>
  );
}