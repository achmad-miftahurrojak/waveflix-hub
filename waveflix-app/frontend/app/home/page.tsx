import {
  getTrendingGlobal,
  getTrendingIndonesia,
  getTrendingKDrama,
  getLatest,
  getLatestEpisodes,
  getHeroSlides,
} from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import MovieRow from "@/components/MovieRow";
import PlatformRow from "@/components/PlatformRow";
import AsianDramaRow from "@/components/AsianDramaRow";
import EpisodeRow from "@/components/EpisodeRow";

export default async function Home() {
  const [trendingGlobal, trendingIndonesia, trendingKDrama, latestMovies, latestSeries, latestEpisodes] =
    await Promise.all([
      getTrendingGlobal(),
      getTrendingIndonesia(),
      getTrendingKDrama(),
      getLatest("movie"),
      getLatest("tv"),
      getLatestEpisodes(14),
    ]);

  const combinedTrending = [...trendingIndonesia];
  for (const item of trendingKDrama) {
    if (!combinedTrending.find((x) => x.id === item.id)) {
      combinedTrending.push(item);
    }
  }

  const heroSlides = await getHeroSlides(combinedTrending, 15);

  return (
    <main>
      <HeroCarousel slides={heroSlides} />

      <div className="relative z-[2] pt-4">
        <MovieRow title="Trending Now" items={trendingGlobal} href="/browse?sort_by=terpopuler" />
        <MovieRow title="Trending in Indonesia" items={combinedTrending} href="/browse?country=ID&sort_by=terpopuler" />
        <PlatformRow />
        <AsianDramaRow />
        <MovieRow title="Latest Movies" items={latestMovies} href="/browse?media=movie&sort_by=terbaru" />
        <MovieRow title="Latest Series" items={latestSeries} href="/browse?media=tv&sort_by=terbaru" />
        <EpisodeRow episodes={latestEpisodes} />
      </div>
    </main>
  );
}