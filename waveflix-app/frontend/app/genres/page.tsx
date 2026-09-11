import type { MediaType } from "@/lib/types";
import { genresFor } from "@/lib/catalog";
import CategoryTiles from "@/components/CategoryTiles";

export default async function GenresPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | undefined>>;
}) {
  const sp = await searchParams;
  const media: MediaType = (sp.media === "tv" || sp.media === "all") ? (sp.media as MediaType) : "movie";
  const G = media === "all" ? [...MOVIE_GENRES, ...TV_GENRES].filter((v,i,a)=>a.findIndex(t=>(t.id === v.id))===i) : media === "tv" ? TV_GENRES : MOVIE_GENRES;
  const tiles = G.map((g) => ({
    label: g.name,
    href: `/browse?media=${media}&genre=${g.id}`,
  }));



  return (
    <main className="min-h-screen pb-16">

      <div className="pt-28">
        <CategoryTiles title="Genres" tiles={tiles} disableWrapper={true} />
      </div>
    </main>
  );
}


