import Link from "next/link";
import type { LatestEpisode } from "@/lib/tmdb";
import { stillUrl, itemTitle } from "@/lib/helpers";

const pad = (n: number) => String(n).padStart(2, "0");

export default function LatestEpisodesRow({
  episodes,
}: {
  episodes: LatestEpisode[];
}) {
  if (episodes.length === 0) return null;
  return (
    <section className="mb-6">
      <h2 className="mb-1 px-[4%] text-xl font-bold">Latest Episodes</h2>
      <div className="no-scrollbar flex gap-3 overflow-x-auto scroll-smooth px-[4%] py-8">
        {episodes.map((e) => (
          <Link
            key={`${e.show.id}-${e.season}-${e.episode}`}
            href={`/tv/${e.show.id}/season/${e.season}/episode/${e.episode}`}
            className="group w-[280px] shrink-0 text-left"
          >
            <div className="relative aspect-video w-full overflow-hidden rounded-lg bg-surface shadow-lg">
              {e.still ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={stillUrl(e.still)}
                  alt={e.name}
                  loading="lazy"
                  className="h-full w-full object-cover transition duration-300 group-hover:scale-105"
                />
              ) : null}
              <span className="absolute left-2 top-2 rounded bg-black/75 px-1.5 py-0.5 text-[0.65rem] font-bold tracking-wide">
                S{pad(e.season)} E{pad(e.episode)}
              </span>
            </div>
            <h4 className="mt-2 truncate text-sm font-semibold">
              S{pad(e.season)}E{pad(e.episode)}: {e.name}
            </h4>
            <span className="truncate text-xs text-white/50">
              {itemTitle(e.show)}
            </span>
          </Link>
        ))}
      </div>
    </section>
  );
}
