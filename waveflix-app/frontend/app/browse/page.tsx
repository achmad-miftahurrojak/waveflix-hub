import type { MediaType } from "@/lib/types";
import { discoverMany, MAJOR_PROVIDERS, getHeroSlides } from "@/lib/tmdb";
import { fetchBrowsePage } from "./actions";
import { MOVIE_GENRES, TV_GENRES, COUNTRIES, COLLECTIONS } from "@/lib/catalog";
import PosterGrid from "@/components/PosterGrid";
import BrowseControls from "@/components/BrowseControls";
import InfinitePosterGrid from "@/components/InfinitePosterGrid";
import HeroCarousel from "@/components/HeroCarousel";

export const dynamic = "force-dynamic";

export default async function BrowsePage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = (sp.media === "movie" || sp.media === "tv") ? (sp.media as MediaType) : "all";
  const label = media === "tv" ? "Semua Series" : media === "all" ? "Semua Film & Series" : "Semua Film";

  const isCategory = Boolean(sp.genre || sp.year || sp.country || sp.provider);

  const today = new Date().toISOString().slice(0, 10);
  const since = new Date(Date.now() - 180 * 86400000).toISOString().slice(0, 10);
  const since5y = `${new Date().getFullYear() - 5}-01-01`;

  const initialData = await fetchBrowsePage(sp, 1);
  const results = initialData.results;
  const initialHasMore = initialData.hasMore;

  
  const heroMedia = (!isCategory && media !== "all") ? media : undefined;
  const heroSp = { ...sp, sort_by: "terpopuler", media: heroMedia };
  const heroData = await fetchBrowsePage(heroSp, 1);
  const heroSlides = heroData.results && heroData.results.length > 0
    ? await getHeroSlides(heroData.results, 5)
    : [];

  const title = (() => {
    if (sp.collection) {
      const col = COLLECTIONS.find((x) => String(x.id) === sp.collection);
      if (col) return col.name;
    }
    if (sp.genre) {
      const g = [...MOVIE_GENRES, ...TV_GENRES].find((x) => String(x.id) === sp.genre);
      if (g) return g.name;
    }
    if (sp.country) {
      const c = COUNTRIES.find((x) => x.code === sp.country);
      if (c) return c.name;
    }
    if (sp.provider) return "Platform";
    if (sp.year) return `Year ${sp.year}`;
    return `All ${label}`;
  })();

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}

      <div className={`px-[4%] ${heroSlides.length > 0 ? "pt-8" : "pt-28"}`}>
        <div className="relative z-20 mb-6 flex flex-wrap items-center justify-between gap-4">
          <h1 className="text-2xl font-bold">{title}</h1>
          <BrowseControls defaultSort={sp.sort_by ?? "terpopuler"} showMediaTabs={isCategory} />
        </div>
        <InfinitePosterGrid 
          initialItems={results} 
          initialHasMore={initialHasMore} 
          sp={sp} 
        />
      </div>
    </main>
  );
}




