import Icon from "../Icon.jsx";
import { resolveAPIURL } from "../../lib/api.js";
import { episodeLabel, readProgress, clock, formatSize, formatRuntime } from "../../lib/watchContent.js";

/**
 * EpisodeCard — the episode tile used by the watch screen's side rail.
 *
 * The still IS the card: artwork fills the tile edge to edge and every fact sits
 * inside it, on top of a gradient. Nothing is pushed below the image, which keeps
 * each row short enough for a narrow rail while the artwork stays the hero.
 *
 * Placement follows the catalogue card's vocabulary (`UnifiedMediaCard`), which
 * pins its badges inside the poster the same way:
 *
 *   top-right     episode number
 *   top-left      completed tick
 *   bottom-right  duration
 *   bottom-left   resolution · size
 *   bottom edge   resume progress bar
 *
 * The title is not printed on the tile: at rail width a readable line would need
 * three of them and the artwork would shrink to a stamp. The rail's job is picking
 * an episode, and the number plus the still do that — the running title is shown
 * in full by the summary under the player.
 */
export default function EpisodeCard({
  episode,
  index = 0,
  type = "series",
  active = false,
  onPlay,
}) {
  const title = episodeLabel(episode, index, type);
  const still = resolveAPIURL(episode?.still_path) || "/nexora-episode-placeholder.PNG";

  const available = episode?.has_local_file !== false;
  const versions = Number(episode?.file_count || 0);
  const progress = readProgress(episode?.streamId || episode?.file_id || episode?.id);

  // A part is a different chapter of one work (a two-part film); an episode is a
  // chapter of a season. The tile labels them differently so a shelf of film
  // files never reads as a shelf of episodes. `episode_number` wins when both are
  // present, because that is the stronger identity.
  const partNumber = Number(episode?.part_number || 0);
  const episodeNumber = Number(episode?.episode_number || 0);
  const isPart = !episodeNumber && partNumber > 0;
  const number = episodeNumber || (isPart ? partNumber : index + 1);

  // Duration first (what a viewer scans for), then the technical extras.
  const seconds = episode?.duration || (episode?.runtime ? episode.runtime * 60 : 0);
  const duration = seconds ? clock(seconds) : "";
  const size = formatSize(episode?.file_size);
  const meta = [episode?.resolution, size].filter(Boolean);

  // Tooltip carries everything the tile cannot print at rail width: the title, the
  // air date and the runtime. Nothing is invented, and nothing is lost.
  const tooltip = [
    title,
    episode?.air_date,
    formatRuntime(episode?.runtime || episode?.duration, Boolean(episode?.duration && !episode?.runtime)),
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <article
      className={[
        "nexora-episode-card",
        active && "is-active",
        !available && "is-unavailable",
      ]
        .filter(Boolean)
        .join(" ")}
      title={tooltip}
    >
      <img
        className="nexora-episode-still"
        src={still}
        alt=""
        loading="lazy"
        onError={(event) => {
          if (!event.currentTarget.src.includes("nexora-episode-placeholder")) {
            event.currentTarget.src = "/nexora-episode-placeholder.PNG";
          }
        }}
      />

      {/* Gradient plate that makes every overlay legible on any artwork. */}
      <span className="nexora-episode-veil" aria-hidden="true" />

      {/* The episode/part number leads the tile: large, glowing, top-right. It is
          the one fact a viewer scans for when picking an entry. */}
      <span className="nexora-episode-number" aria-label={isPart ? `الجزء ${number}` : `الحلقة ${number}`}>
        {isPart && <b className="nexora-episode-number-kind">جزء</b>}
        <em>{number}</em>
      </span>

      {/* Top-left: completed tick. */}
      {progress?.completed && (
        <span className="nexora-episode-done" title="مكتملة">
          <Icon name="checkbox" className="h-3 w-3" />
        </span>
      )}

      {/* Bottom strip: the facts the catalogue actually holds, above the artwork.
          Duration, resolution and size reuse the work-details card's vocabulary. */}
      {(duration || meta.length > 0 || versions > 1) && (
        <span className="nexora-episode-facts" dir="ltr">
          {duration && <span className="nexora-episode-duration">{duration}</span>}
          {versions > 1 && <span className="nexora-episode-versions">{versions} editions</span>}
          {versions <= 1 && meta.map((fact) => <span key={fact} className="nexora-episode-meta">{fact}</span>)}
        </span>
      )}

      {/* Unavailable marker replaces the play affordance. */}
      {!available && <span className="nexora-episode-missing">غير متوفرة</span>}

      {/* Play affordance (hover / active) and the tile's click target. */}
      {available && (
        <button type="button" className="nexora-episode-play" onClick={() => onPlay?.(episode)} aria-label={`تشغيل ${title}`}>
          <Icon name="play" className="h-4 w-4" />
        </button>
      )}

      {/* Resume bar across the bottom edge. */}
      {available && progress && !progress.completed && (
        <span className="nexora-episode-progress" aria-hidden="true">
          <i style={{ width: `${progress.ratio * 100}%` }} />
        </span>
      )}
    </article>
  );
}