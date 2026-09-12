export default function StatCard({ label, value, hint, delta }) {
  return (
    <div className="stat card">
      <div className="stat__label">{label}</div>
      <div className="stat__value">{value}</div>
      {delta != null && (
        <div className={`stat__delta ${delta >= 0 ? 'stat__delta--up' : 'stat__delta--down'}`}>
          {delta >= 0 ? '↑' : '↓'} {Math.abs(delta)}%
        </div>
      )}
      {hint ? <div className="stat__hint">{hint}</div> : null}
    </div>
  );
}
