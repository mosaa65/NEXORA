export default function GlassCard({ className = "", children, ...props }) {
  return (
    <div className={`glass-panel ${className}`} {...props}>
      {children}
    </div>
  );
}
