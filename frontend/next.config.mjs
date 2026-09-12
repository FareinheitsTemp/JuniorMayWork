/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    return [
      // Проксіємо API на Go-бекстенд, щоб фронт ходив відносними шляхами (/api/...)
      { source: '/api/:path*', destination: 'http://localhost:8080/api/:path*' },
    ];
  },
};

export default nextConfig;
