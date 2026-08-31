import Link from "next/link";
import { ChevronRight } from "./Icons";

interface Tile {
  label: string;
  href: string;
}

export default function CategoryTiles({
  title,
  tiles,
  compact,
}: {
  title: string;
  tiles: Tile[];
  compact?: boolean;
}) {
  return (
    <main className="min-h-screen px-[4%] pb-16 pt-28">
      <h1 className="mb-8 text-3xl font-bold">{title}</h1>
      <div
        className={`grid gap-3 ${
          compact
            ? "grid-cols-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8"
            : "grid-cols-2 sm:grid-cols-3 lg:grid-cols-5"
        }`}
      >
        {tiles.map((t) => (
          <Link
            key={t.href}
            href={t.href}
            className="group flex items-center justify-between rounded-xl bg-white/[0.04] px-5 py-6 ring-1 ring-white/10 transition hover:bg-white/[0.08] hover:ring-accent/40"
          >
            <span className="font-semibold text-white/90 group-hover:text-white">
              {t.label}
            </span>
            <ChevronRight className="text-white/30 transition group-hover:text-accent" />
          </Link>
        ))}
      </div>
    </main>
  );
}
