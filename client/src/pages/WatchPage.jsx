import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import NexoraPlayer from "../components/NexoraPlayer.jsx";
import { usePlayback } from "../context/PlaybackContext.jsx";
import { getMediaDetail, getFileSubtitles, resolveAPIURL } from "../lib/api.js";

const clock = (value = 0) => {
  const seconds = Math.max(0, Math.floor(Number.isFinite(value) ? value : 0));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`
    : `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
};

const readProgress = (fileId) => {
  try {
    const raw = localStorage.getItem(`nexora:playback:${fileId}`);
    if (!raw) return null;
    const saved = JSON.parse(raw);
    if (!saved?.duration || saved.completed) return null;
    return Math.min(1, Math.max(0, saved.position / saved.duration));
  } catch {
    return null;
  }
};

function Icon({ path, className = "h-4 w-4" }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden="true">
      <path d={path} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const ICON = {
  play: "M8 5.4v13.2c0 .78.86 1.26 1.53.86l10.2-6.6a1 1 0 000-1.72L9.53 4.54A1 1 0 008 5.4z",
  info: "M12 16v-4M12 8h.01M12 21a9 9 0 100-18 9 9 0 000 18z",
  minimize: "M5 12h14",
  close: "M6 6l12 12M18 6L6 18",
  expand: "M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5",
};

/**
 * WatchPage — the playback screen, rendered inside the customer layout so it
 * keeps the site's search bar and navigation (same shell as every other page).
 *
 * Layout: episodes on the RIGHT, player on the left, and an under-video panel
 * with the synopsis and the current episode's name. Each episode row carries a
 * "تفاصيل" button (loads it here) and a "مشاهدة" button (plays it fullscreen).
 *
 * Minimise does NOT leave the page: it collapses the player into a small
 * draggable floating window over the page, which can be expanded again.
 */
export default function WatchPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [activeFile, setActiveFile] = useState(null);
  const [subtitles, setSubtitles] = useState([]);
  const [fullscreen, setFullscreen] = useState(false);
  const [tab, setTab] = useState("episodes");

  // The global dock owns the player while minimised, so it survives navigation.
  const { minimize, close: closeDock, isActive: dockActive } = usePlayback();

  const stageRef = useRef(null);
  const initialFileId = params.get("file");

  useEffect(() => {
    let alive = true;
    setLoading(true);
    getMediaDetail(id)
      .then((data) => {
        if (alive) setDetail(data);
      })
      .catch(() => {
        if (alive) setDetail(null);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [id]);

  const seasons = detail?.seasons || [];
  const files = detail?.files || [];
  const hasSeasons = seasons.length > 0;

  const episodes = useMemo(
    () =>
      hasSeasons
        ? seasons.flatMap((s) => (s.episodes || []).map((ep) => ({ ...ep, seasonNumber: s.season_number })))
        : files,
    [hasSeasons, seasons, files]
  );

  useEffect(() => {
    if (episodes.length === 0) return;
    setActiveFile((current) => {
      if (current) return current;
      const wanted = initialFileId ? episodes.find((ep) => String(ep.id) === String(initialFileId)) : null;
      return wanted || episodes[0];
    });
  }, [episodes, initialFileId]);

  const currentFile = activeFile || episodes[0] || null;

  useEffect(() => {
    let alive = true;
    if (currentFile?.id) {
      getFileSubtitles(currentFile.id)
        .then((res) => {
          if (!alive) return;
          setSubtitles(
            (res.subtitles || []).map((sub) => ({
              kind: "captions",
              label: sub.label || (sub.language === "ar" ? "العربية" : sub.language),
              src: resolveAPIURL(`/api/stream/file/${currentFile.id}/subtitles/${sub.index}`),
              srcLang: sub.language || "ar",
              default: sub.language === "ar",
            }))
          );
        })
        .catch(() => alive && setSubtitles([]));
    } else {
      setSubtitles([]);
    }
    return () => {
      alive = false;
    };
  }, [currentFile?.id]);

  const streamSrc = currentFile?.id
    ? resolveAPIURL(`/api/stream/file/${currentFile.id}`)
    : currentFile?.file_path
      ? resolveAPIURL(`/api/stream?path=${encodeURIComponent(currentFile.file_path)}`)
      : "";

  const title = detail?.title_ar || detail?.title_en || "تشغيل الوسائط";
  const poster = resolveAPIURL(detail?.banner_path || detail?.poster_path) || "";
  const plot = detail?.plot_ar || detail?.plot_en || "";
  const genres = Array.isArray(detail?.genres) ? detail.genres : [];
  const year = detail?.release_year;
  const rating = detail?.rating;

  const currentIndex = episodes.findIndex((item) => item.id === currentFile?.id);
  const nextFile = currentIndex >= 0 ? episodes[currentIndex + 1] : null;
  const playNext = () => nextFile && setActiveFile(nextFile);

  const currentLabel =
    currentFile?.title_ar ||
    currentFile?.title_en ||
    (currentFile?.episode_number ? `الحلقة ${currentFile.episode_number}` : currentFile ? `الملف ${currentIndex + 1}` : "");

  // Hand the current playback to the global dock and stop rendering it here.
  // The dock lives above the router, so the video keeps playing across pages.
  const minimizeToDock = useCallback(() => {
    if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
    if (!streamSrc) return;
    minimize({
      mediaId: id,
      fileId: currentFile?.id,
      src: streamSrc,
      title,
      poster,
      tracks: subtitles,
      playlist: episodes,
      onSelectFile: setActiveFile,
      onNext: playNext,
    });
  }, [id, currentFile?.id, streamSrc, title, poster, subtitles, episodes, minimize]);

  const playerNode = streamSrc ? (
    <NexoraPlayer
      key={streamSrc}
      src={streamSrc}
      title={title}
      poster={poster}
      tracks={subtitles}
      fileId={currentFile?.id}
      onNext={playNext}
      playlist={episodes}
      currentFileId={currentFile?.id}
      onSelectFile={setActiveFile}
      fullscreenTarget={stageRef}
      onMinimize={minimizeToDock}
      onExit={() => {
        if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
        goBack();
      }}
    />
  ) : (
    <div className="nexora-empty-stage">
      <p className="text-sm font-bold">
        {loading ? "جارٍ تحضير التشغيل…" : "لا يتوفر ملف فيديو صالح لهذا العمل."}
      </p>
    </div>
  );

  // Play a specific episode fullscreen, without leaving the page.
  const playFullscreen = useCallback((file) => {
    if (file) setActiveFile(file);
    stageRef.current?.requestFullscreen?.().catch(() => {});
  }, []);

  // Return to the page the user came from; fall back to the catalogue root only
  // when this screen was opened directly (no history to pop).
  const goBack = useCallback(() => {
    if (window.history.state?.idx > 0) navigate(-1);
    else navigate("/");
  }, [navigate]);

  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === stageRef.current);
    document.addEventListener("fullscreenchange", sync);
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);

  // «مشاهدة» from the details page arrives with ?play=fs, which asks this page
  // to go straight into fullscreen once the chosen file is actually mounted.
  const wantsFullscreen = params.get("play") === "fs";
  useEffect(() => {
    if (!wantsFullscreen || !currentFile?.id) return;
    const raf = requestAnimationFrame(() => {
      stageRef.current?.requestFullscreen?.().catch(() => {});
    });
    return () => cancelAnimationFrame(raf);
  }, [wantsFullscreen, currentFile?.id]);

  return (
    <div className="nexora-watch">
      <header className="nexora-watch-head">
        <div className="nexora-watch-titles">
          <h1 className="nexora-watch-title">{title}</h1>
          <div className="nexora-watch-meta">
            {year ? <span>{year}</span> : null}
            {rating ? <span className="nexora-chip-rating">★ {Number(rating).toFixed(1)}</span> : null}
            {currentFile?.resolution ? <span className="nexora-chip-live">{currentFile.resolution} · LAN</span> : null}
            {episodes.length > 0 ? <span>{episodes.length} حلقة/ملف</span> : null}
          </div>
        </div>
      </header>

      {dockActive && (
        <div className="nexora-watch-docked-note">
          <span>الفيديو يعمل الآن في النافذة العائمة.</span>
          <button type="button" onClick={closeDock}>إيقاف التشغيل العائم</button>
        </div>
      )}

      <div className="nexora-watch-grid">
          {/* Episodes column — right side in RTL (first grid column). */}
          <aside className="nexora-watch-side">
            <div className="nexora-tabs">
              <button type="button" className={tab === "episodes" ? "is-active" : ""} onClick={() => setTab("episodes")}>
                الحلقات <span>{episodes.length}</span>
              </button>
              <button type="button" className={tab === "about" ? "is-active" : ""} onClick={() => setTab("about")}>
                معلومات
              </button>
            </div>

            {tab === "episodes" ? (
              <div className="nexora-watch-list">
                {episodes.map((item, idx) => {
                  const isActive = currentFile?.id === item.id;
                  const label =
                    item.title_ar ||
                    item.title_en ||
                    (item.episode_number ? `الحلقة ${item.episode_number}` : `الملف ${idx + 1}`);
                  const thumb = item.id ? `/api/stream/file/${item.id}/preview?at=0` : poster || "/nexora-episode-placeholder.PNG";
                  const progress = readProgress(item.id);
                  return (
                    <article
                      key={item.id || idx}
                      className={`nexora-watch-card ${isActive ? "is-active" : ""}`}
                      onClick={() => setActiveFile(item)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          setActiveFile(item);
                        }
                      }}
                    >
                      <span
                        className="nexora-watch-thumb"
                        aria-hidden="true"
                      >
                        <img
                          src={thumb || poster || "/nexora-episode-placeholder.PNG"}
                          alt=""
                          loading="lazy"
                          onError={(event) => {
                            const img = event.currentTarget;
                            // preview → work poster → static placeholder, so a
                            // card always shows an image instead of collapsing.
                            const chain = [poster, "/nexora-episode-placeholder.PNG"].filter(Boolean);
                            const step = Number(img.dataset.step || 0);
                            if (step < chain.length) {
                              img.dataset.step = String(step + 1);
                              img.src = chain[step];
                            }
                          }}
                        />
                        <span className="nexora-ep-num">{idx + 1}</span>
                        {item.duration > 0 && <span className="nexora-ep-duration">{clock(item.duration)}</span>}
                        {progress ? <span className="nexora-ep-progress"><i style={{ width: `${progress * 100}%` }} /></span> : null}
                      </span>

                      <div className="nexora-watch-card-body">
                        <p className="nexora-watch-card-title">{label}</p>
                        <p className="nexora-watch-card-sub">
                          {[item.seasonNumber ? `الموسم ${item.seasonNumber}` : null, item.resolution]
                            .filter(Boolean)
                            .join(" · ") || "فيديو محلي"}
                        </p>
                        <div className="nexora-watch-card-actions">
                          <button
                            type="button"
                            className="nexora-act nexora-act--ghost"
                            onClick={(event) => {
                              event.stopPropagation();
                              setActiveFile(item);
                            }}
                          >
                            <Icon path={ICON.info} />
                            تفاصيل
                          </button>
                          <button
                            type="button"
                            className="nexora-act nexora-act--play"
                            onClick={(event) => {
                              event.stopPropagation();
                              playFullscreen(item);
                            }}
                          >
                            <Icon path={ICON.play} />
                            مشاهدة
                          </button>
                        </div>
                      </div>
                    </article>
                  );
                })}
                {episodes.length === 0 && !loading && (
                  <div className="p-8 text-center text-xs text-white/40">لا توجد حلقات فعلية مسجلة لهذا العمل.</div>
                )}
              </div>
            ) : (
              <div className="nexora-about">
                <dl>
                  <div><dt>النوع</dt><dd>{detail?.type || "—"}</dd></div>
                  <div><dt>السنة</dt><dd>{year || "—"}</dd></div>
                  <div><dt>التقييم</dt><dd>{rating ? `★ ${Number(rating).toFixed(1)}` : "—"}</dd></div>
                  <div><dt>الحلقات/الملفات</dt><dd>{episodes.length}</dd></div>
                  <div><dt>الجودة الحالية</dt><dd>{currentFile?.resolution || "—"}</dd></div>
                </dl>
                {plot && <p className="nexora-plot is-open">{plot}</p>}
                {genres.length > 0 && (
                  <div className="nexora-genres">
                    {genres.map((genre) => <span key={genre} className="nexora-genre">{genre}</span>)}
                  </div>
                )}
              </div>
            )}
          </aside>

          {/* Player column — left side in RTL. Hidden while the dock owns playback. */}
          <main className="nexora-watch-main">
            <div ref={stageRef} className={`nexora-watch-stage ${fullscreen ? "is-fullscreen" : ""} ${dockActive ? "is-docked-out" : ""}`}>
              <div className="nexora-watch-player">
                {dockActive ? (
                  <div className="nexora-empty-stage">
                    <p className="text-sm font-bold">الفيديو يعمل في النافذة العائمة</p>
                    <button type="button" className="nexora-act nexora-act--play mt-3" onClick={closeDock}>
                      إعادة التشغيل هنا
                    </button>
                  </div>
                ) : (
                  playerNode
                )}
              </div>
            </div>

            <section className="nexora-watch-summary">
              <div className="nexora-watch-summary-head">
                <span className="nexora-watch-now">
                  <span className="nexora-watch-now-dot" />
                  {currentLabel || "—"}
                </span>
                {currentFile?.duration > 0 && <span className="nexora-watch-len">{clock(currentFile.duration)}</span>}
              </div>
              {plot && <p className="nexora-watch-plot">{plot}</p>}
              {genres.length > 0 && (
                <div className="nexora-genres">
                  {genres.map((genre) => (
                    <span key={genre} className="nexora-genre">{genre}</span>
                  ))}
                </div>
              )}
            </section>
          </main>
        </div>
    </div>
  );
}
