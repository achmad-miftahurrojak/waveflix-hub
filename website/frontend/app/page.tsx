import {
  getTrendingGlobal,
  getTrendingIndonesia,
  getLatest,
  getLatestEpisodes,
  getHeroSlides,
} from "@/lib/tmdb";
import { mediaTypeOf } from "@/lib/helpers";
import HeroCarousel from "@/components/HeroCarousel";
import TrendingSection from "@/components/TrendingSection";
import MovieRow from "@/components/MovieRow";
import SwitchableCarousel, {
  type SwitchGroup,
} from "@/components/SwitchableCarousel";
import LatestEpisodesRow from "@/components/LatestEpisodesRow";

// Section "Originals" — switch antar platform besar.
const ORIGINALS: SwitchGroup[] = [
  { label: "Netflix Originals", id: 8 },
  { label: "Disney+ Originals", id: 122 },
  { label: "HBO Originals", id: 1899 },
  { label: "Prime Video Originals", id: 119 },
  { label: "Apple TV+ Originals", id: 350 },
].map(
  (p): SwitchGroup => ({
    label: p.label,
    sources: [
      { media: "movie", params: `provider=${p.id}&sort_by=popularity.desc&min_votes=50` },
      { media: "tv", params: `provider=${p.id}&sort_by=popularity.desc&min_votes=50` },
    ],
  })
);

// Section "Drama by country" — switch antar negara/anime.
const REGIONS: SwitchGroup[] = [
  { label: "Korean Drama", sources: [{ media: "tv", params: "country=KR&sort_by=popularity.desc&min_votes=10" }] },
  { label: "Japanese Drama", sources: [{ media: "tv", params: "country=JP&sort_by=popularity.desc&min_votes=5" }] },
  { label: "Chinese Drama", sources: [{ media: "tv", params: "country=CN&sort_by=popularity.desc&min_votes=5" }] },
  { label: "Thai Drama", sources: [{ media: "tv", params: "country=TH&sort_by=popularity.desc&min_votes=3" }] },
  { label: "Anime", sources: [{ media: "tv", params: "genre=16&country=JP&sort_by=popularity.desc&min_votes=20" }] },
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

  // Hero: 10 trending global lalu 10 trending Indonesia (dedupe), ganti tiap 10 dtk.
  const globalSlides = await getHeroSlides(trendingGlobal, 10);
  const gKeys = new Set(
    globalSlides.map((s) => `${mediaTypeOf(s.item)}-${s.item.id}`)
  );
  const indoSource = trendingIndonesia.filter(
    (m) => !gKeys.has(`${mediaTypeOf(m)}-${m.id}`)
  );
  const indoSlides = await getHeroSlides(indoSource, 10);
  const heroSlides = [...globalSlides, ...indoSlides];

  return (
    <main>
      <HeroCarousel slides={heroSlides} />

      <div className="relative z-[2] pt-4">
        <TrendingSection title="Trending Now" items={trendingGlobal} />
        <TrendingSection
          title="Trending in Indonesia"
          items={trendingIndonesia}
        />
        <SwitchableCarousel groups={ORIGINALS} />
        <SwitchableCarousel groups={REGIONS} />
        <MovieRow title="Latest Movies" items={latestMovies} />
        <MovieRow title="Latest Series" items={latestSeries} />
        <LatestEpisodesRow episodes={latestEpisodes} />
      </div>
    </main>
  );
}
