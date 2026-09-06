import puppeteer from 'puppeteer-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

puppeteer.use(StealthPlugin());

async function getStreamWithPuppeteer(id, mediaType, season, episode) {
  let url = mediaType === 'tv' 
    ? `https://vidsrc.in/embed/tv/${id}/${season}/${episode}` 
    : `https://vidsrc.in/embed/movie/${id}`;
    
  console.log('[puppeteer] Launching browser for', url);
  const browser = await puppeteer.launch({ 
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,720'],
    channel: 'chrome' // uses system chrome
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
    
    // Some embeds start playing automatically, but vidsrc.in sometimes needs a click
    // We will wait 3 seconds, if no stream is found, try clicking the center
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 5000, 'timeout'));
    
    let result = await Promise.race([streamPromise, timeoutPromise]);
    
    if (result === 'timeout' && !streamUrl) {
        console.log('[puppeteer] Clicking center to trigger play...');
        
        // Wait another 3s just in case iframe hasn't loaded
        await new Promise(r => setTimeout(r, 3000));
        await page.mouse.click(640, 360);
        
        result = await Promise.race([
            streamPromise, 
            new Promise(resolve => setTimeout(resolve, 8000, 'timeout2'))
        ]);
        
        if (result === 'timeout2' && !streamUrl) {
            console.log('[puppeteer] Second click attempt (closing popups)...');
            const pages = await browser.pages();
            if (pages.length > 2) {
               await pages[2].close();
            }
            await page.bringToFront();
            await page.mouse.click(640, 360);
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

// simple test
getStreamWithPuppeteer('862', 'movie').then(res => {
    console.log('Result:', res);
});
