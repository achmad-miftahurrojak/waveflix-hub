import express from "express";
import cors from "cors";
import { createCipheriv, createDecipheriv, randomBytes } from "crypto";

const app = express();
app.use(cors());
app.use(express.json());

const PORT = 8000;

// ─── Embed.su ────────────────────────────────────────────────────────────────

async function embedSuGetVideo(id, season, episode) {
  let url;
  if (season !== undefined && episode !== undefined) {
    url = `https://embed.su/embed/tv/${id}/${season}/${episode}`;
  } else {
    url = `https://embed.su/embed/movie/${id}`;
  }

  const res = await fetch(url, {
    headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
    signal: AbortSignal.timeout(10000),
  });
  if (!res.ok) return null;
  const html = await res.text();

  const match = html.match(/window\.vConfig = JSON\.parse\(atob\(`(.+?)`\)\)/);
  if (!match) return null;

  const decodedData = JSON.parse(Buffer.from(match[1], "base64").toString());
  const firstDecode = atob(decodedData.hash).split(".").map((item) => item.split("").reverse().join(""));
  const servers = JSON.parse(atob(firstDecode.join("").split("").reverse().join("")));
  return { ...decodedData, servers };
}

async function embedSuGetStream(hash) {
  const res = await fetch(`https://embed.su/api/e/${hash}`, {
    headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
    signal: AbortSignal.timeout(10000),
  });
  if (!res.ok) return null;
  return await res.json();
}

async function tryEmbedSu(id, season, episode) {
  try {
    const video = await embedSuGetVideo(id, season, episode);
    if (!video?.servers?.length) return null;

    const stream = await embedSuGetStream(video.servers[0].hash);
    if (!stream?.source) return null;

    return {
      provider: "embed.su",
      sources: [{ file: stream.source, type: "hls" }],
      tracks: (stream.subtitles || []).map((s) => ({
        file: s.file,
        label: s.label,
        kind: "captions",
      })),
    };
  } catch (e) {
    console.warn("[embed.su] failed:", e.message);
    return null;
  }
}

// ─── VidSrc.rip ───────────────────────────────────────────────────────────────

function parseVidsrcConfig(configString) {
  const content = configString.slice(1, -1).trim();
  const config = {};
  const regex = /(\w+):\s*(?:'([^']*)'|"([^"]*)"|(\[[^\]]*\])|([^,}]+))/g;
  let match;
  while ((match = regex.exec(content)) !== null) {
    const [, key, singleQ, doubleQ, arrVal, unquoted] = match;
    let value = singleQ || doubleQ || unquoted;
    if (arrVal) value = JSON.parse(arrVal);
    if (key === "servers" && typeof value === "string") value = [value];
    config[key] = value;
  }
  return config;
}

function xorEncryptDecrypt(key, message) {
  const keyCodes = Array.from(key, (c) => c.charCodeAt(0));
  const msgCodes = Array.from(message, (c) => c.charCodeAt(0));
  return String.fromCharCode(...msgCodes.map((code, i) => code ^ keyCodes[i % keyCodes.length]));
}

function generateVRF(key, encodedMessage) {
  const decoded = decodeURIComponent(encodedMessage);
  const xored = xorEncryptDecrypt(key, decoded);
  return encodeURIComponent(Buffer.from(xored).toString("base64"));
}

async function tryVidsrcRip(id, season, episode) {
  try {
    const url =
      season !== undefined && episode !== undefined
        ? `https://vidsrc.rip/embed/tv/${id}/${season}/${episode}`
        : `https://vidsrc.rip/embed/movie/${id}`;

    const res = await fetch(url, {
      headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
      signal: AbortSignal.timeout(12000),
    });
    if (!res.ok) return null;
    const html = await res.text();
    const match = html.match(/window\.config\s*=\s*(\{.*?\});/s);
    if (!match) return null;

    const config = parseVidsrcConfig(match[1]);
    const servers = Array.isArray(config.servers) ? config.servers : [config.server];

    for (const server of servers.slice(0, 2)) {
      const apiPath = `/api/source/${server}/${id}`;
      const keyRes = await fetch("https://vidsrc.rip/images/skip-button.png", {
        signal: AbortSignal.timeout(5000),
      });
      const key = await keyRes.text();
      const vrf = generateVRF(key, apiPath);

      let apiUrl = `https://vidsrc.rip${apiPath}?vrf=${vrf}`;
      if (season !== undefined && episode !== undefined) {
        apiUrl += `&s=${season}&e=${episode}`;
      }

      const streamRes = await fetch(apiUrl, {
        headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
        signal: AbortSignal.timeout(10000),
      });
      if (!streamRes.ok) continue;
      const data = await streamRes.json();
      if (data?.sources?.length) {
        return {
          provider: "vidsrc.rip",
          sources: data.sources.map((s) => ({ file: s.file, type: "hls", label: s.label })),
          tracks: [],
        };
      }
    }
    return null;
  } catch (e) {
    console.warn("[vidsrc.rip] failed:", e.message);
    return null;
  }
}

// ─── Vidlink.pro ──────────────────────────────────────────────────────────────

const VIDLINK_KEY_HEX = "2de6e6ea13a9df9503b11a6117fd7e51941e04a0c223dfeacfe8a1dbb6c52783";
const VIDLINK_ALGO = "aes-256-cbc";

function vidlinkEncrypt(data) {
  const iv = randomBytes(16);
  const key = Buffer.from(VIDLINK_KEY_HEX, "hex").slice(0, 32);
  const cipher = createCipheriv(VIDLINK_ALGO, key, iv);
  let encrypted = cipher.update(data);
  encrypted = Buffer.concat([encrypted, cipher.final()]);
  return `${iv.toString("hex")}:${encrypted.toString("hex")}`;
}

function vidlinkDecrypt(data) {
  const [ivHex, encHex] = data.split(":");
  const iv = Buffer.from(ivHex, "hex");
  const encrypted = Buffer.from(encHex, "hex");
  const key = Buffer.from(VIDLINK_KEY_HEX, "hex").slice(0, 32);
  const decipher = createDecipheriv(VIDLINK_ALGO, key, iv);
  let dec = decipher.update(encrypted);
  dec = Buffer.concat([dec, decipher.final()]);
  return dec.toString();
}

async function tryVidlink(id, season, episode) {
  try {
    const encodedId = Buffer.from(vidlinkEncrypt(String(id))).toString("base64");
    const url =
      season !== undefined && episode !== undefined
        ? `https://vidlink.pro/api/b/tv/${encodedId}/${season}/${episode}`
        : `https://vidlink.pro/api/b/movie/${encodedId}`;

    const res = await fetch(url, {
      headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
      signal: AbortSignal.timeout(12000),
    });
    if (!res.ok) return null;

    const rawText = await res.text();
    const decrypted = vidlinkDecrypt(rawText);
    const data = JSON.parse(decrypted);

    if (!data?.stream?.playlist) return null;

    return {
      provider: "vidlink.pro",
      sources: [{ file: data.stream.playlist, type: "hls" }],
      tracks: (data.stream.captions || []).map((c) => ({
        file: c.url,
        label: c.language,
        kind: "captions",
      })),
    };
  } catch (e) {
    console.warn("[vidlink.pro] failed:", e.message);
    return null;
  }
}

// ─── Routes ───────────────────────────────────────────────────────────────────

app.get("/", (req, res) => {
  res.json({ status: "ok", service: "WaveFlix Stream Extractor" });
});

// GET /stream?media=movie&id=862
// GET /stream?media=tv&id=1399&season=1&episode=1
app.get("/stream", async (req, res) => {
  const { media, id, season, episode } = req.query;

  if (!id || !media) {
    return res.status(400).json({ error: "Missing required params: id, media" });
  }

  const s = season ? parseInt(season) : undefined;
  const e = episode ? parseInt(episode) : undefined;
  const isTV = media === "tv";

  console.log(`[extractor] ${media.toUpperCase()} id=${id}${isTV ? ` S${s}E${e}` : ""}`);

  // Try providers in order: embed.su → vidsrc.rip → vidlink.pro
  let result = await tryEmbedSu(id, isTV ? s : undefined, isTV ? e : undefined);

  if (!result) {
    console.log(`[extractor] embed.su failed, trying vidsrc.rip...`);
    result = await tryVidsrcRip(id, isTV ? s : undefined, isTV ? e : undefined);
  }

  if (!result) {
    console.log(`[extractor] vidsrc.rip failed, trying vidlink.pro...`);
    result = await tryVidlink(id, isTV ? s : undefined, isTV ? e : undefined);
  }

  if (!result) {
    console.warn(`[extractor] All providers failed for ${media} id=${id}`);
    return res.status(404).json({ error: "No stream found from any provider" });
  }

  console.log(`[extractor] ✓ Got stream from ${result.provider}`);
  return res.json(result);
});

// Availability check endpoint (called by Go backend for filtering)
// GET /check?media=tv&id=1399&season=1&episode=1
app.get("/check", async (req, res) => {
  const { media, id, season, episode } = req.query;
  if (!id || !media) return res.status(400).json({ available: false });

  const s = season ? parseInt(season) : undefined;
  const e = episode ? parseInt(episode) : undefined;
  const isTV = media === "tv";

  // Fast check: just embedsu video (cheapest request)
  try {
    const video = await embedSuGetVideo(id, isTV ? s : undefined, isTV ? e : undefined);
    if (video?.servers?.length) {
      return res.json({ available: true, provider: "embed.su" });
    }
  } catch (_) {}

  // Fallback check: vidsrc.rip page
  try {
    const url = isTV
      ? `https://vidsrc.rip/embed/tv/${id}/${s}/${e}`
      : `https://vidsrc.rip/embed/movie/${id}`;
    const r = await fetch(url, {
      headers: { "User-Agent": "Mozilla/5.0" },
      signal: AbortSignal.timeout(8000),
    });
    if (r.ok) {
      const html = await r.text();
      if (html.includes("window.config")) return res.json({ available: true, provider: "vidsrc.rip" });
    }
  } catch (_) {}

  return res.json({ available: false });
});

app.listen(PORT, () => {
  console.log(`[extractor] WaveFlix Extractor running on http://0.0.0.0:${PORT}`);
});
