import { motion } from "framer-motion";
import NexoraPlayer from "./NexoraPlayer.jsx";
import { usePlayback } from "../context/PlaybackContext.jsx";

function Icon({ path }) {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path d={path} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const ICON = {
  expand: "M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5",
  close: "M6 6l12 12M18 6L6 18",
};

/**
 * MiniPlayerDock — the floating player that survives navigation.
 *
 * It lives above the router, so moving between pages never unmounts the video.
 * It shows whenever the playback context holds a payload (i.e. the user pressed
 * minimise on the watch screen). Dragging moves the window; expand returns to
 * the watch screen; close stops playback and clears the dock.
 */
export default function MiniPlayerDock({ onExpand }) {
  const { payload, isActive, close } = usePlayback();
  if (!isActive) return null;

  return (
    <motion.div
      drag
      dragMomentum={false}
      dragElastic={0.05}
      initial={{ opacity: 0, scale: 0.92 }}
      animate={{ opacity: 1, scale: 1 }}
      className="nexora-dock"
      dir="rtl"
    >
      <div className="nexora-dock-head">
        <button type="button" className="nexora-dock-btn" onClick={() => onExpand?.(payload)} aria-label="تكبير" title="تكبير">
          <Icon path={ICON.expand} />
        </button>
        <span className="nexora-dock-title">{payload.title || "تشغيل"}</span>
        <button type="button" className="nexora-dock-btn nexora-dock-btn--exit" onClick={close} aria-label="إغلاق" title="إغلاق">
          <Icon path={ICON.close} />
        </button>
      </div>
      <div className="nexora-dock-stage">
        <NexoraPlayer
          key={payload.src}
          src={payload.src}
          title={payload.title}
          poster={payload.poster}
          tracks={payload.tracks || []}
          fileId={payload.fileId}
          onNext={payload.onNext}
          playlist={payload.playlist || []}
          currentFileId={payload.fileId}
          onSelectFile={payload.onSelectFile}
          onMinimize={close}
          onExit={close}
          autoResume
        />
      </div>
    </motion.div>
  );
}
