'use client';

// Claude Code Palette for Donut Segments: Terracotta, Sage, Amber, Slate, Olive
const ANTHROPIC_PALETTE = ['#d97757', '#788c5d', '#d4a017', '#6b8fa3', '#b3806d', '#8a9a86'];

export default function Donut({ parts = [], size = 110, strokeWidth = 14 }) {
  const total = parts.reduce((acc, p) => acc + (p.value || 0), 0);
  const r = (size - strokeWidth) / 2;
  const c = 2 * Math.PI * r;

  let offset = 0;

  return (
    <div style={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ transform: 'rotate(-90deg)' }}>
        {/* Background track */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="var(--bg-soft)"
          strokeWidth={strokeWidth}
        />
        {total > 0 && parts.map((p, i) => {
          if (!p.value) return null;
          const strokeDash = (p.value / total) * c;
          const strokeDashoffset = -offset;
          offset += strokeDash;
          const color = p.color || ANTHROPIC_PALETTE[i % ANTHROPIC_PALETTE.length];
          return (
            <circle
              key={p.label || i}
              cx={size / 2}
              cy={size / 2}
              r={r}
              fill="none"
              stroke={color}
              strokeWidth={strokeWidth}
              strokeDasharray={`${strokeDash} ${c - strokeDash}`}
              strokeDashoffset={strokeDashoffset}
              strokeLinecap="butt"
              style={{ transition: 'stroke-dasharray 0.3s ease' }}
            />
          );
        })}
      </svg>
      {/* Center value in monospace */}
      <div style={{
        position: 'absolute',
        inset: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        fontFamily: 'var(--font-mono)',
        fontSize: 14,
        fontWeight: 600,
        color: 'var(--text)',
      }}>
        {total}
      </div>
    </div>
  );
}
