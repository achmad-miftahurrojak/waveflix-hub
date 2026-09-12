"use client";

import React, { useRef, useState } from "react";
import Link from "next/link";
import gsap from "gsap";
import { useGSAP } from "@gsap/react";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, posterUrl, detailHref } from "@/lib/helpers";

gsap.registerPlugin(ScrollTrigger);

interface Props {
  item: TmdbItem;
  disableLink?: boolean;
}

export default function MovieCard({ item, disableLink }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  

  const content = (
    <div
      ref={cardRef}
      className="relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-surface shadow-md"
    >
      <img
        src={posterUrl(item)}
        alt={itemTitle(item)}
        loading="lazy"
        className="absolute inset-0 h-full w-full object-cover"
      />
      
      {}
      <div 
        className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" 
      />
      
      <span 
        className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-lg" 
      />
    </div>
  );

  if (disableLink) {
    return (
      <div ref={containerRef} className="group block w-full text-left transition-transform duration-300 hover:scale-105" aria-label={itemTitle(item)}>
        {content}
      </div>
    );
  }

  return (
    <Link
      ref={containerRef as any}
      href={detailHref(item)}
      className="group block w-full text-left transition-transform duration-300 hover:scale-105"
      aria-label={itemTitle(item)}
    >
      {content}
    </Link>
  );
}