// WaveFlix Hub Service Worker for Performance Optimization
const CACHE_NAME = 'waveflix-v1.0.0';
const STATIC_CACHE_NAME = 'waveflix-static-v1.0.0';
const API_CACHE_NAME = 'waveflix-api-v1.0.0';
const IMAGE_CACHE_NAME = 'waveflix-images-v1.0.0';

// Assets to cache on install
const STATIC_ASSETS = [
  '/',
  '/offline.html',
  '/manifest.json',
  // Add your critical CSS and JS files here
];

// API endpoints to cache
const CACHED_API_PATTERNS = [
  /\/api\/trending/,
  /\/api\/popular/,
  /\/api\/detail/,
  /\/api\/discover/,
];

// Image patterns to cache
const IMAGE_PATTERNS = [
  /image\.tmdb\.org/,
  /images\.unsplash\.com/,
  /\/uploads\//,
  /\/cache\/images\//,
];

// Install event - cache critical assets
self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker...');
  
  event.waitUntil(
    caches.open(STATIC_CACHE_NAME)
      .then((cache) => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker...');
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME && 
              cacheName !== STATIC_CACHE_NAME && 
              cacheName !== API_CACHE_NAME && 
              cacheName !== IMAGE_CACHE_NAME) {
            console.log('[SW] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// Fetch event - implement caching strategies
self.addEventListener('fetch', (event) => {
  const request = event.request;
  const url = new URL(request.url);
  
  // Skip non-GET requests
  if (request.method !== 'GET') {
    return;
  }
  
  // Handle different types of requests
  if (isStaticAsset(url)) {
    event.respondWith(handleStaticAsset(request));
  } else if (isAPIRequest(url)) {
    event.respondWith(handleAPIRequest(request));
  } else if (isImageRequest(url)) {
    event.respondWith(handleImageRequest(request));
  } else if (isNavigationRequest(request)) {
    event.respondWith(handleNavigationRequest(request));
  }
});

// Check if request is for static asset
function isStaticAsset(url) {
  const staticExtensions = ['.js', '.css', '.woff', '.woff2', '.ttf', '.ico'];
  return staticExtensions.some(ext => url.pathname.endsWith(ext));
}

// Check if request is for API
function isAPIRequest(url) {
  return url.pathname.startsWith('/api/') || 
         CACHED_API_PATTERNS.some(pattern => pattern.test(url.pathname));
}

// Check if request is for image
function isImageRequest(url) {
  const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.avif', '.svg'];
  return imageExtensions.some(ext => url.pathname.endsWith(ext)) ||
         IMAGE_PATTERNS.some(pattern => pattern.test(url.href));
}

// Check if request is navigation
function isNavigationRequest(request) {
  return request.mode === 'navigate';
}

// Handle static assets - Cache First strategy
async function handleStaticAsset(request) {
  try {
    const cachedResponse = await caches.match(request, { cacheName: STATIC_CACHE_NAME });
    
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(STATIC_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.error('[SW] Static asset fetch failed:', error);
    throw error;
  }
}

// Handle API requests - Network First with cache fallback
async function handleAPIRequest(request) {
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // Cache successful responses
      const cache = await caches.open(API_CACHE_NAME);
      cache.put(request, networkResponse.clone());
      
      // Set TTL for API cache (15 minutes for trending, 1 hour for details)
      setTimeout(() => {
        caches.open(API_CACHE_NAME).then(cache => {
          if (request.url.includes('trending')) {
            // Shorter cache for trending content
          }
        });
      }, getTTLForAPI(request.url));
    }
    
    return networkResponse;
  } catch (error) {
    console.log('[SW] Network failed, trying cache for:', request.url);
    
    const cachedResponse = await caches.match(request, { cacheName: API_CACHE_NAME });
    
    if (cachedResponse) {
      // Add header to indicate cached response
      const response = cachedResponse.clone();
      response.headers.set('X-Cache', 'sw-cache');
      return response;
    }
    
    throw error;
  }
}

// Handle image requests - Cache First with network fallback
async function handleImageRequest(request) {
  try {
    const cachedResponse = await caches.match(request, { cacheName: IMAGE_CACHE_NAME });
    
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(IMAGE_CACHE_NAME);
      
      // Only cache images smaller than 1MB
      const contentLength = networkResponse.headers.get('content-length');
      if (!contentLength || parseInt(contentLength) < 1024 * 1024) {
        cache.put(request, networkResponse.clone());
      }
    }
    
    return networkResponse;
  } catch (error) {
    console.error('[SW] Image fetch failed:', error);
    
    // Return placeholder or cached offline image
    return new Response(
      '<svg width="300" height="200" xmlns="http://www.w3.org/2000/svg"><rect width="100%" height="100%" fill="#f0f0f0"/><text x="50%" y="50%" text-anchor="middle" dy=".3em">Image Unavailable</text></svg>',
      { 
        headers: { 
          'Content-Type': 'image/svg+xml',
          'Cache-Control': 'no-cache'
        }
      }
    );
  }
}

// Handle navigation requests - Network First with offline fallback
async function handleNavigationRequest(request) {
  try {
    return await fetch(request);
  } catch (error) {
    console.log('[SW] Navigation failed, serving offline page');
    
    const offlineResponse = await caches.match('/offline.html');
    return offlineResponse || new Response('Offline');
  }
}

// Get TTL for different API endpoints
function getTTLForAPI(url) {
  if (url.includes('trending')) {
    return 15 * 60 * 1000; // 15 minutes
  } else if (url.includes('detail')) {
    return 60 * 60 * 1000; // 1 hour
  } else if (url.includes('search')) {
    return 30 * 60 * 1000; // 30 minutes
  }
  return 30 * 60 * 1000; // default 30 minutes
}

// Background sync for offline actions
self.addEventListener('sync', (event) => {
  if (event.tag === 'background-sync') {
    console.log('[SW] Background sync triggered');
    event.waitUntil(doBackgroundSync());
  }
});

// Perform background synchronization
async function doBackgroundSync() {
  try {
    // Sync offline actions when back online
    const offlineActions = await getOfflineActions();
    
    for (const action of offlineActions) {
      try {
        await fetch(action.url, action.options);
        await removeOfflineAction(action.id);
      } catch (error) {
        console.error('[SW] Failed to sync action:', error);
      }
    }
  } catch (error) {
    console.error('[SW] Background sync failed:', error);
  }
}

// Placeholder functions for offline action management
async function getOfflineActions() {
  // Implement offline action storage
  return [];
}

async function removeOfflineAction(id) {
  // Implement offline action removal
}

// Push notification handling
self.addEventListener('push', (event) => {
  const options = {
    body: event.data ? event.data.text() : 'New content available!',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/badge-72x72.png',
    data: {
      url: '/',
    },
  };
  
  event.waitUntil(
    self.registration.showNotification('WaveFlix Hub', options)
  );
});

// Notification click handling
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  const url = event.notification.data?.url || '/';
  
  event.waitUntil(
    clients.openWindow(url)
  );
});

// Periodic background sync for cache updates
self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'cache-update') {
    event.waitUntil(updateCriticalCache());
  }
});

// Update critical content in background
async function updateCriticalCache() {
  try {
    const cache = await caches.open(API_CACHE_NAME);
    
    // Update trending content
    const trendingResponse = await fetch('/api/trending');
    if (trendingResponse.ok) {
      await cache.put('/api/trending', trendingResponse);
    }
    
    console.log('[SW] Critical cache updated');
  } catch (error) {
    console.error('[SW] Cache update failed:', error);
  }
}