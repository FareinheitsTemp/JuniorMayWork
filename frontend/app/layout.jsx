import Nav from '@/components/Nav';
import '@/styles/globals.scss';

export const metadata = {
  title: 'JuniorMayWork',
  description: 'Реальний час — трекер дрібних фриланс-замовлень (React/JS, до $100)',
};

export default function RootLayout({ children }) {
  return (
    <html lang="uk">
      <body className="app">
        <Nav />
        <main className="app__main">{children}</main>
      </body>
    </html>
  );
}
