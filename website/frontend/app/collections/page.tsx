import type { MediaType } from "@/lib/types";
import { COLLECTIONS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function CollectionsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  
  const tiles = COLLECTIONS.map((c) => ({
    label: c.name,
    href: `/browse?media=movie&collection=${c.id}`,
  }));

  if (media === "tv") {
    return (
      <main className="min-h-screen pb-16 flex items-center justify-center">
        <p className="text-white/60">Collections are only available for movies.</p>
      </main>
    );
  }

  return (
    <main className="min-h-screen pb-16">
      <div className="pt-28">
        <CategoryTiles title="Collections" tiles={tiles} disableWrapper={true} />
      </div>
    </main>
  );
}
