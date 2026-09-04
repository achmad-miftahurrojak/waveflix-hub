"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";
import { fetchBrowsePage } from "@/app/browse/actions";

export default function InfinitePosterGrid({
  initialItems,
  initialHasMore,
  sp,
}: {
  initialItems: TmdbItem[];
  initialHasMore: boolean;
  sp: Record<string, string | undefined>;
}) {
  // Create a stable cache key based on URL search params
  const cacheKey = typeof window !== "undefined" 
    ? `waveflix:browse:${new URLSearchParams(sp as Record<string, string>).toString()}` 
    : "";

  const [items, setItems] = useState<TmdbItem[]>(() => {
    if (typeof window !== "undefined" && cacheKey) {
      const cached = sessionStorage.getItem(cacheKey);
      if (cached) {
        try {
          const parsed = JSON.parse(cached);
          if (parsed.items && parsed.items.length > 0) return parsed.items;
        } catch (e) {}
      }
    }
    return initialItems;
  });

  const [hasMore, setHasMore] = useState(() => {
    if (typeof window !== "undefined" && cacheKey) {
      const cached = sessionStorage.getItem(cacheKey);
      if (cached) {
        try {
          const parsed = JSON.parse(cached);
          if (parsed.hasMore !== undefined) return parsed.hasMore;
        } catch (e) {}
      }
    }
    return initialHasMore;
  });

  const [page, setPage] = useState(() => {
    if (typeof window !== "undefined" && cacheKey) {
      const cached = sessionStorage.getItem(cacheKey);
      if (cached) {
        try {
          const parsed = JSON.parse(cached);
          if (parsed.page) return parsed.page;
        } catch (e) {}
      }
    }
    return 1;
  });

  const [loading, setLoading] = useState(false);
  const observerTarget = useRef<HTMLDivElement>(null);

  // Save to sessionStorage whenever state changes
  useEffect(() => {
    if (cacheKey && items.length > 0) {
      sessionStorage.setItem(cacheKey, JSON.stringify({ items, hasMore, page }));
    }
  }, [items, hasMore, page, cacheKey]);

  useEffect(() => {
    if (page === 1 && items.length === initialItems.length) {
      setItems(initialItems);
      setHasMore(initialHasMore);
    }
  }, [initialItems, initialHasMore]);

  const loadMore = useCallback(async () => {
    if (loading || !hasMore) return;
    setLoading(true);
    try {
      const nextPage = page + 1;
      const res = await fetchBrowsePage(sp, nextPage);

      setItems((prev) => {

        const existingIds = new Set(prev.map(i => i.id));
        const newItems = res.results.filter(i => !existingIds.has(i.id));
        return [...prev, ...newItems];
      });
      setHasMore(res.hasMore);
      setPage(nextPage);
    } catch (err) {
      console.error("Failed to load more:", err);
      setHasMore(false); 
    } finally {
      setLoading(false);
    }
  }, [loading, hasMore, page, sp]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMore();
        }
      },
      { rootMargin: "400px" } 
    );

    if (observerTarget.current) {
      observer.observe(observerTarget.current);
    }

    return () => observer.disconnect();
  }, [loadMore]);

  if (!items || items.length === 0) {
    return (
      <p className="py-16 text-center text-white/50">
        Tidak ada konten yang tersedia.
      </p>
    );
  }

  return (
    <div>
      <div className="grid grid-cols-3 gap-x-3 gap-y-6 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7">
        {items.map((m, i) => (
          <MovieCard key={`${m.id}-${i}`} item={m} />
        ))}
      </div>

      {hasMore && (
        <div ref={observerTarget} className="flex justify-center py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent border-t-transparent"></div>
        </div>
      )}

      {!hasMore && items.length > 0 && (
        <p className="py-8 text-center text-sm text-white/40">
          Semua konten telah dimuat.
        </p>
      )}
    </div>
  );
}
