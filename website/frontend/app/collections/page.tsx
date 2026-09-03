import type { MediaType } from "@/lib/types";
import { COLLECTIONS } from "@/lib/catalog";
import { CoverflowCarousel, CoverflowSlide } from "@/components/ui/coverflow-carousel";
import { getCollectionMoviesWithLogos } from "@/lib/tmdb";
import { IMG } from "@/lib/helpers";

export default async function CollectionsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";

  if (media === "tv") {
    return (
      <main className="min-h-screen pb-16 flex items-center justify-center">
        <p className="text-white/60">Collections are only available for movies.</p>
      </main>
    );
  }

  // Fetch movies for all collections in parallel
  const collectionsData = await Promise.all(
    COLLECTIONS.map(async (collection) => {
      const movies = await getCollectionMoviesWithLogos(collection.id, 15);
      
      const slides: CoverflowSlide[] = movies.map(m => ({
        src: m.poster_path ? `${IMG}/w500${m.poster_path}` : "https://images.unsplash.com/photo-1594909122845-11baa439b7bf?q=80&w=500",
        alt: m.title || m.name || "",
        title: m.title || m.name || "",
        logoSrc: m.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
        subtitle: m.release_date ? new Date(m.release_date).getFullYear().toString() : undefined,
      }));

      return {
        ...collection,
        slides,
      };
    })
  );

  return (
    <main className="min-h-screen pb-24 overflow-x-hidden pt-28">
      {collectionsData.map((collection, idx) => {
        if (collection.slides.length === 0) return null;

        return (
          <div key={collection.id} className="mb-20">
            <h2 className="text-3xl font-bold tracking-tight text-white mb-6 px-[4%]">
              {collection.name}
            </h2>
            <CoverflowCarousel
              slides={collection.slides}
              showCaption={true}
              showNavigation={true}
              rotate={40}
              depth={0.8}
              perspective={3}
              fade={0.15}
              gap={0.08}
            />
          </div>
        );
      })}
    </main>
  );
}
