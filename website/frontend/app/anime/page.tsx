import { discover, getHeroSlides, MAJOR_PROVIDERS } from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import InfiniteGrid from "@/components/InfiniteGrid";

export const dynamic = "force-dynamic";

export default async function AnimePage() {
  const today = new Date().toISOString().slice(0, 10);

  const gridParams: Record<string, string> = {
    media: "tv",
    genre: "16",
    country: "JP",
    sort_by: "popularity.desc",
  };

  const [heroData, gridData] = await Promise.all([
    discover({
      media: "tv",
      genre: "16",
      country: "JP",
      sort_by: "popularity.desc", // Populer untuk hero, bukan cuma yang baru rilis agar bagus posternya
      released_before: today,
      provider: MAJOR_PROVIDERS,
    }),
    discover({
      media: "tv",
      genre: "16",
      country: "JP",
      sort_by: "popularity.desc",
      page: 1,
    }),
  ]);

  const heroSlides = await getHeroSlides(heroData.results, 6);

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}

      <div className={`px-[4%] ${heroSlides.length > 0 ? "pt-8" : "pt-28"}`}>
        <h1 className="mb-2 text-2xl font-bold">Anime</h1>
        <p className="mb-6 text-sm text-white/55">
          Japanese animation series.
        </p>
        <InfiniteGrid
          params={gridParams}
          media="tv"
          initial={gridData.results}
          totalPages={gridData.total_pages ?? 1}
        />
      </div>
    </main>
  );
}
