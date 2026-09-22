import { memo, useState } from "react";
import PlayerIcon from "./PlayerIcons.jsx";
import PlayerScrub from "./PlayerScrub.jsx";

/**
 * PlayerControls — the single NEXORA control bar.
 *
 * Video.js' own bar is disabled at creation, so nothing here is duplicated. The
 * timeline is delegated to `PlayerScrub`, which owns the high-frequency updates.
 *
 * Layout reads right-to-left: transport on the right, then volume and the menu
 * buttons running to the left edge. The volume slider itself stays LTR because a
 * RTL range input inverts the direction the arrow keys move it.
 */
function PlayerControls({
  playerRef,
  liveStateRef,
  playing,
  duration,
  muted,
  volume,
  rate,
  isFullscreen,
  supportsPiP,
  captionsAvailable,
  activeCaptionLabel,
  episodeListAvailable,
  episodeListOpen,
  onToggleEpisodeList,
  onTogglePlay,
  onSeekStart,
  onSeek,
  onToggleMute,
  onSetVolume,
  onToggleFullscreen,
  onTogglePiP,
  onOpenSettings,
  onExit,
  onMinimize,
  fileId,
  hasNext,
  hasPrevious,
  onNext,
  onPrevious,
  menuOpen,
  settingsMenu,
  currentTimeLabel,
  remainingLabel,
  compact = false,
}) {
  const [volumeOpen, setVolumeOpen] = useState(false);

  return (
    <div
      dir="rtl"
      className={`nexora-bar absolute inset-x-0 bottom-0 z-30 px-3 pb-2 pt-10 ${compact ? "nexora-bar--compact" : ""}`}
    >
      <PlayerScrub
        playerRef={playerRef}
        liveStateRef={liveStateRef}
        duration={duration}
        onSeekStart={onSeekStart}
        onSeek={onSeek}
        fileId={fileId}
        currentTimeLabel={currentTimeLabel}
        remainingLabel={remainingLabel}
      />

      {isFullscreen && episodeListAvailable && (
        <div className="nexora-player-episodes-slot">
          <button
            type="button"
            className={`nexora-player-episodes-button ${episodeListOpen ? "is-open" : ""}`}
            aria-expanded={episodeListOpen}
            onClick={onToggleEpisodeList}
          >
            <PlayerIcon name="playlist" className="h-4 w-4" />
            الحلقات <span>{episodeListAvailable}</span>
          </button>
        </div>
      )}

      <div className="relative flex items-center justify-between gap-2">
        {/* Transport (leading edge in RTL) */}
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            className="nexora-bar-button"
            onClick={onPrevious}
            disabled={!hasPrevious}
            aria-label="الحلقة السابقة"
            title="الحلقة السابقة"
          >
            <PlayerIcon name="skipPrev" className="h-4 w-4" />
          </button>

          <button
            type="button"
            className="nexora-bar-button nexora-bar-button--primary"
            onClick={onTogglePlay}
            aria-label={playing ? "إيقاف مؤقت" : "تشغيل"}
            title={`${playing ? "إيقاف مؤقت" : "تشغيل"} (Space)`}
          >
            <PlayerIcon name={playing ? "pause" : "play"} className="h-4 w-4" />
          </button>

          <button
            type="button"
            className="nexora-bar-button"
            onClick={onNext}
            disabled={!hasNext}
            aria-label="الحلقة التالية"
            title="الحلقة التالية"
          >
            <PlayerIcon name="skipNext" className="h-4 w-4" />
          </button>

          {/* Volume — expands on hover/focus so the bar stays short on mobile. */}
          <div
            className="nexora-volume-host"
            onMouseEnter={() => setVolumeOpen(true)}
            onMouseLeave={() => setVolumeOpen(false)}
          >
            <button
              type="button"
              className={`nexora-bar-button ${muted ? "nexora-bar-button--active" : ""}`}
              onClick={onToggleMute}
              onFocus={() => setVolumeOpen(true)}
              aria-label={muted ? "إلغاء كتم الصوت" : "كتم الصوت"}
              title="كتم الصوت (M)"
            >
              <PlayerIcon name={muted || volume === 0 ? "mute" : "volume"} className="h-4 w-4" />
            </button>
            <input
              dir="ltr"
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={muted ? 0 : volume}
              onChange={(event) => onSetVolume(Number(event.target.value))}
              aria-label="مستوى الصوت"
              className={`nexora-vol ${volumeOpen ? "is-open" : ""}`}
            />
          </div>
        </div>

        {/* Menus (trailing edge in RTL) */}
        <div className="flex items-center gap-1.5">
          {captionsAvailable > 0 && onOpenSettings && (
            <button
              type="button"
              className={`nexora-bar-button ${activeCaptionLabel ? "nexora-bar-button--active" : ""}`}
              onClick={() => onOpenSettings("subtitle")}
              aria-label="الترجمة"
              title="الترجمة (C)"
            >
              <PlayerIcon name="captions" className="h-4 w-4" />
            </button>
          )}

          <span className="nexora-rate-chip" title="سرعة التشغيل">
            {Number(rate).toFixed(rate % 1 === 0 ? 0 : 2).replace(/0$/, "")}×
          </span>

          {supportsPiP && (
            <button
              type="button"
              className="nexora-bar-button nexora-bar-button--hide-sm"
              onClick={onTogglePiP}
              aria-label="نافذة عائمة"
              title="نافذة عائمة"
            >
              <PlayerIcon name="pip" className="h-4 w-4" />
            </button>
          )}

          {/* The settings surface is omitted on the compact dock, where there is no
              room to read a menu — the watch screen owns it. */}
          {onOpenSettings && (
            <button
              type="button"
              className={`nexora-bar-button ${menuOpen ? "nexora-bar-button--active" : ""}`}
              onClick={() => onOpenSettings(menuOpen === "root" ? null : "root")}
              aria-label="الإعدادات"
              aria-expanded={Boolean(menuOpen)}
              title="الإعدادات"
            >
              <PlayerIcon name="settings" className="h-4 w-4" />
            </button>
          )}

          <button
            type="button"
            className="nexora-bar-button"
            onClick={onToggleFullscreen}
            aria-label={isFullscreen ? "الخروج من ملء الشاشة" : "ملء الشاشة"}
            title={`ملء الشاشة (F)`}
          >
            <PlayerIcon name={isFullscreen ? "fullscreenExit" : "fullscreen"} className="h-4 w-4" />
          </button>
          {isFullscreen && (
            <>
              <button
                type="button"
                className="nexora-bar-button"
                onClick={onMinimize}
                aria-label="تصغير"
                title="تصغير"
              >
                <PlayerIcon name="minimize" className="h-4 w-4" />
              </button>
              <button
                type="button"
                className="nexora-bar-button nexora-bar-button--danger"
                onClick={onExit}
                aria-label="خروج"
                title="خروج"
              >
                <PlayerIcon name="close" className="h-4 w-4" />
              </button>
            </>
          )}
        </div>
      </div>

      {settingsMenu}
    </div>
  );
}

export default memo(PlayerControls);