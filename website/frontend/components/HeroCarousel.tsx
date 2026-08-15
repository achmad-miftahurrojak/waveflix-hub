"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import type { HeroSlide } from "@/lib/tmdb";
import {
  backdropUrl,
  itemTitle,
  itemYear,
  ratingText,
  detailHref,
  isTv,
} from "@/lib/helpers";
import { StarIcon, PlayIcon } from "./Icons";

const AUTOPLAY_MS = 10000;

/** Hero carousel ala IDLIX: beberapa judul trending, titik navigasi, auto-geser. */
export default function HeroCarousel({ slides }: { slides: HeroSlide[] }) {
  const [active, setActive] = useState(0);
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  const total = slides.length;

  const start = () => {
    if (timer.current) clearInterval(timer.current);
    if (total <= 1) return;
    timer.current = setInterval(
      () => setActive((i) => (i + 1) % total),
      AUTOPLAY_MS
    );
  };

  useEffect(() => {
    start();
    return () => {
      if (timer.current) clearInterval(timer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [total]);

  const go = (i: number) => {
    setActive(i);
    start();
  };

  if (total === 0) return null;

  return (
    <section className="relative h-[86vh] min-h-[560px] w-full overflow-hidden">
      {/* Lapisan background per-slide (fade) */}
      {slides.map((s, i) => (
        <div
          key={s.item.id}
          className={`absolute inset-0 transition-opacity duration-700 ${
            i === active ? "opacity-100" : "opacity-0"
          }`}
          style={{
            backgroundImage: `url('${backdropUrl(s.item)}')`,
            backgroundSize: "cover",
            backgroundPosition: "center top",
          }}
        />
      ))}

      {/* Redup seragam tipis supaya teks lebih jelas (bukan vignette) */}
      <div className="pointer-events-none absolute inset-0 bg-black/35" />

      {/* Fade bawah saja supaya menyatu mulus dengan baris konten di bawahnya */}
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-bg to-transparent" />

      {/* Konten slide aktif — dianchor ke bawah ala IDLIX */}
      <div className="relative z-[2] flex h-full items-end">
        <div className="w-full max-w-2xl px-[4%] pb-[8vh]">
          {slides.map((s, i) => {
            const tv = isTv(s.item);
            const desc =
              s.overview.length > 220
                ? s.overview.slice(0, 220) + "…"
                : s.overview;
            return (
              <div
                key={s.item.id}
                className={
                  i === active ? "block animate-[fadeIn_.5s_ease]" : "hidden"
                }
              >
                <span className="mb-4 inline-block rounded bg-accent px-2.5 py-1 text-xs font-bold uppercase tracking-wider text-black">
                  {tv ? "TV Series" : "Movie"}
                </span>

                {s.tagline && (
                  <p className="mb-2 text-sm italic text-white/70 drop-shadow md:text-base">
                    {s.tagline}
                  </p>
                )}

                {s.logo ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={s.logo}
                    alt={itemTitle(s.item)}
                    className="mb-4 max-h-16 w-auto max-w-[220px] object-contain object-left drop-shadow-[0_2px_10px_rgba(0,0,0,0.8)] md:max-h-24 md:max-w-[280px]"
                  />
                ) : (
                  <h1 className="mb-4 max-w-xl text-3xl font-extrabold leading-tight drop-shadow-lg md:text-4xl">
                    {itemTitle(s.item)}
                  </h1>
                )}

                <div className="mb-4 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-white/90 drop-shadow-[0_1px_5px_rgba(0,0,0,0.9)]">
                  <span className="flex items-center gap-1 font-semibold text-[#f5c518]">
                    <StarIcon /> {ratingText(s.item)}
                  </span>
                  <span className="text-white/40">&bull;</span>
                  <span>{itemYear(s.item)}</span>
                  {s.duration && (
                    <>
                      <span className="text-white/40">&bull;</span>
                      <span>{s.duration}</span>
                    </>
                  )}
                  {s.genres.length > 0 && (
                    <>
                      <span className="text-white/40">&bull;</span>
                      <span>{s.genres.join(", ")}</span>
                    </>
                  )}
                  {s.status && (
                    <span className="rounded border border-emerald-500/70 px-2 py-0.5 text-xs font-bold text-emerald-400">
                      {s.status}
                    </span>
                  )}
                </div>

                {desc && (
                  <p className="mb-6 max-w-lg text-xs leading-relaxed text-white/75 drop-shadow md:text-sm">
                    {desc}
                  </p>
                )}

                <Link
                  href={detailHref(s.item)}
                  className="inline-flex items-center gap-2 rounded-md bg-accent px-7 py-3 text-base font-semibold text-black transition hover:scale-105 hover:bg-accent-dark"
                >
                  <PlayIcon className="text-black" />
                  Watch Now
                </Link>
              </div>
            );
          })}

          {/* Titik navigasi ala IDLIX: bulat kecil, aktif memanjang */}
          {total > 1 && (
            <div className="mt-8 flex items-center gap-2">
              {slides.map((s, i) => (
                <button
                  key={s.item.id}
                  onClick={() => go(i)}
                  aria-label={`Slide ${i + 1}`}
                  className={`h-2 rounded-full transition-all ${
                    i === active
                      ? "w-7 bg-accent"
                      : "w-2 bg-white/35 hover:bg-white/60"
                  }`}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
