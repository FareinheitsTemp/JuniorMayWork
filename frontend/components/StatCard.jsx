// Claude Code / Anthropic Stat Card widget
// Monospace numbers, muted uppercase terminal labels, indicator dots.

export default function StatCard({ label, value, delta, hint }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <span style={{
        fontSize: 11,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.05em',
        color: 'var(--text-dim)',
      }}>
        {label}
      </span>
      <div style={{
        display: 'flex',
        alignItems: 'baseline',
        gap: 8,
        fontFamily: 'var(--font-mono)',
        fontSize: 24,
        fontWeight: 600,
        color: 'var(--text)',
        letterSpacing: '-0.02em',
      }}>
        <span>{value ?? '—'}</span>
        {delta != null && (
          <span style={{
            fontSize: 12,
            fontFamily: 'var(--font-mono)',
            fontWeight: 500,
            color: delta > 0 ? 'var(--good)' : delta < 0 ? 'var(--bad)' : 'var(--text-muted)',
          }}>
            {delta > 0 ? `+${delta}%` : `${delta}%`}
          </span>
        )}
      </div>
      {hint && (
        <span style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>
          {hint}
        </span>
      )}
    </div>
  );
}
