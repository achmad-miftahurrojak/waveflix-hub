"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import type { CollectionCardItem } from "./CollectionCards";
import { CollectionCards } from "./CollectionCards";
import { fetchMoreCollections } from "@/app/collections/actions";

export default function InfiniteCollectionGrid({
  initialItems,
  initialHasMore,
  sp,
}: {
  initialItems: CollectionCardItem[];
  initialHasMore: boolean;
  sp: Record<string, string | undefined>;
}) {
  // Create a stable cache key based on URL search params
  const cacheKey = typeof window !== "undefined" 
    ? `waveflix:collections:${new URLSearchParams(sp as Record<string, string>).toString()}` 
    : "";

  const [items, setItems] = useState<CollectionCardItem[]>(() => {
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
    // Only reset if we are not restoring from cache or if initialItems changed significantly 
    // (though initialItems changing means the URL params probably changed, so cacheKey changed and state initialized correctly)
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
      const res = await fetchMoreCollections(sp, nextPage);

      setItems((prev) => {
        const existingIds = new Set(prev.map((i) => i.id));
        const newItems = res.results.filter((i) => !existingIds.has(i.id));
        return [...prev, ...newItems];
      });
      setHasMore(res.hasMore);
      setPage(nextPage);
    } catch (err) {
      console.error("Failed to load more collections:", err);
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
      { rootMargin: "600px" } // Load well before reaching the bottom
    );

    if (observerTarget.current) {
      observer.observe(observerTarget.current);
    }

    return () => observer.disconnect();
  }, [loadMore]);

  return (
    <>
      <CollectionCards items={items} />
      
      {hasMore && (
        <div ref={observerTarget} className="flex justify-center py-10 mt-8">
          <div className="flex space-x-2">
            <div className="w-3 h-3 bg-white/20 rounded-full animate-bounce" style={{ animationDelay: "0ms" }}></div>
            <div className="w-3 h-3 bg-white/40 rounded-full animate-bounce" style={{ animationDelay: "150ms" }}></div>
            <div className="w-3 h-3 bg-white/60 rounded-full animate-bounce" style={{ animationDelay: "300ms" }}></div>
          </div>
        </div>
      )}
      
      {!hasMore && items.length > 0 && (
        <div className="text-center py-10 text-white/30 text-sm mt-8">
          Kamu sudah mencapai akhir katalog koleksi.
        </div>
      )}
    </>
  );
}
