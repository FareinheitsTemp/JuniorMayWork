export default function StatCard({ label, value, hint }) {
  return (
    <div className="stat card">
      <div className="stat__label">{label}</div>
      <div className="stat__value">{value}</div>
      {hint ? <div className="stat__hint">{hint}</div> : null}
    </div>
  );
}
