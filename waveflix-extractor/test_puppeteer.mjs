import puppeteer from 'puppeteer-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

puppeteer.use(StealthPlugin());

async function extractM3u8(url) {
  console.log('Launching browser...');
  const browser = await puppeteer.launch({ 
    headless: "new", 
    args: ['--no-sandbox'],
    channel: "chrome"
  });
  const page = await browser.newPage();
  
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
    await page.goto(url, { waitUntil: 'networkidle2', timeout: 15000 });
  } catch (e) {
    console.log('Timeout or error during goto:', e.message);
  }
  
  if (!streamUrl) {
    console.log('Waiting extra time just in case...');
    await new Promise(r => setTimeout(r, 5000));
  }

  await browser.close();
  return streamUrl;
}

extractM3u8('https://vidsrc.in/embed/movie/862').then(console.log);
