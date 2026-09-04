"use client";

import Link from "next/link";
import type { LatestEpisode } from "@/lib/tmdb";
import { stillUrl, itemTitle } from "@/lib/helpers";
import { useTranslation } from "@/lib/i18n";

const pad = (n: number) => String(n).padStart(2, "0");
const getYear = (dateStr?: string | null) => dateStr ? dateStr.substring(0, 4) : "";

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
        {episodes.map((e) => {
          const showName = itemTitle(e.show);
          const year = getYear(e.show.first_air_date || (e.show as any).release_date);
          const epStr = `S${pad(e.season)} E${pad(e.episode)}`;
          const titleLine = `${epStr}: ${e.name}`;
          const subLine = year ? `${showName} (${year})` : showName;
          
          return (
            <Link
              key={`${e.show.id}-${e.season}-${e.episode}`}
              href={`/tv/${e.show.id}/season/${e.season}/episode/${e.episode}`}
              className="group w-[280px] shrink-0 snap-start text-left flex flex-col gap-2"
              aria-label={`${showName} - ${epStr}`}
            >
              <div className="relative aspect-video w-full overflow-hidden rounded-lg bg-surface shadow-md">
                {e.still ? (
                  <img
                    src={stillUrl(e.still)}
                    alt={e.name}
                    loading="lazy"
                    className="absolute inset-0 h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                  />
                ) : (
                  <div className="absolute inset-0 flex items-center justify-center bg-gray-800 text-gray-400">
                    No Image
                  </div>
                )}
                
                <div className="absolute inset-0 bg-black/10 transition-colors duration-300 group-hover:bg-transparent" />
                <span className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition-all duration-300 group-hover:ring-1 group-hover:ring-white/30 rounded-lg" />
                
                {/* Top-left Badge */}
                <span className="absolute left-2 top-2 rounded bg-black/80 px-2 py-0.5 text-xs font-bold tracking-wider text-gray-200 shadow-sm border border-white/10">
                  {epStr}
                </span>

                {/* Bottom-right Comment Icon Mock */}
                <div className="absolute bottom-2 right-2 flex items-center gap-1 rounded bg-black/80 px-1.5 py-0.5 text-[10px] font-medium text-gray-300 shadow-sm border border-white/10">
                  <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                  0
                </div>
              </div>
              
              {/* Text Below */}
              <div className="px-0.5">
                <h3 className="line-clamp-1 text-sm font-bold text-gray-100 transition-colors group-hover:text-white" title={titleLine}>
                  {titleLine}
                </h3>
                <p className="line-clamp-1 mt-0.5 text-xs font-medium text-gray-400">
                  {subLine}
                </p>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}