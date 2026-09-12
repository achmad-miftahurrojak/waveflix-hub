import Link from "next/link";
import { MouseEvent } from "react";
import type { HistoryItem } from "@/lib/types";
import { itemTitle, posterUrl, detailHref } from "@/lib/helpers";

interface Props {
  item: HistoryItem;
  onRemove?: (id: number) => void;
}

export default function ContinueWatchingCard({ item, onRemove }: Props) {
  const handleRemove = (e: MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (onRemove) {
      onRemove(item.id);
    }
  };

  const runtime = item.runtime || 120;
  const progress = item.progress || 0;
  const percent = Math.min(100, Math.max(0, (progress / runtime) * 100));

  return (
    <Link
      href={detailHref(item)}
      className="group block w-full text-left focus:outline-none relative"
      aria-label={itemTitle(item)}
    >
      <div className="relative aspect-[16/9] overflow-hidden rounded-lg bg-surface shadow-lg transition-transform duration-300 ease-out group-hover:scale-[1.05] group-hover:shadow-card">
        {}
        <img
          src={item.backdrop_path ? `https://image.tmdb.org/t/p/w500${item.backdrop_path}` : posterUrl(item)}
          alt={itemTitle(item)}
          loading="lazy"
          className="h-full w-full object-cover"
        />

        {}
        <div className="absolute bottom-0 left-0 right-0 h-1 bg-white/20 backdrop-blur-sm">
          <div 
            className="h-full bg-accent transition-all duration-300"
            style={{ width: `${percent}%` }}
          />
        </div>

        {}
        <div className="pointer-events-none absolute inset-0 bg-black/0 transition duration-300 group-hover:bg-black/20" />

        {}
        <button
          onClick={handleRemove}
          className="absolute right-2 top-2 z-10 flex h-6 w-6 items-center justify-center rounded-full bg-black/50 text-white opacity-0 backdrop-blur-sm transition duration-300 hover:bg-red-500 focus-visible:opacity-100 group-hover:opacity-100 group-focus-within:opacity-100"
          aria-label="Remove from history"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

    </Link>
  );
}
