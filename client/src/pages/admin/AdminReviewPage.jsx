import ResolutionReviewCenter from "../../components/admin/ResolutionReviewCenter.jsx";

/**
 * AdminReviewPage — شاشة مراجعة الحالات الغامضة
 *
 * Route: /admin/review
 *
 * تعرض الملفات التي رفض Entity Resolution الحسم فيها بدل تخمين عنوان لها.
 * كل قرار يُنفّذ فعليًا ويمكن ترقيته إلى alias دائم، فيتعلّم النظام ولا يتكرر
 * نفس الغموض.
 */
export default function AdminReviewPage() {
  return (
    <div className="space-y-6 text-right animate-fadeIn" dir="rtl">
      <ResolutionReviewCenter />
    </div>
  );
}
