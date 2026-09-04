"use client";

import Link from "next/link";
import type { LatestEpisode } from "@/lib/tmdb";
import { stillUrl, itemTitle } from "@/lib/helpers";
import { useTranslation } from "@/lib/i18n";

const pad = (n: number) => String(n).padStart(2, "0");

export default function LatestEpisodesRow({
  episodes,
}: {
  episodes: LatestEpisode[];
}) {
  const { t } = useTranslation();
  if (episodes.length === 0) return null;
  return (
    <section className="mb-3">
      <h2 className="mb-1 px-[4%] text-xl font-bold">{t("ui.latestEpisodes")}</h2>
      <div className="no-scrollbar flex snap-x snap-mandatory scroll-pl-[4%] gap-3 overflow-x-auto scroll-smooth px-[4%] py-2">
        {episodes.map((e) => (
          <Link
            key={`${e.show.id}-${e.season}-${e.episode}`}
            href={`/tv/${e.show.id}/season/${e.season}/episode/${e.episode}`}
            className="group w-[280px] shrink-0 snap-start text-left"
          >
            <div className="relative aspect-video w-full overflow-hidden rounded-lg bg-surface shadow-lg">
              {e.still ? (

                <img
                  src={stillUrl(e.still)}
                  alt={e.name}
                  loading="lazy"
                  className="h-full w-full object-cover transition duration-300 group-hover:scale-105"
                />
              ) : null}
              <span className="absolute left-2 top-2 rounded bg-black/75 px-1.5 py-0.5 text-xs font-bold tracking-wide">
                S{pad(e.season)} E{pad(e.episode)}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}
