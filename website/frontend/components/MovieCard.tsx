import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, itemYear, posterUrl, detailHref } from "@/lib/helpers";

interface Props {
  item: TmdbItem;
  rank?: number; 
}

export default function MovieCard({ item, rank }: Props) {
  const isTV = item.media_type === "tv" || !item.release_date;
  const rating = item.vote_average ? item.vote_average.toFixed(1) : "N/A";
  const commentCount = item.vote_count || 0;

  return (
    <Link
      href={detailHref(item)}
      className="group block w-full text-left"
      aria-label={itemTitle(item)}
    >
      <div className="relative aspect-[2/3] w-full overflow-hidden rounded-lg bg-surface transition-transform duration-300 ease-out group-hover:scale-[1.03] group-hover:shadow-xl">
        <img
          src={posterUrl(item)}
          alt={itemTitle(item)}
          loading="lazy"
          className="h-full w-full object-cover"
        />

        {/* Top Badges */}
        <div className="absolute top-2 left-2 flex flex-col gap-1">
          {rank !== undefined ? (
            <span className="select-none font-sans text-4xl font-extrabold leading-none text-white drop-shadow-[0_2px_6px_rgba(0,0,0,0.85)]">
              {rank}
            </span>
          ) : (
            <span className="rounded bg-[#2a62c6] px-1.5 py-0.5 text-[10px] font-bold text-white shadow">
              {isTV ? "TV" : "MOVIE"}
            </span>
          )}
        </div>
        <div className="absolute top-2 right-2 flex flex-col gap-1 text-right">
          <span className="rounded bg-[#c82222] px-1.5 py-0.5 text-[10px] font-bold text-white shadow">
            HD
          </span>
        </div>

        {/* Bottom Badges */}
        <div className="absolute bottom-2 left-2">
          <span className="flex items-center gap-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] font-bold text-white backdrop-blur-sm">
            <span className="text-yellow-400">⭐</span> {rating}
          </span>
        </div>
        <div className="absolute bottom-2 right-2">
          <span className="flex items-center gap-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] font-bold text-white backdrop-blur-sm">
            <span className="text-gray-300">💬</span> {commentCount}
          </span>
        </div>

        <span className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition duration-300 group-hover:ring-1 group-hover:ring-white/20" />
      </div>

      {/* Title & Year (Below Poster) */}
      <div className="mt-2 flex flex-col px-1">
        <h3 className="line-clamp-2 text-sm font-semibold leading-tight text-gray-100 transition-colors group-hover:text-accent">
          {itemTitle(item)}
        </h3>
        <p className="mt-0.5 text-xs font-medium text-gray-400">
          {itemYear(item)}
        </p>
      </div>
    </Link>
  );
}
