import { searchMulti } from "@/lib/tmdb";
import PosterGrid from "@/components/PosterGrid";

export const dynamic = "force-dynamic";

export default async function SearchPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const { q } = await searchParams;
  const query = (q ?? "").trim();
  const results = query ? await searchMulti(query) : [];

  return (
    <main className="min-h-screen px-[4%] pb-16 pt-28">
      <h1 className="mb-6 text-2xl font-bold">
        {query ? (
          <>
            Hasil pencarian:{" "}
            <span className="text-accent">&ldquo;{query}&rdquo;</span>
          </>
        ) : (
          "Cari film atau series"
        )}
      </h1>
      {query ? (
        <PosterGrid items={results} />
      ) : (
        <p className="py-16 text-center text-white/50">
          Ketik kata kunci di kolom pencarian untuk mulai mencari.
        </p>
      )}
    </main>
  );
}
