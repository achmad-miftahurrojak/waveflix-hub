import {
  getTrendingGlobal,
  getTrendingIndonesia,
  discoverByProvider,
  getRecent,
  getHeroSlides,
} from "@/lib/tmdb";
import HeroCarousel from "@/components/HeroCarousel";
import TrendingSection from "@/components/TrendingSection";
import MovieRow from "@/components/MovieRow";
import { mediaTypeOf } from "@/lib/helpers";

// Provider ID TMDB (watch_region=ID).
const NETFLIX = 8;
const DISNEY = 122; // Disney+ Hotstar (region ID)
const APPLE_TV = 350;
const HBO_MAX = 1899;

export default async function Home() {
  const [
    trendingGlobal,
    trendingIndonesia,
    netflix,
    disney,
    apple,
    hbo,
    recentMovies,
    recentSeries,
  ] = await Promise.all([
    getTrendingGlobal(),
    getTrendingIndonesia(),
    discoverByProvider(NETFLIX, "movie"),
    discoverByProvider(DISNEY, "movie"),
    discoverByProvider(APPLE_TV, "tv"),
    discoverByProvider(HBO_MAX, "movie"),
    getRecent("movie"),
    getRecent("tv"),
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

      {/* Hero penuh 1 layar; baris konten mulai di bawahnya (scroll untuk lihat) */}
      <div className="relative z-[2] pt-4">
        <TrendingSection items={trendingGlobal} title="Trending Now" />
        <TrendingSection
          items={trendingIndonesia}
          title="Trending in Indonesia"
        />
        <MovieRow
          title="Netflix"
          items={netflix}
          viewAll
          href={`/browse?media=movie&provider=${NETFLIX}`}
        />
        <MovieRow
          title="Disney+"
          items={disney}
          viewAll
          href={`/browse?media=movie&provider=${DISNEY}`}
        />
        <MovieRow
          title="Apple TV+ Originals"
          items={apple}
          viewAll
          href={`/browse?media=tv&provider=${APPLE_TV}`}
        />
        <MovieRow
          title="HBO Max"
          items={hbo}
          viewAll
          href={`/browse?media=movie&provider=${HBO_MAX}`}
        />
        <MovieRow
          title="Newly Added Movies"
          items={recentMovies}
          viewAll
          href="/browse?media=movie&sort_by=primary_release_date.desc"
        />
        <MovieRow
          title="Newly Added Series"
          items={recentSeries}
          viewAll
          href="/browse?media=tv&sort_by=first_air_date.desc"
        />
      </div>
    </main>
  );
}
