import type { MediaType } from "@/lib/types";
import { YEARS } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function YearsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const tiles = YEARS.map((y) => ({
    label: y,
    href: `/browse?media=${media}&year=${y}`,
  }));
  return <CategoryTiles title="Browse by Year" tiles={tiles} compact />;
}
