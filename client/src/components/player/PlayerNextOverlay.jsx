import { memo } from "react";
import PlayerIcon from "./PlayerIcons.jsx";
import { clock } from "../../lib/watchContent.js";

/**
 * PlayerNextOverlay — the "next episode" card shown as the current one ends.
 *
 * The card is only rendered when the catalogue actually has a next file, so the
 * player never offers a transition that would fail. `countdown` is null while the
 * card waits for the video's real end; once the video ends it becomes a number and
 * counts down, and reaching zero advances automatically.
 *
 * When there is no next file at all, `endOfWork` renders the honest ending instead
 * ("انتهى الموسم" / "انتهى العمل") with a link to the related rail.
 */
export const PlayerNextOverlay = memo(function PlayerNextOverlay({
  next,
  countdown,
  onPlayNow,
  onCancel,
}) {
  if (!next) return null;

  const label = next.episode_number
    ? `S${String(next.season_number || 1).padStart(2, "0")}E${String(next.episode_number).padStart(2, "0")}`
    : "";
  const title = next.title_ar || next.title_en || "الحلقة التالية";

  return (
    <section className="nexora-next" dir="rtl" aria-live="polite" aria-label="الحلقة التالية">
      <div className="nexora-next-head">
        <PlayerIcon name="next" className="h-4 w-4" />
        <span className="nexora-next-kicker">الحلقة التالية</span>
        {label ? <span className="nexora-next-code">{label}</span> : null}
      </div>

      <p className="nexora-next-title">{title}</p>

      {next.duration > 0 ? <p className="nexora-next-meta">{clock(next.duration)}</p> : null}

      <div className="nexora-next-actions">
        <button type="button" className="nexora-next-play" onClick={onPlayNow}>
          <PlayerIcon name="play" className="h-3.5 w-3.5" />
          {countdown === null ? "تشغيل الآن" : `تشغيل الآن (${countdown})`}
        </button>
        {countdown !== null ? (
          <button type="button" className="nexora-next-cancel" onClick={onCancel}>
            إلغاء
          </button>
        ) : null}
      </div>

      {countdown !== null ? (
        <span className="nexora-next-progress" aria-hidden="true">
          <i style={{ width: `${((10 - countdown) / 10) * 100}%` }} />
        </span>
      ) : null}
    </section>
  );
});

/**
 * EndOfWorkNotice — what the viewer sees instead of a broken auto-advance.
 *
 * "Next episode" must not jump to something that does not exist, so at the end of a
 * season or of a work this is shown in the next-episode card's place.
 */
export const EndOfWorkNotice = memo(function EndOfWorkNotice({ kind, onBack }) {
  const isSeason = kind === "season";
  return (
    <section className="nexora-next nexora-next--end" dir="rtl" aria-live="polite">
      <div className="nexora-next-head">
        <PlayerIcon name="check" className="h-4 w-4" />
        <span className="nexora-next-kicker">{isSeason ? "انتهى الموسم" : "انتهى العمل"}</span>
      </div>
      <p className="nexora-next-title">
        {isSeason
          ? "هذه آخر حلقة متاحة في هذا الموسم."
          : "هذه آخر حلقة متاحة من هذا العمل."}
      </p>
      {onBack ? (
        <div className="nexora-next-actions">
          <button type="button" className="nexora-next-cancel" onClick={onBack}>
            استعراض أعمال مشابهة
          </button>
        </div>
      ) : null}
    </section>
  );
});