"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { TmdbItem, SeasonSummary, Episode } from "@/lib/types";
import { stillUrl, runtimeText, BACKEND } from "@/lib/helpers";
import { PlayIcon, ChevronRight } from "./Icons";
import { useTranslation } from "@/lib/i18n";

function formatAirDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const mon = d.toLocaleString("en-US", { month: "short" });
  return `${d.getDate()} ${mon} ${String(d.getFullYear()).slice(2)}`;
}

interface Props {
  show: TmdbItem;
  seasons: SeasonSummary[];
  initialSeason?: number;
  currentEpisode?: number; 
}

export default function EpisodesSection({
  show,
  seasons,
  initialSeason,
  currentEpisode,
}: Props) {
  const { t } = useTranslation();
  const valid = seasons.filter((s) => s.season_number > 0);
  const [season, setSeason] = useState(
    valid.find((s) => s.season_number === initialSeason)?.season_number ??
      valid[0]?.season_number ??
      1
  );
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [loading, setLoading] = useState(false);
  const [dropdownOpen, setDropdownOpen] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    fetch(`${BACKEND}/api/season?id=${show.id}&season=${season}`)
      .then((r) => r.json())
      .then((d) => {
        if (alive) setEpisodes(d.episodes ?? []);
      })
      .catch(() => alive && setEpisodes([]))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [show.id, season]);

  if (valid.length === 0) return null;

  const now = Date.now();
  const aired = episodes.filter(
    (ep) => ep.air_date && new Date(ep.air_date).getTime() <= now
  );

  return (
    <section className="mt-10">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-xl font-bold">{t("ui.episodes")}</h3>
        {valid.length > 1 && (
          <div className="relative" onMouseLeave={() => setDropdownOpen(false)}>
            <button
              onClick={() => setDropdownOpen(!dropdownOpen)}
              className="flex items-center gap-2 rounded-xl border border-white/20 bg-white/5 px-4 py-2 text-sm font-semibold outline-none backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)] text-white"
            >
              {valid.find(s => s.season_number === season)?.name || `${t("ui.season")} ${season}`}
              <ChevronRight className={`h-4 w-4 transition ${dropdownOpen ? "rotate-90" : ""}`} />
            </button>
            {dropdownOpen && (
              <div className="absolute right-0 top-full z-30 pt-2">
                <div className="w-48 rounded-xl border border-white/20 bg-white/5 p-2 backdrop-blur-[15px] backdrop-saturate-200 shadow-[0_4px_30px_rgba(0,0,0,0.1),inset_0_1px_0_rgba(255,255,255,0.3)] flex flex-col max-h-60 overflow-y-auto no-scrollbar">
                  {valid.map((s) => (
                    <button
                      key={s.season_number}
                      onClick={() => {
                        setSeason(s.season_number);
                        setDropdownOpen(false);
                      }}
                      className={`text-left px-3 py-2 text-sm rounded-md transition ${season === s.season_number ? "bg-white/10 text-accent" : "text-white/80 hover:bg-white/10 hover:text-white"}`}
                    >
                      {s.name || `${t("ui.season")} ${s.season_number}`}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {loading ? (
        <p className="py-6 text-white/50">{t("ui.loadingEpisodes")}</p>
      ) : aired.length === 0 ? (
        <p className="py-6 text-white/50 text-center italic">Tidak ada episode yang tersedia di server saat ini.</p>
      ) : (
        <div className="grid grid-cols-2 gap-x-4 gap-y-6 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {aired.map((ep) => (
            <Link
              key={ep.episode_number}
              href={`/tv/${show.id}/season/${season}/episode/${ep.episode_number}`}
              className="group text-left"
            >
              <div
                className={`relative aspect-video w-full overflow-hidden rounded-lg bg-surface ${
                  currentEpisode === ep.episode_number
                    ? "ring-2 ring-accent"
                    : ""
                }`}
              >
                {ep.still_path ? (

                  <img
                    src={stillUrl(ep.still_path)}
                    alt={ep.name}
                    loading="lazy"
                    className="h-full w-full object-cover transition duration-300 group-hover:scale-105"
                  />
                ) : null}
                <span className="absolute left-2 top-2 rounded bg-black/75 px-1.5 py-0.5 text-xs font-bold tracking-wide">
                  S{String(season).padStart(2, "0")}E
                  {String(ep.episode_number).padStart(2, "0")}
                </span>
                {ep.air_date && (
                  <span className="absolute right-2 top-2 rounded bg-black/75 px-1.5 py-0.5 text-xs font-semibold text-white/80">
                    {formatAirDate(ep.air_date)}
                  </span>
                )}
                {ep.runtime ? (
                  <span className="absolute bottom-2 right-2 rounded bg-black/75 px-1.5 py-0.5 text-xs font-semibold text-white/80">
                    {runtimeText(ep.runtime)}
                  </span>
                ) : null}
                <span className="absolute inset-0 grid place-items-center bg-black/30 opacity-0 transition group-hover:opacity-100">
                  <span className="grid h-11 w-11 place-items-center rounded-full bg-accent text-black">
                    <PlayIcon className="text-black" />
                  </span>
                </span>
              </div>
              <h4 className="mt-2 truncate font-semibold">
                {t("ui.episode")} {ep.episode_number}
              </h4>
              {ep.overview && (
                <p className="mt-1 line-clamp-2 text-sm text-white/55">
                  {ep.overview}
                </p>
              )}
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}
