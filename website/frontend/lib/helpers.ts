import type { TmdbItem, TmdbDetail, MediaType } from "./types";

export const IMG = "https://image.tmdb.org/t/p";

export function isTv(m: TmdbItem): boolean {
  return m.media_type === "tv" || Boolean(m.first_air_date);
}

export function mediaTypeOf(m: TmdbItem): MediaType {
  return isTv(m) ? "tv" : "movie";
}

export function itemTitle(m: TmdbItem): string {
  return m.title || m.name || "Untitled";
}

export function itemYear(m: TmdbItem): string {
  return (m.release_date || m.first_air_date || "").slice(0, 4);
}

export function posterUrl(m: TmdbItem): string {
  return m.poster_path
    ? `${IMG}/w500${m.poster_path}`
    : "https://images.unsplash.com/photo-1594909122845-11baa439b7bf?q=80&w=500";
}

export function backdropUrl(m: TmdbItem): string {
  return m.backdrop_path
    ? `${IMG}/original${m.backdrop_path}`
    : "https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?q=80&w=2070";
}

export function stillUrl(path?: string | null): string {
  return path ? `${IMG}/w300${path}` : "";
}

export function profileUrl(path?: string | null): string {
  return path
    ? `${IMG}/w185${path}`
    : "https://ui-avatars.com/api/?background=1f2833&color=fff&name=%3F";
}

export function ratingText(m: TmdbItem): string {
  return m.vote_average ? m.vote_average.toFixed(1) : "N/A";
}

/** Durasi menit -> "2h 25m". */
export function runtimeText(minutes?: number | null): string {
  if (!minutes || minutes <= 0) return "";
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

/** Ringkasan durasi/musim untuk baris meta. */
export function durationText(d: TmdbDetail): string {
  if (isTv(d)) {
    if (d.number_of_seasons) return `${d.number_of_seasons} Seasons`;
    if (d.episode_run_time?.length) return `${d.episode_run_time[0]}m/ep`;
    return "";
  }
  return runtimeText(d.runtime);
}

export function genreNames(d: TmdbDetail): string[] {
  return (d.genres ?? []).map((g) => g.name);
}

/** Meta jumlah untuk TV: "4 Seasons • 19 Episodes" (ala IDLIX). */
export function tvCountsText(d: TmdbDetail): string {
  const parts: string[] = [];
  if (d.number_of_seasons) parts.push(`${d.number_of_seasons} Seasons`);
  if (d.number_of_episodes) parts.push(`${d.number_of_episodes} Episodes`);
  return parts.join(" • ");
}

/** Status jadi label badge: ONGOING / ENDED. */
export function statusLabel(d: TmdbDetail): string {
  switch ((d.status || "").toLowerCase()) {
    case "returning series":
      return "ONGOING";
    case "ended":
      return "ENDED";
    case "canceled":
      return "CANCELED";
    default:
      return "";
  }
}

export function detailHref(m: TmdbItem): string {
  return `/${mediaTypeOf(m)}/${m.id}`;
}
