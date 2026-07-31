import { useEffect, useRef } from "react";
import Plyr from "plyr";
import "plyr/dist/plyr.css";

const controls = [
  "play-large",
  "play",
  "progress",
  "current-time",
  "duration",
  "mute",
  "volume",
  "captions",
  "settings",
  "pip",
  "airplay",
  "fullscreen"
];

const arabicLabels = {
  restart: "إعادة التشغيل",
  rewind: "رجوع {seektime} ثانية",
  play: "تشغيل",
  pause: "إيقاف مؤقت",
  fastForward: "تقديم {seektime} ثانية",
  seek: "الانتقال",
  seekLabel: "{currentTime} من {duration}",
  played: "تم التشغيل",
  buffered: "تم التحميل",
  currentTime: "الوقت الحالي",
  duration: "المدة",
  volume: "الصوت",
  mute: "كتم",
  unmute: "إلغاء الكتم",
  enableCaptions: "تشغيل الترجمة",
  disableCaptions: "إيقاف الترجمة",
  download: "تنزيل",
  enterFullscreen: "ملء الشاشة",
  exitFullscreen: "الخروج من ملء الشاشة",
  frameTitle: "مشغل {title}",
  captions: "الترجمة",
  settings: "الإعدادات",
  pip: "نافذة مصغرة",
  menuBack: "رجوع",
  speed: "السرعة",
  normal: "عادي",
  quality: "الجودة",
  loop: "تكرار"
};

export default function VideoPlayer({ src, title, poster, tracks = [] }) {
  const videoRef = useRef(null);
  const playerRef = useRef(null);

  useEffect(() => {
    if (!videoRef.current || !src) {
      return undefined;
    }

    playerRef.current?.destroy();
    const player = new Plyr(videoRef.current, {
      controls,
      i18n: arabicLabels,
      keyboard: { focused: true, global: false },
      tooltips: { controls: true, seek: true },
      captions: { active: true, language: "ar", update: true }
    });
    playerRef.current = player;

    return () => {
      player.destroy();
      playerRef.current = null;
    };
  }, [src]);

  return (
    <div className="nexora-player overflow-hidden rounded-[1.5rem] border border-white/10 bg-black shadow-panel">
      <video
        key={src}
        ref={videoRef}
        controls
        playsInline
        preload="metadata"
        poster={poster || undefined}
        aria-label={title}
        className="aspect-video w-full bg-black"
      >
        <source src={src} />
        {tracks.map((track) => (
          <track
            key={track.src}
            kind={track.kind || "subtitles"}
            label={track.label || "العربية"}
            src={track.src}
            srcLang={track.srcLang || "ar"}
            default={Boolean(track.default)}
          />
        ))}
      </video>
    </div>
  );
}
