import express from "express";
import cors from "cors";
import puppeteer from 'puppeteer';
import { getTorrentInfo, activeEngines } from './torrent_engine.js';

const app = express();
app.use(cors());
app.use(express.json());

const PORT = 8000;

async function extractStreamWithPuppeteer(id, mediaType, season, episode) {
  let url = mediaType === 'tv' 
    ? `https://vidsrc.in/embed/tv/${id}/${season}/${episode}` 
    : `https://vidsrc.in/embed/movie/${id}`;
    
  console.log('[puppeteer] Launching browser for', url);
  const browser = await puppeteer.launch({ 
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,720'],
    channel: 'chrome'
  });
  
  try {
    const pages = await browser.pages();
    const page = pages.length > 0 ? pages[0] : await browser.newPage();
    await page.setViewport({ width: 1280, height: 720 });
    
    let streamUrl = null;
    let resolveStream;
    const streamPromise = new Promise(resolve => { resolveStream = resolve; });

    page.on('request', req => {
      const rUrl = req.url();
      if ((rUrl.includes('.m3u8') || rUrl.includes('.mp4')) && !rUrl.includes('skip-button') && !rUrl.includes('empty')) {
        if (!streamUrl) {
            streamUrl = rUrl;
            console.log('[puppeteer] Found stream:', streamUrl);
            resolveStream(streamUrl);
        }
      }
    });

    // Auto-close any new popup tabs immediately
    console.log('[puppeteer] Navigating to', url);
    await page.goto(url, { waitUntil: 'networkidle2', timeout: 30000 });
    
    // Aggressive click loop to bypass invisible overlay ads
    if (!streamUrl) {
        console.log('[puppeteer] Starting aggressive click loop...');
        for (let i = 0; i < 6; i++) {
            if (streamUrl) break;
            
            const x = 640 + Math.random() * 10;
            const y = 360 + Math.random() * 10;
            try { await page.mouse.click(x, y, {delay: 50}); } catch(e) {}
            
            // Wait to see if stream appears
            await Promise.race([
                streamPromise,
                new Promise(r => setTimeout(() => r('wait'), 1500))
            ]);
            
            if (streamUrl) break;
            
            // close any new tabs
            try {
                const currentPages = await browser.pages();
                for (let j = 1; j < currentPages.length; j++) {
                    await currentPages[j].close();
                }
                await page.bringToFront();
            } catch (e) {}
        }
    }
    
    await browser.close();
    
    if (streamUrl) {
        return {
            provider: 'vidsrc.in (puppeteer)',
            sources: [{ file: streamUrl, type: streamUrl.includes('.m3u8') ? 'hls' : 'mp4' }],
            tracks: []
        };
    }
    
    return null;
  } catch (e) {
    console.error('[puppeteer] Error:', e.message);
    await browser.close();
    return null;
  }
}


// ─── Routes ───────────────────────────────────────────────────────────────────

app.get("/", (req, res) => {
  res.json({ status: "ok", service: "WaveFlix Puppeteer Extractor" });
});

// GET /stream?media=movie&id=862
// GET /stream?media=tv&id=1399&season=1&episode=1
app.get("/stream", async (req, res) => {
  const { media, id, season, episode } = req.query;

  if (!id || !media) {
    return res.status(400).json({ error: "Missing required params: id, media" });
  }

  console.log(`[extractor] Extracting ${media.toUpperCase()} id=${id}`);

  let result = null;

  if (media === 'movie') {
    console.log(`[extractor] Using Torrent for Movie id=${id}`);
    const hostUrl = `${req.protocol}://${req.get('host')}`;
    result = await getTorrentInfo(id, hostUrl);
    
    // Fallback to Puppeteer if torrent fails
    if (!result) {
        console.log(`[extractor] WebTorrent failed, falling back to Puppeteer...`);
        result = await extractStreamWithPuppeteer(id, media, season, episode);
    }
  } else {
    // TV Shows default to Puppeteer
    result = await extractStreamWithPuppeteer(id, media, season, episode);
  }

  if (!result) {
    console.warn(`[extractor] Failed to extract stream for ${media} id=${id}`);
    return res.status(404).json({ error: "No stream found from any provider" });
  }

  console.log(`[extractor] ✓ Got stream successfully`);
  return res.json(result);
});

// GET /stream-video/:tmdbId
// Handles HTTP Range requests to stream the video file
app.get("/stream-video/:tmdbId", (req, res) => {
    const tmdbId = req.params.tmdbId;
    if (!activeEngines.has(tmdbId)) {
        return res.status(404).send('Torrent not found or expired');
    }
    
    const { file } = activeEngines.get(tmdbId);
    
    const range = req.headers.range;
    if (!range) {
        // Fallback for no-range request
        res.writeHead(200, {
            'Content-Length': file.length,
            'Content-Type': 'video/mp4'
        });
        file.createReadStream().pipe(res);
        return;
    }
    
    const parts = range.replace(/bytes=/, "").split("-");
    const start = parseInt(parts[0], 10);
    const end = parts[1] ? parseInt(parts[1], 10) : file.length - 1;
    const chunksize = (end - start) + 1;
    
    res.writeHead(206, {
        'Content-Range': `bytes ${start}-${end}/${file.length}`,
        'Accept-Ranges': 'bytes',
        'Content-Length': chunksize,
        'Content-Type': 'video/mp4'
    });
    
    const stream = file.createReadStream({ start, end });
    stream.pipe(res);
});

app.listen(PORT, () => {
  console.log(`[extractor] WaveFlix Extractor running on http://0.0.0.0:${PORT}`);
});
