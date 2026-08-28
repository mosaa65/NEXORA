import { useEffect, useState } from "react";
import Icon from "../components/Icon.jsx";
import { getMediaDetail, enrichMedia, getMediaMetadataSnapshot, getMediaSeasonMetadata, resolveAPIURL } from "../lib/api.js";
import { horizontalWheel } from "../lib/horizontalScroll.js";

const hasArabicText = (value) => /[\u0600-\u06FF]/.test(value || "");

export default function MediaDetailsPage({
  media,
  searchQuery = "",
  onSearchChange = () => {},
  onOpenCategory,
  onQuickPlay
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
              plot: hasArabicText(data.plot_ar) ? data.plot_ar : (data.plot_en || media.plot || "عمل سينمائي متاح في مكتبة NEXORA المحلية."),
              posterPath: data.poster_path || media.posterPath,
              bannerPath: data.banner_path || media.bannerPath,
              categorySlug: data.category_slug || media.categorySlug || "movies",
              highlights: data.genres?.length > 0 ? data.genres : [],
              seasons: data.seasons || [],
              files: data.files || []
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
        getMediaMetadataSnapshot(media.id, "en-US")
      ]).then(([arabicSnapshot, englishSnapshot]) => {
        if (!alive) return;
        const arabicPayload = arabicSnapshot.status === "fulfilled" ? arabicSnapshot.value?.payload : null;
        const englishPayload = englishSnapshot.status === "fulfilled" ? englishSnapshot.value?.payload : null;
        setTmdb(arabicPayload || englishPayload || null);
        setTmdbEnglish(englishPayload || null);
      });
      Promise.allSettled([
        getMediaSeasonMetadata(media.id, "ar-SA"),
        getMediaSeasonMetadata(media.id, "en-US")
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
          highlights: arabic?.genres?.length > 0 ? arabic.genres : (english.genres?.length > 0 ? english.genres : prev.highlights)
        }));
        Promise.allSettled([
          getMediaMetadataSnapshot(current.id, "ar-SA"),
          getMediaMetadataSnapshot(current.id, "en-US")
        ]).then(([arabicSnapshot, englishSnapshot]) => {
          const arabicPayload = arabicSnapshot.status === "fulfilled" ? arabicSnapshot.value?.payload : null;
          const englishPayload = englishSnapshot.status === "fulfilled" ? englishSnapshot.value?.payload : null;
          setTmdb(arabicPayload || englishPayload || null);
          setTmdbEnglish(englishPayload || null);
        });
        Promise.allSettled([
          getMediaSeasonMetadata(current.id, "ar-SA"),
          getMediaSeasonMetadata(current.id, "en-US")
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
            onClick={() => onOpenCategory ? onOpenCategory("movies") : window.history.back()}
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
            onClick={() => onOpenCategory ? onOpenCategory("movies") : window.history.back()}
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
  const trailers = [...(tmdb?.videos?.results || []), ...(tmdbEnglish?.videos?.results || [])]
    .filter((video, index, videos) => String(video.site || "").toLowerCase() === "youtube" && video.key && videos.findIndex((item) => item.key === video.key) === index);
  const keywords = tmdb?.keywords?.keywords || tmdb?.keywords?.results || [];
  const related = tmdb?.recommendations?.results || tmdb?.similar?.results || [];
  const englishRelatedByID = new Map([...(tmdbEnglish?.recommendations?.results || []), ...(tmdbEnglish?.similar?.results || [])].map((item) => [item.id, item]));
  const alternativeTitles = tmdb?.alternative_titles?.titles || [];
  const reviews = tmdb?.reviews?.results || [];
  const translations = tmdb?.translations?.translations || [];
  const releaseCountry = (tmdb?.release_dates?.results || []).find((entry) => entry.iso_3166_1 === "SA") || (tmdb?.release_dates?.results || [])[0];
  const providerResults = tmdb?.watch_providers?.results || {};
  const providerRegion = providerResults.SA || providerResults.US || Object.values(providerResults)[0];
  const providers = [
    ...(providerRegion?.flatrate || []), ...(providerRegion?.free || []), ...(providerRegion?.ads || []), ...(providerRegion?.rent || []), ...(providerRegion?.buy || [])
  ].filter((provider, index, list) => list.findIndex((item) => item.provider_id === provider.provider_id) === index);
  const crew = tmdb?.credits?.crew || tmdb?.aggregate_credits?.crew || [];
  const imageGallery = [
    ...(tmdbEnglish?.images?.backdrops || tmdb?.images?.backdrops || []).map((image) => ({ ...image, kind: "backdrop", localPath: image.local_backdrop_path })),
    ...(tmdbEnglish?.images?.posters || tmdb?.images?.posters || []).map((image) => ({ ...image, kind: "poster", localPath: image.local_poster_path })),
    ...(tmdbEnglish?.images?.logos || tmdb?.images?.logos || []).map((image) => ({ ...image, kind: "logo", localPath: image.local_logo_path }))
  ];
  const productionCompanies = tmdb?.production_companies || tmdbEnglish?.production_companies || [];
  const productionCountries = tmdb?.production_countries || tmdbEnglish?.production_countries || [];
  const spokenLanguages = tmdb?.spoken_languages || tmdbEnglish?.spoken_languages || [];
  const collection = tmdb?.belongs_to_collection || tmdbEnglish?.belongs_to_collection;
  const publicLists = tmdb?.lists?.results || tmdbEnglish?.lists?.results || [];
  const localizedSeasons = seasonSnapshots.map((snapshot) => snapshot.payload || {}).filter((season) => season.season_number !== undefined);
  const englishSeasonsByNumber = new Map(englishSeasonSnapshots.map((snapshot) => [snapshot.seasonNumber, snapshot.payload || {}]));
  const selectedRemoteSeason = localizedSeasons[selectedMetadataSeason] || localizedSeasons[0];
  const englishSelectedRemoteSeason = selectedRemoteSeason ? englishSeasonsByNumber.get(selectedRemoteSeason.season_number) : null;
  const remoteEpisodes = selectedRemoteSeason?.episodes || [];
  const englishEpisodesByNumber = new Map((englishSelectedRemoteSeason?.episodes || []).map((episode) => [episode.episode_number, episode]));
  const infoFields = [
    ["العنوان الأصلي", tmdb?.original_title || tmdb?.original_name || tmdbEnglish?.original_title || tmdbEnglish?.original_name], ["الشعار", tmdb?.tagline || tmdbEnglish?.tagline], ["الحالة", tmdb?.status || tmdbEnglish?.status], ["اللغة الأصلية", tmdb?.original_language || tmdbEnglish?.original_language],
    ["تاريخ الإصدار", tmdb?.release_date || tmdbEnglish?.release_date || tmdb?.first_air_date || tmdbEnglish?.first_air_date], ["مدة الفيلم", (tmdb?.runtime || tmdbEnglish?.runtime) ? `${tmdb?.runtime || tmdbEnglish?.runtime} دقيقة` : null], ["الميزانية", (tmdb?.budget || tmdbEnglish?.budget) ? `$${Number(tmdb?.budget || tmdbEnglish?.budget).toLocaleString("en-US")}` : null], ["الإيرادات", (tmdb?.revenue || tmdbEnglish?.revenue) ? `$${Number(tmdb?.revenue || tmdbEnglish?.revenue).toLocaleString("en-US")}` : null], ["الشعبية", tmdb?.popularity || tmdbEnglish?.popularity], ["عدد الأصوات", tmdb?.vote_count || tmdbEnglish?.vote_count], ["الموقع الرسمي", tmdb?.homepage || tmdbEnglish?.homepage]
  ].filter(([, value]) => value);
  const posterURL = resolveAPIURL(current.posterPath) || "/nexora-poster-placeholder.PNG";
  const bannerURL = resolveAPIURL(current.bannerPath) || "/nexora-library-backdrop.PNG";
  const englishTitle = current.titleEn || current.titleAr || "Untitled";
  const arabicTitle = hasArabicText(current.titleAr) ? current.titleAr : "لا تتوفر ترجمة عربية لهذا العنوان";
  const tmdbImageURL = (path, size = "w342") => path ? `https://image.tmdb.org/t/p/${size}${path}` : "";

  return (
    <div className="relative mx-auto max-w-[1500px] space-y-6 pb-12 text-right" dir="rtl">
      <div className="flex flex-wrap items-center justify-between gap-3 pb-2">
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
              {current.categorySlug === "movies" ? "الأفلام" : current.categorySlug === "series" ? "المسلسلات" : current.categorySlug === "anime" ? "الأنمي" : current.categorySlug}
            </span>
          </button>
        )}
      </div>

      <section className="luminous-hero relative overflow-hidden rounded-[28px] bg-[#0d0b18] shadow-[0_30px_80px_rgba(17,12,30,0.85)]">
        <div className="absolute inset-0 bg-cover bg-center opacity-40" style={{ backgroundImage: `url('${bannerURL}')` }} />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(236,72,153,0.18),transparent_32%),linear-gradient(90deg,rgba(13,11,24,0.94),rgba(13,11,24,0.73),rgba(13,11,24,0.88))]" />

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
                <div className="rounded-full border border-amber-400/30 bg-amber-500/10 px-3 py-1.5 text-sm font-black text-amber-300">
                  ★ {Number(current.rating || 8.5).toFixed(1)}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 text-[11px] font-bold text-white">
                <span className="rounded-full bg-white/15 px-2.5 py-1 text-white">{current.year || 2023}</span>
                <span className="rounded-full bg-white/15 px-2.5 py-1 text-white">HD</span>
                {(current.highlights || []).slice(0, 4).map((h) => (
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
                className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-violet-600 px-5 py-3 text-sm font-black text-white shadow-lg shadow-fuchsia-900/40 transition hover:brightness-110"
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

      {tmdb && (
        <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
          <div className="rounded-[24px] border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]">
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-lg font-black text-[var(--text-primary)]">معلومات أساسية</h2>
              <span className="rounded-full bg-[var(--bg-elevated)] px-2.5 py-1 text-[10px] text-[var(--text-muted)] font-bold">TMDB</span>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              {[
                ["الحالة", tmdb.status],
                ["اللغة الأصلية", tmdb.original_language],
                ["المدة", tmdb.runtime ? `${tmdb.runtime} دقيقة` : null],
                ["الحلقات", tmdb.number_of_episodes],
                ["المواسم", tmdb.number_of_seasons],
                ["التقييمات", tmdb.vote_count],
                ["تاريخ الإصدار", tmdb.release_date || tmdb.first_air_date],
                ["اللغة المنطوقة", spokenLanguages?.[0]?.name || spokenLanguages?.[0]?.english_name || null]
              ].filter(([, value]) => value).map(([label, value]) => (
                <div key={label} className="rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3">
                  <p className="text-[10px] text-[var(--text-muted)]">{label}</p>
                  <p className="mt-1 text-sm font-bold text-[var(--text-primary)]">{value}</p>
                </div>
              ))}
            </div>

            {keywords.length > 0 && (
              <div className="mt-5">
                <p className="mb-2 text-[11px] font-bold text-[var(--text-muted)]">الكلمات المفتاحية</p>
                <div className="flex flex-wrap gap-2">
                  {keywords.slice(0, 12).map((keyword) => (
                    <span key={keyword.id || keyword.name} className="rounded-full bg-[var(--color-info-light)] border border-cyan-500/20 px-2.5 py-1 text-[11px] font-bold text-[var(--color-info)]">
                      #{keyword.name}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="rounded-[24px] border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]">
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-lg font-black text-[var(--text-primary)]">المقاطع الدعائية</h2>
              <span className="rounded-full bg-fuchsia-500/10 px-2.5 py-1 text-[10px] text-[var(--color-accent)] font-bold">YouTube</span>
            </div>

            {trailers.length > 0 ? (
              <div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2">
                {trailers.slice(0, 4).map((video) => (
                  <article key={video.id || video.key} className="w-[260px] shrink-0 overflow-hidden rounded-2xl border border-[var(--border-default)] bg-[var(--bg-elevated)]">
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
                    <p className="truncate px-3 py-2 text-[11px] font-bold text-[var(--text-primary)]">{video.name || "المقطع الدعائي"}</p>
                  </article>
                ))}
              </div>
            ) : (
              <p className="text-sm text-[var(--text-muted)]">لا توجد مقاطع دعائية متاحة لهذا العمل.</p>
            )}
          </div>
        </section>
      )}

      {cast.length > 0 && (
        <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]">
          <h2 className="text-lg font-black text-[var(--text-primary)]">طاقم العمل</h2>
          <div onWheel={horizontalWheel} className="mt-4 flex gap-3 overflow-x-auto pb-2">
            {cast.slice(0, 20).map((person) => {
              const englishPerson = englishCastByID.get(person.id);
              const profileURL = resolveAPIURL(englishPerson?.local_profile_path || person.local_profile_path) || tmdbImageURL(person.profile_path || englishPerson?.profile_path, "w185");
              return (
                <article key={person.id} className="w-28 sm:w-32 shrink-0 overflow-hidden rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] shadow-sm">
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
                    <p className="truncate text-xs font-bold text-[var(--text-primary)]">{person.name}</p>
                    <p className="mt-0.5 truncate text-[10px] text-[var(--text-muted)]">{person.character || person.roles?.[0]?.character || "طاقم العمل"}</p>
                  </div>
                </article>
              );
            })}
          </div>
        </section>
      )}

      {related.length > 0 && <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">قد يعجبك أيضاً</h2><div onWheel={horizontalWheel} className="mt-4 flex gap-3 overflow-x-auto pb-2">{related.slice(0, 12).map((item) => { const englishItem = englishRelatedByID.get(item.id); const titleEN = englishItem?.title || englishItem?.name || item.original_title || item.original_name || item.title || item.name; const titleAR = hasArabicText(item.title || item.name) ? (item.title || item.name) : "لا تتوفر ترجمة عربية"; const relatedPoster = resolveAPIURL(englishItem?.local_poster_path || item.local_poster_path) || tmdbImageURL(item.poster_path || englishItem?.poster_path); return <article key={item.id} className="w-40 shrink-0 overflow-hidden rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)]"><div className="aspect-[2/3] bg-[var(--bg-surface)]">{relatedPoster ? <img src={relatedPoster} alt={titleEN} className="h-full w-full object-cover" loading="lazy" /> : <div className="flex h-full items-center justify-center text-xs text-[var(--text-muted)]">لا توجد صورة</div>}</div><div className="p-3"><p dir="ltr" className="line-clamp-2 text-left text-xs font-bold text-[var(--text-primary)]">{titleEN}</p><p className="mt-1 line-clamp-2 text-xs text-[var(--text-secondary)]">{titleAR}</p><p className="mt-2 text-xs text-amber-500 font-bold">★ {Number(item.vote_average || englishItem?.vote_average || 0).toFixed(1)}</p></div></article>; })}</div></section>}

      {imageGallery.length > 0 && <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">معرض الصور</h2><p className="mt-1 text-xs text-[var(--text-muted)]">بوسترات وخلفيات وشعارات العمل المحفوظة محلياً عند توفرها.</p><div onWheel={horizontalWheel} className="mt-4 flex gap-3 overflow-x-auto pb-2">{imageGallery.map((image, index) => { const imageURL = resolveAPIURL(image.localPath) || tmdbImageURL(image.file_path, image.kind === "poster" ? "w342" : "w780"); return <figure key={`${image.file_path}-${index}`} className={`shrink-0 overflow-hidden rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] ${image.kind === "poster" ? "w-28" : image.kind === "logo" ? "w-48" : "w-64"}`}><div className={image.kind === "poster" ? "aspect-[2/3]" : "aspect-video"}>{imageURL ? <img src={imageURL} alt={`${image.kind} ${index + 1}`} className="h-full w-full object-cover" loading="lazy" /> : null}</div><figcaption className="px-2 py-1.5 text-[10px] text-[var(--text-muted)]">{image.kind === "poster" ? "بوستر" : image.kind === "logo" ? "شعار" : "خلفية"}</figcaption></figure>; })}</div></section>}

      {crew.length > 0 && <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">فريق الإنتاج</h2><div onWheel={horizontalWheel} className="mt-4 flex gap-3 overflow-x-auto pb-2">{crew.slice(0, 30).map((person, index) => <article key={`${person.credit_id || person.id}-${index}`} className="w-36 shrink-0 rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-3"><p className="truncate text-xs font-bold text-[var(--text-primary)]">{person.name}</p><p className="mt-1 line-clamp-2 text-[11px] text-[var(--color-accent)]">{person.job || person.jobs?.[0]?.job || person.department || "فريق الإنتاج"}</p></article>)}</div></section>}

      {(collection || publicLists.length > 0) && <section className="grid gap-4 lg:grid-cols-2"><div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">السلسلة</h2>{collection ? <div className="mt-4 flex items-center gap-3"><div className="h-16 w-12 shrink-0 overflow-hidden rounded-lg bg-[var(--bg-elevated)]">{collection.poster_path && <img src={tmdbImageURL(collection.poster_path)} alt={collection.name} className="h-full w-full object-cover" loading="lazy" />}</div><p className="text-sm font-bold text-[var(--text-primary)]">{collection.name}</p></div> : <p className="mt-4 text-sm text-[var(--text-muted)]">ليس ضمن سلسلة مسجلة في TMDB.</p>}</div><div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">القوائم العامة</h2>{publicLists.length > 0 ? <div className="mt-4 flex flex-wrap gap-2">{publicLists.map((list) => <span key={list.id} className="rounded-lg bg-[var(--bg-elevated)] px-3 py-1.5 text-xs text-[var(--text-secondary)]">{list.name}</span>)}</div> : <p className="mt-4 text-sm text-[var(--text-muted)]">لا توجد قوائم عامة مرتبطة بهذا العمل.</p>}</div></section>}

      {tmdb && <section className="grid gap-4 xl:grid-cols-3">
        <div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)] xl:col-span-2">
          <h2 className="text-lg font-black text-[var(--text-primary)]">بيانات الفيلم الموسّعة</h2>
          <dl className="mt-4 grid gap-x-6 gap-y-3 text-xs sm:grid-cols-2">{infoFields.map(([label, value]) => <div key={label} className="border-b border-[var(--border-subtle)] pb-2"><dt className="text-[var(--text-muted)]">{label}</dt><dd className="mt-1 break-words font-bold text-[var(--text-primary)]">{value}</dd></div>)}</dl>
          {productionCompanies.length > 0 && <p className="mt-5 text-xs text-[var(--text-secondary)]"><span className="text-[var(--text-muted)]">شركات الإنتاج: </span>{productionCompanies.map((company) => company.name).join("، ")}</p>}
          {productionCountries.length > 0 && <p className="mt-2 text-xs text-[var(--text-secondary)]"><span className="text-[var(--text-muted)]">بلد الإنتاج: </span>{productionCountries.map((country) => country.name).join("، ")}</p>}
          {spokenLanguages.length > 0 && <p className="mt-2 text-xs text-[var(--text-secondary)]"><span className="text-[var(--text-muted)]">لغات الحوار: </span>{spokenLanguages.map((language) => language.name || language.english_name || language.iso_639_1).join("، ")}</p>}
        </div>
        <div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">الإصدارات والتصنيف</h2>{releaseCountry ? <div className="mt-4 space-y-2 text-xs text-[var(--text-secondary)]"><p>المنطقة: <span className="font-bold text-[var(--text-primary)]">{releaseCountry.iso_3166_1}</span></p>{releaseCountry.release_dates?.slice(0, 3).map((release, index) => <p key={`${release.release_date}-${index}`}>{release.release_date?.slice(0, 10)} {release.certification && <span className="mr-2 rounded bg-[var(--bg-elevated)] px-1.5 py-0.5">{release.certification}</span>}</p>)}</div> : <p className="mt-4 text-sm text-[var(--text-muted)]">لا توجد بيانات إصدار محلية لهذا العمل.</p>}</div>
      </section>}

      {tmdb && <section className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">العناوين والترجمات</h2>{alternativeTitles.length > 0 && <div className="mt-4 flex flex-wrap gap-2">{alternativeTitles.slice(0, 16).map((title, index) => <span key={`${title.title}-${index}`} className="rounded-lg bg-[var(--bg-elevated)] px-3 py-1.5 text-xs text-[var(--text-secondary)]">{title.title} <span className="text-[var(--text-muted)]">{title.iso_3166_1}</span></span>)}</div>}<p className="mt-4 text-xs text-[var(--text-muted)]">الترجمات المتاحة في TMDB: {translations.length || 0}</p>{translations.length > 0 && <p className="mt-2 line-clamp-2 text-xs text-[var(--text-secondary)]">{translations.slice(0, 14).map((translation) => translation.name || translation.english_name || translation.iso_639_1).join("، ")}</p>}</div>
        <div className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">المعرّفات الخارجية</h2><div className="mt-4 space-y-2 text-xs">{tmdb?.external_ids?.imdb_id && <a className="block text-[var(--color-info)] hover:underline" target="_blank" rel="noreferrer" href={`https://www.imdb.com/title/${tmdb.external_ids.imdb_id}/`}>IMDB: {tmdb.external_ids.imdb_id} ↗</a>}{Object.entries(tmdb?.external_ids || {}).filter(([key, value]) => value && key !== "imdb_id" && key !== "id").slice(0, 8).map(([key, value]) => <p key={key} className="break-all text-[var(--text-secondary)]">{key}: {String(value)}</p>)}</div></div>
      </section>}

      {providers.length > 0 && <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">أماكن المشاهدة المتاحة</h2><p className="mt-1 text-xs text-[var(--text-muted)]">بيانات مزود المشاهدة مقدمة من JustWatch.</p><div className="mt-4 flex flex-wrap gap-2">{providers.map((provider) => <span key={provider.provider_id} className="rounded-lg bg-[var(--bg-elevated)] px-3 py-1.5 text-xs font-bold text-[var(--text-primary)]">{provider.provider_name}</span>)}</div>{providerRegion?.link && <a href={providerRegion.link} target="_blank" rel="noreferrer" className="mt-4 inline-block text-xs font-bold text-[var(--color-info)] hover:underline">عرض عبر JustWatch ↗</a>}</section>}

      {reviews.length > 0 && <section className="rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><h2 className="text-lg font-black text-[var(--text-primary)]">مراجعات الجمهور</h2><div className="mt-4 grid gap-3 lg:grid-cols-2">{reviews.slice(0, 4).map((review) => <article key={review.id} className="rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-elevated)] p-4"><p className="text-xs font-bold text-[var(--color-accent)]">{review.author_details?.name || review.author || "مستخدم TMDB"}</p><p className="mt-2 line-clamp-4 text-xs leading-6 text-[var(--text-secondary)]">{review.content}</p>{review.url && <a href={review.url} target="_blank" rel="noreferrer" className="mt-3 inline-block text-[11px] text-[var(--color-info)]">المراجعة الكاملة ↗</a>}</article>)}</div></section>}

      {localizedSeasons.length > 0 && <section className="space-y-4 rounded-2xl border border-[var(--border-default)] bg-[var(--bg-card)] p-5 shadow-[var(--shadow-sm)]"><div><h2 className="text-lg font-black text-[var(--text-primary)]">مواسم وحلقات TMDB</h2><p className="mt-1 text-xs text-[var(--text-muted)]">تفاصيل المواسم والحلقات الوصفية محفوظة محلياً؛ التشغيل يظل من ملفات مكتبتك فقط.</p></div><div onWheel={horizontalWheel} className="flex gap-3 overflow-x-auto pb-2">{localizedSeasons.map((season, index) => { const englishSeason = englishSeasonsByNumber.get(season.season_number); const seasonPoster = tmdbImageURL(season.poster_path || englishSeason?.poster_path) || "/nexora-poster-placeholder.PNG"; return <button type="button" key={season.season_number} onClick={() => setSelectedMetadataSeason(index)} className={`w-40 shrink-0 overflow-hidden rounded-xl border text-right transition ${selectedRemoteSeason?.season_number === season.season_number ? "border-[var(--color-accent)] bg-[var(--color-accent-light)]" : "border-[var(--border-default)] bg-[var(--bg-elevated)] hover:border-[var(--color-accent)]"}`}><div className="aspect-[2/3] bg-[var(--bg-surface)]"><img src={seasonPoster} alt={englishSeason?.name || season.name} className="h-full w-full object-cover" loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-poster-placeholder.PNG"; }} /></div><div className="p-3"><p dir="ltr" className="truncate text-left text-xs font-bold text-[var(--text-primary)]">{englishSeason?.name || season.name || `Season ${season.season_number}`}</p><p className="mt-1 truncate text-xs text-[var(--text-muted)]">{hasArabicText(season.name) ? season.name : "لا تتوفر ترجمة عربية"}</p><p className="mt-2 text-[11px] text-[var(--color-accent)] font-bold">{season.episode_count || season.episodes?.length || 0} حلقة</p></div></button>; })}</div>{selectedRemoteSeason && <div><h3 className="text-sm font-black text-[var(--text-primary)]">حلقات: {hasArabicText(selectedRemoteSeason.name) ? selectedRemoteSeason.name : englishSelectedRemoteSeason?.name || selectedRemoteSeason.name}</h3><div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{remoteEpisodes.map((episode) => { const englishEpisode = englishEpisodesByNumber.get(episode.episode_number); const titleEN = englishEpisode?.name || episode.name || `Episode ${episode.episode_number}`; const titleAR = hasArabicText(episode.name) ? episode.name : "لا تتوفر ترجمة عربية"; const still = tmdbImageURL(episode.still_path || englishEpisode?.still_path, "w342") || "/nexora-episode-placeholder.PNG"; return <article key={episode.id || episode.episode_number} className="overflow-hidden rounded-xl border border-[var(--border-default)] bg-[var(--bg-elevated)]"><div className="aspect-video bg-[var(--bg-surface)]"><img src={still} alt={titleEN} className="h-full w-full object-cover" loading="lazy" onError={(event) => { event.currentTarget.src = "/nexora-episode-placeholder.PNG"; }} /></div><div className="p-3"><p dir="ltr" className="truncate text-left text-xs font-bold text-[var(--text-primary)]">{episode.episode_number}. {titleEN}</p><p className="mt-1 truncate text-xs text-[var(--text-muted)]">{titleAR}</p><p className="mt-2 text-[11px] text-[var(--text-muted)]">{episode.air_date || ""} {episode.runtime ? `· ${episode.runtime} دقيقة` : ""}</p><p className="mt-2 line-clamp-2 text-[11px] leading-5 text-[var(--text-secondary)]">{hasArabicText(episode.overview) ? episode.overview : (englishEpisode?.overview || episode.overview || "")}</p></div></article>; })}</div></div>}</section>}

      {/* Seasons Tab & Episode List */}
      {seasonsList.length > 0 && (
        <section className="space-y-4">
          <div className="border-b border-[var(--border-default)] pb-2">
            <h2 className="text-lg font-black text-[var(--text-primary)] sm:text-xl">المواسم والحلقات</h2>
          </div>

          {/* Season Selector Tabs */}
          <div className="flex flex-wrap gap-2">
            {seasonsList.map((season, idx) => (
              <button
                key={season.id || idx}
                type="button"
                onClick={() => setSelectedSeasonIdx(idx)}
                className={`rounded-xl px-4 py-2 text-xs font-bold transition ${
                  selectedSeasonIdx === idx
                    ? "bg-gradient-to-r from-purple-800 to-fuchsia-700 text-white shadow-md shadow-purple-900/40"
                    : "border border-[var(--border-default)] bg-[var(--bg-card)] text-[var(--text-secondary)] hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]"
                }`}
              >
                {season.title_ar || season.title_en || `الموسم ${season.season_number || idx + 1}`}
                <span className="mr-1.5 opacity-60">({season.episodes?.length || 0} حلقة)</span>
              </button>
            ))}
          </div>

          {/* Episodes Grid */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {activeEpisodes.map((ep, epIdx) => (
              <button
                key={ep.id || epIdx}
                type="button"
                onClick={() => onQuickPlay(current, ep)}
                className="group flex items-center justify-between gap-3 rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] p-3 text-right transition hover:border-[var(--color-accent)] hover:bg-[var(--bg-elevated)] shadow-[var(--shadow-sm)]"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-fuchsia-500/10 text-[var(--color-accent)] group-hover:bg-[var(--color-accent)] group-hover:text-white transition">
                  ▶
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-[var(--color-accent)]">
                    {ep.title_ar || ep.title_en || `الحلقة ${ep.episode_number || epIdx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] font-medium text-[var(--text-muted)]">
                    {ep.resolution || "1080p"} • {ep.video_codec || "HEVC/H264"}
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}

      {/* Direct Video Files list if no seasons */}
      {seasonsList.length === 0 && current.files && current.files.length > 0 && (
        <section className="space-y-4">
          <div className="border-b border-[var(--border-default)] pb-2">
            <h2 className="text-lg font-black text-[var(--text-primary)] sm:text-xl">ملفات الفيديو المتاحة</h2>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {current.files.map((file, idx) => (
              <button
                key={file.id || idx}
                type="button"
                onClick={() => onQuickPlay(current, file)}
                className="group flex items-center justify-between gap-3 rounded-xl border border-[var(--border-default)] bg-[var(--bg-card)] p-3 text-right transition hover:border-[var(--color-accent)] hover:bg-[var(--bg-elevated)] shadow-[var(--shadow-sm)]"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-fuchsia-500/10 text-[var(--color-accent)] group-hover:bg-[var(--color-accent)] group-hover:text-white transition">
                  ▶
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-bold text-[var(--text-primary)] group-hover:text-[var(--color-accent)]">
                    {file.title_ar || file.title_en || `ملف التشغيل #${idx + 1}`}
                  </p>
                  <p className="mt-0.5 text-[10px] font-medium text-[var(--text-muted)]">
                    {file.resolution || "1080p"} • {(Number(file.file_size || 0) / (1024 * 1024)).toFixed(1)} MB
                  </p>
                </div>
              </button>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
