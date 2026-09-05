"use client";

import { useEffect, useState } from "react";
import type { TmdbItem, MediaType } from "@/lib/types";
import Carousel from "./Carousel";
import { ChevronRight } from "./Icons";
import { useTranslation } from "@/lib/i18n";
import { BACKEND } from "@/lib/helpers";

interface Source {
  media: MediaType;
  params: string;
}

interface PlatformOption {
  label: string;
  sources: Source[];
}

const PLATFORMS: (PlatformOption & { logo?: string })[] = [
  { label: "Netflix", logo: "/networks/netflix.png", sources: [{ media: "movie", params: "provider=8&sort_by=popularity.desc" }, { media: "tv", params: "provider=8&sort_by=popularity.desc" }] },
  { label: "Disney+ Originals", logo: "/networks/disney.png", sources: [{ media: "movie", params: "provider=122&sort_by=popularity.desc" }, { media: "tv", params: "provider=122&sort_by=popularity.desc" }] },
  { label: "HBO MAX", logo: "/networks/hbo.png", sources: [{ media: "movie", params: "provider=384&sort_by=popularity.desc" }, { media: "tv", params: "provider=384&sort_by=popularity.desc" }] },
  { label: "Prime Video", logo: "/networks/prime.png", sources: [{ media: "movie", params: "provider=119&sort_by=popularity.desc" }, { media: "tv", params: "provider=119&sort_by=popularity.desc" }] },
  { label: "Apple TV", logo: "/networks/apple.png", sources: [{ media: "movie", params: "provider=350&sort_by=popularity.desc" }, { media: "tv", params: "provider=350&sort_by=popularity.desc" }] },
];

export default function PlatformRow() {
  const { t } = useTranslation();
  const [active, setActive] = useState(0);
  const [open, setOpen] = useState(false);
  const [cache, setCache] = useState<Record<number, TmdbItem[]>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (cache[active]) return;
    let alive = true;
    setLoading(true);
    Promise.all(
      PLATFORMS[active].sources.map((s) =>
        fetch(`${BACKEND}/api/discover?media=${s.media}&${s.params}&lang=en`)
          .then((r) => r.json())
          .then((d: { results?: TmdbItem[] }) =>
            (d.results ?? [])
              .filter((m) => m.poster_path)
              .map((m) => ({ ...m, media_type: s.media }))
          )
          .catch(() => [] as TmdbItem[])
      )
    ).then((lists) => {
      if (!alive) return;
      const merged = lists
        .flat()
        .sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0))
        .slice(0, 18);
      setCache((c) => ({ ...c, [active]: merged }));
      setLoading(false);
    });
    return () => { alive = false; };
  }, [active]);

  const items = cache[active] ?? [];

  return (
    <section className="mb-8">
      <div className="mb-3 px-[4%]">
        <div className="relative inline-block" onMouseLeave={() => setOpen(false)}>
          <button
            onClick={() => setOpen((o) => !o)}
            onMouseEnter={() => setOpen(true)}
            className="flex items-center gap-2 text-xl font-bold"
          >
            {PLATFORMS[active].logo ? (
              <img src={PLATFORMS[active].logo} alt={t(PLATFORMS[active].label)} className="h-16 object-contain" />
            ) : (
              t(PLATFORMS[active].label)
            )}
            <ChevronRight className={`h-5 w-5 transition ${open ? "rotate-90" : ""}`} />
          </button>
          {open && (
            <div className="absolute left-0 top-full z-30 pt-4">
              <div className="w-56 rounded-xl border border-white/20 bg-white/5 p-2 backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)]">
                {PLATFORMS.map((g, i) => (
                  <button
                    key={g.label}
                    onClick={() => { setActive(i); setOpen(false); }}
                    className={`block w-full rounded-md px-3 py-2 text-left text-sm transition ${
                      i === active
                        ? "bg-white/10 text-accent"
                        : "text-white/75 hover:bg-white/10 hover:text-white"
                    }`}
                  >
                    {g.logo ? (
                      <img src={g.logo} alt={t(g.label)} className="h-12 object-contain inline-block" />
                    ) : (
                      t(g.label)
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {loading && items.length === 0 ? (
        <div className="px-[4%] py-10 text-white/50">Loading…</div>
      ) : (
        <Carousel items={items} />
      )}
    </section>
  );
}