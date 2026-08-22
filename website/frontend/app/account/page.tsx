"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import Link from "next/link";
import { useRouter } from "next/navigation";

const pillSpring = { type: "spring", stiffness: 380, damping: 30 } as const;
import type { TmdbItem } from "@/lib/types";
import { useAuth } from "@/components/AuthProvider";
import PosterGrid from "@/components/PosterGrid";
import { nameFontCss, profileFontVars } from "@/lib/profileFonts";

type Tab = "overview" | "favorites" | "mylist" | "history";

function toItem(r: any): TmdbItem {
  return {
    id: r.tmdb_id ?? r.id,
    title: r.title,
    name: r.title,
    media_type: r.media_type,
    poster_path: r.poster_path,
    vote_average: r.vote_average,
    first_air_date: r.media_type === "tv" ? " " : undefined,
  };
}

export default function AccountPage() {
  const { user, ready, authFetch, activeProfile } = useAuth();
  const router = useRouter();

  const [tab, setTab] = useState<Tab>("overview");
  const [fav, setFav] = useState<TmdbItem[]>([]);
  const [list, setList] = useState<TmdbItem[]>([]);
  const [hist, setHist] = useState<TmdbItem[]>([]);

  useEffect(() => {
    if (ready && !user) router.replace("/masuk");
  }, [ready, user, router]);

  useEffect(() => {
    if (!user) return;
    const load = (path: string, set: (x: TmdbItem[]) => void) =>
      authFetch(path).then((r) => r.json()).then((d) => set((d.results ?? []).map(toItem))).catch(() => {});
    load("/api/favorites", setFav);
    load("/api/watchlist", setList);
    load("/api/history", setHist);
  }, [user, authFetch]);

  if (!ready || !user) {
    return <main className="min-h-screen pt-28 text-center text-white/50">Loading…</main>;
  }

  const handle = user.username.toLowerCase().replace(/\s+/g, "");
  const joined = user.joined
    ? new Date(user.joined.replace(" ", "T")).toLocaleDateString("en-US", { month: "long", year: "numeric" })
    : null;

  const TABS: { key: Tab; label: string; count?: number }[] = [
    { key: "overview", label: "Overview" },
    { key: "favorites", label: "Favorites", count: fav.length },
    { key: "mylist", label: "My List", count: list.length },
    { key: "history", label: "History", count: hist.length },
  ];

  return (
    <main className={`min-h-screen pb-16 ${profileFontVars}`}>
      {/* HERO */}
      <div className="relative flex h-[86vh] min-h-[560px] items-end overflow-hidden">
        {(activeProfile as any)?.banner || user.banner ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={(activeProfile as any)?.banner || user.banner} alt="" className="absolute inset-0 h-full w-full object-cover" />
        ) : (
          <div className="absolute inset-0 bg-gradient-to-br from-accent/25 via-bg to-bg" />
        )}
        <div className="pointer-events-none absolute inset-0 bg-black/40" />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-bg to-transparent" />

        <div className="relative z-[2] w-full px-[4%] pb-12">
          <div className="mx-auto flex max-w-5xl flex-wrap items-end justify-between gap-4">
            <div className="flex items-end gap-5">
              {(activeProfile as any)?.avatar || user.avatar ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={(activeProfile as any)?.avatar || user.avatar} alt="Avatar" className="h-32 w-32 shrink-0 rounded-full border-4 border-bg object-cover md:h-36 md:w-36" />
              ) : (
                <span className="grid h-32 w-32 shrink-0 place-items-center rounded-full border-4 border-bg bg-accent text-4xl font-bold text-black md:h-36 md:w-36">
                  {user.username.charAt(0).toUpperCase()}
                </span>
              )}
              <div className="mb-1">
                <h1 className="pb-1 text-4xl font-extrabold drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)] md:text-5xl" style={{ fontFamily: nameFontCss(user.name_font), lineHeight: 1.45 }}>
                  {user.username}
                </h1>
                <p className="text-sm text-white/70 drop-shadow">
                  @{handle}
                  {joined && <span className="ml-2">· Joined {joined}</span>}
                </p>
              </div>
            </div>

            <Link href="/profil/settings" className="mb-2 rounded-lg border border-white/25 bg-black/30 px-5 py-2.5 text-sm font-semibold backdrop-blur transition hover:bg-white/10">
              Edit Profil
            </Link>
          </div>
        </div>
      </div>

      {/* BELOW: bio + tabs + list */}
      <div className="mx-auto max-w-5xl px-[4%] py-10">
        {(activeProfile as any)?.bio || user.bio ? (
          <p className="max-w-2xl text-[15px] italic leading-7 text-white/70">{(activeProfile as any)?.bio || user.bio}</p>
        ) : (
          <p className="text-sm text-white/40">Belum ada bio. Klik Edit Profil untuk menambahkan.</p>
        )}

        <div className="mt-6 flex flex-wrap gap-2 border-b border-white/10 pb-3">
          {TABS.map((t) => (
            <button key={t.key} onClick={() => setTab(t.key)} className={`relative rounded-full px-4 py-2 text-sm font-semibold transition ${tab === t.key ? "text-black" : "text-white/60 hover:bg-white/10 hover:text-white"}`}>
              {tab === t.key && <motion.span layoutId="profileTab" transition={pillSpring} className="absolute inset-0 rounded-full bg-accent" />}
              <span className="relative z-10">
                {t.label}
                {t.count != null && t.count > 0 && <span className={`ml-1.5 ${tab === t.key ? "text-black/60" : "text-white/40"}`}>{t.count}</span>}
              </span>
            </button>
          ))}
        </div>

        <div className="mt-6">
          {tab === "overview" && (
            <div className="space-y-8">
              <div className="grid grid-cols-3 gap-3">
                {[
                  { label: "Favorites", n: fav.length },
                  { label: "My List", n: list.length },
                  { label: "Watched", n: hist.length },
                ].map((s) => (
                  <div key={s.label} className="rounded-xl bg-white/[0.04] px-4 py-5 text-center ring-1 ring-white/10">
                    <div className="text-2xl font-bold text-accent">{s.n}</div>
                    <div className="text-xs text-white/50">{s.label}</div>
                  </div>
                ))}
              </div>
              {fav.length > 0 && (
                <div>
                  <h3 className="mb-3 text-lg font-bold">Favorites</h3>
                  <PosterGrid items={fav.slice(0, 12)} />
                </div>
              )}
              {hist.length > 0 && (
                <div>
                  <h3 className="mb-3 text-lg font-bold">Recently watched</h3>
                  <PosterGrid items={hist.slice(0, 12)} />
                </div>
              )}
            </div>
          )}
          {tab === "favorites" && <PosterGrid items={fav} />}
          {tab === "mylist" && <PosterGrid items={list} />}
          {tab === "history" && <PosterGrid items={hist} />}
        </div>
      </div>
    </main>
  );
}
