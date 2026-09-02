"use client";

import { useRef, useState, useEffect } from "react";
import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";
import QuickViewModal from "./QuickViewModal";
import { itemTitle, posterUrl } from "@/lib/helpers";
import { ChevronLeft, ChevronRight } from "./Icons";

interface Props {
  items: TmdbItem[];
  noPadding?: boolean;
  small?: boolean;
  quickView?: boolean;
}

export default function Carousel({ items, noPadding, small, quickView }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [selectedItem, setSelectedItem] = useState<TmdbItem | null>(null);
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(true);

  const updateScrollButtons = () => {
    if (!ref.current) return;
    const { scrollLeft, scrollWidth, clientWidth } = ref.current;
    setCanScrollLeft(scrollLeft > 0);
    setCanScrollRight(Math.ceil(scrollLeft) < scrollWidth - clientWidth);
  };

  useEffect(() => {
    updateScrollButtons();
    window.addEventListener("resize", updateScrollButtons);
    return () => window.removeEventListener("resize", updateScrollButtons);
  }, [items]);

  const scroll = (dir: number) => {
    const el = ref.current;
    if (!el || el.children.length === 0) return;

    const perPage = small ? 6 : Math.max(1, Math.floor(el.clientWidth / 197)); 
    const positions = Array.from(el.children).map((c) => (c as HTMLElement).offsetLeft - (el.firstElementChild as HTMLElement).offsetLeft);
    const currentIdx = positions.reduce(
      (best, pos, i) => (Math.abs(pos - el.scrollLeft) < Math.abs(positions[best] - el.scrollLeft) ? i : best),
      0
    );
    const targetIdx = Math.min(Math.max(0, currentIdx + dir * perPage), positions.length - 1);
    el.scrollTo({ left: positions[targetIdx], behavior: "smooth" });
  };

  if (items.length === 0) return null;

  const arrows = (side: "left" | "right") => {
    if (side === "left" && !canScrollLeft) return null;
    if (side === "right" && !canScrollRight) return null;
    
    return (
      <button
        onClick={() => scroll(side === "left" ? -1 : 1)}
        aria-label={side === "left" ? "Previous" : "Next"}
        className={`absolute ${side === "left" ? "left-0" : "right-0"} top-0 bottom-0 z-20 hidden w-12 items-center justify-center ${
          side === "left"
            ? "bg-gradient-to-r from-bg to-transparent"
            : "bg-gradient-to-l from-bg to-transparent"
        } text-white md:flex hover:text-accent`}
      >
        {side === "left" ? <ChevronLeft /> : <ChevronRight />}
      </button>
    );
  };

  if (small) {
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
          onScroll={updateScrollButtons}
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
      {canScrollLeft && (
        <button
          onClick={() => scroll(-1)}
          aria-label="Previous"
          className="absolute left-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-r from-bg to-transparent text-white md:flex hover:text-accent"
        >
          <ChevronLeft />
        </button>
      )}

      <div
        ref={ref}
        onScroll={updateScrollButtons}
        className={`no-scrollbar flex snap-x snap-mandatory gap-3 overflow-x-auto py-2 ${noPadding ? "" : "scroll-pl-[4%] px-[4%]"}`}
      >
        {items.map((m, i) => (
          <div key={`${m.id}-${i}`} className="w-[185px] shrink-0 snap-start">
            <MovieCard item={m} />
          </div>
        ))}
      </div>

      {canScrollRight && (
        <button
          onClick={() => scroll(1)}
          aria-label="Next"
          className="absolute right-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-l from-bg to-transparent text-white md:flex hover:text-accent"
        >
          <ChevronRight />
        </button>
      )}
    </div>
  );
}
