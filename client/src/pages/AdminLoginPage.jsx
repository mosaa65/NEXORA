import React, { useState } from "react";
import { motion } from "framer-motion";
import { adminLogin } from "../lib/api";
import Icon from "../components/Icon";

export default function AdminLoginPage({ onLoginSuccess }) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  async function handleSubmit(e) {
    e.preventDefault();
    if (!password) {
      setError("الرجاء إدخال كلمة المرور");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const res = await adminLogin(username, password);
      if (res?.ok && res?.token) {
        localStorage.setItem("nexora_admin_token", res.token);
        localStorage.setItem("nexora_admin_user", JSON.stringify(res.user || {}));
        if (onLoginSuccess) {
          onLoginSuccess(res.user);
        }
      } else {
        setError(res?.error || "فشل تسجيل الدخول");
      }
    } catch (err) {
      setError(err.message || "اسم المستخدم أو كلمة المرور غير صحيحة");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-[85vh] flex items-center justify-center p-4" dir="rtl">
      <motion.div
        initial={{ opacity: 0, y: 20, scale: 0.96 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        className="w-full max-w-md p-8 rounded-[2.5rem] bg-[#0D0C1C]/90 border border-fuchsia-500/20 backdrop-blur-2xl shadow-2xl space-y-6 text-right relative overflow-hidden"
      >
        {/* Ambient Top Glow */}
        <div className="absolute top-0 right-1/2 translate-x-1/2 w-48 h-20 bg-fuchsia-600/20 blur-3xl pointer-events-none" />

        {/* Brand Header */}
        <div className="text-center space-y-2">
          <div className="inline-flex p-4 rounded-2xl bg-gradient-to-br from-fuchsia-600 to-purple-800 text-white shadow-xl shadow-purple-900/50 mb-2">
            <Icon name="settings" className="w-8 h-8" />
          </div>
          <h1 className="text-2xl font-black text-white">بوابة الإدارة والتحكم</h1>
          <p className="text-xs text-gray-400">سجل الدخول للوصول إلى أدوات الفهرسة وإعدادات TMDB والسيرفر</p>
        </div>

        {/* Error Notification */}
        {error && (
          <div className="p-3.5 rounded-2xl bg-rose-950/60 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
            <Icon name="AlertTriangle" className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Login Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-bold text-gray-300">اسم المستخدم</label>
            <div className="relative">
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full bg-slate-900/90 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-white focus:outline-none focus:border-fuchsia-500 transition shadow-inner"
                placeholder="admin"
                required
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-bold text-gray-300">كلمة المرور</label>
            <div className="relative">
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-900/90 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-white focus:outline-none focus:border-fuchsia-500 transition shadow-inner"
                placeholder="••••••••"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 rounded-2xl bg-gradient-to-r from-fuchsia-600 via-purple-600 to-indigo-600 hover:from-fuchsia-500 hover:to-indigo-500 text-white font-bold text-sm shadow-xl shadow-purple-900/40 transition-all flex items-center justify-center gap-2 disabled:opacity-50 mt-2"
          >
            {loading ? (
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            ) : (
              <Icon name="spark" className="w-4 h-4" />
            )}
            <span>{loading ? "جاري التحقق..." : "تسجيل الدخول للإدارة"}</span>
          </button>
        </form>

        <div className="pt-2 text-center border-t border-white/5">
          <p className="text-[11px] text-gray-500">
            نظام NEXORA LAN v0.2.0 · اتصال محلي مشفر
          </p>
        </div>
      </motion.div>
    </div>
  );
}
