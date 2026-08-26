"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { TmdbItem, MediaType } from "@/lib/types";
import PosterGrid from "./PosterGrid";
import { BACKEND } from "@/lib/helpers";

interface Props {
  params: Record<string, string>; // query discover tanpa `page`
  media: MediaType;
  initial: TmdbItem[];
  totalPages: number;
}

/** Grid dengan auto load-more saat di-scroll ke bawah (tanpa tombol). */
export default function InfiniteGrid({
  params,
  media,
  initial,
  totalPages,
}: Props) {
  const [items, setItems] = useState<TmdbItem[]>(initial);
  const [loading, setLoading] = useState(false);
  const sentinel = useRef<HTMLDivElement>(null);
  
  // Gunakan ref agar IntersectionObserver tidak ter-reset setiap kali page bertambah
  const pageRef = useRef(1);
  const loadingRef = useRef(false);

  const done = pageRef.current >= totalPages;

  const loadMore = useCallback(async () => {
    if (loadingRef.current || pageRef.current >= totalPages) return;
    
    loadingRef.current = true;
    setLoading(true);
    const next = pageRef.current + 1;
    
    try {
      const qs = new URLSearchParams({ ...params, page: String(next) });
      const r = await fetch(`${BACKEND}/api/discover?${qs}`);
      const d: { results?: TmdbItem[] } = await r.json();
      const fresh = (d.results ?? [])
        .filter((m) => m.poster_path)
        .map((m) => ({ ...m, media_type: media }));
      setItems((prev) => {
        const seen = new Set(prev.map((p) => p.id));
        return [...prev, ...fresh.filter((m) => !seen.has(m.id))];
      });
      pageRef.current = next;
    } catch {
      /* abaikan; coba lagi saat scroll berikutnya */
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, [params, media, totalPages]);

  useEffect(() => {
    const el = sentinel.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) loadMore();
      },
      { rootMargin: "400px" } // Lebih stabil, memuat sebelum ujung
    );
    io.observe(el);
    return () => io.disconnect();
  }, [loadMore]);

  return (
    <>
      <PosterGrid items={items} />
      {!done && <div ref={sentinel} className="h-8" />}
      {loading && (
        <p className="py-6 text-center text-sm text-white/50">Loading more…</p>
      )}
    </>
  );
}
