"use client";

import { useMemo, useState } from "react";
import type { TmdbItem } from "@/lib/types";
import { mediaTypeOf } from "@/lib/helpers";
import Carousel from "./Carousel";

type Tab = "all" | "movie" | "tv";

const TABS: { id: Tab; label: string }[] = [
  { id: "all", label: "All" },
  { id: "movie", label: "Movies" },
  { id: "tv", label: "TV Series" },
];

/** Baris trending dengan toggle All/Movies/TV + penomoran ranking 1-N. */
export default function TrendingSection({
  items,
  title = "Trending Now",
}: {
  items: TmdbItem[];
  title?: string;
}) {
  const [tab, setTab] = useState<Tab>("all");

  const filtered = useMemo(
    () => (tab === "all" ? items : items.filter((m) => mediaTypeOf(m) === tab)),
    [items, tab]
  );

  return (
    <section className="mb-6">
      <div className="mb-1 flex items-center justify-between px-[4%]">
        <h2 className="text-xl font-bold">{title}</h2>
        <div className="flex rounded-full bg-white/5 p-1">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-full px-4 py-1 text-sm font-medium transition ${
                tab === t.id
                  ? "bg-accent text-black"
                  : "text-white/60 hover:text-white"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
      <Carousel items={filtered} ranked />
    </section>
  );
}
