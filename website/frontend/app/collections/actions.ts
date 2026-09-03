"use server";

import { discover, getItemsWithLogos } from "@/lib/tmdb";
import type { CollectionCardItem } from "@/components/CollectionCards";
import { IMG } from "@/lib/helpers";

export async function fetchMoreCollections(
  sp: Record<string, string | undefined>,
  page: number
): Promise<{ results: CollectionCardItem[], hasMore: boolean }> {
  // `page` in this context starts at 2.
  // Frontend Page 1 fetched TMDB pages 1-3.
  // Page 2 fetches TMDB pages 4-5.
  // Page 3 fetches TMDB pages 6-7.
  const tmdbStartPage = (page - 2) * 2 + 4; 
  
  const activeTypes = sp.type ? sp.type.split(",").filter(Boolean) : [];
  const activeCountries = sp.country ? sp.country.split(",").filter(Boolean) : [];

  // If filtered to movies only, we don't fetch dynamic TV shows on pagination
  if (activeTypes.length > 0 && !activeTypes.includes("tv") && !activeTypes.includes("animation")) {
    return { results: [], hasMore: false };
  }

  const qs: any = { media: "tv", sort_by: "popularity.desc" };
  if (activeCountries.length > 0) {
    qs.country = activeCountries.join("|"); 
  }
  if (activeTypes.includes("animation") && !activeTypes.includes("tv")) {
    qs.genre = "16"; // TMDB animation genre id
  }

  try {
    const page1Data = await discover({ ...qs, page: tmdbStartPage });
    const page2Data = await discover({ ...qs, page: tmdbStartPage + 1 });
    
    const tvItems = [...(page1Data.results || []), ...(page2Data.results || [])];
    if (tvItems.length === 0) {
      return { results: [], hasMore: false };
    }
    
    const tvWithLogos = await getItemsWithLogos(tvItems, "tv");
    const filteredTv = tvWithLogos.filter(m => (m.number_of_seasons || 1) > 1);
    
    const collectionsPreview = filteredTv.map(m => ({
      id: String(m.id),
      name: m.name || m.title || "",
      logoSrc: m.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
      backdropSrc: m.backdrop_path ? `${IMG}/w780${m.backdrop_path}` : undefined,
      href: `/collections?c=${m.id}&t=tv`,
    } as CollectionCardItem));

    const totalPages = page2Data.total_pages || page1Data.total_pages || 0;
    const hasMore = totalPages > (tmdbStartPage + 1);

    return {
      results: collectionsPreview,
      hasMore
    };

  } catch (e) {
    console.error("Failed to fetch more collections", e);
    return { results: [], hasMore: false };
  }
}
