import { useEffect, useRef } from "react";
import PlayerIcon from "./PlayerIcons.jsx";
import { formatSize } from "../../lib/watchContent.js";

const RATES = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2];

/**
 * PlayerSettingsMenu — the single advanced surface of the player.
 *
 * Everything that is not transport lives here so the control bar stays short:
 * quality/versions, audio tracks, subtitles, speed, subtitle styling and the
 * technical panel. Sections are only rendered when the data actually exists — an
 * empty "الجودة" list would imply a choice that does not exist.
 *
 * @param {object} props
 * @param {string} props.section  active section key, or null when closed
 * @param {object} props.options  { quality, audio, subtitle, rate, sources, technical }
 * @param {object} props.current  { quality, audio, subtitle, rate }
 */
export default function PlayerSettingsMenu({
  section,
  onClose,
  onSelectSection,
  options,
  current,
  onApply,
  subtitleStyle,
  onSubtitleStyle,
}) {
  const menuRef = useRef(null);

  // Close on outside click or Escape. Registered only while open, so a closed menu
  // never adds document-level listeners.
  useEffect(() => {
    if (!section) return undefined;
    const onDown = (event) => {
      if (!event.target.closest?.(".nexora-popover, .nexora-bar-button")) onClose();
    };
    const onKey = (event) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        onClose();
      }
    };
    document.addEventListener("pointerdown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [section, onClose]);

  if (!section) return null;

  const sections = [
    { key: "quality", label: "الجودة والإصدارات", icon: "quality", count: options.sources?.length || 0 },
    { key: "audio", label: "الصوت", icon: "audio", count: options.audio?.length || 0 },
    { key: "subtitle", label: "الترجمة", icon: "captions", count: options.subtitle?.length || 0 },
    { key: "rate", label: "سرعة التشغيل", icon: "settings" },
    { key: "info", label: "معلومات الملف", icon: "info" },
  ];

  return (
    <div ref={menuRef} className="nexora-popover nexora-popover--settings" role="dialog" aria-label="إعدادات المشغل">
      {section === "root" ? (
        <>
          <p className="nexora-popover-title">الإعدادات</p>
          {sections.map((item) => (
            <button
              key={item.key}
              type="button"
              className="nexora-popover-item"
              onClick={() => onSelectSection(item.key)}
            >
              <span className="nexora-popover-item-label">
                <PlayerIcon name={item.icon} className="h-4 w-4" />
                {item.label}
              </span>
              <span className="nexora-popover-item-value">
                {summaryFor(item.key, options, current)}
                {item.count > 1 ? <span className="nexora-popover-badge">{item.count}</span> : null}
              </span>
            </button>
          ))}
        </>
      ) : (
        <>
          <div className="nexora-popover-head">
            <button type="button" className="nexora-popover-back" onClick={() => onSelectSection("root")}>
              <PlayerIcon name="forward" className="h-3.5 w-3.5" />
              رجوع
            </button>
            <p className="nexora-popover-title nexora-popover-title--inline">
              {sections.find((s) => s.key === section)?.label}
            </p>
          </div>
          {renderSection(section, options, current, onApply, subtitleStyle, onSubtitleStyle)}
        </>
      )}
    </div>
  );
}

function summaryFor(key, options, current) {
  if (key === "quality") return current.quality || "المصدر الحالي";
  if (key === "audio") return current.audio || "افتراضي";
  if (key === "subtitle") return current.subtitle || "متوقف";
  if (key === "rate") return `${current.rate || 1}×`;
  return "";
}

function renderSection(section, options, current, onApply, subtitleStyle, onSubtitleStyle) {
  if (section === "quality") {
    return (
      <>
        {(options.sources || []).length === 0 ? (
          <p className="nexora-popover-empty">هذا العمل يحتوي على مصدر واحد فقط.</p>
        ) : (
          options.sources.map((source) => (
            <button
              key={source.video_file_id}
              type="button"
              className={`nexora-popover-item ${current.qualityId === source.video_file_id ? "is-on" : ""}`}
              onClick={() => onApply("quality", source)}
            >
              <span className="nexora-popover-item-label">
                {current.qualityId === source.video_file_id ? <PlayerIcon name="check" className="h-3.5 w-3.5" /> : null}
                {source.label}
              </span>
              <span className="nexora-popover-item-value">
                {source.file_size ? formatSize(source.file_size) : ""}
              </span>
            </button>
          ))
        )}
        {options.technical?.resolution ? (
          <p className="nexora-popover-foot">
            المصدر الحالي: {options.technical.resolution}
            {options.technical.video_codec ? ` · ${options.technical.video_codec.toUpperCase()}` : ""}
          </p>
        ) : null}
      </>
    );
  }

  if (section === "audio") {
    if ((options.audio || []).length === 0) {
      return <p className="nexora-popover-empty">لا توجد مسارات صوت متعددة في هذا الملف.</p>;
    }
    return (
      <>
        {options.audio.map((track, index) => (
          <button
            key={`${track.index}-${index}`}
            type="button"
            className={`nexora-popover-item ${current.audioIndex === index ? "is-on" : ""}`}
            onClick={() => onApply("audio", { index, track })}
            disabled={!track.selectable}
          >
            <span className="nexora-popover-item-label">
              {current.audioIndex === index ? <PlayerIcon name="check" className="h-3.5 w-3.5" /> : null}
              {track.label}
            </span>
            {track.detail ? <span className="nexora-popover-item-value">{track.detail}</span> : null}
          </button>
        ))}
        {options.audioUnsupported ? (
          <p className="nexora-popover-foot nexora-popover-foot--warn">
            المتصفح لا يعرض مسارات الصوت في هذا الحاوية، لذا هذه تسميات وصفية من فحص الملف.
          </p>
        ) : null}
      </>
    );
  }

  if (section === "subtitle") {
    return (
      <>
        <button
          type="button"
          className={`nexora-popover-item ${current.subtitleIndex === -1 ? "is-on" : ""}`}
          onClick={() => onApply("subtitle", -1)}
        >
          <span className="nexora-popover-item-label">
            {current.subtitleIndex === -1 ? <PlayerIcon name="check" className="h-3.5 w-3.5" /> : null}
            إيقاف
          </span>
        </button>
        {(options.subtitle || []).length === 0 ? (
          <p className="nexora-popover-empty">لا توجد ترجمات متاحة لهذا الملف.</p>
        ) : (
          options.subtitle.map((track, index) => (
            <button
              key={`${track.label}-${index}`}
              type="button"
              className={`nexora-popover-item ${current.subtitleIndex === index ? "is-on" : ""}`}
              onClick={() => onApply("subtitle", index)}
            >
              <span className="nexora-popover-item-label">
                {current.subtitleIndex === index ? <PlayerIcon name="check" className="h-3.5 w-3.5" /> : null}
                {track.label}
              </span>
              {track.source ? <span className="nexora-popover-item-value">{track.source}</span> : null}
            </button>
          ))
        )}

        {/* Subtitle styling lives with the subtitles — one place to look. */}
        <div className="nexora-popover-styles">
          <p className="nexora-popover-title">حجم الترجمة</p>
          <div className="nexora-popover-chips">
            {[
              { key: "sm", label: "صغير" },
              { key: "md", label: "متوسط" },
              { key: "lg", label: "كبير" },
              { key: "xl", label: "كبير جدًا" },
            ].map((option) => (
              <button
                key={option.key}
                type="button"
                className={`nexora-popover-chip ${subtitleStyle?.size === option.key ? "is-on" : ""}`}
                onClick={() => onSubtitleStyle({ ...subtitleStyle, size: option.key })}
              >
                {option.label}
              </button>
            ))}
          </div>
          <p className="nexora-popover-title">خلفية الترجمة</p>
          <div className="nexora-popover-chips">
            {[
              { key: "none", label: "بدون" },
              { key: "soft", label: "خفيفة" },
              { key: "solid", label: "معتمة" },
            ].map((option) => (
              <button
                key={option.key}
                type="button"
                className={`nexora-popover-chip ${subtitleStyle?.background === option.key ? "is-on" : ""}`}
                onClick={() => onSubtitleStyle({ ...subtitleStyle, background: option.key })}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      </>
    );
  }

  if (section === "rate") {
    return RATES.map((rate) => (
      <button
        key={rate}
        type="button"
        className={`nexora-popover-item ${Number(current.rate) === rate ? "is-on" : ""}`}
        onClick={() => onApply("rate", rate)}
      >
        <span className="nexora-popover-item-label">
          {Number(current.rate) === rate ? <PlayerIcon name="check" className="h-3.5 w-3.5" /> : null}
          {rate}×
        </span>
        {rate === 1 ? <span className="nexora-popover-item-value">عادي</span> : null}
      </button>
    ));
  }

  if (section === "info") {
    const info = options.technical || {};
    return (
      <dl className="nexora-popover-info">
        <div><dt>الدقة</dt><dd>{info.resolution || "—"}</dd></div>
        <div><dt>ترميز الفيديو</dt><dd>{info.video_codec ? info.video_codec.toUpperCase() : "—"}</dd></div>
        <div><dt>الحاوية</dt><dd>{info.container || "—"}</dd></div>
        <div><dt>المدة</dt><dd>{info.durationLabel || "—"}</dd></div>
        <div><dt>الحجم</dt><dd>{info.fileSize ? formatSize(info.file_size) : "—"}</dd></div>
        <div><dt>مسارات الصوت</dt><dd>{info.audioCount ?? 0}</dd></div>
        <div><dt>مسارات الترجمة</dt><dd>{info.subtitleCount ?? 0}</dd></div>
        {info.audioSummary ? <div><dt>الصوت</dt><dd>{info.audioSummary}</dd></div> : null}
      </dl>
    );
  }

  return null;
}