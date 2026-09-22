import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import NexoraPlayer from "../components/NexoraPlayer.jsx";
import EpisodeCard from "../components/watch/EpisodeCard.jsx";
import RelatedRail from "../components/watch/RelatedRail.jsx";
import Icon from "../components/Icon.jsx";
import { usePlayback } from "../context/PlaybackContext.jsx";
import { getMediaDetail, getFileSubtitles, searchAllEpisodes, resolveAPIURL } from "../lib/api.js";
import {
  clock,
  isEpisodic,
  itemNoun,
  kindLabel,
  listTitle,
} from "../lib/watchContent.js";

/**
 * WatchPage — the playback screen for every kind of work in the library.
 *
 * It adapts to `media.type` (movie / series / anime / documentary / …):
 *   - the side list becomes episodes, film parts or files, with the right title;
 *   - a related rail ("watch next") sits beneath the player, scoped to the kind;
 *   - the header carries the kind, year, rating, status and a resume action.
 *
 * Episodes render as square `EpisodeCard` tiles; the player stays in the left
 * column and can be minimised into the global floating dock without leaving the
 * page. Rendered inside the customer layout, so it keeps the site's search bar.
 */
export default function WatchPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [indexEpisodes, setIndexEpisodes] = useState(null); // episodes from the episode index
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

  // Episodes come from the dedicated episode index (the same source the details
  // page uses); `detail.seasons`/`detail.files` are only a fallback for works
  // the index does not describe.
  useEffect(() => {
    let alive = true;
    searchAllEpisodes(id, { pageSize: 200 })
      .then((hits) => {
        if (!alive) return;
        const list = (hits || []).filter((h) => h && h.id != null);
        setIndexEpisodes(list.length > 0 ? list : null);
      })
      .catch(() => {
        if (alive) setIndexEpisodes(null);
      });
    return () => {
      alive = false;
    };
  }, [id]);

  const type = detail?.type || "movie";
  const episodic = isEpisodic(type);
  const seasons = detail?.seasons || [];
  const files = detail?.files || [];
  const hasSeasons = seasons.length > 0;

  const detailEpisodes = useMemo(
    () =>
      hasSeasons
        ? seasons.flatMap((s) => (s.episodes || []).map((ep) => ({ ...ep, seasonNumber: s.season_number })))
        : files,
    [hasSeasons, seasons, files]
  );

  // Prefer the episode index; fall back to the work's own seasons/files. Index
  // hits carry the playable file id separately from the episode id, so normalise
  // a single `streamId` both sources expose for streaming and subtitles.
  const episodes = useMemo(() => {
    const source = (indexEpisodes && indexEpisodes.length > 0) ? indexEpisodes : detailEpisodes;
    return source.map((item) => ({
      ...item,
      streamId: item.file_id || item.fileId || item.id,
    }));
  }, [indexEpisodes, detailEpisodes]);

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
    if (currentFile?.streamId) {
      getFileSubtitles(currentFile.streamId)
        .then((res) => {
          if (!alive) return;
          setSubtitles(
            (res.subtitles || []).map((sub) => ({
              kind: "captions",
              label: sub.label || (sub.language === "ar" ? "العربية" : sub.language),
              src: resolveAPIURL(`/api/stream/file/${currentFile.streamId}/subtitles/${sub.index}`),
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
  }, [currentFile?.streamId]);

  const streamSrc = currentFile?.streamId
    ? resolveAPIURL(`/api/stream/file/${currentFile.streamId}`)
    : currentFile?.file_path
      ? resolveAPIURL(`/api/stream?path=${encodeURIComponent(currentFile.file_path)}`)
      : "";

  const title = detail?.title_ar || detail?.title_en || "تشغيل الوسائط";
  const titleEn = detail?.title_en || "";
  const poster = resolveAPIURL(detail?.poster_path || detail?.banner_path) || "";
  const backdrop = resolveAPIURL(detail?.banner_path || detail?.poster_path) || poster;
  const plot = detail?.plot_ar || detail?.plot_en || "";
  const genres = Array.isArray(detail?.genres) ? detail.genres : [];
  const year = detail?.release_year;
  const rating = detail?.rating;
  const status = detail?.status;

  const currentIndex = episodes.findIndex((item) => item.id === currentFile?.id);
  const nextFile = currentIndex >= 0 ? episodes[currentIndex + 1] : null;
  const playNext = useCallback(() => nextFile && setActiveFile(nextFile), [nextFile]);

  const currentLabel = currentFile
    ? currentFile.title_ar ||
      currentFile.title_en ||
      (currentFile.episode_number ? `الحلقة ${currentFile.episode_number}` : `${itemNoun(type)} ${currentIndex + 1}`)
    : "";

  const resumeFrom = (() => {
    if (!currentFile?.streamId) return null;
    try {
      const saved = JSON.parse(localStorage.getItem(`nexora:playback:${currentFile.streamId}`) || "null");
      if (saved?.position > 30 && !saved.completed) return saved.position;
    } catch {}
    return null;
  })();

  const goBack = useCallback(() => {
    if (window.history.state?.idx > 0) navigate(-1);
    else navigate("/");
  }, [navigate]);

  const playFullscreen = useCallback((file) => {
    if (file) setActiveFile(file);
    stageRef.current?.requestFullscreen?.().catch(() => {});
  }, []);

  // Hand the current playback to the global dock and stop rendering it here.
  const minimizeToDock = useCallback(() => {
    if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
    if (!streamSrc) return;
    minimize({
      mediaId: id,
      fileId: currentFile?.streamId,
      src: streamSrc,
      title,
      poster,
      tracks: subtitles,
      playlist: episodes,
      onSelectFile: setActiveFile,
      onNext: playNext,
    });
  }, [id, currentFile?.streamId, streamSrc, title, poster, subtitles, episodes, minimize, playNext]);

  const playerNode = streamSrc ? (
    <NexoraPlayer
      key={streamSrc}
      src={streamSrc}
      title={title}
      poster={poster}
      tracks={subtitles}
      fileId={currentFile?.streamId}
      onNext={playNext}
      playlist={episodes}
      currentFileId={currentFile?.streamId}
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

  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === stageRef.current);
    document.addEventListener("fullscreenchange", sync);
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);

  const wantsFullscreen = params.get("play") === "fs";
  useEffect(() => {
    if (!wantsFullscreen || !currentFile?.streamId) return;
    const raf = requestAnimationFrame(() => {
      stageRef.current?.requestFullscreen?.().catch(() => {});
    });
    return () => cancelAnimationFrame(raf);
  }, [wantsFullscreen, currentFile?.streamId]);

  const sideTitle = listTitle(type);

  return (
    <div className="nexora-watch">
      {/* Ambient backdrop from the work's own art. */}
      {backdrop ? <div className="nexora-watch-ambient" style={{ backgroundImage: `url(${backdrop})` }} aria-hidden="true" /> : null}

      <header className="nexora-watch-head">
        <button type="button" className="nexora-watch-back" onClick={goBack}>
          <Icon name="arrowRight" className="h-4 w-4" />
          رجوع
        </button>
        <div className="nexora-watch-titles">
          <h1 className="nexora-watch-title">{title}</h1>
          {titleEn && titleEn !== title ? <p className="nexora-watch-title-en" dir="ltr">{titleEn}</p> : null}
          <div className="nexora-watch-meta">
            <span className="nexora-chip-kind">{kindLabel(type)}</span>
            {year ? <span>{year}</span> : null}
            {rating ? <span className="nexora-chip-rating">★ {Number(rating).toFixed(1)}</span> : null}
            {episodic && seasons.length > 0 ? <span>{seasons.length} مواسم</span> : null}
            {episodes.length > 0 ? <span>{episodes.length} {itemNoun(type)}</span> : null}
            {currentFile?.resolution ? <span className="nexora-chip-live">{currentFile.resolution} · LAN</span> : null}
          </div>
        </div>
        {resumeFrom !== null && (
          <span className="nexora-watch-resume-chip">متابعة من {clock(resumeFrom)}</span>
        )}
      </header>

      {dockActive && (
        <div className="nexora-watch-docked-note">
          <span>الفيديو يعمل الآن في النافذة العائمة.</span>
          <button type="button" onClick={closeDock}>إيقاف التشغيل العائم</button>
        </div>
      )}

      <div className="nexora-watch-grid">
        {/* Side column: episodes/parts/files + about. */}
        <aside className="nexora-watch-side">
          <div className="nexora-tabs">
            <button type="button" className={tab === "episodes" ? "is-active" : ""} onClick={() => setTab("episodes")}>
              {sideTitle} <span>{episodes.length}</span>
            </button>
            <button type="button" className={tab === "about" ? "is-active" : ""} onClick={() => setTab("about")}>
              معلومات
            </button>
          </div>

          {tab === "episodes" ? (
            <div className="nexora-watch-list">
              {episodes.map((item, idx) => (
                <EpisodeCard
                  key={item.id || idx}
                  episode={item}
                  index={idx}
                  type={type}
                  active={currentFile?.id === item.id}
                  onPlay={playFullscreen}
                  onDetails={setActiveFile}
                />
              ))}
              {episodes.length === 0 && !loading && (
                <div className="p-8 text-center text-xs text-white/40">
                  لا توجد {itemNoun(type)} متوفرة لهذا العمل.
                </div>
              )}
            </div>
          ) : (
            <div className="nexora-about">
              <dl>
                <div><dt>النوع</dt><dd>{kindLabel(type)}</dd></div>
                <div><dt>السنة</dt><dd>{year || "—"}</dd></div>
                <div><dt>التقييم</dt><dd>{rating ? `★ ${Number(rating).toFixed(1)}` : "—"}</dd></div>
                {status ? <div><dt>الحالة</dt><dd>{status}</dd></div> : null}
                <div><dt>{episodic ? "المواسم" : `عدد ${itemNoun(type)}ات`}</dt><dd>{episodic ? seasons.length : episodes.length}</dd></div>
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

        {/* Player column (left in RTL). */}
        <main className="nexora-watch-main">
          <div
            ref={stageRef}
            className={`nexora-watch-stage ${fullscreen ? "is-fullscreen" : ""} ${dockActive ? "is-docked-out" : ""}`}
          >
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
            {currentFile?.overview_ar || currentFile?.overview_en ? (
              <p className="nexora-watch-plot">{currentFile.overview_ar || currentFile.overview_en}</p>
            ) : plot ? (
              <p className="nexora-watch-plot">{plot}</p>
            ) : null}
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

      {/* Watch next — suggestions scoped to the kind of work. */}
      {detail?.id && (
        <RelatedRail
          mediaId={detail.id}
          type={type}
          excludeIds={[detail.id]}
        />
      )}
    </div>
  );
}