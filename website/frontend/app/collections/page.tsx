import type { MediaType } from "@/lib/types";
import { COLLECTIONS } from "@/lib/catalog";
import { CoverflowCarousel, CoverflowSlide } from "@/components/ui/coverflow-carousel";
import { getCollectionMoviesWithLogos } from "@/lib/tmdb";
import { IMG } from "@/lib/helpers";
import Link from "next/link";
import { cn } from "@/lib/utils";

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

  const currentCollectionId = sp.c || COLLECTIONS[0].id;
  const currentCollection = COLLECTIONS.find((c) => c.id === currentCollectionId) || COLLECTIONS[0];

  const movies = await getCollectionMoviesWithLogos(currentCollection.id, 20);
  
  const slides: CoverflowSlide[] = movies.map(m => ({
    src: m.poster_path ? `${IMG}/w500${m.poster_path}` : "https://images.unsplash.com/photo-1594909122845-11baa439b7bf?q=80&w=500",
    alt: m.title || m.name || "",
    title: m.title || m.name || "",
    logoSrc: m.logo_path ? `${IMG}/w500${m.logo_path}` : undefined,
    subtitle: m.release_date ? new Date(m.release_date).getFullYear().toString() : undefined,
  }));

  return (
    <main className="min-h-screen pb-24 overflow-x-hidden pt-28">
      {/* Collection Selector */}
      <div className="mb-12 px-[4%]">
        <h1 className="text-3xl font-black mb-6">Collections</h1>
        <div className="flex gap-3 overflow-x-auto scrollbar-hide pb-2">
          {COLLECTIONS.map(c => (
            <Link 
              key={c.id} 
              href={`/collections?c=${c.id}`} 
              className={cn(
                "px-5 py-2.5 rounded-full whitespace-nowrap text-sm font-bold transition",
                currentCollectionId === c.id 
                  ? "bg-accent text-black shadow-lg shadow-accent/20" 
                  : "bg-white/10 text-white hover:bg-white/20"
              )}
            >
              {c.name}
            </Link>
          ))}
        </div>
      </div>

      {slides.length > 0 && (
        <div key={currentCollection.id} className="animate-in fade-in slide-in-from-bottom-10 duration-500">
          <CoverflowCarousel
            slides={slides}
            showCaption={true}
            showNavigation={true}
            rotate={40}
            depth={0.8}
            perspective={3}
            fade={0.15}
            gap={0.08}
          />
        </div>
      )}
    </main>
  );
}
