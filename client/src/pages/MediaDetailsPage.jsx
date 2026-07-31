import { useEffect, useMemo, useState } from "react";
import GlassCard from "../components/GlassCard.jsx";
import Icon from "../components/Icon.jsx";
import VideoPlayer from "../components/VideoPlayer.jsx";
import { detailEpisodes, getCategoryTitleAr, getMediaTypeLabel } from "../data/library.js";
import { getMediaFiles, resolveAPIURL } from "../lib/api.js";

export default function MediaDetailsPage({ media, onOpenCategory }) {
  const seasons = media?.seasons?.length ? media.seasons : [1, 2, 3, 4];
  const categoryLabel = getCategoryTitleAr(media.categorySlug);
  const [videoFiles, setVideoFiles] = useState([]);
  const [filesState, setFilesState] = useState("idle");
  const [activeFileID, setActiveFileID] = useState(null);
  const activeFile = videoFiles.find((file) => file.id === activeFileID) || videoFiles[0];
  const streamSrc = activeFile?.stream_url ? resolveAPIURL(activeFile.stream_url) : "";
  const subtitleTracks = useMemo(() => normalizeSubtitleTracks(activeFile?.subtitles), [activeFile]);

  useEffect(() => {
    let alive = true;
    setFilesState("loading");

    getMediaFiles(media.id)
      .then((payload) => {
        if (alive) {
          const files = payload.files || [];
          setVideoFiles(files);
          setActiveFileID(files[0]?.id || null);
          setFilesState("ready");
        }
      })
      .catch(() => {
        if (alive) {
          setVideoFiles([]);
          setActiveFileID(null);
          setFilesState("error");
        }
      });

    return () => {
      alive = false;
    };
  }, [media.id]);

  return (
    <div className="space-y-6">
      <GlassCard className="relative overflow-hidden p-0">
        <div className="absolute inset-0" style={{ backgroundImage: media.gradient }} />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,255,255,0.2),transparent_28%),radial-gradient(circle_at_bottom_left,rgba(25,183,255,0.14),transparent_30%),linear-gradient(180deg,rgba(4,7,18,0.08),rgba(4,7,18,0.86))]" />

        <div className="relative grid gap-6 p-6 lg:grid-cols-[0.78fr_1.22fr] lg:p-8">
          <div className="relative min-h-[30rem] overflow-hidden rounded-[2rem] border border-white/10 bg-black/28 p-5 shadow-panel">
            <div className="absolute inset-0" style={{ backgroundImage: media.gradient }} />
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.14),transparent_30%),radial-gradient(circle_at_bottom_right,rgba(25,183,255,0.18),transparent_34%),linear-gradient(180deg,rgba(4,7,18,0.08),rgba(4,7,18,0.86))]" />
            <div className="absolute inset-0 ring-1 ring-inset ring-white/10" />

            <div className="relative flex h-full flex-col justify-between">
              <div className="flex items-center justify-between">
                <button
                  type="button"
                  onClick={() => onOpenCategory(media.categorySlug)}
                  className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/30 px-3 py-2 text-sm font-semibold text-white/80 transition hover:bg-white/[0.06]"
                >
                  <Icon name="arrowLeft" className="h-4 w-4" />
                  العودة
                </button>
                <div className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-xs text-white/75">
                  {media.year}
                </div>
              </div>

              <div className="rounded-[1.75rem] border border-white/10 bg-black/24 p-5 text-right backdrop-blur-sm">
                <p className="text-xs font-semibold text-white/45">{categoryLabel}</p>
                <h1 className="mt-3 text-4xl font-black text-white md:text-5xl">{media.titleAr}</h1>
                <p className="mt-2 text-sm font-semibold text-white/60">{media.titleEn}</p>
                <p className="mt-4 line-clamp-4 text-sm leading-7 text-white/72">{media.plot}</p>
              </div>

              <div className="flex flex-wrap justify-end gap-2">
                {[getMediaTypeLabel(media.type), media.resolution, `${media.rating.toFixed(1)} ★`, `${media.fileCount} ملف`].map((chip) => (
                  <span
                    key={chip}
                    className="rounded-full border border-white/12 bg-black/28 px-3 py-2 text-xs font-semibold text-white/75"
                  >
                    {chip}
                  </span>
                ))}
              </div>
            </div>
          </div>

          <div className="flex flex-col justify-between gap-5 text-right">
            <div>
              <p className="text-xs font-semibold text-electric/80">تفاصيل العمل</p>
              <h2 className="mt-4 text-3xl font-black text-white md:text-6xl">{media.titleAr}</h2>
              <p className="mt-3 text-lg text-white/65">{media.titleEn}</p>
            </div>

            <div className="grid gap-3 md:grid-cols-2">
              <div className="rounded-2xl border border-white/10 bg-black/24 p-4">
                <p className="text-xs font-semibold text-white/35">النوع</p>
                <p className="mt-2 text-xl font-bold text-white">{getMediaTypeLabel(media.type)}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-black/24 p-4">
                <p className="text-xs font-semibold text-white/35">الجودة</p>
                <p className="mt-2 text-xl font-bold text-white">{media.resolution}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-black/24 p-4">
                <p className="text-xs font-semibold text-white/35">التقييم</p>
                <p className="mt-2 text-xl font-bold text-white">{media.rating}</p>
              </div>
              <div className="rounded-2xl border border-white/10 bg-black/24 p-4">
                <p className="text-xs font-semibold text-white/35">الملفات</p>
                <p className="mt-2 text-xl font-bold text-white">{media.fileCount}</p>
              </div>
            </div>

            <div className="flex flex-wrap justify-end gap-3">
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-2xl border border-electric/25 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric transition hover:bg-electric/18"
              >
                <Icon name="play" className="h-4 w-4" />
                تشغيل الآن
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-white/80 transition hover:bg-white/[0.08]"
              >
                <Icon name="spark" className="h-4 w-4" />
                إضافة إلى المفضلة
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-white/80 transition hover:bg-white/[0.08]"
              >
                <Icon name="bell" className="h-4 w-4" />
                مشاركة
              </button>
            </div>

            <p className="text-base leading-8 text-white/70">{media.plot}</p>
          </div>
        </div>
      </GlassCard>

      <GlassCard className="p-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div className="text-right">
            <p className="text-xs font-semibold text-electric/80">مشغل الفيديو</p>
            <h2 className="mt-2 text-2xl font-bold text-white">البث المحلي لهذا العمل</h2>
          </div>
          <span className="w-fit rounded-2xl border border-white/10 bg-black/24 px-4 py-2 text-sm font-semibold text-white/65">
            {filesState === "loading" ? "جارٍ جلب الملفات" : filesState === "error" ? "الخادم غير متاح" : `${videoFiles.length} ملف`}
          </span>
        </div>

        <div className="mt-6 grid gap-5 xl:grid-cols-[1.25fr_0.75fr]">
          {streamSrc ? (
            <VideoPlayer
              src={streamSrc}
              title={fileDisplayName(activeFile, media)}
              poster={media.posterPath ? resolveAPIURL(media.posterPath) : undefined}
              tracks={subtitleTracks}
            />
          ) : (
            <div className="grid aspect-video place-items-center rounded-[1.5rem] border border-white/10 bg-[linear-gradient(135deg,rgba(0,0,0,0.62),rgba(25,183,255,0.08))] p-6 text-center shadow-panel">
              <div>
                <div className="mx-auto grid h-16 w-16 place-items-center rounded-2xl border border-electric/20 bg-electric/12 text-electric">
                  <Icon name="play" className="h-7 w-7" />
                </div>
                <h3 className="mt-4 text-xl font-bold text-white">ينتظر ملف فيديو مستورد</h3>
                <p className="mx-auto mt-2 max-w-md text-sm leading-7 text-white/58">
                  عند وصول ملفات هذا العمل من الماسح ستظهر هنا مباشرة مع رابط البث المحلي.
                </p>
              </div>
            </div>
          )}

          <div className="rounded-[1.5rem] border border-white/10 bg-black/20 p-4 text-right">
            <div className="flex items-center justify-between">
              <Icon name="library" className="h-5 w-5 text-electric" />
              <div>
                <p className="text-xs font-semibold text-white/40">ملفات العمل</p>
                <h3 className="mt-1 text-lg font-bold text-white">{media.titleAr}</h3>
              </div>
            </div>

            <div className="mt-4 space-y-3">
              {videoFiles.length > 0 ? (
                videoFiles.map((file) => (
                  <button
                    type="button"
                    key={file.id}
                    onClick={() => setActiveFileID(file.id)}
                    className={`w-full rounded-2xl border px-4 py-3 text-right transition ${
                      activeFile?.id === file.id
                        ? "border-electric/25 bg-electric/12"
                        : "border-white/10 bg-white/[0.04]"
                    }`}
                  >
                    <p className="text-sm font-semibold text-white">{fileDisplayName(file, media)}</p>
                    <div className="mt-2 flex flex-wrap justify-end gap-2 text-xs text-white/48">
                      <span>{file.resolution || media.resolution}</span>
                      <span>{formatBytes(file.file_size)}</span>
                      {file.episode_number ? <span>الحلقة {file.episode_number}</span> : null}
                    </div>
                  </button>
                ))
              ) : (
                <div className="rounded-2xl border border-white/10 bg-white/[0.04] p-4 text-sm leading-7 text-white/58">
                  لا توجد ملفات مرتبطة بهذا العمل داخل قاعدة البيانات حالياً.
                </div>
              )}
            </div>
          </div>
        </div>
      </GlassCard>

      <section className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <GlassCard className="p-6">
          <div className="flex items-end justify-between gap-4">
            <div className="text-right">
              <p className="text-xs font-semibold text-white/40">لوحة القصة</p>
              <h2 className="mt-2 text-2xl font-bold text-white">الملخص وملاحظات الرف</h2>
            </div>
            <button
              type="button"
              onClick={() => onOpenCategory(media.categorySlug)}
              className="rounded-2xl border border-electric/20 bg-electric/12 px-4 py-3 text-sm font-semibold text-electric"
            >
              العودة إلى {categoryLabel}
            </button>
          </div>
          <p className="mt-5 text-base leading-8 text-white/70">{media.plot}</p>

          <div className="mt-6 flex flex-wrap justify-end gap-3">
            {media.highlights?.map((highlight) => (
              <span
                key={highlight}
                className="rounded-full border border-white/12 bg-black/25 px-4 py-2 text-xs font-semibold text-white/70"
              >
                {highlight}
              </span>
            ))}
          </div>

          <div className="mt-8 grid gap-4 md:grid-cols-3">
            <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
              <p className="text-xs font-semibold text-white/35">المدة</p>
              <p className="mt-2 text-lg font-semibold text-white">{media.duration || "تختلف حسب الحلقة"}</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
              <p className="text-xs font-semibold text-white/35">الصيغة</p>
              <p className="mt-2 text-lg font-semibold text-white">{media.resolution}</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-right">
              <p className="text-xs font-semibold text-white/35">المواسم</p>
              <p className="mt-2 text-lg font-semibold text-white">{seasons.length} مواسم</p>
            </div>
          </div>
        </GlassCard>

        <GlassCard className="p-6">
          <div className="flex items-center justify-between">
            <div className="text-right">
              <p className="text-xs font-semibold text-white/40">شبكة الحلقات</p>
              <h2 className="mt-2 text-2xl font-bold text-white">المواسم والمسارات</h2>
            </div>
            <Icon name="library" className="h-5 w-5 text-electric" />
          </div>

          <div className="mt-5 flex flex-wrap justify-end gap-3">
            {seasons.map((season) => (
              <span
                key={season}
                className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2 text-sm font-semibold text-white/75"
              >
                الموسم {season}
              </span>
            ))}
          </div>

          <div className="mt-6 grid grid-cols-2 gap-3 md:grid-cols-3">
            {detailEpisodes.map((episode) => (
              <div
                key={episode.number}
                className="rounded-2xl border border-white/10 bg-black/20 p-4 transition hover:border-electric/20 hover:bg-white/[0.05]"
              >
                <p className="text-xs font-semibold text-white/35">الحلقة {episode.number}</p>
                <p className="mt-2 text-sm font-semibold text-white">{episode.title}</p>
              </div>
            ))}
          </div>
        </GlassCard>
      </section>
    </div>
  );
}

function normalizeSubtitleTracks(subtitles) {
  if (!Array.isArray(subtitles)) {
    return [];
  }
  return subtitles
    .map((subtitle) => {
      const src = subtitle.src || subtitle.url || subtitle.path;
      if (!src) {
        return null;
      }
      return {
        src: resolveAPIURL(src),
        label: subtitle.label || subtitle.language || "العربية",
        srcLang: subtitle.srcLang || subtitle.lang || "ar",
        kind: subtitle.kind || "subtitles",
        default: Boolean(subtitle.default)
      };
    })
    .filter(Boolean);
}

function fileDisplayName(file, media) {
  if (!file) {
    return media.titleAr;
  }
  if (file.title_ar || file.title_en) {
    return file.title_ar || file.title_en;
  }
  return `${media.titleAr} - ملف ${file.id}`;
}

function formatBytes(bytes = 0) {
  if (!bytes) {
    return "حجم غير معروف";
  }
  const units = ["بايت", "KB", "MB", "GB", "TB"];
  let value = Number(bytes);
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}
