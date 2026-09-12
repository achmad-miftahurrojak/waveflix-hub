"use server";

import { discover, MAJOR_PROVIDERS } from "@/lib/tmdb";
import type { MediaType, TmdbItem } from "@/lib/types";

export async function fetchBrowsePage(
  sp: Record<string, string | undefined>,
  page: number
): Promise<{ results: TmdbItem[]; hasMore: boolean }> {
  const media: MediaType = (sp.media === "movie" || sp.media === "tv") ? (sp.media as MediaType) : "all";
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
      popularity_gte: "15",
      page,
    };

    const sortUi = sp.sort_by ?? "terpopuler";
    let sortKey = isNetwork ? "popularity.desc" : "popularity.desc";

    if (sortUi === "terlama") {
      sortKey = media === "tv" ? "first_air_date.asc" : "primary_release_date.asc";
    } else if (sortUi === "terbaru") {
      sortKey = media === "tv" ? "first_air_date.desc" : "primary_release_date.desc";
    }

    if (media === "all") {
      // Fetch both and merge
      const movieSortKey = sortUi === "terlama" ? "primary_release_date.asc" : sortUi === "terbaru" ? "primary_release_date.desc" : "popularity.desc";
      const tvSortKey = sortUi === "terlama" ? "first_air_date.asc" : sortUi === "terbaru" ? "first_air_date.desc" : "popularity.desc";
      
      const [movieData, tvData] = await Promise.all([
        discover({ ...base, media: "movie", sort_by: movieSortKey }),
        discover({ ...base, media: "tv", sort_by: tvSortKey })
      ]);
      
      const combined = [...(movieData.results || []), ...(tvData.results || [])];
      
      if (sortUi === "terbaru") {
        combined.sort((a, b) => {
          const dA = new Date((a.release_date || a.first_air_date) ?? 0).getTime();
          const dB = new Date((b.release_date || b.first_air_date) ?? 0).getTime();
          return dB - dA;
        });
      } else if (sortUi === "terlama") {
        combined.sort((a, b) => {
          const dA = new Date((a.release_date || a.first_air_date) ?? 9999999999999).getTime();
          const dB = new Date((b.release_date || b.first_air_date) ?? 9999999999999).getTime();
          return dA - dB;
        });
      } else {
        combined.sort((a, b) => (b.popularity || 0) - (a.popularity || 0));
      }
      
      console.log(`[actions] fetchBrowsePage returned ${combined.length} combined items for page ${page}`);
      return {
        results: combined,
        hasMore: ((movieData.total_pages || 0) > page) || ((tvData.total_pages || 0) > page)
      };
    } else {
      const data = await discover({ ...base, media, sort_by: sortKey });
      console.log(`[actions] fetchBrowsePage returned ${data.results?.length} items for page ${page}`);
      return { 
        results: data.results || [], 
        hasMore: (data.total_pages || 0) > page 
      };
    }
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
      popularity_gte: "15",
      page,
    };
    if (media === "all") {
      const [movieData, tvData] = await Promise.all([
        discover({ ...params, media: "movie", sort_by: sortUi === "terbaru" ? "primary_release_date.desc" : sortUi === "terlama" ? "primary_release_date.asc" : "popularity.desc" }),
        discover({ ...params, media: "tv", sort_by: sortUi === "terbaru" ? "first_air_date.desc" : sortUi === "terlama" ? "first_air_date.asc" : "popularity.desc" })
      ]);
      const combined = [...(movieData.results || []), ...(tvData.results || [])];
      
      if (sortUi === "terbaru") {
        combined.sort((a, b) => new Date((b.release_date || b.first_air_date) ?? 0).getTime() - new Date((a.release_date || a.first_air_date) ?? 0).getTime());
      } else if (sortUi === "terlama") {
        combined.sort((a, b) => new Date((a.release_date || a.first_air_date) ?? 9999999999999).getTime() - new Date((b.release_date || b.first_air_date) ?? 9999999999999).getTime());
      } else {
        combined.sort((a, b) => (b.popularity || 0) - (a.popularity || 0));
      }
      return { results: combined, hasMore: ((movieData.total_pages || 0) > page) || ((tvData.total_pages || 0) > page) };
    } else {
      const data = await discover(params);
      return { results: data.results || [], hasMore: (data.total_pages || 0) > page };
    }
  }
}
