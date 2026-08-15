import type { MediaType } from "@/lib/types";
import { PROVIDERS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function NetworksPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const tiles = PROVIDERS.map((p) => ({
    label: p.name,
    href: `/browse?media=${media}&provider=${p.id}`,
  }));
  return <CategoryTiles title="Networks" tiles={tiles} />;
}
