import type { TmdbItem } from "@/lib/types";
import MovieCard from "./MovieCard";

export default function PosterGrid({ items }: { items: TmdbItem[] }) {
  if (items.length === 0) {
    return (
      <p className="py-16 text-center text-white/50">
        No results. Try adjusting the filters or keywords.
      </p>
    );
  }
  return (
    <div className="grid grid-cols-3 gap-x-3 gap-y-6 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7">
      {items.map((m, i) => (
        <MovieCard key={`${m.id}-${i}`} item={m} />
      ))}
    </div>
  );
}
