import puppeteer from 'puppeteer-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

puppeteer.use(StealthPlugin());

async function extractM3u8(url) {
  console.log('Launching browser...');
  const browser = await puppeteer.launch({ 
    headless: "new", 
    args: ['--no-sandbox', '--window-size=1280,720'],
    channel: "chrome"
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 720 });
  
  let streamUrl = null;
  
  page.on('request', req => {
    const rUrl = req.url();
    if (rUrl.includes('.m3u8') || (rUrl.includes('.mp4') && !rUrl.includes('skip-button'))) {
      if (!streamUrl) {
          streamUrl = rUrl;
          console.log('Found stream:', streamUrl);
      }
    }
  });

  console.log('Navigating to', url);
  try {
    await page.goto(url, { waitUntil: 'networkidle2', timeout: 30000 });
    
    console.log('Clicking center to trigger play...');
    await page.mouse.click(640, 360);
    
    // Wait for a few seconds to let any popups open and the stream to load
    await new Promise(r => setTimeout(r, 5000));
    
    // If it opens a popup, focus back and click again
    const pages = await browser.pages();
    if (pages.length > 2) {
       console.log('Popup detected, closing it and clicking again...');
       await pages[2].close();
       await page.bringToFront();
       await page.mouse.click(640, 360);
       await new Promise(r => setTimeout(r, 5000));
    }
    
    // Check if there are iframes and click in their center
    const frames = page.frames();
    console.log('Frames count:', frames.length);
    
    await page.screenshot({ path: 'screenshot_vidsrc.png' });
  } catch (e) {
    console.log('Error:', e.message);
  }
  
  await browser.close();
  return streamUrl;
}

extractM3u8('https://vidsrc.in/embed/movie/862').then(console.log);
