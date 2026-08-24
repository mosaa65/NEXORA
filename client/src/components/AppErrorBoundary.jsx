import React from "react";

export default class AppErrorBoundary extends React.Component {
  constructor(props) { super(props); this.state = { error: null }; }
  static getDerivedStateFromError(error) { return { error }; }
  render() {
    if (!this.state.error) return this.props.children;
    return <main className="flex min-h-dvh items-center justify-center bg-[var(--bg-base)] p-6 text-center text-[var(--text-primary)]" dir="rtl">
      <section className="max-w-md rounded-2xl border border-rose-500/30 bg-[var(--surface)] p-6 shadow-xl">
        <h1 className="text-xl font-black">تعذر عرض هذه الصفحة</h1>
        <p className="mt-2 text-sm leading-7 text-[var(--text-muted)]">تم منع الشاشة الفارغة. حدّث الصفحة أو ارجع إلى لوحة الإدارة، وإن استمر الخطأ أرسل رسالة الخطأ الظاهرة هنا.</p>
        <p className="mt-3 rounded-lg bg-black/15 p-2 text-left text-xs text-rose-300" dir="ltr">{this.state.error.message}</p>
        <button type="button" onClick={() => window.location.reload()} className="mt-5 min-h-11 rounded-xl bg-fuchsia-600 px-5 text-sm font-bold text-white">تحديث الصفحة</button>
      </section>
    </main>;
  }
}
