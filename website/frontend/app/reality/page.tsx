import { discover, getHeroSlides, MAJOR_PROVIDERS } from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import InfiniteGrid from "@/components/InfiniteGrid";

export const dynamic = "force-dynamic";

// Reality/variety show Korea saja: TV, genre Reality(10764)|Talk(10767), origin KR.
export default async function RealityPage() {
  const today = new Date().toISOString().slice(0, 10);

  const gridParams: Record<string, string> = {
    media: "tv",
    genre: "10764|10767",
    country: "KR",
    sort_by: "popularity.desc",
    min_votes: "1", // Penting: Jangan dihapus! TMDB mem-filter min 30 votes secara default, padahal Variety Korea votes-nya sedikit.
    // Sengaja provider dilepas agar mencakup semua Variety Korea (ratusan halaman)
  };

  const [heroData, gridData] = await Promise.all([
    // Hero = yang BARU rilis (bukan yang sudah punya nama seperti Running Man).
    discover({
      media: "tv",
      genre: "10764|10767",
      country: "KR",
      sort_by: "first_air_date.desc",
      released_before: today,
      provider: MAJOR_PROVIDERS, // Hero tetap dibatasi provider besar agar poster HD
      min_votes: "1",
    }),
    discover({
      media: "tv",
      genre: "10764|10767",
      country: "KR",
      sort_by: "popularity.desc",
      min_votes: "1",
      page: 1,
    }),
  ]);

  const heroSlides = await getHeroSlides(heroData.results, 6);

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}

      <div className={`px-[4%] ${heroSlides.length > 0 ? "pt-8" : "pt-28"}`}>
        <h1 className="mb-2 text-2xl font-bold">Reality</h1>
        <p className="mb-6 text-sm text-white/55">
          Korean variety &amp; reality shows.
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
