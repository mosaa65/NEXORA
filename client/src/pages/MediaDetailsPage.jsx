import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "../components/Icon.jsx";
import RelatedMediaRail from "../components/RelatedMediaRail.jsx";
import RangeSelectionBar from "../components/RangeSelectionBar.jsx";
import { useTransfer } from "../context/TransferContext.jsx";
import { getMediaDetail, enrichMedia, getMediaMetadataSnapshot, getMediaSeasonMetadata, getMediaRelated, searchAllEpisodes, resolveAPIURL } from "../lib/api.js";
import { horizontalWheel } from "../lib/horizontalScroll.js";

const hasArabicText = (value) => /[\u0600-\u06FF]/.test(value || "");
// TMDB orders cast by billing priority. The first 24 are the featured cast
// presented by NEXORA; the complete cast count remains visible in the badge.
const FEATURED_CAST_LIMIT = 24;
// The episode index returns a page at a time, and the endpoint caps a page at
// 200 hits. A long-running show exceeds that, so the whole work is fetched
// across pages rather than truncated to the first one.
const EPISODE_PAGE_SIZE = 200;

function formatSize(bytes) {
  const value = Number(bytes || 0);
  if (!Number.isFinite(value) || value <= 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const unit = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** unit;
  return `${amount >= 10 || unit === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unit]}`;
}

function formatRuntime(minutes) {
  const value = Number(minutes || 0);
  if (!Number.isFinite(value) || value <= 0) return "";
  return value < 60 ? `${value} دقيقة` : `${Math.floor(value / 60)}س${value % 60 ? ` ${value % 60}د` : ""}`;
}

/**
 * Groups the flat episode result into seasons.
 *
 * The episode index returns one row per episode with its season number, so the
 * grouping is a fold over ONE source rather than a reconciliation between local
 * seasons and provider snapshots — which is what the previous screen did, and
 * why it could disagree with itself.
 */
function groupIntoSeasons(hits) {
  const byNumber = new Map();
  for (const episode of hits) {
    const seasonNumber = Number(episode.season_number ?? 0);
    if (!byNumber.has(seasonNumber)) byNumber.set(seasonNumber, []);
    byNumber.get(seasonNumber).push(episode);
  }
  return [...byNumber.entries()]
    .sort((left, right) => left[0] - right[0])
    .map(([seasonNumber, episodes]) => ({
      seasonNumber,
      episodes: [...episodes].sort((left, right) => (left.episode_number || 0) - (right.episode_number || 0)),
      localCount: episodes.filter((episode) => episode.has_local_file).length,
    }));
}

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
  const navigate = useNavigate();

  const handleBack = () => {
    if (window.history.state?.idx > 0 || window.history.length > 1) {
      navigate(-1);
    } else {
      navigate("/");
    }
  };

  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [isEnriching, setIsEnriching] = useState(false);
  const [enrichMsg, setEnrichMsg] = useState("");
  const [tmdb, setTmdb] = useState(null);
  const [tmdbEnglish, setTmdbEnglish] = useState(null);
  // Season posters only. The episode index does not carry artwork, so the
  // provider snapshots supply presentation data — never episode facts.
  const [seasonSnapshots, setSeasonSnapshots] = useState([]);
  // The single season/episode source, exactly as the episode index returned it.
  const [episodeHits, setEpisodeHits] = useState([]);
  const [selectedSeasonNumber, setSelectedSeasonNumber] = useState(null);
  const [episodeQuery, setEpisodeQuery] = useState("");
  const [relatedItems, setRelatedItems] = useState([]);
  // Episode selection drives the existing USB/phone copy flow. A Set of episode
  // ids rather than an index, so filtering or a season switch cannot move the
  // selection onto a different episode.
  const [selectedEpisodeIds, setSelectedEpisodeIds] = useState(() => new Set());
  const [rangeSelection, setRangeSelection] = useState(null);
  const { openTransferModal, selectMultipleFiles } = useTransfer();

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

      // The single season/episode source: every episode of this work, across
      // pages, already enriched and merged by the backend.
      searchAllEpisodes(media.id, { pageSize: EPISODE_PAGE_SIZE })
        .then((hits) => {
          if (alive) setEpisodeHits(hits);
        })
        .catch(() => {
          if (alive) setEpisodeHits([]);
        });

      // Season posters are presentation-only data the episode index does not
      // carry, so they are read separately and are NOT an episode source.
      getMediaSeasonMetadata(media.id, "ar-SA")
        .then((data) => {
          if (alive) setSeasonSnapshots(data?.items || []);
        })
        .catch(() => {
          if (alive) setSeasonSnapshots([]);
        });

      getMediaRelated(media.id).then((data) => {
        if (alive) setRelatedItems(data.items || []);
      }).catch(() => {
        if (alive) setRelatedItems([]);
      });
    } else {
      setLoading(false);
      setRelatedItems([]);
    }

    return () => {
      alive = false;
    };
  }, [media?.id]);

  // Seasons derived from the single source.
  const seasons = useMemo(() => groupIntoSeasons(episodeHits), [episodeHits]);

  // The first season, or the one the viewer picked. Kept as a NUMBER rather
  // than an index, so a re-group cannot silently point at a different season.
  //
  // Season 0 is the provider's bucket for specials, and it sorts first. Opening
  // a show on its specials rather than on season 1 is the wrong default, so the
  // first real season is preferred whenever one exists.
  const defaultSeasonNumber = seasons.find((season) => season.seasonNumber > 0)?.seasonNumber
    ?? seasons[0]?.seasonNumber
    ?? null;
  const activeSeasonNumber = selectedSeasonNumber ?? defaultSeasonNumber;
  const activeSeason = seasons.find((season) => season.seasonNumber === activeSeasonNumber) || null;

  // A local filter over the already-loaded episodes, so typing does not hit the
  // server on every keystroke.
  const visibleEpisodes = useMemo(() => {
    if (!activeSeason) return [];
    const needle = episodeQuery.trim().toLowerCase();
    if (!needle) return activeSeason.episodes;
    return activeSeason.episodes.filter((episode) => {
      const haystack = [episode.episode_title_ar, episode.episode_title_en, String(episode.episode_number)]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(needle);
    });
  }, [activeSeason, episodeQuery]);

  // A work can hold files the episode index does not describe (a file with no
  // season or episode number). They stay reachable through the file list.
  const orphanFiles = useMemo(
    () => (detail?.files || []).filter((file) => !(Number(file.season_number) > 0)),
    [detail]
  );

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
        getMediaSeasonMetadata(current.id, "ar-SA")
          .then((data) => setSeasonSnapshots(data?.items || []))
          .catch(() => {});
        // Enrichment filled episode fields the index may not have carried yet,
        // so the single source is re-read rather than patched locally.
        searchAllEpisodes(detail.id, { pageSize: EPISODE_PAGE_SIZE })
          .then((hits) => setEpisodeHits(hits))
          .catch(() => {});
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
            onClick={handleBack}
            className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-5 py-2.5 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-md)] backdrop-blur-xl"
          >
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-[var(--text-primary)]">‹</span>
            <span>العودة</span>
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
            onClick={handleBack}
            className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] hover:bg-[var(--bg-elevated)] px-5 py-2.5 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-md)] backdrop-blur-xl transition"
          >
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-[var(--text-primary)]">‹</span>
            <span>العودة</span>
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
  const cast = tmdb?.aggregate_credits?.cast || tmdb?.credits?.cast || [];
  const featuredCast = [...cast]
    .sort((left, right) => (Number.isFinite(left?.order) ? left.order : Number.MAX_SAFE_INTEGER) - (Number.isFinite(right?.order) ? right.order : Number.MAX_SAFE_INTEGER))
    .slice(0, FEATURED_CAST_LIMIT);
  const englishCastByID = new Map((tmdbEnglish?.aggregate_credits?.cast || tmdbEnglish?.credits?.cast || []).map((person) => [person.id, person]));
  const trailers = [...(tmdb?.videos?.results || []), ...(tmdbEnglish?.videos?.results || [])].filter(
    (video, index, videos) => String(video.site || "").toLowerCase() === "youtube" && video.key && videos.findIndex((item) => item.key === video.key) === index
  );
  const keywords = tmdb?.keywords?.keywords || tmdb?.keywords?.results || [];
  const crew = tmdb?.credits?.crew || tmdb?.aggregate_credits?.crew || [];
  const imageGallery = [
    ...(tmdbEnglish?.images?.backdrops || tmdb?.images?.backdrops || []).map((image) => ({ ...image, kind: "backdrop", localPath: image.local_backdrop_path })),
    ...(tmdbEnglish?.images?.posters || tmdb?.images?.posters || []).map((image) => ({ ...image, kind: "poster", localPath: image.local_poster_path })),
    ...(tmdbEnglish?.images?.logos || tmdb?.images?.logos || []).map((image) => ({ ...image, kind: "logo", localPath: image.local_logo_path })),
  ];
  const productionCompanies = tmdb?.production_companies || tmdbEnglish?.production_companies || [];
  const networks = tmdb?.networks || tmdbEnglish?.networks || [];
  const createdBy = tmdb?.created_by || tmdbEnglish?.created_by || [];
  const tagline = tmdb?.tagline || tmdbEnglish?.tagline || "";
  const originalTitle = tmdb?.original_title || tmdb?.original_name || tmdbEnglish?.original_title || tmdbEnglish?.original_name || "";
  const imdbId = tmdb?.imdb_id || tmdb?.external_ids?.imdb_id || tmdbEnglish?.imdb_id || tmdbEnglish?.external_ids?.imdb_id || "";
  const productionCountries = tmdb?.production_countries || tmdbEnglish?.production_countries || [];
  const spokenLanguages = tmdb?.spoken_languages || tmdbEnglish?.spoken_languages || [];
  const collection = tmdb?.belongs_to_collection || tmdbEnglish?.belongs_to_collection;
  const contentRating = current.contentRating || current.content_rating || tmdb?.content_rating || "";
  const ratingInfo = getContentRatingInfo(contentRating);
  const posterURL = resolveAPIURL(current.posterPath) || "/nexora-poster-placeholder.PNG";
  const bannerURL = resolveAPIURL(current.bannerPath) || "/nexora-library-backdrop.PNG";
  const englishTitle = current.titleEn || current.titleAr || "Untitled";
  const arabicTitle = hasArabicText(current.titleAr) ? current.titleAr : "لا تتوفر ترجمة عربية لهذا العنوان";
  const tmdbImageURL = (path, size = "w342") => (path ? `https://image.tmdb.org/t/p/${size}${path}` : "");

  // Season posters are keyed by season number and used for presentation only.
  const seasonPosterByNumber = new Map(
    seasonSnapshots.map((snapshot) => [
      Number(snapshot.seasonNumber ?? snapshot.payload?.season_number),
      snapshot.payload?.poster_path || "",
    ])
  );

  const totalEpisodes = seasons.reduce((sum, season) => sum + season.episodes.length, 0);
  const localEpisodes = seasons.reduce((sum, season) => sum + season.localCount, 0);

  // Audio / Subtitles flags
  const hasArAudio = current.hasArabicAudio || (current.highlights || []).some((h) => h.includes("مدبلج") || h.includes("دبلجة") || h.includes("سبيستون") || h.includes("عربي"));

  // An episode without a file cannot play — that is the meaning of the
  // "coming soon" state, so it renders a label instead of a play button.
  const playEpisode = (episode) => {
    if (!episode.has_local_file) return;
    onQuickPlay(current, episode);
  };

  // Only an episode the library holds can be copied; a coming-soon episode has
  // no file to send.
  const copyableEpisodes = visibleEpisodes.filter((episode) => episode.has_local_file);
  const selectedEpisodes = copyableEpisodes.filter((episode) => selectedEpisodeIds.has(episode.id));

  const toggleEpisode = (episode) => {
    if (!episode.has_local_file) return;
    setSelectedEpisodeIds((previous) => {
      const next = new Set(previous);
      if (next.has(episode.id)) next.delete(episode.id);
      else next.add(episode.id);
      return next;
    });
  };

  const clearEpisodeSelection = () => setSelectedEpisodeIds(new Set());

  // "من حلقة → إلى حلقة", by episode NUMBER and within the visible season, which
  // is how an operator thinks about a range.
  const applyEpisodeRange = (from, to) => {
    setRangeSelection(from && to ? { from, to } : null);
    if (!from || !to) return;
    setSelectedEpisodeIds(new Set(
      copyableEpisodes
        .filter((episode) => episode.episode_number >= from && episode.episode_number <= to)
        .map((episode) => episode.id)
    ));
  };

  // Hands the selection to the existing local Copy Bridge flow, which is the
  // same destination the file explorer used to feed.
  const copyEpisodes = (episodes) => {
    const list = (episodes || selectedEpisodes).filter((episode) => episode.has_local_file);
    if (list.length === 0) return;
    const mediaTitle = current.titleAr || current.titleEn || "";
    if (list.length === 1) {
      openTransferModal(list[0], mediaTitle, posterURL);
    } else {
      selectMultipleFiles(list, mediaTitle, posterURL);
      openTransferModal();
    }
  };

  return (
    <div className="relative mx-auto max-w-[1500px] space-y-7 pb-16 text-right" dir="rtl">
      {/* Navigation Top Action */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-1">
        <button
          type="button"
          onClick={handleBack}
          className="group inline-flex items-center gap-2.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-2 text-xs font-bold text-[var(--text-primary)] shadow-[var(--shadow-sm)] backdrop-blur-xl transition hover:border-[var(--color-accent)] hover:bg-[var(--bg-elevated)]"
        >
          <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[var(--bg-elevated)] text-lg leading-none text-[var(--text-primary)] transition group-hover:bg-[var(--color-accent)] group-hover:text-white">‹</span>
          <span>العودة</span>
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
      {/* 1. Main Hero Container (Vertically Centered Poster, Single Rating Badge, Full Plot on Mobile, Single-Row Responsive Buttons) */}
      {/* ========================================================================= */}
      <section className="luminous-hero relative overflow-hidden rounded-[26px] sm:rounded-[32px] bg-[#0c0a17] shadow-[0_30px_80px_rgba(10,8,22,0.9)] border border-[var(--border-default)]">
        {/* Bright, Cinematic Clear Backdrop Artwork */}
        <div
          className="absolute inset-0 bg-cover bg-center transition-all duration-700 opacity-75 sm:opacity-80 scale-105"
          style={{ backgroundImage: `url('${bannerURL}')` }}
        />
        {/* Balanced Cinematic Gradient */}
        <div className="absolute inset-0 bg-gradient-to-t from-[#0c0a17] via-[#0c0a17]/75 to-[#0c0a17]/40 sm:bg-[radial-gradient(ellipse_at_top_right,rgba(236,72,153,0.15),transparent_40%),linear-gradient(90deg,rgba(12,10,23,0.96)_0%,rgba(12,10,23,0.72)_50%,rgba(12,10,23,0.88)_100%)]" />

        {/* Unified Responsive 2-Column Grid with Vertical Centering for Mobile & Desktop */}
        <div className="relative z-10 grid grid-cols-[105px_1fr] sm:grid-cols-[180px_1fr] lg:grid-cols-[220px_1fr] gap-3.5 sm:gap-6 p-4 sm:p-6 lg:p-8 items-center">
          {/* Vertically Centered Poster Column */}
          <div className="luminous-card w-full self-center shrink-0 overflow-hidden rounded-[18px] sm:rounded-[24px] bg-black/40 shadow-[0_20px_50px_rgba(0,0,0,0.6)] border border-white/15 ring-1 ring-white/10">
            <img
              src={posterURL}
              alt={englishTitle}
              className="aspect-[2/3] w-full object-cover shadow-2xl"
              onError={(e) => {
                e.target.src = "/nexora-poster-placeholder.PNG";
              }}
            />
          </div>

          {/* Details Column */}
          <div className="flex flex-col justify-center gap-3 sm:gap-4 min-w-0">
            <div className="space-y-1.5 sm:space-y-2.5">
              {/* Titles Section */}
              <div className="min-w-0">
                <p dir="ltr" className="text-left text-lg font-black tracking-tight text-white sm:text-3xl lg:text-5xl drop-shadow-md truncate">
                  {englishTitle}
                </p>
                <p className="mt-0.5 text-xs font-semibold text-white/95 sm:text-base drop-shadow-sm truncate">
                  {arabicTitle}
                </p>
                {originalTitle && originalTitle !== englishTitle && originalTitle !== arabicTitle && (
                  <p dir="ltr" className="mt-0.5 text-[10px] font-medium text-white/60 sm:text-xs text-left italic truncate">
                    العنوان الأصلي: {originalTitle}
                  </p>
                )}
                {tagline && (
                  <p className="mt-1 text-[11px] sm:text-sm font-medium italic text-fuchsia-300/90">
                    "{tagline}"
                  </p>
                )}
              </div>

              {/* Badges Row (Single Rating Badge Beside Genres, Year, Content Rating) */}
              <div className="flex flex-wrap items-center gap-1.5 sm:gap-2 text-[10px] sm:text-[11px] font-bold text-white pt-0.5">
                <span className="rounded-full bg-white/20 px-2 sm:px-2.5 py-0.5 sm:py-1 text-white shadow-sm">
                  {current.year || 2024}
                </span>

                {/* Single Master Rating Badge */}
                <span className="inline-flex items-center gap-1 rounded-full border border-amber-400/50 bg-amber-500/25 px-2 sm:px-2.5 py-0.5 sm:py-1 font-black text-amber-300 shadow-sm">
                  ★ {Number(current.rating || tmdb?.vote_average || 8.5).toFixed(1)}
                </span>

                {ratingInfo && (
                  <span
                    className={`rounded-full border px-2 sm:px-3 py-0.5 sm:py-1 text-[10px] sm:text-[11px] font-black tracking-wider shadow-sm ${ratingInfo.badgeClass}`}
                    title={ratingInfo.desc}
                  >
                    {ratingInfo.label}
                    <span className="mr-1 text-[9px] sm:text-[10px] font-medium opacity-85 hidden md:inline">({ratingInfo.desc})</span>
                  </span>
                )}

                <span className="rounded-full bg-white/20 px-2 sm:px-2.5 py-0.5 sm:py-1 text-white shadow-sm">
                  {current.type === "series" ? "مسلسل" : "فيلم"}
                </span>

                {(current.highlights || []).slice(0, 5).map((h) => (
                  <span key={h} className="rounded-full border border-fuchsia-400/40 bg-fuchsia-500/25 px-2 sm:px-2.5 py-0.5 sm:py-1 text-fuchsia-100 font-bold shadow-sm">
                    {h}
                  </span>
                ))}
              </div>

              {/* Plot Description - Fully visible on mobile and desktop without truncation */}
              <p className="max-w-3xl text-xs leading-relaxed text-white/95 sm:text-[14px] sm:leading-7 drop-shadow">
                {current.plot}
              </p>
            </div>

            {/* Quick Action Buttons - Guaranteed Single Clean Row on Small Mobile */}
            <div className="flex flex-nowrap items-center gap-1.5 sm:gap-3 pt-1 overflow-x-auto scrollbar-none">
              <button
                type="button"
                onClick={() => onQuickPlay(current)}
                className="inline-flex items-center gap-1.5 shrink-0 rounded-xl sm:rounded-2xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-violet-600 px-3 sm:px-5 py-2 sm:py-3 text-[11px] sm:text-xs font-black text-white shadow-md shadow-fuchsia-900/40 transition hover:brightness-110 active:scale-95 whitespace-nowrap"
              >
                <Icon name="play" className="h-3.5 w-3.5 fill-current text-white" />
                <span>تشغيل الآن</span>
              </button>

              <button
                type="button"
                onClick={() => {
                  const firstEpisode = visibleEpisodes.find((episode) => episode.has_local_file) || visibleEpisodes[0];
                  const playable = (firstEpisode && firstEpisode.has_local_file ? firstEpisode : null)
                    || (current.files && current.files[0])
                    || null;
                  if (seasons.length > 0) {
                    setSelectedSeasonNumber(seasons[0].seasonNumber);
                  } else if (playable) {
                    onQuickPlay(current, playable);
                  }
                }}
                className="inline-flex items-center gap-1.5 shrink-0 rounded-xl sm:rounded-2xl border border-emerald-500/40 bg-emerald-950/60 px-3 sm:px-4 py-2 sm:py-3 text-[11px] sm:text-xs font-black text-emerald-200 transition hover:bg-emerald-900/80 active:scale-95 shadow-md whitespace-nowrap"
              >
                <span>📲</span>
                <span>نسخ إلى الهاتف</span>
              </button>

              <button
                type="button"
                onClick={handleEnrichMetadata}
                disabled={isEnriching}
                className="inline-flex items-center gap-1 shrink-0 rounded-xl sm:rounded-2xl border border-fuchsia-500/40 bg-fuchsia-950/50 px-2.5 sm:px-3.5 py-2 sm:py-3 text-[11px] sm:text-xs font-bold text-fuchsia-200 transition hover:bg-fuchsia-900/60 disabled:cursor-not-allowed disabled:opacity-60 whitespace-nowrap"
              >
                <span>✨</span>
                <span>تحديث TMDB</span>
              </button>

              {imdbId && (
                <a
                  href={`https://www.imdb.com/title/${imdbId}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 shrink-0 rounded-xl sm:rounded-2xl border border-amber-400/40 bg-amber-950/40 px-2.5 sm:px-3 py-2 sm:py-3 text-[11px] sm:text-xs font-black text-amber-300 hover:bg-amber-900/50 transition whitespace-nowrap"
                  title="صفحة IMDb"
                >
                  <span>IMDb</span>
                  <span className="text-[9px]">↗</span>
                </a>
              )}
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
      {/* ========================================================================= */}
      {/* 3. Priority 2: seasons and episodes, in the platform card template */}
      {/* ========================================================================= */}
      {seasons.length > 0 && (
        <section className="space-y-5 rounded-3xl border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3">
            <div>
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-amber-500/20 text-amber-400 text-xs">📺</span>
                المواسم والحلقات
              </h2>
              <p className="mt-0.5 text-xs text-[var(--text-muted)]">
                {localEpisodes} من {totalEpisodes} حلقة متوفرة في المكتبة
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="relative">
                <Icon name="search" className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-muted)]" />
                <input
                  type="search"
                  value={episodeQuery}
                  onChange={(event) => setEpisodeQuery(event.target.value)}
                  placeholder="ابحث في حلقات هذا العمل..."
                  aria-label="filter episodes of this work"
                  className="w-44 sm:w-56 rounded-xl border-[var(--border-default)] bg-[var(--bg-elevated)] py-2 pr-9 pl-3 text-xs text-[var(--text-primary)] outline-none transition focus:border-[var(--color-accent)]"
                />
              </div>
              <RangeSelectionBar
                items={copyableEpisodes}
                onApplyRange={applyEpisodeRange}
                onCopy={copyEpisodes}
              />
            </div>
          </div>

          {/* Season cards — the platform card template, as a horizontal rail */}
          <div onWheel={horizontalWheel} className="flex gap-3.5 overflow-x-auto pb-3 scrollbar-thin">
            {seasons.map((season) => {
              const isActive = season.seasonNumber === activeSeasonNumber;
              const poster = seasonPosterByNumber.get(Number(season.seasonNumber));
              const seasonPoster = resolveAPIURL(poster) || tmdbImageURL(poster, "w342") || "/nexora-poster-placeholder.PNG";
              return (
                <button
                  type="button"
                  key={season.seasonNumber}
                  onClick={() => {
                    setSelectedSeasonNumber(season.seasonNumber);
                    clearEpisodeSelection();
                  }}
                  aria-pressed={isActive}
                  aria-label={`Season ${season.seasonNumber}`}
                  className={`group relative flex w-32 sm:w-40 shrink-0 flex-col overflow-hidden rounded-2xl border text-right transition-all duration-300 ${
                    isActive
                      ? "border-amber-400 bg-amber-950/20 shadow-lg shadow-amber-900/30 ring-2 ring-amber-400 scale-[1.03]"
                      : "border border-[var(--border-default)] bg-[var(--bg-card)] hover:border-amber-400/60 hover:shadow-md"
                  }`}
                >
                  <div className="relative aspect-[3/4] w-full overflow-hidden bg-[#151225]">
                    <img
                      src={seasonPoster}
                      alt={`Season ${season.seasonNumber}`}
                      className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
                      loading="lazy"
                      onError={(event) => {
                        event.currentTarget.src = "/nexora-poster-placeholder.PNG";
                      }}
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/30" />

                    <div className="absolute top-2 right-2">
                      <span className="rounded-md border-white/20 bg-black/60 backdrop-blur-md px-1.5 py-0.5 text-[9px] font-black text-amber-300">
                        {season.episodes.length} حلقة
                      </span>
                    </div>

                    {isActive && (
                      <div className="absolute bottom-2 inset-x-2 flex justify-center">
                        <span className="rounded-full bg-amber-500 px-2.5 py-0.5 text-[9px] font-black text-black shadow">
                          المحدد حالياً ✓
                        </span>
                      </div>
                    )}
                  </div>

                  <div className="p-2 sm:p-2.5 flex-col justify-between flex-1">
                    <p className="truncate text-xs font-black text-[var(--text-primary)] group-hover:text-amber-300 transition">
                      {season.seasonNumber === 0 ? "حلقات خاصة" : `الموسم ${season.seasonNumber}`}
                    </p>
                    <p dir="ltr" className="mt-0.5 truncate text-left text-[10px] text-[var(--text-muted)]">
                      {season.seasonNumber === 0 ? "Specials" : `Season ${season.seasonNumber}`}
                    </p>
                  </div>
                </button>
              );
            })}
          </div>

          {/* Selected season episodes, in the same card template as the catalogue */}
          {activeSeason && (
            <div className="pt-2 border-t border-[var(--border-subtle)]">
              <div className="mb-3 flex-wrap items-center justify-between gap-2">
                <h3 className="text-xs sm:text-sm font-black text-[var(--text-primary)] flex items-center gap-2">
                  <span>حلقات:</span>
                  <span className="text-amber-400 font-extrabold">
                    {activeSeason.seasonNumber === 0 ? "حلقات خاصة" : `الموسم ${activeSeason.seasonNumber}`}
                  </span>
                </h3>
                <div className="flex items-center gap-2">
                  {selectedEpisodes.length > 0 && (
                    <span className="rounded-full bg-[var(--color-accent)]/15 px-2.5 py-1 text-[10px] font-black text-[var(--color-accent)]">
                      {selectedEpisodes.length} محدد
                    </span>
                  )}
                  <span className="text-[11px] font-semibold text-[var(--text-muted)]">
                    {activeSeason.localCount} من {activeSeason.episodes.length} متوفرة
                  </span>
                  {selectedEpisodes.length > 0 && (
                    <button
                      type="button"
                      onClick={() => copyEpisodes(selectedEpisodes)}
                      className="inline-flex items-center gap-1.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 px-3 py-1.5 text-[11px] font-black text-white shadow-sm transition hover:brightness-110 active:scale-95"
                    >
                      <span>📲</span>
                      نسخ المحدد ({selectedEpisodes.length})
                    </button>
                  )}
                </div>
              </div>

              {visibleEpisodes.length === 0 && (
                <p className="py-8 text-center text-xs text-[var(--text-muted)]">
                  {episodeQuery ? "لا توجد حلقات تطابق بحثك في هذا الموسم." : "لا توجد حلقات في هذا الموسم."}
                </p>
              )}

              <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2.5 scrollbar-thin">
                {visibleEpisodes.map((episode) => {
                  const stillURL = resolveAPIURL(episode.still_path) || tmdbImageURL(episode.still_path, "w342") || "/nexora-episode-placeholder.PNG";
                  const titleAR = hasArabicText(episode.episode_title_ar) ? episode.episode_title_ar : null;
                  const titleEN = episode.episode_title_en || `Episode ${episode.episode_number}`;
                  const available = Boolean(episode.has_local_file);
                  const selected = selectedEpisodeIds.has(episode.id);

                  return (
                    <article
                      key={episode.id}
                      className={`w-52 sm:w-60 shrink-0 overflow-hidden rounded-2xl border bg-[var(--bg-elevated)] flex-col justify-between transition ${
                        selected
                          ? "border-[var(--color-accent)] ring-2 ring-[var(--color-accent)]/40"
                          : available
                            ? "border border-[var(--border-default)] hover:border-amber-500/50"
                            : "border border-[var(--border-subtle)] opacity-70"
                      }`}
                    >
                      <div className="aspect-video bg-[var(--bg-surface)] overflow-hidden relative">
                        <img
                          src={stillURL}
                          alt={titleEN}
                          className={`h-full w-full object-cover ${available ? "" : "opacity-60 grayscale"}`}
                          loading="lazy"
                          onError={(event) => {
                            event.currentTarget.src = "/nexora-episode-placeholder.PNG";
                          }}
                        />
                        <span className="absolute top-2 right-2 rounded-md bg-black/80 backdrop-blur-md px-2 py-0.5 text-[10px] font-black text-amber-300 border-white/10">
                          حلقة {episode.episode_number}
                        </span>
                        {episode.file_count > 1 && (
                              <span className="absolute bottom-2 left-2 rounded-md bg-cyan-950/85 px-1.5 py-0.5 text-[9px] font-black text-cyan-200 border-cyan-300/25">
                            {episode.file_count} <span>إصدارات</span>
                          </span>
                        )}
                        {available ? (
                          <button
                            type="button"
                            onClick={() => toggleEpisode(episode)}
                            aria-pressed={selected}
                            aria-label={selected ? "إلغاء التحديد" : "تحديد"}
                            className={`absolute top-2 left-2 flex h-6 w-6 items-center justify-center rounded-lg border-2 backdrop-blur-sm transition ${
                              selected
                                ? "border-[var(--color-accent)] bg-[var(--color-accent)] text-white shadow-md"
                                : "border-white/70 bg-black/45 text-transparent hover:border-[var(--color-accent)]"
                            }`}
                          >
                            <Icon name="checkbox" className="h-3.5 w-3.5" />
                          </button>
                        ) : null}
                      </div>

                      <div className="flex flex-1 flex-col justify-between p-2.5">
                        <div>
                          <p dir="ltr" className="truncate text-left text-xs font-black text-[var(--text-primary)]">{titleEN}</p>
                          <p className="mt-0.5 truncate text-[11px] font-bold text-amber-300/90">{titleAR || "لا تتوفر ترجمة عربية"}</p>
                          {(episode.overview_ar || episode.overview_en) && (
                            <p className="mt-1 line-clamp-2 text-[10px] leading-relaxed text-[var(--text-secondary)]">
                              {episode.overview_ar || episode.overview_en}
                            </p>
                          )}
                        </div>

                        <div className="mt-2 border-t border-[var(--border-subtle)] pt-1.5">
                          {/* dir=ltr on the fact row so a latin value is not
                              reordered by the surrounding RTL context —
                              "2.0 KB" must not render as "KB 2.0". */}
                          <div dir="ltr" className="flex flex-wrap items-center justify-start gap-1.5 text-[9.5px] font-bold text-[var(--text-muted)]">
                            {episode.air_date && <span className="tabular-nums">{episode.air_date}</span>}
                            {formatRuntime(episode.runtime || episode.duration) && <span>{formatRuntime(episode.runtime || episode.duration)}</span>}
                            {episode.resolution && (
                              <span className="rounded border-cyan-300/30 bg-cyan-950/70 px-1.5 py-0.5 text-[9px] font-black text-cyan-100">
                                {episode.resolution}
                              </span>
                            )}
                            {formatSize(episode.file_size) && (
                              <span className="rounded bg-[var(--bg-surface)] px-1.5 py-0.5 tabular-nums">{formatSize(episode.file_size)}</span>
                            )}
                          </div>

                          <div className="mt-2 flex items-center justify-between">
                            {available ? (
                              <button
                                type="button"
                                onClick={() => playEpisode(episode)}
                                className="inline-flex items-center gap-1.5 rounded-lg bg-gradient-to-r from-fuchsia-600 to-purple-600 px-2.5 py-1.5 text-[10.5px] font-black text-white shadow-sm transition hover:brightness-110 active:scale-95"
                              >
                                <Icon name="play" className="h-3 w-3 fill-current text-white" />
                                تشغيل
                              </button>
                            ) : (
                              <span className="rounded-lg border-[var(--border-subtle)] bg-[var(--bg-card)] px-2.5 py-1.5 text-[10px] font-bold text-[var(--text-muted)]">
                                قيد الإضافة
                              </span>
                            )}
                            <span className="text-[9px] font-black text-amber-400">TMDB</span>
                          </div>
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
      {/* Files the episode index does not describe stay reachable here. */}
      {orphanFiles.length > 0 && (
        <section className="space-y-3 rounded-3xl border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <h3 className="border-b border-[var(--border-subtle)] pb-3 text-xs font-black text-[var(--text-muted)]">
            ملفات أخرى في هذا العمل
          </h3>
          <div className="flex flex-wrap gap-2">
            {orphanFiles.map((file) => (
              <button
                key={file.id || file.path || file.file_name}
                type="button"
                onClick={() => onQuickPlay(current, file)}
                className="inline-flex items-center gap-2 rounded-xl border-[var(--border-default)] bg-[var(--bg-elevated)] px-3 py-2 text-[11px] font-bold text-[var(--text-primary)] transition hover:border-[var(--color-accent)]"
              >
                <Icon name="film" className="h-3.5 w-3.5 text-[var(--text-muted)]" />
                <span className="max-w-[220px] truncate">{file.file_name || file.title_en || file.path}</span>
              </button>
            ))}
          </div>
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
                    ترجمة عربية معتمدة + English
                  </p>
                </div>
              </div>
            </div>

            {/* Responsive Specifications Grid (Mobile, Tablet, Laptop) */}
            <div className="grid gap-2.5 grid-cols-2 sm:grid-cols-3">
              {[
                ["نوع العمل", current.type === "series" ? "مسلسل تلفزيوني" : "فيلم سينمائي"],
                ["الحالة الفنية", tmdb.status || "مكتمل"],
                ["إجمالي الأجزاء", seasons.length > 0 ? `${seasons.length} مواسم • ${totalEpisodes} حلقة` : null],
                ["اللغة الأصلية", (tmdb.original_language || "en").toUpperCase()],
                ["مدة العرض", tmdb.runtime ? `${tmdb.runtime} دقيقة` : (tmdb.episode_run_time?.[0] ? `${tmdb.episode_run_time[0]} دقيقة` : "غير محدد")],
                ["تاريخ الإصدار", tmdb.release_date || tmdb.first_air_date || current.year],
                ["مؤشر الرواج", tmdb.popularity ? `🔥 ${Number(tmdb.popularity).toFixed(0)} نقطة تفاعل` : null],
                ["الميزانية", tmdb.budget ? `$${Number(tmdb.budget).toLocaleString("en-US")}` : "غير معلنة"],
                ["الإيرادات", tmdb.revenue ? `$${Number(tmdb.revenue).toLocaleString("en-US")}` : "غير معلنة"],
                ["عدد المقيمين", tmdb.vote_count ? `${Number(tmdb.vote_count).toLocaleString("en-US")} تقييم` : null],
                ["بلد الإنتاج", productionCountries?.map((c) => c.name).slice(0, 2).join("، ") || "عالمي"],
                ["لغات الحوار", spokenLanguages?.map((l) => l.name || l.english_name).slice(0, 3).join("، ") || "متعددة"],
              ]
                .filter(([, val]) => val)
                .map(([label, value]) => (
                  <div key={label} className="rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3 transition hover:border-[var(--color-accent)]/40">
                    <p className="text-[10px] font-bold text-[var(--text-muted)]">{label}</p>
                    <p className="mt-1 text-xs sm:text-sm font-black text-[var(--text-primary)] truncate">{value}</p>
                  </div>
                ))}
            </div>

            {/* Creators & Showrunners if available */}
            {createdBy.length > 0 && (
              <div className="pt-2 border-t border-[var(--border-subtle)]">
                <p className="mb-2 text-xs font-bold text-[var(--text-muted)]">مبتكرو وصناع العمل (Creators)</p>
                <div className="flex flex-wrap gap-2">
                  {createdBy.map((creator) => (
                    <div key={creator.id} className="flex items-center gap-2 rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-subtle)] px-3 py-1.5">
                      <span className="text-xs">✍️</span>
                      <span className="text-xs font-black text-[var(--text-primary)]">{creator.name}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

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

          {/* Companies & Networks Card (With Official Logos) */}
          <div className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)] space-y-4 flex flex-col justify-between">
            <div className="space-y-4">
              {/* Broadcast Networks (For TV Series) */}
              {networks.length > 0 && (
                <div>
                  <div className="border-b border-[var(--border-subtle)] pb-2.5 mb-3">
                    <h2 className="text-xs sm:text-sm font-black text-[var(--text-primary)] flex items-center gap-2">
                      <span className="flex h-6 w-6 items-center justify-center rounded-lg bg-sky-500/20 text-sky-400 text-xs">📡</span>
                      شبكات وقنوات البث الأصلية
                    </h2>
                  </div>
                  <div className="space-y-2">
                    {networks.map((network) => {
                      const netLogo = network.logo_path ? tmdbImageURL(network.logo_path, "w185") : null;
                      return (
                        <div key={network.id} className="flex items-center gap-3 rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-2.5 transition hover:border-sky-500/40">
                          {netLogo ? (
                            <div className="flex h-9 w-16 shrink-0 items-center justify-center rounded-xl bg-white/95 p-1 border border-white/20 shadow-sm">
                              <img src={netLogo} alt={network.name} className="max-h-full max-w-full object-contain" loading="lazy" />
                            </div>
                          ) : (
                            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-sky-500/10 text-sky-300 font-black text-xs">📺</div>
                          )}
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-xs font-black text-[var(--text-primary)]">{network.name}</p>
                            {network.origin_country && (
                              <p className="text-[10px] text-[var(--text-muted)] font-bold">المنشأ: {network.origin_country}</p>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Production Companies & Studios */}
              <div>
                <div className="border-b border-[var(--border-subtle)] pb-2.5 mb-3">
                  <h2 className="text-xs sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                    <span className="flex h-6 w-6 sm:h-7 sm:w-7 items-center justify-center rounded-xl bg-indigo-500/20 text-indigo-400 text-xs">🏢</span>
                    الشركات والاستوديوهات المنتجة
                  </h2>
                  <p className="mt-0.5 text-[11px] sm:text-xs text-[var(--text-muted)]">استوديوهات صناعة هذا العمل (مع شعاراتها الرسمية).</p>
                </div>

                {productionCompanies.length > 0 ? (
                  <div className="space-y-2.5 max-h-[380px] overflow-y-auto pr-1 scrollbar-thin">
                    {productionCompanies.map((company) => {
                      const logoURL = company.logo_path ? tmdbImageURL(company.logo_path, "w185") : null;
                      return (
                        <div
                          key={company.id}
                          className="flex items-center gap-3 rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-2.5 sm:p-3 transition hover:border-fuchsia-500/40"
                        >
                          {logoURL ? (
                            <div className="flex h-10 w-18 sm:h-11 sm:w-20 shrink-0 items-center justify-center rounded-xl bg-white/95 p-1.5 border border-white/20 shadow-sm">
                              <img
                                src={logoURL}
                                alt={company.name}
                                className="max-h-full max-w-full object-contain"
                                loading="lazy"
                              />
                            </div>
                          ) : (
                            <div className="flex h-10 w-10 sm:h-11 sm:w-11 shrink-0 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-300 font-black text-sm border border-indigo-500/20">
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
                  <p className="text-xs text-[var(--text-muted)] py-4 text-center">لا توجد بيانات استوديوهات مسجلة.</p>
                )}
              </div>
            </div>
          </div>
        </section>
      )}

      {/* ========================================================================= */}
      {/* 5. Priority 4: طاقم التمثيل وصناع العمل (Cast & Production Crew) */}
      {/* ========================================================================= */}
      {featuredCast.length > 0 && (
        <section className="rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3 mb-4">
            <div>
              <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-teal-500/20 text-teal-400 text-xs">🎭</span>
                طاقم التمثيل ونجوم العمل
              </h2>
              <p className="mt-1 text-[11px] font-semibold text-[var(--text-muted)]">أبرز الممثلين حسب ترتيب TMDB الرسمي — اختر ممثلًا لاستعراض أعماله المحلية</p>
            </div>
            <span className="shrink-0 rounded-full bg-teal-500/10 px-3 py-1 text-xs text-teal-300 font-black border border-teal-500/20">
              {cast.length} ممثل
            </span>
          </div>

          <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto px-1 pt-4 pb-5 scrollbar-thin">
            {featuredCast.map((person, index) => {
              const englishPerson = englishCastByID.get(person.id);
              const profileURL = resolveAPIURL(englishPerson?.local_profile_path || person.local_profile_path) || tmdbImageURL(person.profile_path || englishPerson?.profile_path, "w185");
              return (
                <button
                  type="button"
                  key={person.credit_id || `${person.id}-${index}`}
                  onClick={() => {
                    if (person.id) window.location.hash = `#/person/tmdb-person-${person.id}`;
                  }}
                  className="group w-32 sm:w-36 lg:w-40 shrink-0 overflow-visible rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] text-right shadow-[var(--shadow-sm)] transition-all duration-300 hover:-translate-y-1 hover:border-teal-400/70 hover:shadow-xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-300"
                  aria-label={`استعراض أعمال ${person.name || "الممثل"} المحلية`}
                >
                  <div className="relative aspect-[3/4] overflow-hidden rounded-t-2xl bg-gradient-to-br from-cyan-950/40 via-purple-950/20 to-fuchsia-950/30">
                    {profileURL ? (
                      <img
                        src={profileURL}
                        alt={person.name}
                        className="h-full w-full object-cover object-top transition duration-300 group-hover:brightness-110"
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
                    <span className="pointer-events-none absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-black/70 via-black/10 to-transparent" />
                    <span className="pointer-events-none absolute bottom-2 right-2 inline-flex translate-y-1 items-center gap-1 rounded-lg border border-white/15 bg-black/45 px-1.5 py-1 text-[9px] font-black text-white opacity-0 backdrop-blur-sm transition duration-200 group-hover:translate-y-0 group-hover:opacity-100 group-focus-visible:translate-y-0 group-focus-visible:opacity-100">
                      <Icon name="film" className="h-3 w-3 text-teal-200" />
                      أعماله
                    </span>
                  </div>
                  <div className="rounded-b-2xl px-2.5 py-3 sm:px-3">
                    <p className="truncate text-xs font-black text-[var(--text-primary)] sm:text-sm">{person.name}</p>
                    <p className="mt-1 truncate text-[10px] font-medium text-[var(--text-muted)] sm:text-[11px]">{person.character || person.roles?.[0]?.character || "طاقم التمثيل"}</p>
                  </div>
                </button>
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

      {/* ========================================================================= */}
      {/* 7. Collection (السلسلة السينمائية) & Related Movies (الأعمال المقترحة) */}
      {/* Fully Mobile-Optimized with Dual Arabic/English Titles & Luxury Cards */}
      {/* ========================================================================= */}
      {(collection || relatedItems.length > 0) && (
        <section className="space-y-6">
          {/* Movie Collection Banner Card */}
          {collection && (
            <div className="relative overflow-hidden rounded-3xl border border-[var(--border-default)] bg-[var(--bg-card)] p-4 sm:p-6 shadow-[var(--shadow-sm)]">
              {collection.backdrop_path && (
                <div
                  className="absolute inset-0 bg-cover bg-center opacity-30 blur-sm scale-105 pointer-events-none"
                  style={{ backgroundImage: `url('${tmdbImageURL(collection.backdrop_path, "w780")}')` }}
                />
              )}
              <div className="relative z-10">
                <div className="flex items-center justify-between gap-3 border-b border-[var(--border-subtle)] pb-3 mb-4">
                  <h2 className="text-base sm:text-lg font-black text-[var(--text-primary)] flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-xl bg-amber-500/20 text-amber-400 text-xs">🍿</span>
                    السلسلة والأجزاء السينمائية
                  </h2>
                  <span className="rounded-full bg-amber-500/10 px-3 py-1 text-xs font-black text-amber-300 border border-amber-500/20">
                    عالم سينمائي متكامل
                  </span>
                </div>

                <div className="flex items-center gap-3.5 sm:gap-5 rounded-2xl bg-[var(--bg-elevated)]/90 border border-[var(--border-subtle)] p-3 sm:p-4 backdrop-blur-md">
                  {collection.poster_path ? (
                    <div className="h-20 w-14 sm:h-24 sm:w-16 shrink-0 overflow-hidden rounded-xl bg-black/40 border border-white/10 shadow-md">
                      <img src={tmdbImageURL(collection.poster_path)} alt={collection.name} className="h-full w-full object-cover" loading="lazy" />
                    </div>
                  ) : null}
                  <div className="min-w-0 flex-1">
                    <span className="rounded-md bg-amber-500/20 px-2 py-0.5 text-[10px] font-black text-amber-300 border border-amber-500/30">
                      سلسلة أفلام
                    </span>
                    <h3 className="mt-1.5 text-sm sm:text-base font-black text-[var(--text-primary)] truncate">
                      {collection.name}
                    </h3>
                    <p className="mt-0.5 text-[11px] text-[var(--text-muted)]">
                      كافة أجزاء وأفلام السلسلة السينمائية المرتبطة بهذا العمل.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          <RelatedMediaRail items={relatedItems} onOpen={(mediaID) => { window.location.hash = `#/media/${mediaID}`; }} />
        </section>
      )}

    </div>
  );
}

