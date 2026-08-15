import type { CastMember } from "@/lib/types";
import { profileUrl } from "@/lib/helpers";

export default function CastRow({ cast }: { cast: CastMember[] }) {
  const list = cast.filter((c) => c.profile_path).slice(0, 12);
  if (list.length === 0) return null;

  return (
    <section className="mt-10">
      <h3 className="mb-4 text-xl font-bold">Cast</h3>
      <div className="no-scrollbar flex gap-5 overflow-x-auto pb-2">
        {list.map((c) => (
          <div key={c.id} className="w-24 shrink-0 text-center">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={profileUrl(c.profile_path)}
              alt={c.name}
              loading="lazy"
              className="mb-2 h-24 w-24 rounded-full object-cover"
            />
            <div className="truncate text-sm font-semibold">{c.name}</div>
            {c.character && (
              <div className="truncate text-xs text-white/50">{c.character}</div>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}
