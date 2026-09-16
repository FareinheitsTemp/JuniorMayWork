'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import '@/styles/blocks/nav-enhance.scss';
import {
  TerminalIcon,
  DashboardIcon,
  OrdersIcon,
  BranchesIcon,
  DatabaseIcon,
  ReportsIcon,
  SettingsIcon,
} from '@/components/Icons';

const LINKS = [
  { href: '/', label: 'Дашборд', icon: DashboardIcon },
  { href: '/orders', label: 'Замовлення', icon: OrdersIcon },
  { href: '/branches', label: 'Вітки', icon: BranchesIcon },
  { href: '/database', label: 'База даних', icon: DatabaseIcon },
  { href: '/reports', label: 'Звіти', icon: ReportsIcon },
  { href: '/settings', label: 'Налаштування', icon: SettingsIcon },
];

export default function Nav() {
  const pathname = usePathname();

  return (
    <header className="nav">
      <div className="nav__brand">
        <TerminalIcon size={16} style={{ color: 'var(--accent)' }} />
        <span>Junior<span>May</span>Work</span>
        <span className="nav__live-dot" suppressHydrationWarning />
      </div>
      <nav className="nav__links">
        {LINKS.map((l) => {
          const Icon = l.icon;
          const isActive = pathname === l.href;
          return (
            <Link
              key={l.href}
              href={l.href}
              className={`nav__link ${isActive ? 'nav__link--active' : ''}`}
            >
              <Icon size={14} style={{ opacity: isActive ? 1 : 0.7 }} />
              <span>{l.label}</span>
            </Link>
          );
        })}
      </nav>
    </header>
  );
}
