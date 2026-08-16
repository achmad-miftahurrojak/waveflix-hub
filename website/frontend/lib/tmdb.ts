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

const BACKEND = process.env.BACKEND_URL ?? "http://localhost:8080";
const REVALIDATE = 60 * 15; 

async function api<T>(path: string, fallback: T, revalidate = REVALIDATE): Promise<T> {
  try {
    const res = await fetch(`${BACKEND}${path}`, { next: { revalidate } });
    if (!res.ok) throw new Error(`Backend ${res.status} @ ${path}`);
    return (await res.json()) as T;
  } catch (e) {
    console.error("[tmdb] gagal:", path, e);
    return fallback;
  }
}

const withPoster = (items: TmdbItem[] = []) => items.filter((m) => m.poster_path);
const tag = (items: TmdbItem[], media: MediaType) =>
  items.map((m) => ({ ...m, media_type: media }));

/**
 * Trending di Indonesia: yang banyak ditonton penonton Indonesia. Menganyam
 * (interleave) konten Korea (K-drama/K-movie — sangat digemari di Indonesia)
 * dengan judul populer global, jadi berbeda dari "Trending Now" yang murni global.
 */
export async function getTrendingIndonesia(): Promise<TmdbItem[]> {
  const krSince = `${new Date().getFullYear() - 3}-01-01`;
  const [movies, tv, krTv, krMovie] = await Promise.all([
    api<TmdbListResponse>(
      `/api/discover?media=movie&sort_by=popularity.desc&min_votes=150`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=tv&sort_by=popularity.desc&min_votes=150`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=tv&country=KR&sort_by=popularity.desc&min_votes=30&released_after=${krSince}`,
      { page: 1, results: [] }
    ),
    api<TmdbListResponse>(
      `/api/discover?media=movie&country=KR&sort_by=popularity.desc&min_votes=30&released_after=${krSince}`,
      { page: 1, results: [] }
    ),
  ]);

  const byPop = (list: TmdbItem[]) =>
    list.sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0));
  const globalPool = byPop([
    ...tag(withPoster(movies.results), "movie"),
    ...tag(withPoster(tv.results), "tv"),
  ]);
  const krPool = byPop([
    ...tag(withPoster(krTv.results), "tv"),
    ...tag(withPoster(krMovie.results), "movie"),
  ]);

  // Anyam: 1 Korea, 1 global, dst. supaya K-drama tampil menonjol + tetap ada hit global.
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

/** Trending global (worldwide) dari endpoint /trending TMDB. */
export async function getTrendingGlobal(): Promise<TmdbItem[]> {
  const [p1, p2] = await Promise.all([
    api<TmdbListResponse>(`/api/trending?media=all&page=1`, { page: 1, results: [] }),
    api<TmdbListResponse>(`/api/trending?media=all&page=2`, { page: 1, results: [] }),
  ]);
  const items = [...(p1.results ?? []), ...(p2.results ?? [])].filter(
    (m) => m.media_type === "movie" || m.media_type === "tv"
  );
  return withPoster(items).slice(0, 20);
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

// Platform legal besar di Indonesia (Netflix, Disney+, Prime, Apple, HBO, Viu, Vidio).
export const MAJOR_PROVIDERS = "8|122|119|350|1899|158|489";

/** Rilisan terbaru (film/series) di platform besar — untuk "Latest Movies/Series". */
export async function getLatest(media: MediaType): Promise<TmdbItem[]> {
  const today = new Date().toISOString().slice(0, 10);
  const data = await api<TmdbListResponse>(
    `/api/discover?media=${media}&sort_by=${
      media === "tv" ? "first_air_date.desc" : "primary_release_date.desc"
    }&released_before=${today}&provider=${MAJOR_PROVIDERS}&min_votes=1`,
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

/** Episode terbaru dari serial ongoing (pakai last_episode_to_air TMDB). */
export async function getLatestEpisodes(count = 14): Promise<LatestEpisode[]> {
  const today = new Date().toISOString().slice(0, 10);
  const list = await api<TmdbListResponse>(
    `/api/discover?media=tv&sort_by=first_air_date.desc&released_before=${today}&provider=${MAJOR_PROVIDERS}&min_votes=1`,
    { page: 1, results: [] }
  );
  const shows = withPoster(list.results).slice(0, count);
  const details = await Promise.all(
    shows.map((s) => getDetail("tv", String(s.id)))
  );
  const out: LatestEpisode[] = [];
  for (const d of details) {
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
    `/api/discover?media=${media}&sort_by=${sort}&released_before=${today}`,
    { page: 1, results: [] }
  );
  return tag(withPoster(data.results), media);
}

/** Discover generik untuk halaman /browse. */
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
  page?: number;
}): Promise<TmdbListResponse> {
  const qs = new URLSearchParams({
    media: params.media,
    min_votes: params.min_votes ?? "30",
  });
  if (params.genre) qs.set("genre", params.genre);
  if (params.year) qs.set("year", params.year);
  if (params.country) qs.set("country", params.country);
  if (params.provider) qs.set("provider", params.provider);
  if (params.sort_by) qs.set("sort_by", params.sort_by);
  if (params.released_after) qs.set("released_after", params.released_after);
  if (params.released_before) qs.set("released_before", params.released_before);
  if (params.page) qs.set("page", String(params.page));
  const data = await api<TmdbListResponse>(`/api/discover?${qs}`, {
    page: 1,
    results: [],
    total_pages: 0,
  });
  data.results = tag(withPoster(data.results), params.media);
  return data;
}

/** Discover banyak halaman sekaligus → grid "tampilkan semua" di /browse. */
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

/** Data satu slide hero (logo + meta), disiapkan di server. */
export interface HeroSlide {
  item: TmdbItem;
  logo: string | null;
  overview: string;
  tagline: string;
  genres: string[];
  duration: string;
  status: string;
  trailer: string | null; // YouTube key
}

/** Siapkan beberapa slide hero (untuk carousel ala IDLIX). */
export async function getHeroSlides(
  items: TmdbItem[],
  count = 5
): Promise<HeroSlide[]> {
  const picks = withPoster(items).filter((m) => m.backdrop_path).slice(0, count);
  return Promise.all(
    picks.map(async (item) => {
      const media = mediaTypeOf(item);
      const [logo, detail] = await Promise.all([
        getHeroLogo(item),
        getDetail(media, String(item.id)),
      ]);
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

export async function getDetail(media: MediaType, id: string): Promise<TmdbDetail | null> {
  const data = await api<TmdbDetail | { error: string }>(
    `/api/detail?media=${media}&id=${id}`,
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

export async function getHeroLogo(item: TmdbItem): Promise<string | null> {
  const media = mediaTypeOf(item);
  const data = await api<TmdbImagesResponse>(
    `/api/images?media=${media}&id=${item.id}`,
    { logos: [] }
  );
  const logos = data.logos ?? [];
  if (logos.length === 0) return null;
  const en = logos.find((l) => l.iso_639_1 === "en") ?? logos[0];
  return `https://image.tmdb.org/t/p/w500${en.file_path}`;
}
