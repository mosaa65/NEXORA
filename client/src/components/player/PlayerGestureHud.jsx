import { memo } from "react";
import PlayerIcon from "./PlayerIcons.jsx";

/**
 * PlayerGestureHud — the on-screen readout for the touch volume/brightness
 * gestures.
 *
 * It is deliberately small and near the middle of the screen: the finger covering
 * the surface is on the edge, so a corner HUD would be hidden by the hand. It
 * fades out on its own shortly after the gesture ends (the hook clears the state),
 * so nothing has to be dismissed.
 */
function PlayerGestureHud({ hud }) {
  if (!hud) return null;

  const isVolume = hud.kind === "volume";
  const label = isVolume ? "الصوت" : "السطوع";

  return (
    <div className="nexora-gesture-hud" role="status" aria-live="polite">
      <PlayerIcon name={isVolume ? "volume" : "brightness"} className="nexora-gesture-icon" />
      <div className="nexora-gesture-meter" aria-hidden="true">
        <i style={{ height: `${hud.percent}%` }} />
      </div>
      <span className="nexora-gesture-value">
        {label} {hud.percent}%
      </span>
    </div>
  );
}

export default memo(PlayerGestureHud);
