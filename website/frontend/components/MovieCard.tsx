"use client";

import React, { useRef, useState, useEffect } from "react";
import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, posterUrl, detailHref, isTv, BACKEND, IMG } from "@/lib/helpers";

interface Props {
  item: TmdbItem;
  disableLink?: boolean;
}

interface TmdbLogo {
  file_path: string;
  iso_639_1: string | null;
}

export default function MovieCard({ item, disableLink }: Props) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [style, setStyle] = useState<React.CSSProperties>({
    transform: "perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)",
    transition: "transform 0.4s ease-in-out",
  });
  const [logoPath, setLogoPath] = useState<string | null>(null);
  const [logoFetched, setLogoFetched] = useState(false);

  const fetchLogo = () => {
    if (logoFetched) return;
    setLogoFetched(true);
    const media = isTv(item) ? "tv" : "movie";
    const imgLang = item.original_language || "en";
    fetch(`${BACKEND}/api/images?media=${media}&id=${item.id}&lang=${imgLang}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => {
        const logos: TmdbLogo[] = d?.logos ?? [];
        const best =
          logos.find((l) => l.iso_639_1 === "en") ??
          logos.find((l) => l.iso_639_1 === null) ??
          logos[0];
        setLogoPath(best?.file_path ?? null);
      })
      .catch(() => setLogoPath(null));
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    fetchLogo();
    if (!cardRef.current) return;

    const { left, top, width, height } = cardRef.current.getBoundingClientRect();
    const x = e.clientX - left;
    const y = e.clientY - top;

    // Max rotation 8deg
    const rotateX = ((y - height / 2) / (height / 2)) * -8;
    const rotateY = ((x - width / 2) / (width / 2)) * 8;

    setStyle({
      transform: `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale3d(1.05, 1.05, 1.05)`,
      transition: "transform 0.1s ease-out",
    });
  };

  const handleMouseLeave = () => {
    setStyle({
      transform: "perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)",
      transition: "transform 0.4s ease-in-out",
    });
  };

  const content = (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      style={{ ...style, transformStyle: "preserve-3d" }}
      className="relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-surface shadow-md"
    >
      <img
        src={posterUrl(item)}
        alt={itemTitle(item)}
        loading="lazy"
        className="absolute inset-0 h-full w-full object-cover transition-transform duration-300"
        style={{ transform: "translateZ(-20px) scale(1.1)" }}
      />
      
      {/* Subtle shadow overlay that reacts to the hover */}
      <div 
        className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" 
        style={{ transform: "translateZ(10px)" }}
      />

      {/* Logo overlay */}
      <div 
        className="absolute inset-x-0 bottom-4 flex items-end justify-center px-4 opacity-0 transition-all duration-300 group-hover:opacity-100 group-hover:bottom-6"
        style={{ transform: "translateZ(40px)" }}
      >
        {logoPath ? (
          <img
            src={`${IMG}/w500${logoPath}`}
            alt={itemTitle(item)}
            className="max-h-16 w-auto object-contain drop-shadow-logo"
            loading="lazy"
          />
        ) : (
          <h3 className="text-center text-sm font-bold text-white drop-shadow-md line-clamp-2">
            {itemTitle(item)}
          </h3>
        )}
      </div>
      
      <span 
        className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-lg" 
        style={{ transform: "translateZ(50px)" }}
      />
    </div>
  );

  if (disableLink) {
    return (
      <div className="group block w-full text-left" aria-label={itemTitle(item)}>
        {content}
      </div>
    );
  }

  return (
    <Link
      href={detailHref(item)}
      className="group block w-full text-left"
      aria-label={itemTitle(item)}
    >
      {content}
    </Link>
  );
}