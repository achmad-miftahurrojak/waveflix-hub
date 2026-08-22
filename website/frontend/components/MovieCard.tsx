import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, itemYear, posterUrl, detailHref } from "@/lib/helpers";

interface Props {
  item: TmdbItem;
  rank?: number; // kalau diisi → tampilkan angka ranking di dalam card ala IDLIX
}

export default function MovieCard({ item, rank }: Props) {
  return (
    <Link
      href={detailHref(item)}
      className="group block w-full text-left focus:outline-none"
      aria-label={itemTitle(item)}
    >
      <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-surface shadow-lg transition-transform duration-300 ease-out group-hover:scale-[1.05] group-hover:shadow-card">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={posterUrl(item)}
          alt={itemTitle(item)}
          loading="lazy"
          className="h-full w-full object-cover"
        />

        {rank !== undefined && (
          <span
            className="absolute left-1.5 top-0 select-none font-sans text-4xl font-black leading-none text-white drop-shadow-[0_2px_6px_rgba(0,0,0,0.85)]"
            aria-hidden
          >
            {rank}
          </span>
        )}


        <span className="pointer-events-none absolute inset-0 ring-0 ring-accent/0 transition group-hover:ring-2 group-hover:ring-accent/60" />
      </div>

    </Link>
  );
}
