import type { MediaType } from "@/lib/types";
import { COLLECTIONS } from "@/lib/catalog";
import { CoverflowCarousel, CoverflowSlide } from "@/components/ui/coverflow-carousel";
import { getCollectionMoviesWithLogos } from "@/lib/tmdb";
import { IMG } from "@/lib/helpers";
import CategoryTiles from "@/components/CategoryTiles";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";

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

  // If no specific collection is selected, show the grid of cards
  if (!sp.c) {
    const tiles = COLLECTIONS.map((c) => ({
      label: c.name,
      href: `/collections?c=${c.id}`, // Link to this same page but with 'c' param
    }));

    return (
      <main className="min-h-screen pb-16">
        <div className="pt-28">
          <CategoryTiles title="Collections" tiles={tiles} disableWrapper={true} />
        </div>
      </main>
    );
  }

  // A collection is selected, show the Coverflow Carousel
  const currentCollectionId = sp.c;
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
      <div className="px-[4%] mb-10 flex items-center gap-4">
        <Link href="/collections" className="p-2 rounded-full bg-white/10 hover:bg-white/20 transition text-white">
          <ArrowLeft className="size-5" />
        </Link>
        <h1 className="text-3xl font-black">{currentCollection.name}</h1>
      </div>

      {slides.length > 0 && (
        <div key={currentCollection.id} className="animate-in fade-in zoom-in-95 duration-500">
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
