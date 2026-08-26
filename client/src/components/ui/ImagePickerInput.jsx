import React, { useState, useRef } from "react";
import Icon from "../Icon.jsx";
import { resolveAPIURL } from "../../lib/api.js";

/**
 * ImagePickerInput — مكوّن موحد لاختيار ورفع الصور في لوحة التحكم
 *
 * يدعم طريقتين معاً:
 * 1. رفع / اختيار صورة مباشرة من الجهاز (مع دعم السحب والإفلات ومعاينتها فوراً).
 * 2. كتابة أو لصق رابط / مسار الصورة يدوياً (URLs, Server Paths, Relative Paths).
 *
 * @param {string} label - عنوان الحقل
 * @param {string} value - مسار الصورة أو الـ DataURL الحالي
 * @param {Function} onChange - دالة التحديث عند تغيير الصورة
 * @param {"poster"|"banner"|"square"|"auto"} aspectRatio - نسبة العرض إلى الارتفاع للمعاينة
 * @param {string} placeholder - نص التلميح
 * @param {string} helperText - نص إرشادي إضافي
 * @param {boolean} required - هل الحقل إلزامي
 */
export default function ImagePickerInput({
  label,
  value = "",
  onChange,
  aspectRatio = "poster",
  placeholder = "مثال: https://... أو /posters/image.png",
  helperText,
  required = false,
  className = "",
}) {
  const [mode, setMode] = useState(value && value.startsWith("data:") ? "upload" : "path");
  const [dragOver, setDragOver] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const fileInputRef = useRef(null);

  const previewURL = value ? resolveAPIURL(value) : "";

  // Aspect ratio classes for preview
  const aspectClasses = {
    poster: "aspect-[2/3] w-24 sm:w-28",
    banner: "aspect-[16/9] w-48 sm:w-56",
    square: "aspect-square w-24 sm:w-28",
    auto: "max-h-36 w-auto",
  }[aspectRatio] || "aspect-[2/3] w-24 sm:w-28";

  // Handle File Selection and Convert to Data URL
  function handleFileProcess(file) {
    if (!file) return;

    if (!file.type.startsWith("image/")) {
      setErrorMsg("يرجى اختيار ملف صورة صالح (PNG, JPG, WEBP...)");
      return;
    }

    // Limit size to 10MB
    if (file.size > 10 * 1024 * 1024) {
      setErrorMsg("حجم الصورة كبير جداً (الحد الأقصى 10 ميغابايت)");
      return;
    }

    setErrorMsg("");
    const reader = new FileReader();
    reader.onload = (e) => {
      const dataUrl = e.target.result;
      onChange?.(dataUrl);
    };
    reader.onerror = () => {
      setErrorMsg("حدث خطأ أثناء قراءة ملف الصورة");
    };
    reader.readAsDataURL(file);
  }

  function handleFileChange(e) {
    const file = e.target.files?.[0];
    if (file) handleFileProcess(file);
  }

  function handleDrop(e) {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer?.files?.[0];
    if (file) handleFileProcess(file);
  }

  function handleClear() {
    onChange?.("");
    setErrorMsg("");
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  return (
    <div className={`space-y-2 text-right ${className}`} dir="rtl">
      {/* Label and Mode Switcher */}
      <div className="flex items-center justify-between gap-2">
        <label className="block text-xs font-bold text-[var(--text-primary)]">
          {label} {required && <span className="text-rose-400">*</span>}
        </label>

        {/* Mode Selector Tabs */}
        <div className="inline-flex rounded-xl bg-white/5 p-0.5 border border-white/10 text-[11px] font-bold">
          <button
            type="button"
            onClick={() => setMode("upload")}
            className={`px-2.5 py-1 rounded-lg transition ${
              mode === "upload"
                ? "bg-fuchsia-600 text-white shadow-sm"
                : "text-white/60 hover:text-white"
            }`}
          >
            📂 من الجهاز
          </button>
          <button
            type="button"
            onClick={() => setMode("path")}
            className={`px-2.5 py-1 rounded-lg transition ${
              mode === "path"
                ? "bg-fuchsia-600 text-white shadow-sm"
                : "text-white/60 hover:text-white"
            }`}
          >
            🔗 مسار / رابط
          </button>
        </div>
      </div>

      {/* Input Area (Upload vs Path) */}
      <div className="space-y-3">
        {mode === "upload" ? (
          <div>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              onChange={handleFileChange}
              className="hidden"
            />
            <div
              onDragOver={(e) => {
                e.preventDefault();
                setDragOver(true);
              }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className={`
                group relative flex flex-col items-center justify-center p-4 rounded-2xl border-2 border-dashed
                cursor-pointer transition-all duration-200 text-center
                ${
                  dragOver
                    ? "border-fuchsia-400 bg-fuchsia-950/30 scale-[1.01]"
                    : "border-white/15 bg-black/40 hover:border-fuchsia-500/50 hover:bg-black/60"
                }
              `}
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 text-fuchsia-400 group-hover:scale-110 transition">
                <Icon name="upload" className="h-5 w-5" />
              </div>
              <p className="mt-2 text-xs font-bold text-white">
                اضغط لاختيار صورة من جهازك
              </p>
              <p className="text-[10px] text-white/50 mt-0.5">
                أو اسحب وأفلت ملف الصورة هنا (PNG, JPG, WEBP)
              </p>
            </div>
          </div>
        ) : (
          <div className="relative">
            <input
              type="text"
              value={value}
              onChange={(e) => {
                setErrorMsg("");
                onChange?.(e.target.value);
              }}
              placeholder={placeholder}
              dir="ltr"
              className="w-full rounded-xl border border-[var(--border-default)] bg-[var(--bg-input)] px-3.5 py-2.5 text-xs text-[var(--text-primary)] font-mono outline-none focus:border-[var(--color-accent)] focus:ring-1 focus:ring-[var(--color-accent)] transition"
            />
            {value && (
              <button
                type="button"
                onClick={handleClear}
                className="absolute left-2.5 top-1/2 -translate-y-1/2 flex h-5 w-5 items-center justify-center rounded-full bg-white/10 text-white/60 hover:bg-white/20 hover:text-white transition text-xs"
                title="مسح"
              >
                ✕
              </button>
            )}
          </div>
        )}

        {/* Error Message */}
        {errorMsg && (
          <p className="text-[11px] font-bold text-rose-400 animate-fadeIn">
            ⚠️ {errorMsg}
          </p>
        )}

        {/* Live Preview Card (if value exists) */}
        {previewURL && (
          <div className="flex items-center gap-3.5 p-2.5 rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur-sm animate-fadeIn">
            <div className={`relative overflow-hidden rounded-xl border border-white/15 bg-black/60 shadow-md shrink-0 ${aspectClasses}`}>
              <img
                src={previewURL}
                alt="معاينة الصورة"
                className="h-full w-full object-cover"
                onError={(e) => {
                  e.target.src = "/nexora-poster-placeholder.PNG";
                }}
              />
            </div>

            <div className="flex flex-1 flex-col justify-between min-w-0">
              <div>
                <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                  معاينة مباشرة جاهزة
                </span>
                <p className="truncate text-xs font-mono text-white/60 mt-1 max-w-xs" dir="ltr">
                  {value.startsWith("data:") ? "ملف صورة مرفوع من الجهاز" : value}
                </p>
              </div>

              <div className="flex items-center gap-2 mt-2">
                <button
                  type="button"
                  onClick={handleClear}
                  className="inline-flex items-center gap-1 rounded-lg border border-rose-500/30 bg-rose-950/40 px-2.5 py-1 text-[11px] font-bold text-rose-300 hover:bg-rose-900/60 hover:text-white transition"
                >
                  <Icon name="close" className="h-3 w-3" />
                  <span>حذف الصورة</span>
                </button>
                {mode === "upload" && (
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="inline-flex items-center gap-1 rounded-lg border border-white/15 bg-white/10 px-2.5 py-1 text-[11px] font-bold text-white/80 hover:bg-white/20 hover:text-white transition"
                  >
                    <span>تغيير الملف</span>
                  </button>
                )}
              </div>
            </div>
          </div>
        )}

        {helperText && (
          <p className="text-[11px] text-white/40">{helperText}</p>
        )}
      </div>
    </div>
  );
}
