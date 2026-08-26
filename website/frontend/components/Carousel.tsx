"use client";

import { useRef, useState } from "react";
import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";
import QuickViewModal from "./QuickViewModal";
import { itemTitle, posterUrl } from "@/lib/helpers";
import { ChevronLeft, ChevronRight } from "./Icons";

interface Props {
  items: TmdbItem[];
  ranked?: boolean;
  noPadding?: boolean;
  small?: boolean; // ukuran card kecil (untuk landing page)
  quickView?: boolean; // klik buka modal, bukan navigasi ke detail
}

export default function Carousel({ items, ranked, noPadding, small, quickView }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [selectedItem, setSelectedItem] = useState<TmdbItem | null>(null);

  const scroll = (dir: number) => {
    const el = ref.current;
    if (!el || el.children.length === 0) return;
    // Snap point = offsetLeft tiap card, jadi mendarat selalu rata
    const perPage = small ? 6 : Math.max(1, Math.floor(el.clientWidth / 197)); // 185px card + 12px gap
    const positions = Array.from(el.children).map((c) => (c as HTMLElement).offsetLeft - (el.firstElementChild as HTMLElement).offsetLeft);
    const currentIdx = positions.reduce(
      (best, pos, i) => (Math.abs(pos - el.scrollLeft) < Math.abs(positions[best] - el.scrollLeft) ? i : best),
      0
    );
    const targetIdx = Math.min(Math.max(0, currentIdx + dir * perPage), positions.length - 1);
    el.scrollTo({ left: positions[targetIdx], behavior: "smooth" });
  };

  if (items.length === 0) return null;

  const arrows = (side: "left" | "right") => (
    <button
      onClick={() => scroll(side === "left" ? -1 : 1)}
      aria-label={side === "left" ? "Previous" : "Next"}
      className={`absolute ${side === "left" ? "left-0" : "right-0"} top-0 bottom-0 z-20 hidden w-12 items-center justify-center ${
        side === "left"
          ? "bg-gradient-to-r from-bg to-transparent"
          : "bg-gradient-to-l from-bg to-transparent"
      } text-white opacity-0 transition group-hover/car:opacity-100 group-focus-within/car:opacity-100 md:flex hover:text-accent`}
    >
      {side === "left" ? <ChevronLeft /> : <ChevronRight />}
    </button>
  );

  if (small) {
    // Landing: viewport persis 6 card (6×170px + 5×12px gap), sisanya tersembunyi
    return (
      <div className="group/car relative mx-auto w-full max-w-[1080px]">
        {quickView && selectedItem && (
          <QuickViewModal
            isOpen={true}
            item={selectedItem}
            onClose={() => setSelectedItem(null)}
          />
        )}

        {arrows("left")}

        <div
          ref={ref}
          className="no-scrollbar flex snap-x snap-mandatory gap-3 overflow-x-auto py-2"
        >
          {items.map((m, i) => (
            <div
              key={`${m.id}-${i}`}
              className="w-[170px] shrink-0 snap-start"
            >
              {quickView ? (
                <div
                  onClick={() => setSelectedItem(m)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      setSelectedItem(m);
                    }
                  }}
                  role="button"
                  tabIndex={0}
                  aria-label={`Lihat detail: ${itemTitle(m)}`}
                  className="group block w-full cursor-pointer rounded-lg"
                >
                  <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-surface shadow-lg transition-transform duration-300 ease-out group-hover:scale-[1.05] group-hover:shadow-card">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={posterUrl(m)}
                      alt={itemTitle(m)}
                      loading="lazy"
                      className="h-full w-full object-cover"
                    />
                    <span className="pointer-events-none absolute inset-0 ring-0 ring-accent/0 transition group-hover:ring-2 group-hover:ring-accent/60" />
                  </div>
                </div>
              ) : (
                <MovieCard item={m} />
              )}
            </div>
          ))}
        </div>

        {arrows("right")}
      </div>
    );
  }

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
        className={`no-scrollbar flex snap-x snap-mandatory gap-3 overflow-x-auto py-2 ${noPadding ? "" : "scroll-pl-[4%] px-[4%]"}`}
      >
        {items.map((m, i) => (
          <div key={`${m.id}-${i}`} className="w-[185px] shrink-0 snap-start">
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
