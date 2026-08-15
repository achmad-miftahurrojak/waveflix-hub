"use client";

import { useMemo, useState } from "react";
import { motion } from "framer-motion";
import type { TmdbItem } from "@/lib/types";
import { mediaTypeOf } from "@/lib/helpers";
import Carousel from "./Carousel";

const pillSpring = { type: "spring", stiffness: 380, damping: 30 } as const;

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
              className={`relative rounded-full px-4 py-1 text-sm font-medium transition ${
                tab === t.id ? "text-black" : "text-white/60 hover:text-white"
              }`}
            >
              {tab === t.id && (
                <motion.span
                  layoutId="trendingPill"
                  transition={pillSpring}
                  className="absolute inset-0 rounded-full bg-accent"
                />
              )}
              <span className="relative z-10">{t.label}</span>
            </button>
          ))}
        </div>
      </div>
      <Carousel items={filtered} ranked />
    </section>
  );
}
