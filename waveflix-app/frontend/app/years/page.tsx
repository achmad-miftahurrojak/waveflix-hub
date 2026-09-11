import type { MediaType } from "@/lib/types";
import { YEARS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function YearsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = (sp.media === "movie" || sp.media === "tv") ? (sp.media as MediaType) : "all";
  const tiles = YEARS.map((y) => ({
    label: y,
    href: `/browse?media=${media}&year=${y}`,
  }));



  return (
    <main className="min-h-screen pb-16">

      <div className="pt-28">
        <CategoryTiles title="Browse by Year" tiles={tiles} compact disableWrapper={true} />
      </div>
    </main>
  );
}



