const DRAMACOOL_MIRRORS = [
  "https://asianc.to",
  "https://dramanice.so",
  "https://dramacool.com.tr",
  "https://watchasian.sh",
  "https://dramahood.info",
  "https://myasiantv.cc",
];

async function backtestMirrors() {
  console.log("=== Backtesting Scraper Mirrors ===");
  const results = [];

  for (const mirror of DRAMACOOL_MIRRORS) {
    const start = Date.now();
    try {
      const res = await fetch(mirror, {
        headers: {
          "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
        },
        signal: AbortSignal.timeout(5000),
      });
      const duration = Date.now() - start;
      
      results.push({
        mirror,
        status: res.status,
        ok: res.ok,
        latency: `${duration}ms`,
      });
    } catch (e) {
      results.push({
        mirror,
        status: "FAILED",
        ok: false,
        latency: "timeout/error",
        error: e.message,
      });
    }
  }

  console.table(results);
}

backtestMirrors();
