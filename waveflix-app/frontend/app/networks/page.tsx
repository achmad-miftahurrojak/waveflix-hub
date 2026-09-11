import type { MediaType } from "@/lib/types";
import { PROVIDERS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function NetworksPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = (sp.media === "movie" || sp.media === "tv") ? (sp.media as MediaType) : "all";
  const tiles = PROVIDERS.map((p) => ({
    label: p.name,
    href: `/browse?media=${media}&provider=${p.id}`,
    logo: p.logo,
  }));

  return (
    <main className="min-h-screen pb-16">
      <div className="pt-28">
        <CategoryTiles title="Networks" tiles={tiles} disableWrapper={true} />
      </div>
    </main>
  );
}
