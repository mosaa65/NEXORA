import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import NexoraPlayer from "../components/NexoraPlayer.jsx";
import EpisodeCard from "../components/watch/EpisodeCard.jsx";
import RelatedRail from "../components/watch/RelatedRail.jsx";
import Icon from "../components/Icon.jsx";
import { usePlayback } from "../context/PlaybackContext.jsx";
import { getMediaPlayback, getFileSubtitles, resolveAPIURL } from "../lib/api.js";
import { clock, itemNoun, kindLabel, listTitle } from "../lib/watchContent.js";

/**
 * The identifier of a playable file row.
 *
 * The playback API names this field `video_file_id` on `files`, `source`, `next`
 * and `previous` (the repository's `PlaybackFile` type). Older callers pass an
 * `id`, so both are accepted and compared as strings: the field arrives as a
 * number from JSON but as a string in a URL query, and a strict comparison would
 * silently drop the match.
 */
function fileKey(row) {
  const value = row?.video_file_id ?? row?.id;
  return value === undefined || value === null || value === "" ? null : String(value);
}

/**
 * WatchPage — the playback screen for every kind of work in the library.
 *
 * One request (`getMediaPlayback`) supplies the critical path: the work header, the
 * source, the episode list and the next/previous entries. The previous version
 * opened with the media detail call plus a paged episode search, which on a long
 * show meant up to fifty requests before the first frame.
 *
 * The screen adapts to the work's type: films show their parts/files, episodic
 * works get a season selector above the episode grid. Related titles load in the
 * background and never block playback.
 */
export default function WatchPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [params] = useSearchParams();

  const [plan, setPlan] = useState(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [seasonFilter, setSeasonFilter] = useState(null);
  const [subtitles, setSubtitles] = useState([]);
  const [fullscreen, setFullscreen] = useState(false);

  const { minimize, close: closeDock, isActive: dockActive } = usePlayback();

  const stageRef = useRef(null);
  const initialFileId = params.get("file");

  // The global dock owns the player while minimised, so it survives navigation.
  useEffect(() => {
    let alive = true;
    setLoading(true);
    setFailed(false);
    getMediaPlayback(id, initialFileId)
      .then((data) => {
        if (!alive) return;
        setPlan(data);
        // Open on the season of the source, so the list matches what is playing.
        if (data?.source?.season_number) setSeasonFilter(data.source.season_number);
      })
      .catch(() => {
        if (!alive) return;
        setPlan(null);
        setFailed(true);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [id, initialFileId]);

  const type = plan?.type || "movie";
  const episodic = type === "series" || type === "anime" || type === "documentary";
  const title = plan?.title_ar || plan?.title_en || "تشغيل الوسائط";
  const titleEn = plan?.title_en || "";
  const poster = resolveAPIURL(plan?.poster_path || plan?.banner_path) || "";
  const backdrop = resolveAPIURL(plan?.banner_path || plan?.poster_path) || poster;
  const plot = plan?.plot_ar || plan?.plot_en || "";
  const genres = Array.isArray(plan?.genres) ? plan.genres : [];

  const files = plan?.files || [];
  const episodes = plan?.episodes || [];
  const seasons = plan?.seasons || [];

  const currentFile = plan?.source || null;

  useEffect(() => {
    let alive = true;
    if (!currentFile?.video_file_id) {
      setSubtitles([]);
      return () => {
        alive = false;
      };
    }
    getFileSubtitles(currentFile.video_file_id)
      .then((res) => {
        if (!alive) return;
        setSubtitles(
          (res.subtitles || []).map((sub) => ({
            kind: "captions",
            label: sub.label || (sub.language === "ar" ? "العربية" : sub.language),
            src: resolveAPIURL(`/api/stream/file/${currentFile.video_file_id}/subtitles/${sub.index}`),
            srcLang: sub.language || "ar",
            default: sub.language === "ar",
          }))
        );
      })
      .catch(() => alive && setSubtitles([]));
    return () => {
      alive = false;
    };
  }, [currentFile?.video_file_id]);

  const streamSrc = currentFile?.video_file_id
    ? resolveAPIURL(`/api/stream/file/${currentFile.video_file_id}`)
    : "";

  // Which episodes the grid shows: the selected season, or everything when the work
  // has no seasons (films, standalone documentaries).
  const visibleEpisodes = useMemo(() => {
    if (!episodic || seasonFilter === null || seasons.length === 0) return episodes;
    return episodes.filter((episode) => episode.season_number === seasonFilter);
  }, [episodic, seasonFilter, seasons.length, episodes]);

  // The side list for a film: its own playable files shaped like episode rows so
  // one card component renders every kind of work.
  const fileCards = useMemo(
    () =>
      files.map((file) => ({
        id: file.video_file_id ?? file.id,
        video_file_id: file.video_file_id ?? file.id,
        episode_id: file.episode_id,
        episode_number: file.episode_number || file.part_number,
        season_number: file.season_number,
        title_ar: file.title_ar,
        title_en: file.title_en,
        duration: file.duration,
        resolution: file.resolution,
        file_size: file.file_size,
        file_count: 1,
        has_local_file: true,
      })),
    [files]
  );

  /** Point the player at a specific release of the current episode. */
  const selectFile = useCallback(
    (target) => {
      if (!target) return;
      // Accept an episode row or a playable file row from either list.
      const wanted = fileKey(target);
      if (!wanted || wanted === fileKey(currentFile)) return;
      // Match on the file id, then on the episode id: the side list hands over an
      // episode whose `video_file_id` may be absent while its `episode_id` is not.
      const file =
        files.find((item) => fileKey(item) === wanted) ||
        (target.episode_id ? files.find((item) => String(item.episode_id) === String(target.episode_id)) : null);
      if (!file) return;
      setPlan((current) => (current ? { ...current, source: file } : current));
      if (file.season_number) setSeasonFilter(file.season_number);
    },
    [files, currentFile]
  );

  /** Playing an episode starts from its catalogue-selected (lowest id) release. */
  const playEpisode = useCallback(
    (episode) => {
      if (!episode || fileKey(episode) === null) return;
      selectFile(episode);
    },
    [selectFile]
  );

  // Next / previous are derived from the ORDERED file list against the CURRENT
  // source, not read from `plan.next` / `plan.previous`.
  //
  // The server computes those two once, for the file the page opened with. The
  // client then swaps `plan.source` locally on every episode click, so a cached
  // `plan.next` kept pointing at the entry that followed the ORIGINAL file — the
  // "next" button walked back to an already-watched episode. Deriving the pair
  // from `files` keeps them correct however many times the source changes.
  const { nextFile, previousFile } = useMemo(() => {
    const index = files.findIndex((file) => fileKey(file) === fileKey(currentFile));
    if (index < 0) {
      // The current file is not in the list (a film opened by an unknown id): fall
      // back to the server's answer, which is still meaningful for the first play.
      return { nextFile: plan?.next || null, previousFile: plan?.previous || null };
    }
    return {
      nextFile: files[index + 1] || null,
      previousFile: files[index - 1] || null,
    };
  }, [files, currentFile, plan?.next, plan?.previous]);

  // The next-episode card shows the catalogue's own episode copy when the work has
  // episodes, and falls back to the file's title for a film with several parts.
  const nextEpisode = useMemo(() => {
    if (!nextFile) return null;
    const episode = episodes.find(
      (item) => fileKey(item) === fileKey(nextFile) || (item.episode_id && String(item.episode_id) === String(nextFile.episode_id))
    );
    return {
      ...(episode || {}),
      season_number: nextFile.season_number,
      episode_number: nextFile.episode_number || nextFile.part_number,
      title_ar: episode?.title_ar || nextFile.title_ar,
      title_en: episode?.title_en || nextFile.title_en,
      duration: nextFile.duration,
      resolution: nextFile.resolution,
      still_path: episode?.still_path,
    };
  }, [nextFile, episodes]);

  const playNext = useCallback(() => {
    if (nextFile) selectFile(nextFile);
  }, [nextFile, selectFile]);
  const playPrevious = useCallback(() => {
    if (previousFile) selectFile(previousFile);
  }, [previousFile, selectFile]);

  const currentIndex = files.findIndex((file) => fileKey(file) === fileKey(currentFile));
  const currentLabel = currentFile
    ? currentFile.title_ar ||
      currentFile.title_en ||
      (currentFile.episode_number
        ? `الحلقة ${currentFile.episode_number}`
        : `${itemNoun(type)} ${currentIndex + 1}`)
    : "";

  // "season" when no later episode of the current season has a file, else "work".
  const endOfLabel = useMemo(() => {
    if (!currentFile) return "work";
    const laterInSeason = episodes.some(
      (episode) =>
        episode.season_number === currentFile.season_number &&
        episode.episode_number > currentFile.episode_number &&
        episode.has_local_file
    );
    return laterInSeason ? "work" : "season";
  }, [currentFile, episodes]);

  const goBack = useCallback(() => {
    if (window.history.state?.idx > 0) navigate(-1);
    else navigate("/");
  }, [navigate]);

  // Hand the current playback to the global dock and stop rendering it here.
  const minimizeToDock = useCallback(() => {
    if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
    if (!streamSrc) return;
    minimize({
      mediaId: id,
      fileId: currentFile?.video_file_id,
      src: streamSrc,
      title,
      poster,
      tracks: subtitles,
      playlist: episodes,
      onSelectFile: playEpisode,
      onNext: playNext,
    });
  }, [id, currentFile?.video_file_id, streamSrc, title, poster, subtitles, episodes, minimize, playEpisode, playNext]);

  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === stageRef.current);
    document.addEventListener("fullscreenchange", sync);
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);

  const wantsFullscreen = params.get("play") === "fs";
  useEffect(() => {
    if (!wantsFullscreen || !streamSrc) return undefined;
    const raf = requestAnimationFrame(() => {
      stageRef.current?.requestFullscreen?.().catch(() => {});
    });
    return () => cancelAnimationFrame(raf);
  }, [wantsFullscreen, streamSrc]);

  const sideTitle = listTitle(type);

  // Technical facts of the current source: the badges and the player's technical
  // panel read these.
  const technical = {
    resolution: currentFile?.resolution || "",
    video_codec: currentFile?.video_codec || "",
    container: currentFile?.file_path ? `.${String(currentFile.file_path).split(".").pop()}` : "",
    duration: currentFile?.duration || 0,
    file_size: currentFile?.file_size || 0,
    audio_track_count: currentFile?.audio_track_count || 0,
    subtitle_count: currentFile?.subtitle_count || 0,
  };

  const playerNode = streamSrc ? (
    <NexoraPlayer
      key={streamSrc}
      src={streamSrc}
      title={title}
      poster={poster}
      tracks={subtitles}
      fileId={currentFile?.video_file_id}
      onNext={playNext}
      playlist={episodes}
      currentFileId={currentFile?.video_file_id}
      onSelectFile={playEpisode}
      fullscreenTarget={stageRef}
      onMinimize={minimizeToDock}
      onExit={() => {
        if (document.fullscreenElement) document.exitFullscreen?.().catch(() => {});
        goBack();
      }}
      sources={plan?.siblings || []}
      onSelectSource={(option) => selectFile({ video_file_id: option.video_file_id })}
      technical={technical}
      hasNext={Boolean(nextFile)}
      hasPrevious={Boolean(previousFile)}
      nextEpisode={nextEpisode}
      onPlayPrevious={playPrevious}
      endOfLabel={endOfLabel}
      onBrowseMore={() => navigate(`/media/${id}`)}
    />
  ) : (
    <div className="nexora-empty-stage">
      <p className="text-sm font-bold">
        {loading
          ? "جارٍ تحضير التشغيل…"
          : failed
            ? "تعذر تحميل بيانات التشغيل لهذا العمل."
            : "لا يتوفر ملف فيديو صالح لهذا العمل."}
      </p>
      {failed && (
        <button type="button" className="nexora-act nexora-act--ghost mt-3" onClick={() => navigate(`/media/${id}`)}>
          العودة إلى صفحة العمل
        </button>
      )}
    </div>
  );

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
            {plan?.release_year ? <span>{plan.release_year}</span> : null}
            {plan?.rating ? <span className="nexora-chip-rating">★ {Number(plan.rating).toFixed(1)}</span> : null}
            {episodic && seasons.length > 0 ? <span>{seasons.length} مواسم</span> : null}
            {episodes.length > 0 ? <span>{episodes.length} {itemNoun(type)}</span> : null}
            {technical.resolution ? <span className="nexora-chip-live">{technical.resolution} · LAN</span> : null}
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
        {/* Side column: seasons + episodes/parts/files. */}
        <aside className="nexora-watch-side">
          <div className="nexora-tabs">
            <span className="nexora-tabs-label">
              {sideTitle} <span>{episodic ? episodes.length : files.length}</span>
            </span>
          </div>

          <>
              {/* Season selector: switching seasons swaps the grid without a page
                  reload, so a work with 20 seasons never renders 500 cards at once. */}
              {episodic && seasons.length > 1 && (
                <div className="nexora-season-bar" role="tablist" aria-label="المواسم">
                  {seasons.map((season) => (
                    <button
                      key={season.season_id}
                      type="button"
                      role="tab"
                      aria-selected={seasonFilter === season.season_number}
                      className={`nexora-season-chip ${seasonFilter === season.season_number ? "is-active" : ""}`}
                      onClick={() => setSeasonFilter(season.season_number)}
                    >
                      {season.poster_path ? (
                        <img
                          className="nexora-season-chip-art"
                          src={resolveAPIURL(season.poster_path)}
                          alt=""
                          loading="lazy"
                        />
                      ) : null}
                      <span className="nexora-season-chip-body">
                        <span className="nexora-season-chip-name">
                          {season.title_ar || season.title_en || `الموسم ${season.season_number}`}
                        </span>
                        <span className="nexora-season-chip-count">
                          {season.local_count}/{season.episode_count} متوفرة
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              )}

              <div className="nexora-watch-list">
                {(episodic ? visibleEpisodes : fileCards).map((item, idx) => (
                  <EpisodeCard
                    key={item.episode_id || fileKey(item) || idx}
                    episode={{ ...item, streamId: fileKey(item) }}
                    index={idx}
                    type={type}
                    active={fileKey(item) !== null && fileKey(item) === fileKey(currentFile)}
                    onPlay={playEpisode}
                  />
                ))}
                {!loading && (episodic ? visibleEpisodes : files).length === 0 && (
                  <div className="p-8 text-center text-xs text-white/40">
                    لا توجد {itemNoun(type)} متوفرة لهذا العمل.
                  </div>
                )}
              </div>
          </>
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
              {/* The transport already owns jumping between episodes: the player
                  bar carries next/previous next to play. A second pair here was
                  redundant, so the summary is purely descriptive. */}
              {currentFile?.duration ? (
                <div className="nexora-watch-summary-actions">
                  <span className="nexora-watch-len">{clock(currentFile.duration)}</span>
                </div>
              ) : null}
            </div>
            {plot ? <p className="nexora-watch-plot">{plot}</p> : null}
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

      {/* Loaded in the background; playback never waits for it. */}
      {plan?.media_id && <RelatedRail mediaId={plan.media_id} type={type} excludeIds={[plan.media_id]} />}
    </div>
  );
}
