import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import { getMediaDetail, enrichMedia, getMediaMetadataSnapshot, getMediaSeasonMetadata, resolveAPIURL } from "../lib/api.js";
import { horizontalWheel } from "../lib/horizontalScroll.js";

const hasArabicText = (value) => /[\u0600-\u06FF]/.test(value || "");

function getContentRatingInfo(rating) {
  if (!rating) return null;
  const key = String(rating).trim().toUpperCase();
  if (["G", "TV-G", "TV-Y", "ALL"].includes(key)) {
    return {
      label: key,
      desc: "مناسب لجميع الأعمار (عائلي)",
      badgeClass: "border-emerald-400/40 bg-emerald-950/70 text-emerald-200",
    };
  }
  if (["PG", "TV-PG", "TV-Y7"].includes(key)) {
    return {
      label: key,
      desc: "إشراف عائلي موصى به",
      badgeClass: "border-sky-400/40 bg-sky-950/70 text-sky-200",
    };
  }
  if (["PG-13", "TV-14", "13+", "12"].includes(key)) {
    return {
      label: key,
      desc: "غير مناسب لمن هم دون 13 عاماً",
      badgeClass: "border-amber-400/40 bg-amber-950/70 text-amber-200",
    };
  }
  if (["R", "TV-MA", "NC-17", "18+", "18", "MA"].includes(key)) {
    return {
      label: key,
      desc: "للبالغين فقط (+18)",
      badgeClass: "border-rose-500/40 bg-rose-950/80 text-rose-200",
    };
  }
  return {
    label: key,
    desc: `تصنيف عمري: ${key}`,
    badgeClass: "border-white/20 bg-white/10 text-white/90",
  };
}

export default function MediaDetailsPage({
  media,
  onOpenCategory,
  onQuickPlay,
}) {
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [selectedSeasonIdx, setSelectedSeasonIdx] = useState(0);
  const [isEnriching, setIsEnriching] = useState(false);
  const [enrichMsg, setEnrichMsg] = useState("");
  const [tmdb, setTmdb] = useState(null);
  const [tmdbEnglish, setTmdbEnglish] = useState(null);
  const [seasonSnapshots, setSeasonSnapshots] = useState([]);
  const [englishSeasonSnapshots, setEnglishSeasonSnapshots] = useState([]);
  const [selectedMetadataSeason, setSelectedMetadataSeason] = useState(0);

  useEffect(() => {
    let alive = true;
    if (media?.id) {
      setLoading(true);
      getMediaDetail(media.id)
        .then((data) => {
          if (!alive) return;
          if (data && data.id) {
            setDetail({
              id: data.id,
              titleAr: hasArabicText(data.title_ar) ? data.title_ar : (hasArabicText(media.titleAr) ? media.titleAr : ""),
              titleEn: data.title_en || media.titleEn || "",
              type: data.type || media.type || "movie",
              year: data.release_year || media.year || 2024,
              rating: data.rating || media.rating || 0,
              contentRating: data.content_rating || data.contentRating || media.contentRating || media.content_rating || "",
              plot: hasArabicText(data.plot_ar) ? data.plot_ar : (data.plot_en || media.plot || "عمل سينمائي متاح في مكتبة NEXORA المحلية."),
              posterPath: data.poster_path || media.posterPath,
              bannerPath: data.banner_path || media.bannerPath,
              categorySlug: data.category_slug || media.categorySlug || "movies",
              highlights: data.genres?.length > 0 ? data.genres : [],
              seasons: data.seasons || [],
              files: data.files || [],
              hasArabicAudio: data.has_arabic_audio,
              hasArabicSubtitles: data.has_arabic_subtitles,
            });
          } else {
            setDetail(null);
          }
        })
        .catch(() => {
          if (alive) setDetail(null);
        })
        .finally(() => {
          if (alive) setLoading(false);
        });

      Promise.allSettled([
        getMediaMetadataSnapshot(media.id, "ar-SA"),
        getMediaMetadataSnapshot(media.id, "en-US"),
      ]).then(([arabicSnapshot, englishSnapshot]) => {
        if (!alive) return;
        const arabicPayload = arabicSnapshot.status === "fulfilled" ? arabicSnapshot.value?.payload : null;
        const englishPayload = englishSnapshot.status === "fulfilled" ? englishSnapshot.value?.payload : null;
        setTmdb(arabicPayload || englishPayload || null);
        setTmdbEnglish(englishPayload || null);
      });

      Promise.allSettled([
        getMediaSeasonMetadata(media.id, "ar-SA"),
        getMediaSeasonMetadata(media.id, "en-US"),
      ]).then(([arabicSeasons, englishSeasons]) => {
        if (!alive) return;
        setSeasonSnapshots(arabicSeasons.status === "fulfilled" ? arabicSeasons.value?.items || [] : []);
        setEnglishSeasonSnapshots(englishSeasons.status === "fulfilled" ? englishSeasons.value?.items || [] : []);
      });
    } else {
      setLoading(false);
    }

    return () => {
      alive = false;
    };
  }, [media?.id]);

  async function handleEnrichMetadata() {
    if (!current?.id) return;
    setIsEnriching(true);
    setEnrichMsg("جارٍ جلب البيانات والبوستر...");
    try {
      const res = await enrichMedia(current.id);
      const allMetadata = Array.isArray(res.metadata) ? res.metadata : [res.metadata].filter(Boolean);
      const arabic = allMetadata.find((item) => item.locale === "ar-SA");
      const english = allMetadata.find((item) => item.locale === "en-US") || allMetadata[0];
      if (res.ok && english) {
        setEnrichMsg("✅ تم تحديث البيانات والبوستر بنجاح!");
        setDetail((prev) => ({
          ...prev,
          titleEn: english.title || prev.titleEn,
          titleAr: hasArabicText(arabic?.title) ? arabic.title : prev.titleAr,
          plot: hasArabicText(arabic?.overview) ? arabic.overview : (english.overview || prev.plot),
          rating: english.rating || prev.rating,
          year: english.releaseYear || prev.year,
          posterPath: english.cachedPosterPath || english.posterPath || prev.posterPath,
          bannerPath: english.cachedBannerPath || english.bannerPath || prev.bannerPath,
          highlights: arabic?.genres?.length > 0 ? arabic.genres : (english.genres?.length > 0 ? english.genres : prev.highlights),
        }));
        Promise.allSettled([
          getMediaMetadataSnapshot(current.id, "ar-SA"),
          getMediaMetadataSnapshot(current.id, "en-US"),
        ]).then(([arabicSnapshot, englishSnapshot]) => {
          const arabicPayload = arabicSnapshot.status === "fulfilled" ? arabicSnapshot.value?.payload : null;
          const englishPayload = englishSnapshot.status === "fulfilled" ? englishSnapshot.value?.payload : null;
          setTmdb(arabicPayload || englishPayload || null);
          setTmdbEnglish(englishPayload || null);
        });
        Promise.allSettled([
          getMediaSeasonMetadata(current.id, "ar-SA"),
          getMediaSeasonMetadata(current.id, "en-US"),
        ]).then(([arabicSeasons, englishSeasons]) => {
          setSeasonSnapshots(arabicSeasons.status === "fulfilled" ? arabicSeasons.value?.items || [] : []);
          setEnglishSeasonSnapshots(englishSeasons.status === "fulfilled" ? englishSeasons.value?.items || [] : []);
        });
      } else {
        setEnrichMsg("لم يتم العثور على تطابق.");
      }
    } catch (err) {
      const providerError = String(err?.payload?.error || err?.message || "");
      if (providerError.includes("api.themoviedb.org") || providerError.includes("dial tcp") || providerError.includes("socket")) {
        setEnrichMsg("تعذر وصول خادم NEXORA إلى TMDB — تحقق من الإنترنت أو جدار الحماية.");
      } else {
        setEnrichMsg("تعذر تحديث بيانات TMDB: " + (providerError || "خطأ غير معروف"));
      }
    } finally {
      setIsEnriching(false);
      setTimeout(() => setEnrichMsg(""), 4000);
    }
  }

  if (loading) {
    return (
      <div className="space-y-6 text-right animate-pulse pb-16" dir="rtl">
        <div className="flex items-center justify-between gap-4 pb-2">
          <button
            type="button"
            onClick={() => (onOpenCategory ? onOpenCategory("movies") : window.history.back())}
            className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-5 py-2.5 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-md)] backdrop-blur-xl"
          >
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-[var(--text-primary)]">‹</span>
            <span>العودة للمكتبة</span>
          </button>
        </div>
        <div className="h-96 rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] flex items-center justify-center">
          <div className="text-center space-y-3">
            <div className="w-10 h-10 border-4 border-fuchsia-500/30 border-t-fuchsia-500 rounded-full animate-spin mx-auto" />
            <p className="text-xs text-[var(--text-muted)]">جارٍ جلب تفاصيل العمل من المكتبة...</p>
          </div>
        </div>
      </div>
    );
  }

  if (!detail) {
    return (
      <div className="space-y-6 text-right pb-16" dir="rtl">
        <div className="flex items-center justify-between gap-4 pb-2">
          <button
            type="button"
            onClick={() => (onOpenCategory ? onOpenCategory("movies") : window.history.back())}
            className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] hover:bg-[var(--bg-elevated)] px-5 py-2.5 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-md)] backdrop-blur-xl transition"
          >
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-[var(--text-primary)]">‹</span>
            <span>العودة للمكتبة</span>
          </button>
        </div>
        <div className="p-16 rounded-3xl border border-dashed border-[var(--border-default)] bg-[var(--bg-card)] text-center space-y-3">
          <span className="text-4xl">🎬</span>
          <h2 className="text-lg font-bold text-[var(--text-primary)]">لم يتم العثور على العمل المطلوب</h2>
          <p className="text-xs text-[var(--text-muted)] max-w-sm mx-auto">
            قد يكون هذا العمل قد تم حذفه أو نقله من مجلدات المكتبة.
          </p>
        </div>
      </div>
    );
  }

  const current = detail;
  const seasonsList = current.seasons && current.seasons.length > 0 ? current.seasons : [];
  const currentSeason = seasonsList[selectedSeasonIdx] || seasonsList[0];
  const activeEpisodes = currentSeason?.episodes || current.files || [];
  const cast = tmdb?.aggregate_credits?.cast || tmdb?.credits?.cast || [];
  const englishCastByID = new Map((tmdbEnglish?.aggregate_credits?.cast || tmdbEnglish?.credits?.cast || []).map((person) => [person.id, person]));
  const trailers = [...(tmdb?.videos?.results || []), ...(tmdbEnglish?.videos?.results || [])].filter(
    (video, index, videos) => String(video.site || "").toLowerCase() === "youtube" && video.key && videos.findIndex((item) => item.key === video.key) === index
  );
  const keywords = tmdb?.keywords?.keywords || tmdb?.keywords?.results || [];
  const related = tmdb?.recommendations?.results || tmdb?.similar?.results || [];
  const englishRelatedByID = new Map([...(tmdbEnglish?.recommendations?.results || []), ...(tmdbEnglish?.similar?.results || [])].map((item) => [item.id, item]));
  const crew = tmdb?.credits?.crew || tmdb?.aggregate_credits?.crew || [];
  const imageGallery = [
    ...(tmdbEnglish?.images?.backdrops || tmdb?.images?.backdrops || []).map((image) => ({ ...image, kind: "backdrop", localPath: image.local_backdrop_path })),
    ...(tmdbEnglish?.images?.posters || tmdb?.images?.posters || []).map((image) => ({ ...image, kind: "poster", localPath: image.local_poster_path })),
    ...(tmdbEnglish?.images?.logos || tmdb?.images?.logos || []).map((image) => ({ ...image, kind: "logo", localPath: image.local_logo_path })),
  ];
  const productionCompanies = tmdb?.production_companies || tmdbEnglish?.production_companies || [];
  const productionCountries = tmdb?.production_countries || tmdbEnglish?.production_countries || [];
  const spokenLanguages = tmdb?.spoken_languages || tmdbEnglish?.spoken_languages || [];
  const collection = tmdb?.belongs_to_collection || tmdbEnglish?.belongs_to_collection;
  const localizedSeasons = seasonSnapshots.map((snapshot) => snapshot.payload || {}).filter((season) => season.season_number !== undefined);
  const englishSeasonsByNumber = new Map(englishSeasonSnapshots.map((snapshot) => [snapshot.seasonNumber, snapshot.payload || {}]));
  const selectedRemoteSeason = localizedSeasons[selectedMetadataSeason] || localizedSeasons[0];
  const englishSelectedRemoteSeason = selectedRemoteSeason ? englishSeasonsByNumber.get(selectedRemoteSeason.season_number) : null;
  const remoteEpisodes = selectedRemoteSeason?.episodes || [];
  const englishEpisodesByNumber = new Map((englishSelectedRemoteSeason?.episodes || []).map((episode) => [episode.episode_number, episode]));
  const contentRating = current.contentRating || current.content_rating || tmdb?.content_rating || "";
  const ratingInfo = getContentRatingInfo(contentRating);
  const posterURL = resolveAPIURL(current.posterPath) || "/nexora-poster-placeholder.PNG";
  const bannerURL = resolveAPIURL(current.bannerPath) || "/nexora-library-backdrop.PNG";
  const englishTitle = current.titleEn || current.titleAr || "Untitled";
  const arabicTitle = hasArabicText(current.titleAr) ? current.titleAr : "لا تتوفر ترجمة عربية لهذا العنوان";
  const tmdbImageURL = (path, size = "w342") => (path ? `https://image.tmdb.org/t/p/${size}${path}` : "");

  // Audio / Subtitles flags
  const hasArAudio = current.hasArabicAudio || (current.highlights || []).some((h) => h.includes("مدبلج") || h.includes("دبلجة") || h.includes("سبيستون") || h.includes("عربي"));
  const hasArSubs = current.hasArabicSubtitles || true; // NEXORA default subtitle engine

  return (
    <div className="relative mx-auto max-w-[1500px] space-y-7 pb-16 text-right" dir="rtl">
      {/* Navigation Top Action */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-1">
        <button
          type="button"
          onClick={() => {
            if (onOpenCategory) {
              onOpenCategory(current.categorySlug || "movies");
            } else {
              window.history.back();
            }
          }}
          className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-2.5 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-sm)] backdrop-blur-xl transition hover:border-[var(--color-accent)] hover:bg-[var(--bg-elevated)]"
        >
          <span className="flex h-7 w-7 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-lg leading-none text-[var(--text-primary)] transition group-hover:bg-[var(--color-accent)] group-hover:text-white">‹</span>
          <span>العودة للمكتبة</span>
        </button>

        {current.categorySlug && (
          <button
            type="button"
            onClick={() => onOpenCategory && onOpenCategory(current.categorySlug)}
            className="inline-flex items-center gap-2 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-3.5 py-1.5 text-[11px] font-semibold text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition hover:border-[var(--color-accent)] hover:text-[var(--text-primary)]"
          >
            <span>القسم</span>
            <span className="font-black text-[var(--color-accent)]">
              {current.categorySlug === "movies" ? "الأفلام السينمائية" : current.categorySlug === "series" ? "المسلسلات والدراما" : current.categorySlug === "anime" ? "الأنمي والرسوم اليابانية" : current.categorySlug === "kids" ? "الأطفال والكرتون" : current.categorySlug === "family" ? "العائلة والسينما العائلية" : current.categorySlug}
            </span>
          </button>
        )}
      </div>

      {/* ========================================================================= */}
      {/* 1. Main Hero Container (Preserved structure + Added rating badge down beside genres) */}
      {/* ========================================================================= */}
      <section className="luminous-hero relative overflow-hidden rounded-[28px] bg-[#0d0b18] shadow-[0_30px_80px_rgba(17,12,30,0.85)] border border-[var(--border-default)]">
        <div className="absolute inset-0 bg-cover bg-center opacity-40" style={{ backgroundImage: `url('${bannerURL}')` }} />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(236,72,153,0.18),transparent_32%),linear-gradient(90deg,rgba(13,11,24,0.95),rgba(13,11,24,0.75),rgba(13,11,24,0.90))]" />

        <div className="relative z-10 grid gap-6 p-4 sm:p-6 lg:grid-cols-[220px_1fr] lg:p-8">
          <div className="luminous-card mx-auto w-[180px] shrink-0 overflow-hidden rounded-[22px] bg-black/20 shadow-[0_30px_50px_rgba(168,85,247,0.3)] sm:w-[200px] lg:mx-0">
            <img
              src={posterURL}
              alt={englishTitle}
              className="aspect-[2/3] w-full object-cover"
              onError={(e) => {
                e.target.src = "/nexora-poster-placeholder.PNG";
              }}
            />
          </div>

          <div className="flex flex-col justify-between gap-5">
            <div className="space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <p dir="ltr" className="text-left text-2xl font-black tracking-tight text-white sm:text-3xl lg:text-5xl">{englishTitle}</p>
                  <p className="mt-2 text-sm font-medium text-white sm:text-base drop-shadow-sm">{arabicTitle}</p>
                </div>
                <div className="rounded-full border border-amber-400/30 bg-amber-500/10 px-3.5 py-1.5 text-sm font-black text-amber-300 shadow-sm">
                  ★ {Number(current.rating || 8.5).toFixed(1)}
                </div>
              </div>

              {/* Badges Row (Includes Rating Badge down next to genres) */}
              <div className="flex flex-wrap items-center gap-2 text-[11px] font-bold text-white">
                <span className="rounded-full bg-white/15 px-2.5 py-1 text-white">{current.year || 2024}</span>
                
                {/* Rating Badge directly beside genres */}
                <span className="inline-flex items-center gap-1 rounded-full border border-amber-400/40 bg-amber-500/20 px-2.5 py-1 font-black text-amber-300 shadow-sm">
                  ★ {Number(current.rating || 8.5).toFixed(1)}
                </span>

                {ratingInfo && (
                  <span
                    className={`rounded-full border px-3 py-1 text-[11px] font-black tracking-wider shadow-sm ${ratingInfo.badgeClass}`}
                    title={ratingInfo.desc}
                  >
                    {ratingInfo.label}
                    <span className="mr-1.5 text-[10px] font-medium opacity-85 hidden sm:inline">({ratingInfo.desc})</span>
                  </span>
                )}
                <span className="rounded-full bg-white/15 px-2.5 py-1 text-white">HD</span>
                {(current.highlights || []).slice(0, 5).map((h) => (
                  <span key={h} className="rounded-full border border-fuchsia-400/30 bg-fuchsia-500/20 px-2.5 py-1 text-fuchsia-100 font-bold">
                    {h}
                  </span>
                ))}
              </div>

              <p className="max-w-3xl text-sm leading-7 text-white sm:text-[15px] drop-shadow-sm">{current.plot}</p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={() => onQuickPlay(current)}
                className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-violet-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-fuchsia-900/40 transition hover:brightness-110 active:scale-95"
              >
                <Icon name="play" className="h-4 w-4 fill-current text-white" />
                <span>تشغيل الآن</span>
              </button>

              <button
                type="button"
                onClick={handleEnrichMetadata}
                disabled={isEnriching}
                className="inline-flex items-center gap-2 rounded-2xl border border-fuchsia-500/40 bg-fuchsia-950/40 px-4 py-3 text-xs font-bold text-fuchsia-200 transition hover:bg-fuchsia-900/60 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <span>✨</span>
                <span>تحديث بيانات TMDB</span>
              </button>
            </div>

            {enrichMsg && <p className="text-xs font-bold text-fuchsia-200">{enrichMsg}</p>}
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* 2. Priority 1: المقاطع الدعائية الرسمية (Official Trailers & Teasers) */}
      {/* ========================================================================= */}
      {trailers.length > 0 && (
        <section className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="mb-4 flex items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3">
            <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2.5">
              <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-red-500/20 text-red-400 text-xs">▶</span>
              المقاطع الدعائية الرسمية (Trailers)
            </h2>
            <span className="rounded-full bg-red-500/10 px-3 py-1 text-[11px] text-red-400 font-black border border-red-500/20">
              YouTube • {trailers.length} مقاطع
            </span>
          </div>

          <div onWheel={horizontalWheel} className="flex gap-3.5 overflow-x-auto pb-2 scrollbar-thin">
            {trailers.slice(0, 5).map((video) => (
              <article key={video.id || video.key} className="w-[260px] sm:w-[320px] shrink-0 overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] shadow-sm">
                <div className="aspect-video bg-black">
                  <iframe
                    className="h-full w-full"
                    src={`https://www.youtube-nocookie.com/embed/${encodeURIComponent(video.key)}?rel=0`}
                    title={video.name || "YouTube trailer"}
                    loading="lazy"
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                    allowFullScreen
                  />
                </div>
                <div className="p-2.5">
                  <p className="truncate text-xs font-bold text-[var(--text-primary)]">{video.name || "المقطع الدعائي"}</p>
                  <p className="mt-0.5 text-[10px] text-[var(--text-muted)]">{video.type || "Trailer"} • جودة عالية</p>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}

      {/* ========================================================================= */}
      {/* 3. Priority 2: الأجزاء والمواسم والحلقات (Seasons & Compact Episodes List) */}
      {/* ========================================================================= */}
      {/* Local Video Files / Seasons */}
      {seasonsList.length > 0 && (
        <section className="space-y-4 rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3">
            <div>
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-fuchsia-500/20 text-fuchsia-400 text-xs">📺</span>
                المواسم وحلقات التشغيل المتاحة
              </h2>
              <p className="mt-0.5 text-xs text-[var(--text-muted)]">حلقات الفيديو المحلية الجاهزة للدفق المباشر بأعلى جودة.</p>
            </div>
            <span className="rounded-full bg-fuchsia-500/10 px-3.5 py-1 text-xs font-black text-fuchsia-300 border border-fuchsia-500/20">
              {seasonsList.length} مواسم متوفرة
            </span>
          </div>

          {/* Season Selector Tabs */}
          <div className="flex flex-wrap gap-2 pt-1">
            {seasonsList.map((season, idx) => (
              <button
                key={season.id || idx}
                type="button"
                onClick={() => setSelectedSeasonIdx(idx)}
                className={`rounded-2xl px-4 py-2 text-xs font-bold transition flex items-center gap-2 ${
                  selectedSeasonIdx === idx
                    ? "bg-gradient-to-r from-fuchsia-600 to-purple-600 text-white shadow-md shadow-fuchsia-900/40 scale-[1.02]"
                    : "border border-[var(--border-default)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:border-fuchsia-500/50 hover:text-[var(--text-primary)]"
                }`}
              >
                <span>{season.title_ar || season.title_en || `الموسم ${season.season_number || idx + 1}`}</span>
                <span className="rounded-full bg-black/30 px-2 py-0.5 text-[10px] font-black">
                  {season.episodes?.length || 0} حلقة
                </span>
              </button>
            ))}
          </div>

          {/* Compact, Beautiful Episodes Grid (Refined size, prominent episode badge) */}
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 pt-2">
            {activeEpisodes.map((ep, epIdx) => (
              <button
                key={ep.id || epIdx}
                type="button"
                onClick={() => onQuickPlay(current, ep)}
                className="group flex items-center gap-3 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] p-2.5 text-right transition hover:border-fuchsia-500/60 hover:bg-[var(--bg-card)] hover:shadow-md active:scale-95"
              >
                {/* Compact Episode Thumbnail */}
                <div className="relative h-14 w-20 shrink-0 overflow-hidden rounded-xl bg-black/40 border border-white/10">
                  <img
                    src={posterURL}
                    alt=""
                    className="h-full w-full object-cover opacity-75 group-hover:scale-105 transition duration-300"
                  />
                  <div className="absolute inset-0 flex items-center justify-center bg-black/35 group-hover:bg-black/10 transition">
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white shadow text-[10px] font-black">
                      ▶
                    </span>
                  </div>
                </div>

                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5 mb-1">
                    <span className="rounded-md bg-gradient-to-r from-fuchsia-600 to-purple-600 px-2 py-0.5 text-[10px] font-black text-white shadow-sm shrink-0">
                      حلقة {ep.episode_number || epIdx + 1}
                    </span>
                    <span className="rounded bg-white/10 px-1.5 py-0.2 text-[9px] font-bold text-[var(--text-muted)] truncate">
                      {ep.resolution || "1080p"}
                    </span>
                  </div>
                  <p className="truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-fuchsia-300 transition">
                    {ep.title_ar || ep.title_en || `الحلقة ${ep.episode_number || epIdx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] text-[var(--text-muted)] truncate">
                    {ep.video_codec || "HEVC/H264"} • تشغيل فوري
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}

      {/* Direct Video Files list if no seasons */}
      {seasonsList.length === 0 && current.files && current.files.length > 0 && (
        <section className="space-y-4 rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="border-b border-[var(--border-subtle)] pb-3">
            <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-fuchsia-500/20 text-fuchsia-400 text-xs">🎬</span>
              ملفات الفيديو المتاحة للتشغيل
            </h2>
          </div>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-3">
            {current.files.map((file, idx) => (
              <button
                key={file.id || idx}
                type="button"
                onClick={() => onQuickPlay(current, file)}
                className="group flex items-center gap-3 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] p-3 text-right transition hover:border-fuchsia-500/60 hover:bg-[var(--bg-card)] shadow-sm active:scale-95"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-fuchsia-600 to-purple-600 text-white font-black text-xs group-hover:scale-105 transition shadow">
                  ▶
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-fuchsia-300">
                    {file.title_ar || file.title_en || `ملف التشغيل #${idx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] text-[var(--text-muted)]">
                    {file.resolution || "1080p"} • {(Number(file.file_size || 0) / (1024 * 1024)).toFixed(1)} MB
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}

      {/* TMDB Rich Descriptive Seasons & Episodes (Metadata view with dual Arabic/English/Original support) */}
      {localizedSeasons.length > 0 && (
        <section className="space-y-4 rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3">
            <div>
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-amber-500/20 text-amber-400 text-xs">✨</span>
                أدلة ومواسم TMDB التفصيلية
              </h2>
              <p className="mt-0.5 text-xs text-[var(--text-muted)]">دليل وصفي متكامل للقصص والحلقات مع الترجمة العربية والإنجليزية والأصلية.</p>
            </div>
            <span className="rounded-full bg-amber-500/10 px-3.5 py-1 text-xs font-black text-amber-300 border border-amber-500/20">
              {localizedSeasons.length} مواسم في الدليل
            </span>
          </div>

          <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
            {localizedSeasons.map((season, index) => {
              const englishSeason = englishSeasonsByNumber.get(season.season_number);
              const seasonPoster = tmdbImageURL(season.poster_path || englishSeason?.poster_path) || "/nexora-poster-placeholder.PNG";
              const isSelected = selectedRemoteSeason?.season_number === season.season_number;
              const sTitleEN = englishSeason?.name || `Season ${season.season_number}`;
              const sTitleAR = hasArabicText(season.name) ? season.name : "دليل الموسم";
              return (
                <button
                  type="button"
                  key={season.season_number}
                  onClick={() => setSelectedMetadataSeason(index)}
                  className={`w-32 sm:w-36 shrink-0 overflow-hidden rounded-2xl border text-right transition ${
                    isSelected
                      ? "border-fuchsia-500 bg-fuchsia-950/40 shadow-lg ring-1 ring-fuchsia-500 scale-[1.02]"
                      : "border-[var(--border-default)] bg-[var(--bg-elevated)] hover:border-fuchsia-500/50"
                  }`}
                >
                  <div className="aspect-[2/3] bg-[var(--bg-surface)]">
                    <img
                      src={seasonPoster}
                      alt={sTitleEN}
                      className="h-full w-full object-cover"
                      loading="lazy"
                      onError={(e) => {
                        e.currentTarget.src = "/nexora-poster-placeholder.PNG";
                      }}
                    />
                  </div>
                  <div className="p-2.5">
                    <p dir="ltr" className="truncate text-left text-xs font-bold text-[var(--text-primary)]">
                      {sTitleEN}
                    </p>
                    <p className="mt-0.5 truncate text-[10px] text-[var(--text-muted)]">
                      {sTitleAR}
                    </p>
                    <p className="mt-1 text-[10px] text-fuchsia-400 font-black">
                      {season.episode_count || season.episodes?.length || 0} حلقة
                    </p>
                  </div>
                </button>
              );
            })}
          </div>

          {selectedRemoteSeason && (
            <div className="pt-2">
              <h3 className="text-xs sm:text-sm font-black text-[var(--text-primary)] mb-3 flex items-center gap-2">
                <span>حلقات:</span>
                <span className="text-fuchsia-400">{hasArabicText(selectedRemoteSeason.name) ? selectedRemoteSeason.name : (englishSelectedRemoteSeason?.name || selectedRemoteSeason.name)}</span>
                {englishSelectedRemoteSeason?.name && hasArabicText(selectedRemoteSeason.name) && (
                  <span className="text-xs text-[var(--text-muted)] font-normal" dir="ltr">({englishSelectedRemoteSeason.name})</span>
                )}
              </h3>
              <div className="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {remoteEpisodes.map((episode) => {
                  const englishEpisode = englishEpisodesByNumber.get(episode.episode_number);
                  const titleEN = englishEpisode?.name || episode.name || `Episode ${episode.episode_number}`;
                  const titleAR = hasArabicText(episode.name) ? episode.name : null;
                  const originalName = episode.original_name || null;
                  const still = tmdbImageURL(episode.still_path || englishEpisode?.still_path, "w342") || "/nexora-episode-placeholder.PNG";
                  const overviewAR = hasArabicText(episode.overview) ? episode.overview : null;
                  const overviewEN = englishEpisode?.overview || episode.overview || "";
                  
                  return (
                    <article key={episode.id || episode.episode_number} className="overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)] flex flex-col justify-between hover:border-fuchsia-500/40 transition">
                      <div className="aspect-video bg-[var(--bg-surface)] overflow-hidden relative">
                        <img src={still} alt={titleEN} className="h-full w-full object-cover" loading="lazy" onError={(e) => { e.currentTarget.src = "/nexora-episode-placeholder.PNG"; }} />
                        <span className="absolute top-2 right-2 rounded-md bg-black/70 backdrop-blur-md px-2 py-0.5 text-[10px] font-black text-amber-300 border border-white/10">
                          حلقة {episode.episode_number}
                        </span>
                      </div>
                      <div className="p-2.5 flex-1 flex flex-col justify-between">
                        <div>
                          {/* Dual / Multi-lingual Title Display */}
                          <p dir="ltr" className="truncate text-left text-xs font-black text-[var(--text-primary)]">{titleEN}</p>
                          {titleAR ? (
                            <p className="mt-0.5 truncate text-[11px] font-bold text-fuchsia-300">{titleAR}</p>
                          ) : originalName && originalName !== titleEN ? (
                            <p dir="ltr" className="mt-0.5 truncate text-left text-[10px] text-[var(--text-muted)] italic font-semibold">{originalName}</p>
                          ) : (
                            <p className="mt-0.5 truncate text-[10px] text-[var(--text-muted)]">لا تتوفر ترجمة عربية لعنوان الحلقة</p>
                          )}
                          <p className="mt-1 line-clamp-2 text-[10px] leading-4 text-[var(--text-muted)]">
                            {overviewAR || overviewEN || "لا يتوفر وصف نصي لهذه الحلقة."}
                          </p>
                        </div>
                        <div className="mt-2 flex items-center justify-between pt-2 border-t border-[var(--border-subtle)] text-[10px] text-[var(--text-muted)]">
                          <span>{episode.air_date || "تاريخ العرض"}</span>
                          <span>{episode.runtime ? `${episode.runtime} د` : "المدة قياسية"}</span>
                        </div>
                      </div>
                    </article>
                  );
                })}
              </div>
            </div>
          )}
        </section>
      )}

      {/* ========================================================================= */}
      {/* 4. Priority 3: معلومات أساسية وبيانات العمل واللغات والصوتيات */}
      {/* ========================================================================= */}
      {tmdb && (
        <section className="grid gap-4 lg:grid-cols-3">
          {/* Main Info Card (2 columns on large screens) */}
          <div className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)] lg:col-span-2 space-y-5">
            <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3">
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-purple-500/20 text-purple-400 text-xs">ℹ️</span>
                معلومات وبيانات العمل الفني
              </h2>
              <span className="rounded-full bg-[var(--bg-elevated)] px-3 py-1 text-[10px] text-[var(--text-muted)] font-black border border-[var(--border-subtle)]">
                بيانات معتمدة
              </span>
            </div>

            {/* Languages & Audio Dual Banner */}
            <div className="grid gap-2.5 sm:grid-cols-2">
              <div className="flex items-center gap-3 rounded-2xl border border-emerald-500/30 bg-emerald-950/30 p-3 text-emerald-200">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-emerald-500/20 text-lg">
                  🎙️
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-bold text-emerald-300">مسارات الصوت والدبلجة</p>
                  <p className="text-xs font-black text-white truncate">
                    {hasArAudio ? "صوت ودبلجة عربية متوفرة" : "اللغة الأصلية + الإنجليزية"}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-3 rounded-2xl border border-sky-500/30 bg-sky-950/30 p-3 text-sky-200">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-sky-500/20 text-lg">
                  📝
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-bold text-sky-300">الترجمة والنصوص (Subtitles)</p>
                  <p className="text-xs font-black text-white truncate">
                    {hasArSubs ? "ترجمة عربية معتمدة + English" : "الإنجليزية والأصلية"}
                  </p>
                </div>
              </div>
            </div>

            {/* Responsive Specifications Grid (Mobile, Tablet, Laptop) */}
            <div className="grid gap-2.5 grid-cols-2 sm:grid-cols-3">
              {[
                ["الحالة الفنية", tmdb.status || "مكتمل"],
                ["اللغة الأصلية", (tmdb.original_language || "en").toUpperCase()],
                ["مدة العمل", tmdb.runtime ? `${tmdb.runtime} دقيقة` : (tmdb.episode_run_time?.[0] ? `${tmdb.episode_run_time[0]} دقيقة` : "غير محدد")],
                ["تاريخ الإصدار", tmdb.release_date || tmdb.first_air_date || current.year],
                ["الميزانية", tmdb.budget ? `$${Number(tmdb.budget).toLocaleString("en-US")}` : "غير معلنة"],
                ["الإيرادات", tmdb.revenue ? `$${Number(tmdb.revenue).toLocaleString("en-US")}` : "غير معلنة"],
                ["عدد المقيمين", tmdb.vote_count ? `${Number(tmdb.vote_count).toLocaleString("en-US")} تقييم` : null],
                ["بلد الإنتاج", productionCountries?.[0]?.name || "عالمي"],
                ["لغات الحوار", spokenLanguages?.map((l) => l.name || l.english_name).slice(0, 2).join("، ") || "متعددة"],
              ]
                .filter(([, val]) => val)
                .map(([label, value]) => (
                  <div key={label} className="rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3 transition hover:border-[var(--color-accent)]/40">
                    <p className="text-[10px] font-bold text-[var(--text-muted)]">{label}</p>
                    <p className="mt-1 text-xs sm:text-sm font-black text-[var(--text-primary)] truncate">{value}</p>
                  </div>
                ))}
            </div>

            {/* Keywords & Tags */}
            {keywords.length > 0 && (
              <div className="pt-2">
                <p className="mb-2 text-xs font-bold text-[var(--text-muted)]">الكلمات المفتاحية والوسوم</p>
                <div className="flex flex-wrap gap-1.5">
                  {keywords.slice(0, 14).map((keyword) => (
                    <span key={keyword.id || keyword.name} className="rounded-xl bg-fuchsia-500/10 border border-fuchsia-500/20 px-2.5 py-1 text-[11px] font-bold text-fuchsia-300">
                      #{keyword.name}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Companies & Production Studios Card (With Logo Image Display) */}
          <div className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)] space-y-4 flex flex-col justify-between">
            <div>
              <div className="border-b border-[var(--border-subtle)] pb-3 mb-4">
                <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                  <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-indigo-500/20 text-indigo-400 text-xs">🏢</span>
                  الشركات والاستوديوهات المنتجة
                </h2>
                <p className="mt-0.5 text-xs text-[var(--text-muted)]">استوديوهات وشركات صناعة هذا العمل (مع شعاراتها الرسمية).</p>
              </div>

              {productionCompanies.length > 0 ? (
                <div className="space-y-2.5">
                  {productionCompanies.map((company) => {
                    const logoURL = company.logo_path ? tmdbImageURL(company.logo_path, "w185") : null;
                    return (
                      <div
                        key={company.id}
                        className="flex items-center gap-3 rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3 transition hover:border-fuchsia-500/40"
                      >
                        {logoURL ? (
                          <div className="flex h-11 w-20 shrink-0 items-center justify-center rounded-xl bg-white/95 p-1.5 border border-white/20 shadow-sm">
                            <img
                              src={logoURL}
                              alt={company.name}
                              className="max-h-full max-w-full object-contain"
                              loading="lazy"
                            />
                          </div>
                        ) : (
                          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-300 font-black text-sm border border-indigo-500/20">
                            🏢
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-xs font-black text-[var(--text-primary)]">{company.name}</p>
                          {company.origin_country && (
                            <p className="text-[10px] text-[var(--text-muted)] font-bold mt-0.5">الدولة: {company.origin_country}</p>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <p className="text-xs text-[var(--text-muted)] py-6 text-center">لا توجد بيانات استوديوهات مسجلة.</p>
              )}
            </div>
          </div>
        </section>
      )}

      {/* ========================================================================= */}
      {/* 5. Priority 4: طاقم التمثيل وصناع العمل (Cast & Production Crew) */}
      {/* ========================================================================= */}
      {cast.length > 0 && (
        <section className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3 mb-4">
            <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-teal-500/20 text-teal-400 text-xs">🎭</span>
              طاقم التمثيل ونجوم العمل
            </h2>
            <span className="rounded-full bg-teal-500/10 px-3 py-1 text-xs text-teal-300 font-black border border-teal-500/20">
              {cast.length} ممثل
            </span>
          </div>

          <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
            {cast.slice(0, 24).map((person) => {
              const englishPerson = englishCastByID.get(person.id);
              const profileURL = resolveAPIURL(englishPerson?.local_profile_path || person.local_profile_path) || tmdbImageURL(person.profile_path || englishPerson?.profile_path, "w185");
              return (
                <article key={person.id} className="w-28 sm:w-32 shrink-0 overflow-hidden rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] shadow-sm hover:border-[var(--color-accent)]/50 transition">
                  <div className="relative aspect-[4/5] bg-gradient-to-br from-cyan-950/40 via-purple-950/20 to-fuchsia-950/30 overflow-hidden">
                    {profileURL ? (
                      <img
                        src={profileURL}
                        alt={person.name}
                        className="h-full w-full object-cover"
                        loading="lazy"
                        onError={(e) => {
                          e.currentTarget.style.display = "none";
                          if (e.currentTarget.nextElementSibling) e.currentTarget.nextElementSibling.classList.remove("hidden");
                        }}
                      />
                    ) : null}
                    <div className={`${profileURL ? "hidden" : ""} absolute inset-0 flex items-center justify-center p-2`}>
                      <span className="flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] text-[var(--color-info)] shadow-inner">
                        <Icon name="user" className="h-5 w-5 sm:h-6 sm:w-6" />
                      </span>
                    </div>
                  </div>
                  <div className="p-2 sm:p-2.5">
                    <p className="truncate text-xs font-black text-[var(--text-primary)]">{person.name}</p>
                    <p className="mt-0.5 truncate text-[10px] text-[var(--text-muted)]">{person.character || person.roles?.[0]?.character || "طاقم التمثيل"}</p>
                  </div>
                </article>
              );
            })}
          </div>
        </section>
      )}

      {/* Production Crew */}
      {crew.length > 0 && (
        <section className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between border-b border-[var(--border-subtle)] pb-3 mb-4">
            <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-orange-500/20 text-orange-400 text-xs">🎬</span>
              فريق الإخراج والإنتاج
            </h2>
            <span className="rounded-full bg-orange-500/10 px-3 py-1 text-xs text-orange-300 font-black border border-orange-500/20">
              {crew.length} مخرج وفني
            </span>
          </div>

          <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
            {crew.slice(0, 24).map((person, index) => (
              <article key={`${person.credit_id || person.id}-${index}`} className="w-36 shrink-0 rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3 hover:border-orange-500/40 transition">
                <p className="truncate text-xs font-black text-[var(--text-primary)]">{person.name}</p>
                <p className="mt-1 line-clamp-2 text-[11px] text-[var(--color-accent)] font-bold">{person.job || person.jobs?.[0]?.job || person.department || "فريق الإنتاج"}</p>
              </article>
            ))}
          </div>
        </section>
      )}

      {/* ========================================================================= */}
      {/* 6. Priority 5: معرض الصور والسلاسل والأعمال المشابهة */}
      {/* ========================================================================= */}
      {imageGallery.length > 0 && (
        <section className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] mb-3 flex items-center gap-2">
            <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-sky-500/20 text-sky-400 text-xs">🖼️</span>
            معرض الصور والبوسترات والشعارات
          </h2>
          <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
            {imageGallery.map((image, index) => {
              const imageURL = resolveAPIURL(image.localPath) || tmdbImageURL(image.file_path, image.kind === "poster" ? "w342" : "w780");
              return (
                <figure key={`${image.file_path}-${index}`} className={`shrink-0 overflow-hidden rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] ${image.kind === "poster" ? "w-28 sm:w-32" : image.kind === "logo" ? "w-44 sm:w-48" : "w-60 sm:w-64"}`}>
                  <div className={image.kind === "poster" ? "aspect-[2/3]" : "aspect-video"}>
                    {imageURL ? <img src={imageURL} alt="" className="h-full w-full object-cover" loading="lazy" /> : null}
                  </div>
                  <figcaption className="px-2 py-1.5 text-[10px] font-bold text-[var(--text-muted)] text-center bg-[var(--bg-card)]/50">
                    {image.kind === "poster" ? "بوستر رسمي" : image.kind === "logo" ? "شعار العمل" : "خلفية سينمائية"}
                  </figcaption>
                </figure>
              );
            })}
          </div>
        </section>
      )}

      {/* Collection and Related Movies */}
      {(collection || related.length > 0) && (
        <section className="grid gap-4 lg:grid-cols-3">
          {collection && (
            <div className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]">
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] mb-3 flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-amber-500/20 text-amber-400 text-xs">🍿</span>
                السلسلة السينمائية
              </h2>
              <div className="flex items-center gap-3 p-2.5 rounded-2xl bg-[var(--bg-elevated)] border border-[var(--border-subtle)]">
                {collection.poster_path && (
                  <div className="h-16 w-12 shrink-0 overflow-hidden rounded-xl bg-black/20">
                    <img src={tmdbImageURL(collection.poster_path)} alt={collection.name} className="h-full w-full object-cover" loading="lazy" />
                  </div>
                )}
                <p className="text-xs sm:text-sm font-black text-[var(--text-primary)]">{collection.name}</p>
              </div>
            </div>
          )}

          {related.length > 0 && (
            <div className={`rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)] ${collection ? "lg:col-span-2" : "lg:col-span-3"}`}>
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] mb-3 flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-fuchsia-500/20 text-fuchsia-400 text-xs">✨</span>
                أعمال مقترحة ذات صلة
              </h2>
              <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
                {related.slice(0, 10).map((item) => {
                  const englishItem = englishRelatedByID.get(item.id);
                  const titleEN = englishItem?.title || englishItem?.name || item.original_title || item.original_name || item.title || item.name;
                  const titleAR = hasArabicText(item.title || item.name) ? item.title || item.name : "لا تتوفر ترجمة عربية";
                  const relatedPoster = resolveAPIURL(englishItem?.local_poster_path || item.local_poster_path) || tmdbImageURL(item.poster_path || englishItem?.poster_path);
                  return (
                    <article key={item.id} className="w-32 sm:w-36 shrink-0 overflow-hidden rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] hover:border-fuchsia-500/50 transition">
                      <div className="aspect-[2/3] bg-[var(--bg-surface)]">
                        {relatedPoster ? <img src={relatedPoster} alt={titleEN} className="h-full w-full object-cover" loading="lazy" /> : <div className="flex h-full items-center justify-center text-xs text-[var(--text-muted)]">لا توجد صورة</div>}
                      </div>
                      <div className="p-2.5">
                        <p dir="ltr" className="truncate text-left text-xs font-black text-[var(--text-primary)]">{titleEN}</p>
                        <p className="mt-0.5 truncate text-[10px] text-[var(--text-secondary)]">{titleAR}</p>
                        <p className="mt-1 text-[11px] text-amber-400 font-black">★ {Number(item.vote_average || englishItem?.vote_average || 0).toFixed(1)}</p>
                      </div>
                    </article>
                  );
                })}
              </div>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
