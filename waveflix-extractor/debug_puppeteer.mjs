import puppeteer from 'puppeteer-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

puppeteer.use(StealthPlugin());

async function debugExtractor(id) {
  const url = `https://vidsrc.in/embed/movie/${id}`;
  console.log('[debug] Launching browser for', url);
  const browser = await puppeteer.launch({ 
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,720'],
    channel: 'chrome'
  });
  
  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 720 });
  
  page.on('request', req => {
      if (req.url().includes('m3u8') || req.url().includes('mp4')) {
          console.log('[network] stream request:', req.url());
      }
  });

  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  console.log('Taking initial screenshot...');
  await page.screenshot({ path: 'debug_step1.png' });
  
  await new Promise(r => setTimeout(r, 5000));
  console.log('Taking screenshot after 5s...');
  await page.screenshot({ path: 'debug_step2.png' });
  
  console.log('Clicking center...');
  try { await page.mouse.click(640, 360); } catch(e){}
  await new Promise(r => setTimeout(r, 5000));
  
  console.log('Taking screenshot after click...');
  await page.screenshot({ path: 'debug_step3.png' });
  
  const pages = await browser.pages();
  console.log('Total pages open:', pages.length);
  
  await browser.close();
}

debugExtractor('1268609');
