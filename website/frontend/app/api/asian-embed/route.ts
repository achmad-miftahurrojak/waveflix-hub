import { NextRequest, NextResponse } from "next/server";
import * as cheerio from "cheerio";

/**
 * API Route: /api/asian-embed?slug=running-man&ep=715
 *
 * Scraper khusus untuk Variety Show Korea yang tidak bisa dimainkan
 * di server embed biasa (VidLink, VidSrc, dll) karena masalah
 * perbedaan format episode TMDB vs database Asia.
 *
 * Alur kerja:
 * 1. Coba beberapa domain mirror Dramacool/AsianLoad
 * 2. Scrape halaman episode untuk menemukan iframe/video embed
 * 3. Kembalikan URL embed yang bisa langsung dipakai di <iframe>
 */

const DRAMACOOL_MIRRORS = [
  "https://asianc.to",
  "https://dramanice.so",
  "https://dramacool.com.tr",
  "https://watchasian.sh",
  "https://dramahood.info",
  "https://myasiantv.cc",
];

const USER_AGENT =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36";

/**
 * Mencoba mengekstrak URL embed dari halaman episode Dramacool/AsianLoad.
 * Halaman-halaman ini biasanya memiliki:
 * - <iframe> di dalam div.play-video atau div.watch_video
 * - Atribut data-video di div.video-block
 */
function extractEmbedUrl(html: string): string | null {
  const $ = cheerio.load(html);

  // Cari semua kemungkinan elemen yang menyimpan video
  const candidates = [
    $("div.play-video iframe").attr("src"),
    $("div.watch_video iframe").attr("src"),
    $("div.watch-video iframe").attr("src"),
    $("div.anime_muti_link iframe").attr("src"),
    $(".video-block").attr("data-video"),
    $("iframe#main-embed").attr("src"),
    $("iframe#playerframe").attr("src"),
    $("iframe[allowfullscreen]").first().attr("src"),
  ];

  for (const url of candidates) {
    if (url && url.trim().length > 10) {
      let cleaned = url.trim();
      if (cleaned.startsWith("//")) cleaned = "https:" + cleaned;
      return cleaned;
    }
  }

  return null;
}

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const slug = searchParams.get("slug");
  const ep = searchParams.get("ep");

  if (!slug || !ep) {
    return NextResponse.json(
      { error: "Missing slug or ep" },
      { status: 400 }
    );
  }

  if (!/^[a-zA-Z0-9-]+$/.test(slug) || !/^[0-9]+$/.test(ep)) {
    return NextResponse.json(
      { error: "Invalid slug or ep format" },
      { status: 400 }
    );
  }

  // URL pattern yang umum dipakai situs Asia
  const pathVariants = [
    `/${slug}-episode-${ep}.html`,
    `/${slug}-episode-${ep}`,
    `/drama/${slug}/episode-${ep}`,
    `/${slug}-episode-${ep}-english-sub.html`,
  ];

  const urls: string[] = [];
  for (const mirror of DRAMACOOL_MIRRORS) {
    for (const path of pathVariants) {
      urls.push(mirror + path);
    }
  }

  try {
    const result = await Promise.any(
      urls.map(async (url) => {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 6000);

        const res = await fetch(url, {
          headers: {
            "User-Agent": USER_AGENT,
            Accept:
              "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.5",
          },
          signal: controller.signal,
          redirect: "follow",
        });

        clearTimeout(timeout);

        if (!res.ok) throw new Error("Not OK");

        const html = await res.text();

        if (
          html.includes("Page not found") ||
          html.includes("404") ||
          html.length < 1000
        ) {
          throw new Error("Invalid page content");
        }

        const embedUrl = extractEmbedUrl(html);
        if (embedUrl) {
          const sourceMirror = DRAMACOOL_MIRRORS.find(m => url.startsWith(m)) || "unknown";
          return {
            url: embedUrl,
            source: sourceMirror,
          };
        }
        throw new Error("Embed not found");
      })
    );
    
    return NextResponse.json(result);
  } catch (error) {
    return NextResponse.json(
      { error: "Video not found on any mirror" },
      { status: 404 }
    );
  }
}
