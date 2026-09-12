import Link from 'next/link';
import '@/styles/globals.scss';

export const metadata = {
  title: 'JuniorMayWork',
  description: 'Реальний час — трекер дрібних фриланс-замовлень (React/JS, до $100)',
};

export default function RootLayout({ children }) {
  return (
    <html lang="uk">
      <body className="app">
        <header className="nav">
          <div className="nav__brand">
            Junior<span>May</span>Work
          </div>
          <nav className="nav__links">
            <Link className="nav__link" href="/">Дашборд</Link>
            <Link className="nav__link" href="/orders">Замовлення</Link>
            <Link className="nav__link" href="/branches">Вітки</Link>
            <Link className="nav__link" href="/reports">Звіти</Link>
            <Link className="nav__link" href="/settings">Налаштування</Link>
          </nav>
        </header>
        <main className="app__main">{children}</main>
      </body>
    </html>
  );
}
