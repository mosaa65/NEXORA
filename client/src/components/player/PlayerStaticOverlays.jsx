import { memo } from "react";
import PlayerIcon from "./PlayerIcons.jsx";
import { clock } from "../../lib/watchContent.js";

/**
 * PlayerStaticOverlays — the layers that appear on their own: the resume prompt,
 * the buffering indicator, the transient toast and the player's error state.
 *
 * They are grouped because they share one property: each is driven by a single
 * boolean from the player and none of them re-render on playback ticks. Keeping
 * them out of `NexoraPlayer` also keeps the engine component readable.
 */

export const ResumePrompt = memo(function ResumePrompt({ resumeAt, onResume, onRestart }) {
  return (
    <div className="absolute inset-0 z-40 flex items-center justify-center bg-black/55 p-4 backdrop-blur-sm" dir="rtl">
      <div className="nexora-resume-card">
        <p className="text-base font-black text-white">متابعة المشاهدة؟</p>
        <p className="mt-1 text-xs text-white/60">توقفت عند {clock(resumeAt)}</p>
        <div className="mt-4 flex justify-center gap-2">
          <button type="button" className="nexora-resume-primary" onClick={onResume}>
            استئناف
          </button>
          <button type="button" className="nexora-resume-ghost" onClick={onRestart}>
            من البداية
          </button>
        </div>
      </div>
    </div>
  );
});

/** A short buffer must not black out the screen — this is a corner indicator. */
export const BufferingIndicator = memo(function BufferingIndicator({ visible }) {
  if (!visible) return null;
  return (
    <div className="nexora-buffering" role="status" aria-live="polite">
      <span className="nexora-buffering-dot" />
      <span className="nexora-buffering-dot" />
      <span className="nexora-buffering-dot" />
      <span className="sr-only">جارٍ التحميل</span>
    </div>
  );
});

export const PlayerToast = memo(function PlayerToast({ message }) {
  if (!message) return null;
  return (
    <div className="nexora-toast" role="status" aria-live="polite">
      {message}
    </div>
  );
});

/**
 * PlayerErrorState — a useful failure, not "حدث خطأ ما".
 *
 * The message names the real cause when the player can tell (an unsupported
 * container/codec, a missing file, a network read error) and always offers the
 * two things a viewer can actually do: retry, or go back to the details page.
 */
export const PlayerErrorState = memo(function PlayerErrorState({ error, onRetry, onBack, technical }) {
  return (
    <div className="nexora-player-error" dir="rtl" role="alert">
      <span className="nexora-player-error-icon">
        <PlayerIcon name="alert" className="h-6 w-6" />
      </span>
      <h3 className="nexora-player-error-title">تعذر تشغيل الفيديو</h3>
      <p className="nexora-player-error-text">{error?.message || "تعذر قراءة الملف من المصدر."}</p>

      {technical?.resolution || technical?.video_codec ? (
        <p className="nexora-player-error-tech">
          المصدر: {technical.resolution || "—"}
          {technical.video_codec ? ` · ${technical.video_codec.toUpperCase()}` : ""}
        </p>
      ) : null}

      <div className="nexora-player-error-actions">
        <button type="button" className="nexora-resume-primary" onClick={onRetry}>
          إعادة المحاولة
        </button>
        {onBack ? (
          <button type="button" className="nexora-resume-ghost" onClick={onBack}>
            معلومات العمل
          </button>
        ) : null}
      </div>
    </div>
  );
});