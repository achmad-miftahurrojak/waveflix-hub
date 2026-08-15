import type { MediaType } from "@/lib/types";
import { discoverMany, getHeroSlides } from "@/lib/tmdb";
import { MOVIE_GENRES, TV_GENRES, COUNTRIES, PROVIDERS } from "@/lib/catalog";
import PosterGrid from "@/components/PosterGrid";
import HeroCarousel from "@/components/HeroCarousel";

export const dynamic = "force-dynamic";

export default async function BrowsePage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const label = media === "tv" ? "Series" : "Movies";

  // Kategori = filter yang tidak terikat tipe media → tampilkan Film + Series bareng.
  const isCategory = Boolean(sp.genre || sp.year || sp.country || sp.provider);

  const today = new Date().toISOString().slice(0, 10);
  const since = new Date(Date.now() - 180 * 86400000).toISOString().slice(0, 10);
  const since5y = `${new Date().getFullYear() - 5}-01-01`;

  // Batasi ke platform legal besar yang tersedia di Indonesia (Netflix, Disney+,
  // Prime, Apple TV+, HBO Max, Viu, Vidio). Menyaring "film bodong"/short film
  // yang tidak ada di platform mana pun, sekaligus jaga katalog tetap resmi.
  const MAJOR_PROVIDERS = "8|122|119|350|1899|158|489";

  let results;
  let heroPool;
  if (isCategory) {
    // Network (provider) → urut POPULER supaya katalog yang dikenal muncul
    // (Disney = Marvel/Pixar dst), bukan tambahan terbaru acak. Kategori lain
    // (negara/genre/tahun) → urut TERBARU biar rilisan baru langsung muncul.
    const isNetwork = Boolean(sp.provider);
    const noDate = Boolean(sp.year) || isNetwork;
    const base = {
      genre: sp.genre,
      year: sp.year,
      country: sp.country,
      // Network → platform spesifik; kategori lain → dibatasi ke platform besar.
      provider: isNetwork ? sp.provider : MAJOR_PROVIDERS,
      released_after: noDate ? undefined : since5y,
      released_before: noDate ? undefined : today,
      min_votes: isNetwork ? "50" : "1",
    };
    const [mv, tvr] = await Promise.all([
      discoverMany(
        {
          ...base,
          media: "movie",
          sort_by: isNetwork ? "popularity.desc" : "primary_release_date.desc",
        },
        5
      ),
      discoverMany(
        {
          ...base,
          media: "tv",
          sort_by: isNetwork ? "popularity.desc" : "first_air_date.desc",
        },
        5
      ),
    ]);
    const merged = [...mv.results, ...tvr.results];
    if (isNetwork) {
      results = [...merged]
        .sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0))
        .slice(0, 200);
      heroPool = results;
    } else {
      const dateVal = (m: (typeof merged)[number]) =>
        new Date(m.release_date || m.first_air_date || 0).getTime();
      results = [...merged].sort((a, b) => dateVal(b) - dateVal(a)).slice(0, 200);
      heroPool = [...merged].sort(
        (a, b) => (b.popularity ?? 0) - (a.popularity ?? 0)
      );
    }
  } else {
    // Listing polos (Movies/Series) atau sort-only (Newly Added): satu tipe media,
    // tetap dibatasi ke platform besar biar tidak ada judul "bodong".
    const params = {
      media,
      provider: MAJOR_PROVIDERS,
      sort_by: sp.sort_by ?? "popularity.desc",
      ...(sp.sort_by ? {} : { released_after: since, min_votes: "20" }),
    };
    const data = await discoverMany(params, 10);
    results = data.results;
    heroPool = data.results;
  }

  // Hero selalu ada — dari judul populer-terkini sesuai konteks filter.
  const heroSlides = await getHeroSlides(heroPool, 5);

  // Judul kontekstual ala IDLIX.
  const title = (() => {
    if (sp.genre) {
      const g = [...MOVIE_GENRES, ...TV_GENRES].find(
        (x) => String(x.id) === sp.genre
      );
      if (g) return g.name;
    }
    if (sp.country) {
      const c = COUNTRIES.find((x) => x.code === sp.country);
      if (c) return c.name;
    }
    if (sp.provider) {
      const p = PROVIDERS.find((x) => String(x.id) === sp.provider);
      if (p) return p.name;
    }
    if (sp.year) return `Year ${sp.year}`;
    return `All ${label}`;
  })();

  return (
    <main className="min-h-screen pb-16">
      {heroSlides.length > 0 && <HeroCarousel slides={heroSlides} />}

      <div className={`px-[4%] ${heroSlides.length > 0 ? "pt-8" : "pt-28"}`}>
        <h1 className="mb-6 text-2xl font-bold">{title}</h1>
        <PosterGrid items={results} />
      </div>
    </main>
  );
}
