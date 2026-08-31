"use server";

import { discover, MAJOR_PROVIDERS } from "@/lib/tmdb";
import type { MediaType, TmdbItem } from "@/lib/types";

export async function fetchBrowsePage(
  sp: Record<string, string | undefined>,
  page: number
): Promise<{ results: TmdbItem[]; hasMore: boolean }> {
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const isCategory = Boolean(sp.genre || sp.year || sp.country || sp.provider);
  const today = new Date().toISOString().slice(0, 10);
  const since = new Date(Date.now() - 180 * 86400000).toISOString().slice(0, 10);
  const since5y = `${new Date().getFullYear() - 5}-01-01`;

  const without_genres = (sp.genre === "16" || sp.genre === "10764") ? undefined : "16,10764";

  if (isCategory) {
    const isNetwork = Boolean(sp.provider);
    const noDate = Boolean(sp.year) || isNetwork;
    const base = {
      genre: sp.genre,
      year: sp.year,
      country: sp.country,
      provider: isNetwork ? sp.provider : MAJOR_PROVIDERS,
      released_after: noDate ? undefined : since5y,
      released_before: noDate ? undefined : today,
      without_genres,
      page,
    };

    const sortUi = sp.sort_by ?? "";
    let sortMovie = isNetwork ? "popularity.desc" : "primary_release_date.desc";
    let sortTv = isNetwork ? "popularity.desc" : "first_air_date.desc";

    if (sortUi === "terpopuler") {
      sortMovie = "popularity.desc"; sortTv = "popularity.desc";
    } else if (sortUi === "terlama") {
      sortMovie = "primary_release_date.asc"; sortTv = "first_air_date.asc";
    } else if (sortUi === "terbaru") {
      sortMovie = "primary_release_date.desc"; sortTv = "first_air_date.desc";
    }

    const [mv, tvr] = await Promise.all([
      discover({ ...base, media: "movie", sort_by: sortMovie }),
      discover({ ...base, media: "tv", sort_by: sortTv }),
    ]);

    const merged = [...(mv.results || []), ...(tvr.results || [])];

    if (sortMovie.startsWith("popularity")) {
      merged.sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0));
    } else if (sortMovie.endsWith("asc")) {
      const dateVal = (m: TmdbItem) => new Date(m.release_date || m.first_air_date || 0).getTime();
      merged.sort((a, b) => dateVal(a) - dateVal(b));
    } else {
      const dateVal = (m: TmdbItem) => new Date(m.release_date || m.first_air_date || 0).getTime();
      merged.sort((a, b) => dateVal(b) - dateVal(a));
    }

    const hasMore = (mv.total_pages || 0) > page || (tvr.total_pages || 0) > page;
    return { results: merged, hasMore };
  } else {
    let sortUi = sp.sort_by ?? "popularity.desc";
    if (sortUi === "terbaru") sortUi = media === "tv" ? "first_air_date.desc" : "primary_release_date.desc";
    if (sortUi === "terlama") sortUi = media === "tv" ? "first_air_date.asc" : "primary_release_date.asc";
    if (sortUi === "terpopuler") sortUi = "popularity.desc";

    const params = {
      media,
      provider: MAJOR_PROVIDERS,
      sort_by: sortUi,
      without_genres,
      page,
      ...(sortUi.includes("popularity") ? {} : { released_after: since }),
    };
    const data = await discover(params);
    return {
      results: data.results || [],
      hasMore: (data.total_pages || 0) > page
    };
  }
}
