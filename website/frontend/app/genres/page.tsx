import type { MediaType } from "@/lib/types";
import { genresFor } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function GenresPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = sp.media === "tv" ? "tv" : "movie";
  const tiles = genresFor(media).map((g) => ({
    label: g.name,
    href: `/browse?media=${media}&genre=${g.id}`,
  }));
  return <CategoryTiles title="Genres" tiles={tiles} />;
}