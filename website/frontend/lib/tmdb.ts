import "server-only";
import type {
  TmdbItem,
  TmdbListResponse,
  TmdbImagesResponse,
  TmdbDetail,
  Episode,
  MediaType,
} from "./types";
import {
  mediaTypeOf,
  genreNames,
  durationText,
  tvCountsText,
  statusLabel,
  isTv,
} from "./helpers";

import { cookies } from "next/headers";

const BACKEND = process.env.BACKEND_URL ?? "http://localhost:8081";
const REVALIDATE = 60 * 15; 

async function api<T>(path: string, fallback: T, revalidate = REVALIDATE): Promise<T> {
  let lang = "id";
  try {
    const cookieStore = await cookies();
    lang = cookieStore.get("waveflix_lang")?.value || "id";
  } catch (e) {

  }
  const sep = path.includes("?") ? "&" : "?";
  const p = `${path}${sep}lang=${lang}`;

  try {
    const res = await fetch(`${BACKEND}${p}`, { next: { revalidate } });
    if (!res.ok) {
      if (res.status !== 403 && res.status !== 404 && res.status !== 502 && res.status !== 429) {
        console.error(`[tmdb] gagal (${res.status}):`, path);
      }
      return fallback;
    }
    return (await res.json()) as T;
  } catch (e) {
    console.error("[tmdb] gagal:", path, e);
    return fallback;
  }
}

const withPoster = (items: TmdbItem[] = []) => (items || []).filter((m) => m.poster_path);
const tag = (items: TmdbItem[], media: MediaType) =>
  (items || []).map((m) => ({ ...m, media_type: media }));


export async function getTrendingIndonesia(): Promise<TmdbItem[]> {
  const today = new Date().toISOString().slice(0, 10);
  const [movies, tv, krTv, krMovie] = await Promise.all([
    api<TmdbListResponse>(
      `/api/discover?media=movie&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=tv&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=tv&country=KR&sort_by=first_air_date.desc&released_before=${today}&without_genres=10764,10767,10763&provider=${MAJOR_PROVIDERS}`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=movie&country=KR&sort_by=primary_release_date.desc&released_before=${today}&provider=${MAJOR_PROVIDERS}`,
      { page: 1, results: [] }
    ),
  ]);

  const byPop = (list: TmdbItem[]) =>
    list.sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0));
  const globalPool = byPop([
    ...tag(withPoster(movies.results), "movie"),
    ...tag(withPoster(tv.results), "tv"),
  ]);

  const parseDate = (m: TmdbItem) =>
    new Date(m.first_air_date || m.release_date || "1970-01-01").getTime();
  const krPool = [
    ...tag(withPoster(krTv.results), "tv"),
    ...tag(withPoster(krMovie.results), "movie"),
  ].sort((a, b) => parseDate(b) - parseDate(a));

  const out: TmdbItem[] = [];
  const seen = new Set<string>();
  const key = (m: TmdbItem) => `${m.media_type ?? "?"}-${m.id}`;
  const push = (m?: TmdbItem) => {
    if (m && !seen.has(key(m))) {
      seen.add(key(m));
      out.push(m);
    }
  };
  let gi = 0;
  let ki = 0;
  while (out.length < 20 && (gi < globalPool.length || ki < krPool.length)) {
    push(krPool[ki++]);
    push(globalPool[gi++]);
  }
  return out.slice(0, 20);
}

export async function getTrendingGlobal(): Promise<TmdbItem[]> {
  const [p1, p2] = await Promise.all([
    api<TmdbListResponse>(`/api/discover?media=movie&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}`, { page: 1, results: [] }),
    api<TmdbListResponse>(`/api/discover?media=tv&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}`, { page: 1, results: [] }),
  ]);
  const items = [...tag(p1.results ?? [], "movie"), ...tag(p2.results ?? [], "tv")];
  const merged = items.sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0));
  return withPoster(merged).slice(0, 20);
}

export async function getTrendingKDrama(): Promise<TmdbItem[]> {
  const today = new Date().toISOString().slice(0, 10);
  const krTv = await api<TmdbListResponse>(
    `/api/discover?media=tv&country=KR&sort_by=popularity.desc&released_before=${today}&without_genres=16,10764,99,10767,10763&provider=${MAJOR_PROVIDERS}`,
    { page: 1, results: [] }
  );
  return tag(withPoster(krTv.results), "tv").slice(0, 20);
}

export async function discoverByProvider(
  provider: number,
  media: MediaType
): Promise<TmdbItem[]> {
  const data = await api<TmdbListResponse>(
    `/api/discover?media=${media}&provider=${provider}`,
    { page: 1, results: [] }
  );
  return tag(withPoster(data.results), media);
}


export async function getLatest(media: MediaType): Promise<TmdbItem[]> {
  const today = new Date().toISOString().slice(0, 10);
  const data = await api<TmdbListResponse>(
    `/api/discover?media=${media}&sort_by=${
      media === "tv" ? "first_air_date.desc" : "primary_release_date.desc"
    }&released_before=${today}&provider=${MAJOR_PROVIDERS}`,
    { page: 1, results: [] }
  );
  return tag(withPoster(data.results), media).slice(0, 18);
}

export interface LatestEpisode {
  show: TmdbItem;
  season: number;
  episode: number;
  name: string;
  still: string | null;
  air_date: string;
}

export async function getLatestEpisodes(count = 14): Promise<LatestEpisode[]> {
  const today = new Date().toISOString().slice(0, 10);
  const list = await api<TmdbListResponse>(
    `/api/discover?media=tv&sort_by=popularity.desc&released_before=${today}&provider=${MAJOR_PROVIDERS}`,
    { page: 1, results: [] }
  );
  const shows = withPoster(list.results).slice(0, count);
  if (shows.length === 0) return [];

  const ids = shows.map((s) => s.id).join(",");
  const batchResponse = await api<{ results: Record<string, TmdbDetail> }>(
    `/api/batch?media=tv&ids=${ids}`,
    { results: {} }
  );
  const detailsMap = batchResponse.results || {};

  const out: LatestEpisode[] = [];
  for (const s of shows) {
    const d = detailsMap[String(s.id)];
    const ep = d?.last_episode_to_air;
    if (!d || !ep || ep.season_number == null || ep.episode_number == null)
      continue;
    out.push({
      show: { ...d, media_type: "tv" },
      season: ep.season_number,
      episode: ep.episode_number,
      name: ep.name || `Episode ${ep.episode_number}`,
      still: ep.still_path ?? d.backdrop_path ?? null,
      air_date: ep.air_date || "",
    });
  }
  return out;
}

export async function getRecent(media: MediaType): Promise<TmdbItem[]> {
  const today = new Date().toISOString().slice(0, 10);
  const sort = media === "tv" ? "first_air_date.desc" : "primary_release_date.desc";
  const data = await api<TmdbListResponse>(
    `/api/discover?media=${media}&sort_by=${sort}&released_before=${today}&provider=${MAJOR_PROVIDERS}`,
    { page: 1, results: [] }
  );
  return tag(withPoster(data.results), media);
}

export const MAJOR_PROVIDERS = "8|119|350|122|1899"; // Netflix, Prime Video, Apple TV, Disney+, HBO Max

export async function discover(params: {
  media: MediaType;
  genre?: string;
  year?: string;
  country?: string;
  provider?: string;
  sort_by?: string;
  released_after?: string;
  released_before?: string;
  min_votes?: string;
  without_genres?: string;
  page?: number;
}): Promise<TmdbListResponse> {
  const qs = new URLSearchParams({
    media: params.media,
  });
  if (params.genre) qs.set("genre", params.genre);
  if (params.year) qs.set("year", params.year);
  if (params.country) qs.set("country", params.country);
  
  // Default to major providers if no specific provider is selected
  qs.set("provider", params.provider || MAJOR_PROVIDERS);
  
  if (params.sort_by) qs.set("sort_by", params.sort_by);
  if (params.released_after) qs.set("released_after", params.released_after);
  if (params.released_before) qs.set("released_before", params.released_before);
  if (params.without_genres) qs.set("without_genres", params.without_genres);
  if (params.page) qs.set("page", String(params.page));
  
  const data = await api<TmdbListResponse>(`/api/discover?${qs}`, {
    page: 1,
    results: [],
    total_pages: 0,
  });
  data.results = tag(withPoster(data.results), params.media);
  return data;
}

export async function discoverMany(
  params: Parameters<typeof discover>[0],
  pages = 5
): Promise<TmdbListResponse> {
  const first = await discover({ ...params, page: 1 });
  const total = Math.min(first.total_pages ?? 1, pages);
  if (total <= 1) return first;

  const rest = await Promise.all(
    Array.from({ length: total - 1 }, (_, i) =>
      discover({ ...params, page: i + 2 })
    )
  );

  const seen = new Set(first.results.map((m) => m.id));
  for (const r of rest) {
    for (const m of r.results) {
      if (!seen.has(m.id)) {
        seen.add(m.id);
        first.results.push(m);
      }
    }
  }
  return first;
}

export interface HeroSlide {
  item: TmdbItem;
  logo: string | null;
  overview: string;
  tagline: string;
  genres: string[];
  duration: string;
  status: string;
  trailer: string | null; 
}

export async function getHeroSlides(
  items: TmdbItem[],
  count = 5
): Promise<HeroSlide[]> {
  const cookieStore = await cookies();
  const lang = cookieStore.get("waveflix_lang")?.value || "id";
  const picks = withPoster(items).filter((m) => m.backdrop_path).slice(0, count);
  return Promise.all(
    picks.map(async (item) => {
      const media = mediaTypeOf(item);
      const detail = await getDetail(media, String(item.id), "en");
      const logos = detail?.images?.logos ?? [];
      const localizedLogo =
        logos.find((l) => l.iso_639_1 === lang) ??
        logos.find((l) => l.iso_639_1 === 'en') ??
        logos[0];
      const logo = localizedLogo
        ? `https://image.tmdb.org/t/p/w500${localizedLogo.file_path}`
        : await getHeroLogo(item, lang);
      const ov = detail?.overview || item.overview || "";
      const duration = detail
        ? isTv(detail)
          ? tvCountsText(detail)
          : durationText(detail)
        : "";
      const vids = detail?.videos?.results ?? [];
      const yt =
        vids.find((v) => v.site === "YouTube" && v.type === "Trailer") ??
        vids.find((v) => v.site === "YouTube" && v.type === "Teaser") ??
        vids.find((v) => v.site === "YouTube");
      return {
        item,
        logo,
        overview: ov,
        tagline: detail?.tagline || "",
        genres: detail ? genreNames(detail).slice(0, 3) : [],
        duration,
        status: detail ? statusLabel(detail) : "",
        trailer: yt?.key ?? null,
      };
    })
  );
}

export async function searchMulti(query: string): Promise<TmdbItem[]> {
  if (!query.trim()) return [];
  const data = await api<TmdbListResponse>(
    `/api/search?q=${encodeURIComponent(query)}`,
    { page: 1, results: [] },
    0
  );
  return withPoster(
    (data.results ?? []).filter((m) => m.media_type === "movie" || m.media_type === "tv")
  );
}

export async function getDetail(media: MediaType, id: string, lang?: string): Promise<TmdbDetail | null> {
  const path = `/api/detail?media=${media}&id=${id}${lang ? `&lang=${lang}` : ''}`;
  const data = await api<TmdbDetail | { error: string }>(
    path,
    { error: "kosong" } as { error: string }
  );
  if ("error" in data) return null;
  return { ...(data as TmdbDetail), media_type: media };
}

export async function getSeasonEpisodes(id: string, season: number): Promise<Episode[]> {
  const data = await api<{ episodes?: Episode[] }>(
    `/api/season?id=${id}&season=${season}`,
    { episodes: [] }
  );
  return data.episodes ?? [];
}

export async function getHeroLogo(item: TmdbItem, lang?: string): Promise<string | null> {
  let finalLang = lang;
  if (!finalLang) {
    try {
      const cookieStore = await cookies();
      finalLang = cookieStore.get("waveflix_lang")?.value || "id";
    } catch {
      finalLang = "id";
    }
  }

  const media = mediaTypeOf(item);
  const detail = await getDetail(media, String(item.id), finalLang);
  const logos = detail?.images?.logos ?? [];
  if (logos.length === 0) return null;
  const loc = logos.find((l) => l.iso_639_1 === finalLang) ?? logos.find((l) => l.iso_639_1 === 'en') ?? logos[0];
  return `https://image.tmdb.org/t/p/w500${loc.file_path}`;
}

export async function getPerson(id: string): Promise<import('./types').TmdbPerson | null> {
  const data = await api<import('./types').TmdbPerson | { error: string }>(
    `/api/person?id=${id}`,
    { error: "kosong" } as { error: string }
  );
  if ("error" in data) return null;
  return data as import('./types').TmdbPerson;
}
