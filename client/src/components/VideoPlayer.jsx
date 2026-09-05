import { useCallback, useEffect, useRef, useState } from "react";

const SEEK_SECONDS = 10;
const clock = (value = 0) => {
  const seconds = Math.max(0, Math.floor(Number.isFinite(value) ? value : 0));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0 ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}` : `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
};
const keyFor = (fileId, src) => `nexora:playback:${fileId || src}`;

function PlayerIcon({ name, className = "h-5 w-5" }) {
  const common = { viewBox: "0 0 24 24", className, "aria-hidden": true, fill: "none", stroke: "currentColor", strokeWidth: 2, strokeLinecap: "round", strokeLinejoin: "round" };
  if (name === "play") return <svg {...common} fill="currentColor" stroke="none"><path d="M8 5.4v13.2c0 .78.86 1.26 1.53.86l10.2-6.6a1 1 0 000-1.72L9.53 4.54A1 1 0 008 5.4z" /></svg>;
  if (name === "pause") return <svg {...common} fill="currentColor" stroke="none"><rect x="7" y="5" width="3.5" height="14" rx="1" /><rect x="13.5" y="5" width="3.5" height="14" rx="1" /></svg>;
  if (name === "rewind" || name === "forward") return <svg {...common} className={`${className} ${name === "forward" ? "scale-x-[-1]" : ""}`}><path d="M4 10a8.5 8.5 0 1 1 1.35 7.1" /><path d="M4 4.5V10h5.5" /></svg>;
  if (name === "volume") return <svg {...common}><path d="M4 10v4h4l5 4V6L8 10H4z" /><path d="M16 9a4 4 0 010 6M18.5 6.5a7.5 7.5 0 010 11" /></svg>;
  if (name === "mute") return <svg {...common}><path d="M4 10v4h4l5 4V6L8 10H4zM17 10l4 4m0-4l-4 4" /></svg>;
  if (name === "pip") return <svg {...common}><rect x="3.5" y="5" width="17" height="14" rx="2" /><rect x="12.5" y="12" width="5" height="4" rx=".7" fill="currentColor" stroke="none" /></svg>;
  if (name === "fullscreen") return <svg {...common}><path d="M8 3H3v5m13-5h5v5M8 21H3v-5m13 5h5v-5" /></svg>;
  if (name === "playlist") return <svg {...common}><path d="M5 6h14M5 12h14M5 18h9" /><path d="M18 16v5m-2.5-2.5h5" /></svg>;
  return null;
}

export default function VideoPlayer({ src, title, poster, tracks = [], fileId, onNext, playlist = [], currentFileId, onSelectFile }) {
  const rootRef = useRef(null);
  const videoRef = useRef(null);
  const hideTimer = useRef(null);
  const previewTimer = useRef(null);
  const previewBucket = useRef(null);
  const lastSavedAt = useRef(0);
  const [playing, setPlaying] = useState(false);
  const [duration, setDuration] = useState(0);
  const [time, setTime] = useState(0);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [rate, setRate] = useState(1);
  const [visible, setVisible] = useState(true);
  const [hoverTime, setHoverTime] = useState(null);
  const [previewSrc, setPreviewSrc] = useState("");
  const [resumeAt, setResumeAt] = useState(0);
  const [askResume, setAskResume] = useState(false);
  const [captionIndex, setCaptionIndex] = useState(-1);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [episodeDrawerOpen, setEpisodeDrawerOpen] = useState(false);

  const saveProgress = useCallback((force = false) => {
    const video = videoRef.current;
    if (!video?.duration || (!force && Math.abs(video.currentTime - lastSavedAt.current) < 10)) return;
    lastSavedAt.current = video.currentTime;
    const completed = video.currentTime / video.duration >= 0.95;
    try { localStorage.setItem(keyFor(fileId, src), JSON.stringify({ position: completed ? 0 : video.currentTime, duration: video.duration, completed, updatedAt: Date.now() })); } catch {}
  }, [fileId, src]);

  const seekBy = useCallback((seconds) => {
    const video = videoRef.current;
    if (video) video.currentTime = Math.max(0, Math.min(video.duration || 0, video.currentTime + seconds));
  }, []);
  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (video?.paused) video.play().catch(() => {}); else video?.pause();
  }, []);
  const toggleFullscreen = useCallback(() => {
    if (document.fullscreenElement) document.exitFullscreen?.(); else rootRef.current?.requestFullscreen?.().catch(() => {});
  }, []);
  const showControls = useCallback(() => {
    setVisible(true); clearTimeout(hideTimer.current);
    if (!videoRef.current?.paused) hideTimer.current = setTimeout(() => setVisible(false), 2500);
  }, []);
  const hideControls = useCallback(() => {
    clearTimeout(hideTimer.current);
    if (!videoRef.current?.paused && !askResume) setVisible(false);
  }, [askResume]);
  const cycleCaptions = useCallback(() => {
    const textTracks = videoRef.current?.textTracks;
    if (!textTracks?.length) return;
    const next = captionIndex + 1 >= textTracks.length ? -1 : captionIndex + 1;
    Array.from(textTracks).forEach((track, index) => { track.mode = index === next ? "showing" : "disabled"; });
    setCaptionIndex(next);
  }, [captionIndex]);

  useEffect(() => {
    const onKeyDown = (event) => {
      if (!rootRef.current?.contains(document.activeElement) || ["INPUT", "TEXTAREA", "SELECT"].includes(event.target?.tagName)) return;
      const key = event.key.toLowerCase();
      if ([" ", "k", "arrowleft", "arrowright", "j", "l", "m", "f", "c"].includes(key)) event.preventDefault();
      if (key === " " || key === "k") togglePlay();
      if (key === "arrowleft" || key === "j") seekBy(-SEEK_SECONDS);
      if (key === "arrowright" || key === "l") seekBy(SEEK_SECONDS);
      if (key === "m" && videoRef.current) videoRef.current.muted = !videoRef.current.muted;
      if (key === "f") toggleFullscreen();
      if (key === "c") cycleCaptions();
      showControls();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [cycleCaptions, seekBy, showControls, toggleFullscreen, togglePlay]);
  useEffect(() => () => { saveProgress(true); clearTimeout(hideTimer.current); clearTimeout(previewTimer.current); }, [saveProgress]);
  useEffect(() => {
    const syncFullscreen = () => {
      const fullscreen = document.fullscreenElement === rootRef.current;
      setIsFullscreen(fullscreen);
      if (!fullscreen) setEpisodeDrawerOpen(false);
    };
    document.addEventListener("fullscreenchange", syncFullscreen);
    syncFullscreen();
    return () => document.removeEventListener("fullscreenchange", syncFullscreen);
  }, []);

  function onMetadata() {
    const video = videoRef.current;
    if (!video) return;
    setDuration(video.duration || 0);
    try {
      const saved = JSON.parse(localStorage.getItem(keyFor(fileId, src)) || "null");
      if (saved?.position > 30 && saved.position < video.duration - 30 && !saved.completed) { setResumeAt(saved.position); setAskResume(true); }
    } catch {}
  }
  function onSeek(event) { const next = Number(event.target.value); if (videoRef.current) videoRef.current.currentTime = next; setTime(next); }
  function onHoverProgress(event) {
    const rect = event.currentTarget.getBoundingClientRect();
    const nextTime = duration ? Math.max(0, Math.min(duration, ((event.clientX - rect.left) / rect.width) * duration)) : null;
    setHoverTime(nextTime);
    if (!fileId || nextTime === null) return;
    const bucket = Math.floor(nextTime / 10) * 10;
    if (previewBucket.current === bucket) return;
    previewBucket.current = bucket;
    clearTimeout(previewTimer.current);
    previewTimer.current = setTimeout(() => setPreviewSrc(`/api/stream/file/${fileId}/preview?at=${bucket}`), 180);
  }
  const currentPlaylistIndex = playlist.findIndex((item) => item.id === currentFileId || (!item.id && item.file_path === src));
  const remainingEpisodes = currentPlaylistIndex >= 0 ? playlist.slice(currentPlaylistIndex + 1) : playlist;
  return (
    <div ref={rootRef} tabIndex="0" className="nexora-player group relative aspect-video overflow-hidden bg-black shadow-panel" onMouseMove={showControls} onMouseEnter={showControls} onMouseLeave={hideControls} onFocus={showControls}>
      <video key={src} ref={videoRef} playsInline preload="metadata" poster={poster || undefined} aria-label={title} className={`h-full w-full bg-black object-contain ${visible ? "cursor-default" : "cursor-none"}`} onLoadedMetadata={onMetadata} onTimeUpdate={(e) => { setTime(e.currentTarget.currentTime); saveProgress(); }} onPlay={() => { setPlaying(true); showControls(); }} onPause={() => { setPlaying(false); setVisible(true); saveProgress(true); }} onVolumeChange={(e) => { setVolume(e.currentTarget.volume); setMuted(e.currentTarget.muted); }} onRateChange={(e) => setRate(e.currentTarget.playbackRate)} onEnded={() => { saveProgress(true); onNext?.(); }} onClick={togglePlay} onDoubleClick={(e) => { const box = e.currentTarget.getBoundingClientRect(); seekBy(e.clientX < box.left + box.width / 2 ? -SEEK_SECONDS : SEEK_SECONDS); }}>
        <source src={src} />
        {tracks.map((track) => <track key={track.src} kind={track.kind || "subtitles"} label={track.label || "العربية"} src={track.src} srcLang={track.srcLang || "ar"} default={Boolean(track.default)} />)}
      </video>

      <div className={`nexora-player-center-controls absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 transition-all duration-500 ${visible && !askResume ? "scale-100 opacity-100" : "pointer-events-none scale-95 opacity-0"}`} dir="ltr" onClick={(event) => { if (event.target === event.currentTarget) togglePlay(); }}>
        <div className="flex items-center gap-3 sm:gap-5">
          <button type="button" className="nexora-player-center-button nexora-player-seek-button" onClick={() => seekBy(-SEEK_SECONDS)} aria-label="رجوع 10 ثوانٍ"><PlayerIcon name="rewind" /><small>10</small></button>
          <button type="button" className="nexora-player-center-button nexora-player-main-button" onClick={togglePlay} aria-label={playing ? "إيقاف مؤقت" : "تشغيل"}><PlayerIcon name={playing ? "pause" : "play"} className="h-7 w-7" /></button>
          <button type="button" className="nexora-player-center-button nexora-player-seek-button" onClick={() => seekBy(SEEK_SECONDS)} aria-label="تقديم 10 ثوانٍ"><PlayerIcon name="forward" /><small>10</small></button>
        </div>
      </div>

      {askResume && <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/55 p-4 backdrop-blur-sm" dir="rtl"><div className="rounded-2xl border border-white/15 bg-[#151225]/95 p-5 text-center shadow-2xl"><p className="text-base font-black text-white">متابعة المشاهدة؟</p><p className="mt-1 text-xs text-white/60">توقفت عند {clock(resumeAt)}</p><div className="mt-4 flex justify-center gap-2"><button type="button" className="rounded-xl bg-fuchsia-600 px-4 py-2 text-xs font-black text-white" onClick={() => { videoRef.current.currentTime = resumeAt; setAskResume(false); videoRef.current.play().catch(() => {}); }}>استئناف</button><button type="button" className="rounded-xl bg-white/10 px-4 py-2 text-xs font-bold text-white" onClick={() => { videoRef.current.currentTime = 0; setAskResume(false); }}>من البداية</button></div></div></div>}

      {isFullscreen && remainingEpisodes.length > 0 && <section className={`nexora-fullscreen-episodes ${episodeDrawerOpen ? "is-open" : ""}`} dir="rtl" aria-label="الحلقات المتبقية">
        <div className="nexora-fullscreen-episodes-panel">
          <div className="mb-3 flex items-center justify-between"><button type="button" className="nexora-fullscreen-episodes-close" onClick={() => setEpisodeDrawerOpen(false)} aria-label="إغلاق قائمة الحلقات">×</button><span className="text-xs text-white/55">{remainingEpisodes.length} متبقية</span><h3 className="text-sm font-black text-white">الحلقات المتبقية</h3></div>
          <div className="nexora-fullscreen-episodes-list">{remainingEpisodes.map((episode, index) => <button key={episode.id || episode.file_path || index} type="button" onClick={() => { onSelectFile?.(episode); setEpisodeDrawerOpen(false); }}><span className="text-fuchsia-300">{currentPlaylistIndex + index + 2}</span><b>{episode.title_ar || episode.title_en || (episode.episode_number ? `الحلقة ${episode.episode_number}` : `ملف ${currentPlaylistIndex + index + 2}`)}</b>{episode.resolution && <small>{episode.resolution}</small>}</button>)}</div>
        </div>
      </section>}

      <div className={`absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-black via-black/80 to-transparent px-3 pb-3 pt-12 transition-all duration-500 ${visible ? "translate-y-0 opacity-100" : "pointer-events-none translate-y-2 opacity-0"}`} dir="rtl">
        <div className="relative mb-2" onMouseMove={onHoverProgress} onMouseLeave={() => { setHoverTime(null); setPreviewSrc(""); previewBucket.current = null; }}>
          {hoverTime !== null && <span className="absolute bottom-4 z-20 flex w-40 flex-col overflow-hidden rounded-lg border border-white/20 bg-black/95 shadow-xl" style={{ right: `${Math.max(0, Math.min(82, 100 - (hoverTime / duration) * 100))}%` }}>{previewSrc && <img src={previewSrc} alt="معاينة المشهد" className="aspect-video w-full object-cover" onError={() => setPreviewSrc("")} />}<span className="px-2 py-1 text-center text-[11px] font-bold text-white">{clock(hoverTime)}</span></span>}
          <input dir="ltr" aria-label="شريط تقدم الفيديو" type="range" min="0" max={duration || 0} step="0.1" value={time} onChange={onSeek} className="nexora-player-progress w-full" style={{ "--player-progress": `${duration ? (time / duration) * 100 : 0}%` }} />
        </div>
        {isFullscreen && remainingEpisodes.length > 0 && <div className="nexora-player-episodes-slot" dir="rtl"><button type="button" className="nexora-player-episodes-button" onClick={() => setEpisodeDrawerOpen((open) => !open)}><PlayerIcon name="playlist" className="h-4 w-4" />الحلقات المتبقية <span>{remainingEpisodes.length}</span></button></div>}
        <div className="flex items-center gap-1.5 text-white"><span className="rounded-md bg-black/25 px-2 py-1 text-[11px] font-bold tabular-nums text-white/85">{clock(time)} <span className="text-white/40">/</span> {clock(duration)}</span><div className="ml-auto flex items-center gap-1"><button type="button" className="nexora-player-button" onClick={() => { if (videoRef.current) videoRef.current.muted = !muted; }} aria-label={muted ? "إلغاء الكتم" : "كتم الصوت"}><PlayerIcon name={muted || volume === 0 ? "mute" : "volume"} className="h-4 w-4" /></button><input aria-label="الصوت" type="range" min="0" max="1" step="0.05" value={muted ? 0 : volume} onChange={(e) => { if (videoRef.current) { videoRef.current.volume = Number(e.target.value); videoRef.current.muted = Number(e.target.value) === 0; } }} className="w-16 accent-fuchsia-400" />{tracks.length > 0 && <button type="button" className={`nexora-player-button ${captionIndex >= 0 ? "text-fuchsia-300" : ""}`} onClick={cycleCaptions}>CC</button>}<select aria-label="سرعة التشغيل" value={rate} onChange={(e) => { if (videoRef.current) videoRef.current.playbackRate = Number(e.target.value); }} className="rounded-md bg-white/10 px-1.5 py-1 text-[11px] font-bold outline-none"><option value="0.75">0.75×</option><option value="1">1×</option><option value="1.25">1.25×</option><option value="1.5">1.5×</option><option value="2">2×</option></select>{document.pictureInPictureEnabled && <button type="button" className="nexora-player-button" onClick={() => videoRef.current?.requestPictureInPicture?.().catch(() => {})} aria-label="نافذة مصغرة"><PlayerIcon name="pip" className="h-4 w-4" /></button>}<button type="button" className="nexora-player-button" onClick={toggleFullscreen} aria-label="ملء الشاشة"><PlayerIcon name="fullscreen" className="h-4 w-4" /></button></div></div>
      </div>
    </div>
  );
}
