import { memo } from "react";
import PlayerIcon from "./PlayerIcons.jsx";

const SEEK_SECONDS = 10;

/**
 * PlayerCenterControls — the large play/pause with the ±10s jumps.
 *
 * The layer itself never swallows pointer events: clicks still reach the video
 * (single click toggles play, double click toggles fullscreen). Only the buttons
 * re-enable pointer events for themselves.
 */
function PlayerCenterControls({ visible, playing, onTogglePlay, onSeekBy, onToast, seekable = true }) {
  return (
    <div
      dir="ltr"
      className={`nexora-center pointer-events-none absolute inset-0 z-20 items-center justify-center gap-4 ${visible ? "opacity-100" : "opacity-0"
        }`}
      aria-hidden={!visible}
    >
      <div className="pointer-events-auto flex items-center gap-4 sm:gap-6">
        {/* The ±10s pair only makes sense while something is actually playable:
            with no duration there is nothing to seek through, and the rings just
            crowd the tile (which is what made them collide with the numeral on a
            phone). They are hidden until the media is seekable. */}
        {seekable && (
          <button
            type="button"
            className="nexora-center-button nexora-center-seek"
            onClick={() => {
              onSeekBy(-SEEK_SECONDS);
              onToast("−10 ثوانٍ");
            }}
            aria-label="رجوع 10 ثوانٍ"
            tabIndex={visible ? 0 : -1}
          >
            <PlayerIcon name="rewind" />
            <small>10</small>
          </button>
        )}

        <button
          type="button"
          className="nexora-center-button nexora-center-main"
          onClick={onTogglePlay}
          aria-label={playing ? "إيقاف مؤقت" : "تشغيل"}
          tabIndex={visible ? 0 : -1}
        >
          <PlayerIcon name={playing ? "pause" : "play"} />
        </button>

        {seekable && (
          <button
            type="button"
            className="nexora-center-button nexora-center-seek"
            onClick={() => {
              onSeekBy(SEEK_SECONDS);
              onToast("+10 ثوانٍ");
            }}
            aria-label="تقديم 10 ثوانٍ"
            tabIndex={visible ? 0 : -1}
          >
            <PlayerIcon name="forward" />
            <small>10</small>
          </button>
        )}
      </div>
    </div>
  );
}

export default memo(PlayerCenterControls);