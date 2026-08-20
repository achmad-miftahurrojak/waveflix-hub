/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "image.tmdb.org" },
      { protocol: "https", hostname: "images.unsplash.com" },
    ],
  },
  async rewrites() {
    return [
      {
        source: "/uploads/:path*",
        destination: "http://localhost:8080/uploads/:path*",
      },
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          {
            key: "Content-Security-Policy",
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-eval' 'unsafe-inline' https://www.youtube.com https://s.ytimg.com",
              "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
              "img-src 'self' data: https://image.tmdb.org https://images.unsplash.com https://ui-avatars.com",
              "font-src 'self' https://fonts.gstatic.com",
              // Hanya allow domain embed yang dikenal — jangan 'frame-src *'
              "frame-src https://www.youtube.com https://www.youtube-nocookie.com https://vid.srccdn.org https://vidlink.pro https://vidsrc.to https://vidsrc.xyz https://vidsrc.me https://asianc.to https://dramanice.so https://dramacool.com.tr https://watchasian.sh https://myasiantv.cc",
              "connect-src 'self' http://localhost:8080",
            ].join("; "),
          },
          {
            key: "X-Frame-Options",
            value: "DENY",
          },
          {
            key: "X-Content-Type-Options",
            value: "nosniff",
          },
          {
            key: "Referrer-Policy",
            value: "strict-origin-when-cross-origin",
          },
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=(), interest-cohort=()",
          },
        ],
      },
    ];
  },
};

export default nextConfig;
