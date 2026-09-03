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
  const [items, setItems] = useState<CollectionCardItem[]>(initialItems);
  const [hasMore, setHasMore] = useState(initialHasMore);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setItems(initialItems);
    setHasMore(initialHasMore);
    setPage(1);
  }, [initialItems, initialHasMore, sp]);

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
