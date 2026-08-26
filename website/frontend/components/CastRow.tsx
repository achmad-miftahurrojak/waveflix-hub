import Link from "next/link";
import type { CastMember } from "@/lib/types";
import { profileUrl } from "@/lib/helpers";

export default function CastRow({ cast }: { cast: CastMember[] }) {
  const list = cast.filter((c) => c.profile_path).slice(0, 12);
  if (list.length === 0) return null;

  return (
    <section className="mt-10">
      <h3 className="mb-4 text-xl font-bold">Cast</h3>
      <div className="no-scrollbar flex snap-x snap-mandatory gap-5 overflow-x-auto pb-2">
        {list.map((c) => (
          <Link href={`/person/${c.id}`} key={c.id} className="w-24 shrink-0 snap-start text-center group">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={profileUrl(c.profile_path)}
              alt={c.original_name || c.name}
              loading="lazy"
              className="mb-2 h-24 w-24 rounded-full object-cover shadow-lg transition-transform duration-300 ease-out group-hover:scale-[1.05] group-hover:shadow-card"
            />
            <div className="truncate text-sm font-semibold transition-colors duration-300 group-hover:text-white/80">{c.original_name || c.name}</div>
            {c.character && (
              <div className="truncate text-xs text-white/50">{c.character}</div>
            )}
          </Link>
        ))}
      </div>
    </section>
  );
}
