import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, posterUrl, detailHref } from "@/lib/helpers";

interface Props {
  item: TmdbItem;
}

export default function MovieCard({ item }: Props) {
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
        <span className="pointer-events-none absolute inset-0 ring-0 ring-white/0 transition duration-300 group-hover:ring-1 group-hover:ring-white/20" />
      </div>
    </Link>
  );
}