"use client";

import { useRef } from "react";
import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";
import { ChevronLeft, ChevronRight } from "./Icons";

interface Props {
  items: TmdbItem[];
  ranked?: boolean;
  noPadding?: boolean;
}

export default function Carousel({ items, ranked, noPadding }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  const scroll = (dir: number) => {
    if (ref.current) {
      const clientWidth = ref.current.clientWidth;
      const scrollAmount = clientWidth * 0.75; // Scroll by 75% of container width
      ref.current.scrollBy({ left: dir * scrollAmount, behavior: "smooth" });
    }
  };

  if (items.length === 0) return null;

  return (
    <div className="group/car relative">
      <button
        onClick={() => scroll(-1)}
        aria-label="Previous"
        className="absolute left-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-r from-bg to-transparent text-white opacity-0 transition group-hover/car:opacity-100 md:flex hover:text-accent"
      >
        <ChevronLeft />
      </button>

      <div
        ref={ref}
        className={`no-scrollbar flex gap-3 overflow-x-auto py-2 ${noPadding ? '' : 'px-[4%]'}`}
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
        className="absolute right-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-l from-bg to-transparent text-white opacity-0 transition group-hover/car:opacity-100 md:flex hover:text-accent"
      >
        <ChevronRight />
      </button>
    </div>
  );
}
