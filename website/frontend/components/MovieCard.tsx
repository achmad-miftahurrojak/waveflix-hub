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
  const [style, setStyle] = useState<React.CSSProperties>({
    transform: "perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)",
    transition: "transform 0.4s ease-in-out",
  });

  useGSAP(
    () => {
      if (!containerRef.current) return;
      
      gsap.from(containerRef.current, {
        scrollTrigger: {
          trigger: containerRef.current,
          start: "top bottom-=50",
          once: true,
        },
        y: 40,
        opacity: 0,
        duration: 0.8,
        ease: "power3.out",
      });
    },
    { scope: containerRef, dependencies: [] }
  );

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
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
      />
      
      {/* Subtle shadow overlay that reacts to the hover */}
      <div 
        className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" 
        style={{ transform: "translateZ(10px)" }}
      />
      
      <span 
        className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-lg" 
        style={{ transform: "translateZ(30px)" }}
      />
    </div>
  );

  if (disableLink) {
    return (
      <div ref={containerRef} className="group block w-full text-left" aria-label={itemTitle(item)}>
        {content}
      </div>
    );
  }

  return (
    <Link
      ref={containerRef as any}
      href={detailHref(item)}
      className="group block w-full text-left"
      aria-label={itemTitle(item)}
    >
      {content}
    </Link>
  );
}