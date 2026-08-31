import { withSentryConfig } from '@sentry/nextjs';

/** @type {import('next').NextConfig} */
const nextConfig = {
  outputFileTracingRoot: import.meta.dirname,
  output: 'standalone', // Enable standalone build for Docker
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: 'image.tmdb.org' },
      { protocol: 'https', hostname: 'images.unsplash.com' }
    ]
  },
  // Optimize for production
  compress: true,
  poweredByHeader: false,

  async rewrites () {
    return [
      {
        source: '/uploads/:path*',
        destination: `${
          process.env.NEXT_PUBLIC_BACKEND_URL ??
          process.env.BACKEND_URL ??
          'http://localhost:8080'
        }/uploads/:path*`
      }
    ]
  },
  async headers () {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-eval' 'unsafe-inline' https://www.youtube.com https://s.ytimg.com",
              "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
              "img-src 'self' data: https://image.tmdb.org https://images.unsplash.com https://ui-avatars.com",
              "font-src 'self' https://fonts.gstatic.com",
              // Hanya allow domain embed yang dikenal — jangan 'frame-src *'
              'frame-src https://www.youtube.com https://www.youtube-nocookie.com https://vidlink.pro https://vidsrc.to https://multiembed.mov https://2embed.cc',
              `connect-src 'self' ${
                process.env.NEXT_PUBLIC_BACKEND_URL ??
                process.env.BACKEND_URL ??
                'http://localhost:8080'
              }`
            ].join('; ')
          },
          {
            key: 'X-Frame-Options',
            value: 'DENY'
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff'
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin'
          },
          {
            key: 'Permissions-Policy',
            value:
              'camera=(), microphone=(), geolocation=(), interest-cohort=()'
          }
        ]
      }
    ]
  },

  // Performance optimizations
  experimental: {
    optimizePackageImports: ['lucide-react']
  },

  // Production webpack optimizations
  webpack: (config, { dev, isServer }) => {
    if (!dev && !isServer) {
      // Production client-side optimizations
      config.optimization = {
        ...config.optimization,
        splitChunks: {
          chunks: 'all',
          cacheGroups: {
            default: {
              minChunks: 2,
              priority: -20,
              reuseExistingChunk: true
            },
            vendor: {
              test: /[\\/]node_modules[\\/]/,
              name: 'vendors',
              priority: -10,
              chunks: 'all'
            }
          }
        }
      }
    }

    return config
  }
}

export default withSentryConfig(nextConfig, {
  silent: true,
  org: process.env.SENTRY_ORG,
  project: process.env.SENTRY_PROJECT,
}, {
  widenClientFileUpload: true,
  transpileClientSDK: true,
  tunnelRoute: "/monitoring",
  hideSourceMaps: true,
  disableLogger: true,
  automaticVercelMonitors: true,
});
