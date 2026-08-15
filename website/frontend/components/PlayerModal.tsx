"use client";

import { useEffect, useRef } from "react";
import type { TmdbItem } from "@/lib/types";
import { embedUrl } from "@/lib/helpers";
import { CloseIcon, MaximizeIcon } from "./Icons";

interface Props {
  state: { item: TmdbItem; season?: number; episode?: number } | null;
  onClose: () => void;
}

export default function PlayerModal({ state, onClose }: Props) {
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    if (state) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", onKey);
    }
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", onKey);
    };
  }, [state, onClose]);

  const toggleFullscreen = () => {
    const el = wrapRef.current;
    if (!el) return;
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      el.requestFullscreen().catch(() => {});
    }
  };

  if (!state) return null;

  return (
    <div
      className="fixed inset-0 z-[2000] flex items-center justify-center bg-black/95 p-4"
      onClick={onClose}
    >
      <div className="absolute right-6 top-6 flex items-center gap-4">
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleFullscreen();
          }}
          aria-label="Fullscreen"
          className="text-white/80 transition hover:text-accent"
        >
          <MaximizeIcon />
        </button>
        <button
          onClick={onClose}
          aria-label="Close player"
          className="text-white/80 transition hover:text-accent"
        >
          <CloseIcon />
        </button>
      </div>

      <div
        ref={wrapRef}
        className="aspect-video w-full max-w-5xl overflow-hidden rounded-xl bg-black shadow-[0_0_40px_rgba(0,229,255,0.2)] [&:fullscreen]:max-w-none [&:fullscreen]:rounded-none"
        onClick={(e) => e.stopPropagation()}
      >
        <iframe
          src={embedUrl(state.item, state.season, state.episode)}
          className="h-full w-full"
          allow="autoplay; encrypted-media; fullscreen; picture-in-picture"
          allowFullScreen
        />
      </div>
    </div>
  );
}
