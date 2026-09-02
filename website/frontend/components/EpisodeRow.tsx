"use client";

import Link from "next/link";
import type { LatestEpisode } from "@/lib/tmdb";
import { posterUrl, itemTitle } from "@/lib/helpers";
import { useTranslation } from "@/lib/i18n";

const pad = (n: number) => String(n).padStart(2, "0");

export default function EpisodeRow({
  episodes,
}: {
  episodes: LatestEpisode[];
}) {
  const { t } = useTranslation();
  if (episodes.length === 0) return null;
  return (
    <section className="mb-6">
      <div className="mb-3 px-[4%]">
        <h2 className="text-xl font-bold text-gray-100">Recently Added Episodes</h2>
      </div>
      <div className="no-scrollbar flex snap-x snap-mandatory scroll-pl-[4%] gap-3 overflow-x-auto scroll-smooth px-[4%] py-2">
        {episodes.map((e) => (
          <Link
            key={`${e.show.id}-${e.season}-${e.episode}`}
            href={`/tv/${e.show.id}/season/${e.season}/episode/${e.episode}`}
            className="group w-[185px] shrink-0 snap-start text-left block"
            aria-label={`${itemTitle(e.show)} - Season ${e.season} Episode ${e.episode}`}
          >
            <div className="relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-surface shadow-md">
              <img
                src={posterUrl(e.show)}
                alt={e.name}
                loading="lazy"
                className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-110"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/10 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
              <span className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-lg" />
              
              <span className="absolute left-2 top-2 rounded bg-black/80 px-2 py-1 text-xs font-bold tracking-wide text-white shadow-sm">
                EPS {e.episode}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}