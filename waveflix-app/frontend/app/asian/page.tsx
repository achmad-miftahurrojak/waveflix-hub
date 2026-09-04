import { discover, getHeroSlides, MAJOR_PROVIDERS } from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import MovieRow from "@/components/MovieRow";

export const dynamic = "force-dynamic";

export default async function AsianPage() {
  const without = "16,10764,99,10767,10763";
  const [kr, jp, cn, th] = await Promise.all([
    discover({
      media: "tv",
      country: "KR",
      sort_by: "popularity.desc",
      provider: MAJOR_PROVIDERS,
      without_genres: without,
    }),
    discover({
      media: "tv",
      country: "JP",
      sort_by: "popularity.desc",
      provider: MAJOR_PROVIDERS,
      without_genres: without,
    }),
    discover({
      media: "tv",
      country: "CN",
      sort_by: "popularity.desc",
      provider: MAJOR_PROVIDERS,
      without_genres: without,
    }),
    discover({
      media: "tv",
      country: "TH",
      sort_by: "popularity.desc",
      provider: MAJOR_PROVIDERS,
      without_genres: without,
    }),
  ]);

  const mix = [
    ...kr.results.slice(0, 3),
    ...jp.results.slice(0, 2),
    ...cn.results.slice(0, 2),
    ...th.results.slice(0, 1),
  ];

  const heroSlides = await getHeroSlides(mix.slice(0, 6), 6);

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}
      <div className="relative z-[2] pt-4">
        <MovieRow title="category.korean" items={kr.results} />
        <MovieRow title="category.japanese" items={jp.results} />
        <MovieRow title="category.chinese" items={cn.results} />
        <MovieRow title="category.thai" items={th.results} />
      </div>
    </main>
  );
}
