"use client";

import { useMemo, useState } from "react";
import { motion } from "framer-motion";
import type { TmdbItem } from "@/lib/types";
import { mediaTypeOf } from "@/lib/helpers";
import Carousel from "./Carousel";
import { useTranslation } from "@/lib/i18n";

const pillSpring = { type: "spring", stiffness: 380, damping: 30 } as const;

type Tab = "all" | "movie" | "tv";

const TABS: { id: Tab; labelKey: string }[] = [
  { id: "all", labelKey: "ui.all" },
  { id: "movie", labelKey: "nav.movies" },
  { id: "tv", labelKey: "nav.tv" },
];

/** Baris trending dengan toggle All/Movies/TV + penomoran ranking 1-N. */
export default function TrendingSection({
  items,
  title,
}: {
  items: TmdbItem[];
  title?: string;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>("all");
  const displayTitle = title ? t(title) : t("ui.trendingNow");

  const filtered = useMemo(
    () => (tab === "all" ? items : items.filter((m) => mediaTypeOf(m) === tab)),
    [items, tab]
  );

  return (
    <section className="mb-3">
      <div className="mb-1 flex items-center justify-between px-[4%]">
        <h2 className="text-xl font-bold">{displayTitle}</h2>
        <div className="flex rounded-full bg-white/5 p-1">
          {TABS.map((tabItem) => (
            <button
              key={tabItem.id}
              onClick={() => setTab(tabItem.id)}
              className={`relative rounded-full px-4 py-1 text-sm font-medium transition ${
                tab === tabItem.id ? "text-black" : "text-white/60 hover:text-white"
              }`}
            >
              {tab === tabItem.id && (
                <motion.span
                  layoutId={`trendingPill-${displayTitle}`}
                  transition={pillSpring}
                  className="absolute inset-0 rounded-full bg-accent"
                />
              )}
              <span className="relative z-10">{t(tabItem.labelKey)}</span>
            </button>
          ))}
        </div>
      </div>
      <Carousel items={filtered} ranked />
    </section>
  );
}
