
module.exports = {
  siteUrl: process.env.SITE_URL || 'https://localhost:3000',
  generateRobotsTxt: true,
  generateIndexSitemap: false,

  exclude: [
    '/api/*',
    '/admin/*',
    '/dashboard/*',
    '/profile/*',
    '/settings/*',
  ],

  additionalPaths: async (config) => {
    const result = []

    const popularRoutes = [
      '/movie/popular',
      '/tv/popular', 
      '/trending',
      '/browse',
    ]

    popularRoutes.forEach((route) => {
      result.push({
        loc: route,
        changefreq: 'daily',
        priority: 0.8,
        lastmod: new Date().toISOString(),
      })
    })

    return result
  },

  robotsTxtOptions: {
    policies: [
      {
        userAgent: '*',
        allow: '/',
        disallow: [
          '/api/',
          '/admin/',
          '/dashboard/',
          '/profile/',
          '/settings/',
        ],
      },
      {
        userAgent: 'Googlebot',
        allow: '/',
        crawlDelay: 2,
      },
    ],
    additionalSitemaps: [
      'https://yourdomain.com/sitemap-movies.xml',
      'https://yourdomain.com/sitemap-tv.xml',
    ],
  },
}