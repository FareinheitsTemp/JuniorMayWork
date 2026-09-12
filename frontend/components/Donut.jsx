// Донат-діаграма на чистому CSS (conic-gradient) — без бібліотек.
// parts: [{label, value, color}].
export default function Donut({ parts, size = 170 }) {
  const total = parts.reduce((sum, p) => sum + p.value, 0);
  let acc = 0;
  const stops = parts.map((p) => {
    const from = (acc / (total || 1)) * 360;
    acc += p.value;
    const to = (acc / (total || 1)) * 360;
    return `${p.color} ${from}deg ${to}deg`;
  });

  return (
    <div className="donut">
      <div
        className="donut__ring"
        style={{ width: size, height: size, background: `conic-gradient(${stops.join(',')})` }}
      >
        <div className="donut__hole">
          <span className="donut__total">{total}</span>
        </div>
      </div>
      <ul className="donut__legend">
        {parts.map((p) => (
          <li key={p.label} className="donut__item">
            <span className="donut__dot" style={{ background: p.color }} />
            <span className="donut__item-label">{p.label}</span>
            <span className="donut__item-value">{p.value}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
