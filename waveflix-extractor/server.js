import express from "express";
import cors from "cors";
import puppeteer from 'puppeteer-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

puppeteer.use(StealthPlugin());

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
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,720'],
    channel: 'chrome'
  });
  
  try {
    const page = await browser.newPage();
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

    console.log('[puppeteer] Navigating to', url);
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
    
    // Wait for stream to be caught or timeout
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 5000, 'timeout'));
    let result = await Promise.race([streamPromise, timeoutPromise]);
    
    if (result === 'timeout' && !streamUrl) {
        console.log('[puppeteer] Clicking center to trigger play...');
        await new Promise(r => setTimeout(r, 3000));
        try { await page.mouse.click(640, 360); } catch(e) {}
        
        result = await Promise.race([
            streamPromise, 
            new Promise(resolve => setTimeout(resolve, 8000, 'timeout2'))
        ]);
        
        if (result === 'timeout2' && !streamUrl) {
            console.log('[puppeteer] Second click attempt (closing popups)...');
            const pages = await browser.pages();
            if (pages.length > 2) {
               try { await pages[2].close(); } catch(e) {}
            }
            await page.bringToFront();
            try { await page.mouse.click(640, 360); } catch(e) {}
            await Promise.race([
                streamPromise, 
                new Promise(resolve => setTimeout(resolve, 5000, 'timeout3'))
            ]);
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

  const result = await extractStreamWithPuppeteer(id, media, season, episode);

  if (!result) {
    console.warn(`[extractor] Failed to extract stream for ${media} id=${id}`);
    return res.status(404).json({ error: "No stream found from any provider" });
  }

  console.log(`[extractor] ✓ Got stream successfully`);
  return res.json(result);
});

app.listen(PORT, () => {
  console.log(`[extractor] WaveFlix Extractor running on http://0.0.0.0:${PORT}`);
});
