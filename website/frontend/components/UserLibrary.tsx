"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { TmdbItem } from "@/lib/types";
import { useAuth } from "./AuthProvider";
import PosterGrid from "./PosterGrid";

function toItem(r: any): TmdbItem {
  return {
    id: r.tmdb_id ?? r.id,
    title: r.title,
    name: r.title,
    media_type: r.media_type,
    poster_path: r.poster_path,
    vote_average: r.vote_average,
    first_air_date: r.media_type === "tv" ? " " : undefined,
  };
}

interface Props {
  title: string;
  endpoint: "/api/watchlist" | "/api/history";
  /** Untuk watchlist: tampilkan localStorage saat belum login. */
  localFallbackKey?: string;
}

export default function UserLibrary({ title, endpoint, localFallbackKey }: Props) {
  const { user, ready, authFetch } = useAuth();
  const [items, setItems] = useState<TmdbItem[] | null>(null);

  useEffect(() => {
    if (!ready) return;
    if (user) {
      authFetch(endpoint)
        .then((r) => r.json())
        .then((d) => setItems((d.results ?? []).map(toItem)))
        .catch(() => setItems([]));
    } else if (localFallbackKey) {
      try {
        const local = JSON.parse(localStorage.getItem(localFallbackKey) || "[]");
        setItems([...local].reverse());
      } catch {
        setItems([]);
      }
    } else {
      setItems([]);
    }
  }, [user, ready, endpoint, localFallbackKey, authFetch]);

  return (
    <main className="min-h-screen px-[4%] pb-16 pt-28">
      <h1 className="mb-6 text-2xl font-bold">{title}</h1>

      {!ready || items === null ? (
        <p className="py-16 text-center text-white/50">Memuat…</p>
      ) : !user && !localFallbackKey ? (
        <p className="py-16 text-center text-white/50">
          <Link href="/masuk" className="text-accent hover:underline">
            Masuk
          </Link>{" "}
          untuk melihat {title.toLowerCase()}.
        </p>
      ) : items.length === 0 ? (
        <p className="py-16 text-center text-white/50">Belum ada apa-apa di sini.</p>
      ) : (
        <PosterGrid items={items} />
      )}
    </main>
  );
}
