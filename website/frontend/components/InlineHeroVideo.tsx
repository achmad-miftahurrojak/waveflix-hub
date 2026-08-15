"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { embedUrl } from "@/lib/helpers";
import { useUI } from "./UIProvider";
import { MaximizeIcon, MinimizeIcon, CloseIcon } from "./Icons";

interface Props {
  id: number;
  season?: number;
  episode?: number;
  backdrop: string;
  heightClass: string; // mis. "h-[86vh] min-h-[560px]"
  children: ReactNode; // kolom info (judul/meta/tombol) saat tidak diputar
}

/**
 * Hero yang bisa memutar video INLINE (menggantikan backdrop), bukan popup.
 * Aktif kalau target play global cocok dengan judul/episode hero ini.
 */
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

  useEffect(() => {
    const onFs = () => setIsFs(!!document.fullscreenElement);
    document.addEventListener("fullscreenchange", onFs);
    return () => document.removeEventListener("fullscreenchange", onFs);
  }, []);

  // Tampilkan kontrol lalu sembunyikan otomatis setelah diam sejenak.
  const revealControls = () => {
    setControlsVisible(true);
    if (hideTimer.current) clearTimeout(hideTimer.current);
    hideTimer.current = setTimeout(() => setControlsVisible(false), 2600);
  };

  const active =
    !!player &&
    player.item.id === id &&
    player.season === season &&
    player.episode === episode;

  useEffect(() => {
    if (active) revealControls();
    return () => {
      if (hideTimer.current) clearTimeout(hideTimer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  const toggleFullscreen = () => {
    const el = wrapRef.current;
    if (!el) return;
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    else el.requestFullscreen().catch(() => {});
  };

  if (active) {
    return (
      <div
        ref={wrapRef}
        onMouseMove={revealControls}
        onMouseLeave={() => setControlsVisible(false)}
        className={`relative w-full bg-black ${heightClass}`}
      >
        <iframe
          src={embedUrl(player!.item, player!.season, player!.episode)}
          className="absolute inset-0 h-full w-full"
          allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
          allowFullScreen
        />
        {/* Zona atas: iframe menelan event mouse, jadi deteksi gerak di sini. */}
        <div
          className="absolute inset-x-0 top-0 z-10 h-32"
          onMouseMove={revealControls}
        />
        <div
          className={`absolute right-5 top-24 z-20 flex items-center gap-3 transition-opacity duration-300 ${
            controlsVisible ? "opacity-100" : "pointer-events-none opacity-0"
          }`}
        >
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
