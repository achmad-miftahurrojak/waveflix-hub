import type { MediaType } from "@/lib/types";
import { COUNTRIES } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function CountriesPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = (sp.media === "movie" || sp.media === "tv") ? (sp.media as MediaType) : "all";
  const tiles = COUNTRIES.map((c) => ({
    label: c.name,
    href: `/browse?media=${media}&country=${c.code}`,
  }));



  return (
    <main className="min-h-screen pb-16">

      <div className="pt-28">
        <CategoryTiles title="Countries" tiles={tiles} disableWrapper={true} />
      </div>
    </main>
  );
}



