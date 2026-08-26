"use client";

import { useEffect, useState, useRef } from "react";
import type { HistoryItem } from "@/lib/types";
import { useAuth } from "./AuthProvider";
import ContinueWatchingCard from "./ContinueWatchingCard";
import { ChevronLeft, ChevronRight } from "./Icons";
import { useTranslation } from "@/lib/i18n";

export default function ContinueWatchingRow() {
  const { user, ready, authFetch, activeProfile } = useAuth();
  const { t } = useTranslation();
  const [items, setItems] = useState<HistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ready || !user) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setItems([]); // reset saat ganti profil

    authFetch("/api/history")
      .then((r) => r.json())
      .then((d) => {
        const list = (d.results ?? []).map((r: any) => ({
          id: r.tmdb_id ?? r.id,
          title: r.title,
          name: r.title,
          media_type: r.media_type,
          poster_path: r.poster_path,
          backdrop_path: r.poster_path, // fallback if needed, API could be updated
          vote_average: r.vote_average,
          first_air_date: r.media_type === "tv" ? " " : undefined,
          season: r.season,
          episode: r.episode,
          runtime: r.runtime || 120,
          progress: r.progress || 0,
        }));
        
        // Filter out completed items (progress >= runtime * 0.95 or similar). 
        // We simulate unfinished as progress < runtime. 
        const unfinished = list.filter((m: any) => {
          if (m.progress === 0) return true; // Just started
          return m.progress > 0 && m.progress < (m.runtime * 0.95);
        });

        setItems(unfinished);
      })
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [user, ready, authFetch, activeProfile]);

  const handleRemove = (id: number) => {
    // Optimistic UI update
    setItems((prev) => prev.filter((item) => item.id !== id));
    
    // Delete from backend
    authFetch(`/api/history?tmdb_id=${id}`, { method: "DELETE" })
      .catch((err) => console.error("Failed to delete history item", err));
  };

  const scroll = (dir: number) => {
    if (ref.current) {
      const clientWidth = ref.current.clientWidth;
      const scrollAmount = clientWidth * 0.75;
      ref.current.scrollBy({ left: dir * scrollAmount, behavior: "smooth" });
    }
  };

  if (loading || items.length === 0) return null;

  return (
    <section className="mb-3">
      <h2 className="mb-1 px-[4%] text-xl font-bold">{t("ui.continueWatching")}</h2>
      <div className="group/car relative">
        <button
          onClick={() => scroll(-1)}
          aria-label="Previous"
          className="absolute left-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-r from-bg via-bg/80 to-transparent text-white opacity-0 transition-opacity duration-300 group-hover/car:opacity-100 md:flex hover:text-accent"
        >
          <ChevronLeft />
        </button>

        <div
          ref={ref}
          className="no-scrollbar flex snap-x snap-mandatory scroll-pl-[4%] gap-4 overflow-x-auto py-2 px-[4%]"
        >
          {items.map((m, i) => (
            <div key={`${m.id}-${i}`} className="w-[280px] shrink-0 snap-start">
              <ContinueWatchingCard item={m} onRemove={handleRemove} />
            </div>
          ))}
        </div>

        <button
          onClick={() => scroll(1)}
          aria-label="Next"
          className="absolute right-0 top-0 bottom-0 z-20 hidden w-12 items-center justify-center bg-gradient-to-l from-bg via-bg/80 to-transparent text-white opacity-0 transition-opacity duration-300 group-hover/car:opacity-100 md:flex hover:text-accent"
        >
          <ChevronRight />
        </button>
      </div>
    </section>
  );
}
