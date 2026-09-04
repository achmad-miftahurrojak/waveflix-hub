import type { MediaType, TmdbItem } from "@/lib/types";
import { COLLECTIONS } from "@/lib/catalog";
import { CoverflowSlide } from "@/components/ui/coverflow-carousel";
import { getCollectionMoviesWithLogos, getTvShowSeasons, getItemsWithLogos, discoverMany } from "@/lib/tmdb";
import { IMG } from "@/lib/helpers";
import { CollectionCards, CollectionCardItem } from "@/components/CollectionCards";
import InfiniteCollectionGrid from "@/components/InfiniteCollectionGrid";
import CollectionFilters from "@/components/CollectionFilters";
import CollectionViewer from "@/components/CollectionViewer";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";

export default async function CollectionsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;

  // If no specific collection is selected, show the CollectionCards grid
  if (!sp.c) {
    const activeTypes = sp.type ? sp.type.split(",").filter(Boolean) : [];
    const activeCountries = sp.country ? sp.country.split(",").filter(Boolean) : [];

    let collectionsPreview: CollectionCardItem[] = [];

    // 1. Fetch Hardcoded Movie Collections
    if (activeTypes.length === 0 || activeTypes.includes("movie") || activeTypes.includes("animation")) {
      const filteredCollections = COLLECTIONS.filter(c => {
        let matchType = true;
        let matchCountry = true;

        if (activeTypes.length > 0) {
          matchType = c.type.some(t => activeTypes.includes(t));
        }
        if (activeCountries.length > 0) {
          matchCountry = c.country.some(co => activeCountries.includes(co));
        }

        return matchType && matchCountry;
      });

      const movieCollectionsPreview = await Promise.all(
        filteredCollections.map(async (c) => {
          const movies = await getCollectionMoviesWithLogos(c.id, 1);
          const m = movies[0];
          return {
            id: c.id,
            name: c.name,
            logoSrc: m?.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
            backdropSrc: m?.backdrop_path ? `${IMG}/w780${m.backdrop_path}` : undefined,
            href: `/collections?c=${c.id}&t=movie`, // Add type to distinguish
          } as CollectionCardItem;
        })
      );
      collectionsPreview = collectionsPreview.concat(movieCollectionsPreview);
    }

    let hasMoreDynamic = false;

    // 2. Fetch Dynamic TV Shows (that act as collections of seasons)
    if (activeTypes.length === 0 || activeTypes.includes("tv") || activeTypes.includes("animation")) {
      const qs: any = { media: "tv", sort_by: "popularity.desc" };
      
      if (activeCountries.length > 0) {
        qs.country = activeCountries.join("|"); 
      }
      if (activeTypes.includes("animation") && !activeTypes.includes("tv")) {
        qs.genre = "16"; // TMDB animation genre id
      }

      try {
        // Fetch multiple pages so we have enough candidates after filtering out 1-season shows
        const tvData = await discoverMany(qs, 3);
        hasMoreDynamic = (tvData.total_pages || 0) > 3;
        const tvItems = tvData.results || [];
        
        // We only fetch batch logos for the first 40 to avoid heavy load
        const tvWithLogos = await getItemsWithLogos(tvItems.slice(0, 40), "tv");
        
        // Filter out TV shows that have 1 season or less
        const filteredTv = tvWithLogos.filter(m => (m.number_of_seasons || 1) > 1).slice(0, 15);
        
        const tvCollectionsPreview = filteredTv.map(m => ({
          id: String(m.id),
          name: m.name || m.title || "",
          logoSrc: m.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
          backdropSrc: m.backdrop_path ? `${IMG}/w780${m.backdrop_path}` : undefined,
          href: `/collections?c=${m.id}&t=tv`,
        } as CollectionCardItem));
        
        collectionsPreview = collectionsPreview.concat(tvCollectionsPreview);
      } catch (e) {
        console.error("Failed to fetch dynamic TV collections", e);
      }
    }

    // Sort to mix movies and tv (optional), or let movies be first
    
    return (
      <main className="min-h-screen pb-16">
        <div className="pt-28 px-[4%] max-w-[1600px] mx-auto">
          <div className="flex flex-col lg:flex-row lg:items-center justify-between mb-8 gap-4">
            <h1 className="text-3xl font-black text-white">Collections</h1>
            <CollectionFilters />
          </div>
          {collectionsPreview.length > 0 ? (
            <InfiniteCollectionGrid initialItems={collectionsPreview} initialHasMore={hasMoreDynamic} sp={sp} />
          ) : (
            <div className="py-20 text-center text-white/50">
              Tidak ada koleksi yang cocok dengan filter ini.
            </div>
          )}
        </div>
      </main>
    );
  }

  // A collection is selected, show the Coverflow Carousel
  const currentId = sp.c;
  const isTv = sp.t === "tv";
  
  let movies: TmdbItem[] = [];
  let collectionName = "";

  if (isTv) {
    movies = await getTvShowSeasons(currentId);
    collectionName = movies.length > 0 ? (movies[0] as any)?.showName || "TV Series Seasons" : "Seasons";
    // For TV, getTvShowSeasons could be improved to return the show name, but we can also just fetch detail again if needed.
    // Actually, getTvShowSeasons maps the show's logo to all seasons, so we're good.
  } else {
    movies = await getCollectionMoviesWithLogos(currentId, 20);
    const currentCollection = COLLECTIONS.find((c) => c.id === currentId);
    collectionName = currentCollection?.name || "Collection";
  }
  
  const slides: CoverflowSlide[] = movies.map(m => ({
    src: m.poster_path ? `${IMG}/w500${m.poster_path}` : "https://images.unsplash.com/photo-1594909122845-11baa439b7bf?q=80&w=500",
    alt: m.title || m.name || "",
    title: m.title || m.name || "",
    logoSrc: m.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
    subtitle: m.release_date ? new Date(m.release_date).getFullYear().toString() : undefined,
    href: isTv ? `/tv/${currentId}` : `/movie/${m.id}`,
  }));

  return (
    <main className="min-h-screen pb-24 overflow-x-hidden pt-28">
      <div className="px-[4%] mb-10 flex items-center gap-4">
        <Link href="/collections" className="p-2 rounded-full bg-white/10 hover:bg-white/20 transition text-white">
          <ArrowLeft className="size-5" />
        </Link>
        <h1 className="text-3xl font-black">{collectionName}</h1>
      </div>


      {slides.length > 0 && (
        <div key={currentId} className="animate-in fade-in zoom-in-95 duration-500">
          <CollectionViewer 
            slides={slides} 
            movies={movies} 
            isTv={isTv} 
            currentId={currentId} 
          />
        </div>
      )}
    </main>
  );
}
