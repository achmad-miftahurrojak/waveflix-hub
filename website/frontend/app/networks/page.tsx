import type { MediaType } from "@/lib/types";
import { PROVIDERS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";
import HeroCarousel from "@/components/HeroCarousel";
import { getTrendingGlobal, getHeroSlides } from "@/lib/tmdb";

export default async function NetworksPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const tiles = PROVIDERS.map((p) => ({
    label: p.name,
    href: `/browse?media=${media}&provider=${p.id}`,
  }));

  const trending = await getTrendingGlobal();
  const heroSlides = await getHeroSlides(trending, 5);

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}
      <div className={heroSlides.length > 0 ? "pt-8" : "pt-28"}>
        <CategoryTiles title="Networks" tiles={tiles} disableWrapper={true} />
      </div>
    </main>
  );
}
