"use client";

import { useRef, useState, useEffect } from "react";
import Link from "next/link";
import { itemTitle, posterUrl, detailHref } from "@/lib/helpers";
import type { TmdbItem } from "@/lib/types";
import QuickViewModal from "@/components/QuickViewModal";
import { useTranslation } from "@/lib/i18n";

export default function Top10Row({ items }: { items: TmdbItem[] }) {
  const { t } = useTranslation();
  const top10 = items.slice(0, 10);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [canLeft, setCanLeft] = useState(false);
  const [canRight, setCanRight] = useState(false);
  const [selectedItem, setSelectedItem] = useState<TmdbItem | null>(null);

  const updateArrows = () => {
    const el = scrollRef.current;
    if (!el) return;
    setCanLeft(el.scrollLeft > 10);
    setCanRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 10);
  };

  useEffect(() => {
    updateArrows();
    const el = scrollRef.current;
    if (!el) return;
    el.addEventListener("scroll", updateArrows, { passive: true });
    window.addEventListener("resize", updateArrows);
    return () => {
      el.removeEventListener("scroll", updateArrows);
      window.removeEventListener("resize", updateArrows);
    };
  }, [items]);

  const scroll = (dir: "left" | "right") => {
    const el = scrollRef.current;
    if (!el) return;
    const amount = el.clientWidth * 0.75;
    el.scrollBy({ left: dir === "left" ? -amount : amount, behavior: "smooth" });
  };

  return (
    <div className="py-4 relative z-10 w-full px-[4%]">
      {/* Quick view modal */}
      {selectedItem && (
        <QuickViewModal
          isOpen={true}
          item={selectedItem}
          onClose={() => setSelectedItem(null)}
        />
      )}

      <h2 className="mb-6 text-xl font-bold md:text-2xl lg:text-3xl">
        Sedang Tren Sekarang
      </h2>

      <div className="relative group/row">
        {/* Tombol panah kiri */}
        {canLeft && (
          <button
            onClick={() => scroll("left")}
            aria-label="Scroll kiri"
            className="absolute -left-2 top-1/2 -translate-y-1/2 z-20 grid h-12 w-12 place-items-center rounded-full bg-black/70 text-white ring-1 ring-white/20 backdrop-blur transition hover:bg-white/20 hover:ring-white/40"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="w-6 h-6">
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
        )}

        {/* Tombol panah kanan */}
        {canRight && (
          <button
            onClick={() => scroll("right")}
            aria-label="Scroll kanan"
            className="absolute -right-2 top-1/2 -translate-y-1/2 z-20 grid h-12 w-12 place-items-center rounded-full bg-black/70 text-white ring-1 ring-white/20 backdrop-blur transition hover:bg-white/20 hover:ring-white/40"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="w-6 h-6">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        )}

        {/* Daftar film */}
        <div
          ref={scrollRef}
          className="top10-scroll flex gap-0 overflow-x-auto pt-2 pb-2 scroll-smooth"
          style={{ scrollbarWidth: "none", msOverflowStyle: "none" }}
        >
          <style>{`.top10-scroll::-webkit-scrollbar { display: none; }`}</style>
          {top10.map((item, index) => {
            const number = index + 1;
            return (
              <div
                key={item.id}
                className="group relative flex-none transition-transform hover:scale-[1.03]"
              >
                <div className="relative flex items-end pl-6 md:pl-10 py-4">
                  {/* Poster and click handler */}
                  <div 
                    onClick={() => setSelectedItem(item)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        setSelectedItem(item);
                      }
                    }}
                    role="button"
                    tabIndex={0}
                    aria-label={`View details: ${itemTitle(item)}`}
                    className="relative w-[130px] md:w-[160px] lg:w-[200px] aspect-[2/3] rounded-md overflow-hidden bg-white/5 shrink-0 shadow-lg cursor-pointer"
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={posterUrl(item)}
                      alt={itemTitle(item)}
                      className="h-full w-full object-cover"
                      loading="lazy"
                    />
                    <div className="absolute inset-0 bg-black/15 opacity-0 group-hover:opacity-100 transition-opacity" />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
