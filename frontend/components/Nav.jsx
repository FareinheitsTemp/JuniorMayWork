'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const LINKS = [
  { href: '/', label: 'Дашборд' },
  { href: '/orders', label: 'Замовлення' },
  { href: '/branches', label: 'Вітки' },
  { href: '/database', label: 'База даних' },
  { href: '/reports', label: 'Звіти' },
  { href: '/settings', label: 'Налаштування' },
];

export default function Nav() {
  const pathname = usePathname();
  return (
    <header className="nav">
      <div className="nav__brand">
        Junior<span>May</span>Work
      </div>
      <nav className="nav__links">
        {LINKS.map((l) => (
          <Link
            key={l.href}
            href={l.href}
            className={`nav__link ${pathname === l.href ? 'nav__link--active' : ''}`}
          >
            {l.label}
          </Link>
        ))}
      </nav>
    </header>
  );
}
