"use client";

import React, { useRef, useState } from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";

export interface CollectionCardItem {
  id: string;
  name: string;
  logoSrc?: string;
  backdropSrc?: string;
  href: string;
}

function CollectionCard3D({ item }: { item: CollectionCardItem }) {
  const cardRef = useRef<HTMLAnchorElement>(null);
  const [style, setStyle] = useState<React.CSSProperties>({
    transform: "perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)",
    transition: "transform 0.4s ease-in-out",
  });

  const handleMouseMove = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (!cardRef.current) return;

    const { left, top, width, height } = cardRef.current.getBoundingClientRect();
    const x = e.clientX - left;
    const y = e.clientY - top;

    
    const rotateX = ((y - height / 2) / (height / 2)) * -6;
    const rotateY = ((x - width / 2) / (width / 2)) * 6;

    setStyle({
      transform: `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale3d(1.02, 1.02, 1.02)`,
      transition: "transform 0.1s ease-out",
    });
  };

  const handleMouseLeave = () => {
    setStyle({
      transform: "perspective(1000px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)",
      transition: "transform 0.4s ease-in-out",
    });
  };

  return (
    <Link
      href={item.href}
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      style={{ ...style, transformStyle: "preserve-3d" }}
      className="group relative block aspect-[16/9] w-full overflow-hidden rounded-xl bg-background shadow-md border border-white/5 hover:border-white/20"
    >
      {}
      {item.backdropSrc && (
        <div className="absolute inset-0 w-full h-full">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={item.backdropSrc}
            alt={item.name}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        </div>
      )}

      {}
      <div 
        className="absolute inset-0 bg-gradient-to-r from-background from-20% via-background/90 via-50% to-transparent pointer-events-none" 
        style={{ transform: "translateZ(10px)" }}
      />

      {}
      <div 
        className="absolute inset-y-0 left-0 flex w-[65%] flex-col justify-center p-4 pointer-events-none"
        style={{ transform: "translateZ(30px)" }}
      >
        {item.logoSrc ? (
          /* eslint-disable-next-line @next/next/no-img-element */
          <img
            src={item.logoSrc}
            alt={item.name}
            className="max-h-[35px] lg:max-h-[45px] w-auto max-w-[80%] object-contain object-left drop-shadow-2xl filter transition-transform duration-300"
          />
        ) : (
          <h3 className="text-sm lg:text-lg font-black uppercase tracking-tight text-white drop-shadow-lg" style={{ fontFamily: "var(--font-heading)" }}>
            {item.name}
          </h3>
        )}
      </div>

      {}
      <span 
        className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-xl" 
        style={{ transform: "translateZ(30px)" }}
      />
    </Link>
  );
}

export function CollectionCards({
  items,
  className,
}: {
  items: CollectionCardItem[];
  className?: string;
}) {
  return (
    <div className={cn("grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4", className)} style={{ perspective: "1500px" }}>
      {items.map((item) => (
        <CollectionCard3D key={item.id} item={item} />
      ))}
    </div>
  );
}
