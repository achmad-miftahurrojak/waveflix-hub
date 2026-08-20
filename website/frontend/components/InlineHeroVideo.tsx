"use client";

import { useEffect, useRef, useState, useCallback, type ReactNode } from "react";
import { EMBED_SERVERS } from "@/lib/embed-servers";
import { getAsianShowSlug } from "@/lib/verified-shows";
import { useUI } from "./UIProvider";
import { MaximizeIcon, MinimizeIcon, CloseIcon } from "./Icons";

interface Props {
  id: number;
  season?: number;
  episode?: number;
  backdrop: string;
  heightClass: string;
  children: ReactNode;
}

export default function InlineHeroVideo({
  id,
  season,
  episode,
  backdrop,
  heightClass,
  children,
}: Props) {
  const { player, stop } = useUI();
  const wrapRef = useRef<HTMLDivElement>(null);
  const [isFs, setIsFs] = useState(false);
  const [controlsVisible, setControlsVisible] = useState(true);
  const hideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-failover states
  const [serverIdx, setServerIdx] = useState(0);
  const [status, setStatus] = useState<"loading" | "loaded" | "asian-loading" | "asian-failed">("loading");
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Asian scraper states
  const [asianEmbedUrl, setAsianEmbedUrl] = useState<string | null>(null);
  const [serverLabel, setServerLabel] = useState("");

  useEffect(() => {
    const onFs = () => setIsFs(!!document.fullscreenElement);
    document.addEventListener("fullscreenchange", onFs);
    return () => document.removeEventListener("fullscreenchange", onFs);
  }, []);

  const revealControls = useCallback(() => {
    setControlsVisible(true);
    if (hideTimer.current) clearTimeout(hideTimer.current);
    hideTimer.current = setTimeout(() => setControlsVisible(false), 2600);
  }, []);

  const active =
    !!player &&
    player.item.id === id &&
    player.season === season &&
    player.episode === episode;

  // Tentukan apakah ini Variety Show terverifikasi (via Whitelist atau Dinamis)
  const slug = active ? getAsianShowSlug(player!.item) : null;

  // Fetch Asian embed ketika ini variety show terverifikasi
  useEffect(() => {
    if (!active || !slug) return;

    const ep = player!.episode ?? 1;
    setStatus("asian-loading");
    setAsianEmbedUrl(null);
    setServerLabel("Dramacool");

    let cancelled = false;

    const fetchAsianEmbed = async () => {
      try {
        const res = await fetch(`/api/asian-embed?slug=${slug}&ep=${ep}`);
        if (!res.ok) throw new Error("not found");
        const data = await res.json();
        if (!cancelled && data.url) {
          setAsianEmbedUrl(data.url);
          setStatus("loading"); // sekarang loading iframe dari dramacool
        } else {
          throw new Error("empty url");
        }
      } catch {
        if (!cancelled) {
          // Asian scraper gagal, fallback ke sistem embed biasa
          setStatus("loading");
          setServerIdx(0);
        }
      }
    };

    fetchAsianEmbed();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, slug]);

  // Reset untuk NON-variety show
  useEffect(() => {
    if (!active) {
      if (hideTimer.current) clearTimeout(hideTimer.current);
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      return;
    }
    if (!slug) {
      // Bukan variety show → langsung pakai embed server biasa
      setServerIdx(0);
      setStatus("loading");
      setAsianEmbedUrl(null);
    }
    revealControls();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  // Auto-failover timer (hanya untuk embed server biasa, bukan asian)
  useEffect(() => {
    if (!active || status === "loaded" || status === "asian-loading" || asianEmbedUrl) {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      return;
    }

    timeoutRef.current = setTimeout(() => {
      setServerIdx((prev) => (prev + 1) % EMBED_SERVERS.length);
    }, 8000);

    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, [active, status, serverIdx, asianEmbedUrl]);

  const switchServer = () => {
    // Jika sedang di mode Asian, pindah ke embed server biasa
    if (asianEmbedUrl) {
      setAsianEmbedUrl(null);
      setServerIdx(0);
      setStatus("loading");
      return;
    }
    setStatus("loading");
    setServerIdx((prev) => (prev + 1) % EMBED_SERVERS.length);
  };

  const toggleFullscreen = () => {
    const el = wrapRef.current;
    if (!el) return;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    else el.requestFullscreen().catch(() => {});
  };

  // Tentukan URL dan label yang ditampilkan
  const currentSrc = asianEmbedUrl
    ? asianEmbedUrl
    : (active ? EMBED_SERVERS[serverIdx].getUrl(player!.item, player!.season, player!.episode) : "");

  const currentLabel = asianEmbedUrl
    ? "Dramacool"
    : (EMBED_SERVERS[serverIdx]?.name ?? "");

  const isLoading = status === "loading" || status === "asian-loading";

  if (active) {
    return (
      <div
        ref={wrapRef}
        onMouseMove={revealControls}
        onMouseLeave={() => setControlsVisible(false)}
        className={`relative w-full bg-black ${heightClass}`}
      >
        {isLoading && (
          <div className="absolute inset-0 z-0 flex items-center justify-center bg-black">
            <div className="flex flex-col items-center gap-4">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-white/20 border-t-accent" />
              <p className="text-sm text-white/70">
                {status === "asian-loading"
                  ? "Mencari video di Dramacool..."
                  : `Menghubungkan ke server ${currentLabel}...`}
              </p>
            </div>
          </div>
        )}

        {/* Jangan render iframe kalau masih asian-loading (belum punya URL) */}
        {status !== "asian-loading" && currentSrc && (
          <iframe
            src={currentSrc}
            onLoad={() => setStatus("loaded")}
            onError={switchServer}
            className={`absolute inset-0 h-full w-full transition-opacity duration-500 ${
              status === "loaded" ? "opacity-100 z-10" : "opacity-0 -z-10"
            }`}
            allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
            allowFullScreen
          />
        )}

        {/* Zona atas deteksi hover */}
        <div
          className="absolute inset-x-0 top-0 z-20 h-32"
          onMouseMove={revealControls}
        />

        <div
          className={`absolute right-5 top-24 z-30 flex items-center gap-3 transition-opacity duration-300 ${
            controlsVisible ? "opacity-100" : "pointer-events-none opacity-0"
          }`}
        >
          <button
            onClick={switchServer}
            title="Ganti Server (Failover)"
            aria-label="Ganti Server"
            className="flex h-9 items-center gap-2 rounded-full bg-black/60 px-3 text-xs font-semibold text-white/80 transition hover:text-accent"
          >
            <span className="text-lg leading-none">⟳</span>
            <span className="hidden sm:inline">
              {currentLabel}
            </span>
          </button>

          <button
            onClick={toggleFullscreen}
            aria-label="Toggle Fullscreen"
            className="grid h-9 w-9 place-items-center rounded-full bg-black/60 text-white/80 transition hover:text-accent"
          >
            {isFs ? <MinimizeIcon /> : <MaximizeIcon />}
          </button>
          <button
            onClick={stop}
            aria-label="Close player"
            className="grid h-9 w-9 place-items-center rounded-full bg-black/60 text-white/80 transition hover:text-accent"
          >
            <CloseIcon />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      className={`relative flex items-end ${heightClass}`}
      style={{
        backgroundImage: `url('${backdrop}')`,
        backgroundSize: "cover",
        backgroundPosition: "center top",
      }}
    >
      <div className="pointer-events-none absolute inset-0 bg-black/45" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-bg to-transparent" />
      {children}
    </div>
  );
}
