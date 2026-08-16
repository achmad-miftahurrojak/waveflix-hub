"use client";

import { useRef } from "react";
import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";
import { ChevronLeft, ChevronRight } from "./Icons";

interface Props {
  items: TmdbItem[];
  ranked?: boolean;
}

export default function Carousel({ items, ranked }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const scroll = (dir: number) =>
    ref.current?.scrollBy({ left: dir * 640, behavior: "smooth" });

  if (items.length === 0) return null;

  return (
    <div className="group/car relative">
      <button
        onClick={() => scroll(-1)}
        aria-label="Previous"
        className="absolute left-0 top-1/2 z-20 hidden h-full -translate-y-1/2 items-center bg-gradient-to-r from-bg to-transparent px-1 text-white/80 opacity-0 transition group-hover/car:flex group-hover/car:opacity-100 hover:text-accent"
      >
        <ChevronLeft />
      </button>

      <div
        ref={ref}
        className="no-scrollbar flex gap-3 overflow-x-auto scroll-smooth px-[4%] py-8"
      >
        {items.map((m, i) => (
          <div key={`${m.id}-${i}`} className="w-[185px] shrink-0">
            <MovieCard item={m} rank={ranked ? i + 1 : undefined} />
          </div>
        ))}
      </div>

      <button
        onClick={() => scroll(1)}
        aria-label="Next"
        className="absolute right-0 top-1/2 z-20 hidden h-full -translate-y-1/2 items-center bg-gradient-to-l from-bg to-transparent px-1 text-white/80 opacity-0 transition group-hover/car:flex group-hover/car:opacity-100 hover:text-accent"
      >
        <ChevronRight />
      </button>
    </div>
  );
}
