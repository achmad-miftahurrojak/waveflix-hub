import { posterUrl } from "@/lib/helpers";
import type { TmdbItem } from "@/lib/types";

function Tile({ item }: { item: TmdbItem }) {
  return (
    <div className="relative h-20 w-[54px] shrink-0 overflow-hidden rounded bg-white/5 shadow-md shadow-black/50 md:h-24 md:w-16">
      {}
      <img
        src={posterUrl(item)}
        alt=""
        aria-hidden
        className="h-full w-full object-cover"
        loading="lazy"
      />
    </div>
  );
}

export default function PosterWall({ items }: { items: TmdbItem[] }) {
  if (items.length === 0) return null;

  const rows = [
    items,
    [...items].reverse(),
    [...items.slice(6), ...items.slice(0, 6)],
    [...items.slice(3), ...items.slice(0, 3)].reverse(),
    items,
    [...items].reverse(),
  ];

  return (
    <>
      {}
      <div className="absolute inset-0 z-[1] overflow-hidden" aria-hidden>
        <div
          className="absolute -left-96 -right-24 -top-40 bottom-0"
          style={{
            transform:
              "perspective(1200px) rotateX(10deg) rotateY(-4deg) scale(1.25)",
            transformOrigin: "center top",
          }}
        >
          {rows.map((row, i) => (
            <div
              key={i}
              className={`mb-2 flex gap-2 md:mb-3 md:gap-3 ${i % 2 === 0 ? "-ml-10" : ""}`}
            >
              {[...row, ...row].map((item, j) => (
                <Tile key={`${i}-${j}`} item={item} />
              ))}
            </div>
          ))}
        </div>
      </div>

      {}
      <div
        className="pointer-events-none absolute inset-0 z-[2] bg-gradient-to-b from-black/60 via-black/25 to-black"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 z-[2] bg-[radial-gradient(ellipse_at_center,rgba(0,0,0,0.3)_0%,rgba(0,0,0,0.55)_60%,rgba(0,0,0,0.85)_100%)]"
        aria-hidden
      />
    </>
  );
}
