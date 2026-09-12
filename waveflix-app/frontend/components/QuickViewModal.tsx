"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import type { TmdbItem } from "@/lib/types";
import { itemTitle, itemYear, isTv, backdropUrl, IMG, BACKEND } from "@/lib/helpers";

interface TmdbLogo {
  file_path: string;
  iso_639_1: string | null;
}

function premiseOf(overview: string): string {
  const sentences = overview.match(/[^.!?]+[.!?]+["')\]]*\s*|[^.!?]+$/g) ?? [];
  let premise = sentences.slice(0, 2).join(" ").trim();
  if (premise.length > 220) {
    premise = premise.slice(0, 220).replace(/\s+\S*$/, "") + "…";
  }
  return premise;
}

export default function QuickViewModal({
  isOpen,
  onClose,
  item,
  isLanding,
}: {
  isOpen: boolean;
  onClose: () => void;
  item: TmdbItem | null;
  isLanding?: boolean;
}) {
  const [overview, setOverview] = useState<string | null>(null);
  const [tagline, setTagline] = useState<string | null>(null);
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [logoPath, setLogoPath] = useState<string | null>(null);
  const [meta, setMeta] = useState<{
    runtime?: number;
    seasons?: number;
    genres?: string[];
  } | null>(null);

  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  
  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isOpen, onClose]);

  
  useEffect(() => {
    setOverview(null);
    setTagline(null);
    setLogoPath(null);
    setMeta(null);
    if (!isOpen || !item) return;

    let cancelled = false;
    setLoadingOverview(true);
    const media = isTv(item) ? "tv" : "movie";

    const imgLang = item.original_language || "en";

    Promise.all([
      fetch(`${BACKEND}/api/detail?media=${media}&id=${item.id}&lang=en`).then((r) => (r.ok ? r.json() : null)),
      imgLang !== "en"
        ? fetch(`${BACKEND}/api/detail?media=${media}&id=${item.id}&lang=${imgLang}`).then((r) => (r.ok ? r.json() : null))
        : Promise.resolve(null),
    ])
      .then(([dEn, dOrig]) => {
        if (cancelled) return;
        const ov: string = dEn?.overview ?? "";
        setOverview(ov.trim());
        setTagline(typeof dEn?.tagline === "string" ? dEn.tagline.trim() : null);
        setMeta({
          runtime: typeof dEn?.runtime === "number" ? dEn.runtime : undefined,
          seasons: typeof dEn?.number_of_seasons === "number" ? dEn.number_of_seasons : undefined,
          genres: Array.isArray(dEn?.genres)
            ? dEn.genres.map((g: { name?: string }) => g?.name).filter(Boolean).slice(0, 2)
            : undefined,
        });

        const logosEn: TmdbLogo[] = dEn?.images?.logos ?? [];
        const logosOrig: TmdbLogo[] = dOrig?.images?.logos ?? [];
        const logos = [...logosEn, ...logosOrig];

        const best =
          logos.find((l) => l.iso_639_1 === "en") ??
          logos.find((l) => l.iso_639_1 === null) ??
          logos.find((l) => l.iso_639_1 === imgLang) ??
          logos[0];
        setLogoPath(best?.file_path ?? null);
      })
      .catch(() => {
        if (!cancelled) {
          setOverview("");
          setLogoPath(null);
        }
      })
      .finally(() => !cancelled && setLoadingOverview(false));

    return () => {
      cancelled = true;
    };
  }, [isOpen, item]);

  if (!isOpen || !item) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {}
      <div
        className="absolute inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
        onClick={onClose}
      />

      {}
      <div className="relative flex h-[min(520px,90dvh)] w-full max-w-[640px] flex-col overflow-hidden rounded-xl bg-surface-raised shadow-2xl">
        <button
          onClick={onClose}
          aria-label="Tutup"
          className="absolute top-4 right-4 z-20 p-2 bg-black/50 hover:bg-black/80 rounded-full text-white transition"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden>
             <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>

        <div className="flex-1 overflow-y-auto">
          {}
          <div className="relative h-[48%] min-h-56 w-full">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={backdropUrl(item)}
              alt={itemTitle(item)}
              className="w-full h-full object-cover"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-surface-raised via-black/20 to-transparent" />

            <div className="absolute bottom-4 left-5 right-5">
              {logoPath ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={`${IMG}/w500${logoPath}`}
                  alt={itemTitle(item)}
                  className="max-h-20 w-auto max-w-[70%] object-contain object-left drop-shadow-lg"
                />
              ) : (
                <h2
                  className="text-3xl font-bold uppercase tracking-wide text-white drop-shadow-lg"
                  style={{ fontFamily: "var(--font-logo)" }}
                >
                  {itemTitle(item)}
                </h2>
              )}
            </div>
          </div>

          {}
          <div className="px-6 pb-6 pt-4">
            <div className="mb-4 flex flex-wrap items-center gap-2 text-xs font-semibold text-gray-300">
              {itemYear(item) && (
                <span className="rounded bg-white/10 px-2 py-1">{itemYear(item)}</span>
              )}
              {isTv(item)
                ? meta?.seasons && (
                    <span className="rounded bg-white/10 px-2 py-1">
                      {meta.seasons} {meta.seasons > 1 ? "Seasons" : "Season"}
                    </span>
                  )
                : meta?.runtime && (
                    <span className="rounded bg-white/10 px-2 py-1">
                      {Math.floor(meta.runtime / 60) > 0
                        ? `${Math.floor(meta.runtime / 60)}h ${meta.runtime % 60}m`
                        : `${meta.runtime}m`}
                    </span>
                  )}
              <span className="rounded bg-white/10 px-2 py-1">{isTv(item) ? "Serial" : "Film"}</span>
              {meta?.genres?.map((g) => (
                <span key={g} className="rounded bg-white/10 px-2 py-1">
                  {g}
                </span>
              ))}
            </div>

            <p className="mb-5 text-sm leading-relaxed text-gray-200">
              {loadingOverview
                ? "Loading description…"
                : (overview ? premiseOf(overview) : "") || tagline || "No description available for this title."}
            </p>

            <Link
              href={isLanding ? "/daftar" : `/title/${isTv(item) ? "tv" : "movie"}/${item.id}`}
              className="inline-flex items-center gap-2 rounded bg-accent px-6 py-3 text-sm font-semibold text-black transition hover:bg-accent-dark"
            >
              Watch Now
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="h-4 w-4" aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
