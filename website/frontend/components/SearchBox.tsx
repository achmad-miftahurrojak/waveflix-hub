"use client";

import { useRouter } from "next/navigation";
import { useRef, useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { SearchIcon } from "./Icons";
import { posterUrl, itemTitle, detailHref, isTv } from "@/lib/helpers";
import type { TmdbItem } from "@/lib/types";

const BACKEND = process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";

export default function SearchBox() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const [results, setResults] = useState<TmdbItem[]>([]);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  // Debounced search — 300ms delay setelah user berhenti mengetik
  const search = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      const res = await fetch(
        `${BACKEND}/api/search?q=${encodeURIComponent(q)}`
      );
      const data = await res.json();
      const items = (data.results ?? [])
        .filter(
          (m: TmdbItem) =>
            (m.media_type === "movie" || m.media_type === "tv") && m.poster_path
        )
        .slice(0, 8); // Max 8 suggestions
      setResults(items);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => search(value), 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [value, search]);

  // Close dropdown saat klik di luar
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (value.trim()) {
      router.push(`/search?q=${encodeURIComponent(value.trim())}`);
      setOpen(false);
    }
  };

  return (
    <div ref={wrapRef} className="relative">
      <form onSubmit={submit} className="flex items-center">
        <button
          type="button"
          aria-label="Cari"
          onClick={() => {
            setOpen((o) => !o);
            setTimeout(() => inputRef.current?.focus(), 50);
          }}
          className="text-white/80 transition hover:text-accent"
        >
          <SearchIcon />
        </button>
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => {
            setValue(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          placeholder="Cari film atau series…"
          className={`ml-2 rounded-full bg-white/10 text-sm text-white outline-none transition-all duration-300 focus:ring-1 focus:ring-accent ${
            open ? "w-44 px-4 py-1.5 md:w-56" : "w-0 px-0 py-0"
          }`}
        />
      </form>

      {/* Instant results dropdown */}
      {open && value.trim() && (
        <div className="absolute right-0 top-full mt-2 w-80 rounded-xl bg-[#0d0f14] p-2 shadow-xl ring-1 ring-white/10 z-50">
          {loading ? (
            <p className="px-3 py-4 text-sm text-white/50">Mencari…</p>
          ) : results.length === 0 ? (
            <p className="px-3 py-4 text-sm text-white/50">Tidak ada hasil</p>
          ) : (
            results.map((item) => (
              <Link
                key={item.id}
                href={detailHref(item)}
                onClick={() => setOpen(false)}
                className="flex items-center gap-3 rounded-lg px-3 py-2 transition hover:bg-white/10"
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={posterUrl(item)}
                  alt=""
                  className="h-14 w-10 rounded object-cover"
                  loading="lazy"
                />
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">
                    {itemTitle(item)}
                  </p>
                  <p className="text-xs text-white/50">
                    {isTv(item) ? "Series" : "Movie"}
                  </p>
                </div>
              </Link>
            ))
          )}
        </div>
      )}
    </div>
  );
}
