import {
  getTrendingGlobal,
  getTrendingIndonesia,
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

const ORIGINALS: SwitchGroup[] = [
  { label: "category.netflix", id: 8 },
  { label: "category.disney", id: 122 },
  { label: "category.hbo", id: 1899 },
  { label: "category.prime", id: 119 },
  { label: "category.apple", id: 350 },
].map(
  (p): SwitchGroup => ({
    label: p.label,
    sources: [
      { media: "movie", params: `provider=${p.id}&sort_by=popularity.desc` },
      { media: "tv", params: `provider=${p.id}&sort_by=popularity.desc` },
    ],
  })
);

import { MAJOR_PROVIDERS } from "@/lib/tmdb";

const REGIONS: SwitchGroup[] = [
  { label: "category.korean", sources: [{ media: "tv", params: `country=KR&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}` }] },
  { label: "category.japanese", sources: [{ media: "tv", params: `country=JP&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}` }] },
  { label: "category.chinese", sources: [{ media: "tv", params: `country=CN&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}` }] },
  { label: "category.thai", sources: [{ media: "tv", params: `country=TH&without_genres=16,10764,99,10767,10763&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}` }] },
  { label: "category.anime", sources: [{ media: "tv", params: `genre=16&country=JP&sort_by=popularity.desc&provider=${MAJOR_PROVIDERS}` }] },
];

export default async function Home() {
  const [trendingGlobal, trendingIndonesia, latestMovies, latestSeries, latestEpisodes] =
    await Promise.all([
      getTrendingGlobal(),
      getTrendingIndonesia(),
      getLatest("movie"),
      getLatest("tv"),
      getLatestEpisodes(14),
    ]);

  const heroSlides = await getHeroSlides(trendingIndonesia, 10);

  return (
    <main>
      <HeroCarousel slides={heroSlides} />

      <div className="relative z-[2] pt-4">
        <ContinueWatchingRow />
        <TrendingSection title="ui.trendingNow" items={trendingGlobal} />
        <TrendingSection
          title="ui.trendingIndonesia"
          items={trendingIndonesia}
        />
        <SwitchableCarousel groups={ORIGINALS} />
        <SwitchableCarousel groups={REGIONS} />
        <MovieRow title="ui.latestMovies" items={latestMovies} />
        <MovieRow title="ui.latestSeries" items={latestSeries} />
        <LatestEpisodesRow episodes={latestEpisodes} />
      </div>
    </main>
  );
}
