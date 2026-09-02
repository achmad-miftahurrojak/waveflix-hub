import {
  getTrendingGlobal,
  getTrendingIndonesia,
  getTrendingKDrama,
  getLatest,
  getLatestEpisodes,
  getHeroSlides,
} from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import TrendingSection from "@/components/TrendingSection";
import MovieRow from "@/components/MovieRow";
import SwitchableCarousel, {
  type SwitchGroup,
} from "@/components/SwitchableCarousel";
import LatestEpisodesRow from "@/components/LatestEpisodesRow";
import ContinueWatchingRow from "@/components/ContinueWatchingRow";

const PLATFORMS: SwitchGroup[] = [
  { label: "Netflix", id: 8 },
  { label: "Disney+", id: 122 },
  { label: "HBO Max", id: 384 },
  { label: "Apple TV+", id: 350 },
  { label: "Prime Video", id: 119 },
  { label: "Viu", id: 158 },
].map(
  (p): SwitchGroup => ({
    label: p.label,
    sources: [
      { media: "movie", params: `provider=${p.id}&sort_by=popularity.desc` },
      { media: "tv", params: `provider=${p.id}&sort_by=popularity.desc` },
    ],
  })
);

const REGIONS: SwitchGroup[] = [
  { label: "Korean Drama", sources: [{ media: "tv", params: `country=KR&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc` }] },
  { label: "Chinese Drama", sources: [{ media: "tv", params: `country=CN&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc` }] },
  { label: "Japanese Drama", sources: [{ media: "tv", params: `country=JP&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc` }] },
  { label: "Thai Drama", sources: [{ media: "tv", params: `country=TH&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc` }] },
];

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
        <MovieRow title="Trending Now" items={trendingGlobal} showRank={true} viewAll href="/trending" />
        <MovieRow title="Trending in Indonesia" items={trendingIndonesia} />
        
        <SwitchableCarousel groups={PLATFORMS} />
        <SwitchableCarousel groups={REGIONS} />
        
        <MovieRow title="Latest Movies" items={latestMovies} viewAll href="/movies" />
        <MovieRow title="Latest Series" items={latestSeries} viewAll href="/tv" />
        <LatestEpisodesRow episodes={latestEpisodes} />
      </div>
    </main>
  );
}

