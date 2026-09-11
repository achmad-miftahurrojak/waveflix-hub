"use server";

import { discover, MAJOR_PROVIDERS } from "@/lib/tmdb";
import type { MediaType, TmdbItem } from "@/lib/types";

export async function fetchBrowsePage(
  sp: Record<string, string | undefined>,
  page: number
): Promise<{ results: TmdbItem[]; hasMore: boolean }> {
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const isCategory = Boolean(sp.genre || sp.year || sp.country || sp.provider || sp.collection);
  const today = new Date().toISOString().slice(0, 10);
  const since = new Date(Date.now() - 180 * 86400000).toISOString().slice(0, 10);

  const without_genres = (sp.genre === "16" || sp.genre === "10764" || sp.genre === "99") ? undefined : "16,10764,99,10767,10763,10402";

  if (isCategory) {
    const isNetwork = Boolean(sp.provider);
    const noDate = Boolean(sp.year) || isNetwork;
    const base = {
      genre: sp.genre,
      year: sp.year,
      country: sp.country,
      collection: sp.collection,
      provider: isNetwork ? sp.provider : MAJOR_PROVIDERS,
      without_genres,
      page,
    };

    const sortUi = sp.sort_by ?? "terpopuler";
    let sortKey = isNetwork ? "popularity.desc" : "popularity.desc";

    if (sortUi === "terlama") {
      sortKey = media === "tv" ? "first_air_date.asc" : "primary_release_date.asc";
    } else if (sortUi === "terbaru") {
      sortKey = media === "tv" ? "first_air_date.desc" : "primary_release_date.desc";
    }

    const data = await discover({ ...base, media, sort_by: sortKey });
    console.log(`[actions] fetchBrowsePage returned ${data.results?.length} items for page ${page}`);
    return { 
      results: data.results || [], 
      hasMore: (data.total_pages || 0) > page 
    };
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
    };
    const data = await discover(params);
    return {
      results: data.results || [],
      hasMore: (data.total_pages || 0) > page
    };
  }
}
