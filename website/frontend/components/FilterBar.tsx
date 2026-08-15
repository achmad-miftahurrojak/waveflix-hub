"use client";

import { useRouter, useSearchParams } from "next/navigation";
import type { MediaType } from "@/lib/types";
import { genresFor, COUNTRIES, PROVIDERS, SORTS, YEARS } from "@/lib/catalog";

export default function FilterBar() {
  const router = useRouter();
  const sp = useSearchParams();

  const media = (sp.get("media") as MediaType) === "tv" ? "tv" : "movie";
  const genre = sp.get("genre") ?? "";
  const year = sp.get("year") ?? "";
  const country = sp.get("country") ?? "";
  const provider = sp.get("provider") ?? "";
  const sort = sp.get("sort_by") ?? "popularity.desc";

  const update = (key: string, value: string) => {
    const params = new URLSearchParams(sp.toString());
    if (value) params.set(key, value);
    else params.delete(key);
    if (key === "media") params.delete("genre"); // genre beda antar media
    params.delete("page");
    router.push(`/browse?${params.toString()}`);
  };

  const selectCls =
    "rounded-md border border-white/15 bg-surface px-3 py-2 text-sm outline-none focus:border-accent";

  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="flex overflow-hidden rounded-md border border-white/15">
        {(["movie", "tv"] as const).map((m) => (
          <button
            key={m}
            onClick={() => update("media", m)}
            className={`px-4 py-2 text-sm font-medium transition ${
              media === m ? "bg-accent text-black" : "text-white/70 hover:bg-white/10"
            }`}
          >
            {m === "movie" ? "Movies" : "Series"}
          </button>
        ))}
      </div>

      <select value={genre} onChange={(e) => update("genre", e.target.value)} className={selectCls}>
        <option value="">All Genres</option>
        {genresFor(media).map((g) => (
          <option key={g.id} value={g.id}>
            {g.name}
          </option>
        ))}
      </select>

      <select value={year} onChange={(e) => update("year", e.target.value)} className={selectCls}>
        <option value="">All Years</option>
        {YEARS.map((y) => (
          <option key={y} value={y}>
            {y}
          </option>
        ))}
      </select>

      <select value={country} onChange={(e) => update("country", e.target.value)} className={selectCls}>
        <option value="">All Countries</option>
        {COUNTRIES.map((c) => (
          <option key={c.code} value={c.code}>
            {c.name}
          </option>
        ))}
      </select>

      <select value={provider} onChange={(e) => update("provider", e.target.value)} className={selectCls}>
        <option value="">All Platforms</option>
        {PROVIDERS.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </select>

      <select value={sort} onChange={(e) => update("sort_by", e.target.value)} className={selectCls}>
        {SORTS.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </select>
    </div>
  );
}
