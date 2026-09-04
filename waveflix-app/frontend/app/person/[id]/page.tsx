import { notFound } from "next/navigation";
import { getPerson } from "@/lib/tmdb";
import { profileUrl } from "@/lib/helpers";
import Navbar from "@/components/Navbar";
import PersonDetailsTabs from "@/components/PersonDetailsTabs";

export const dynamicParams = true;

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const person = await getPerson(id);
  if (!person) return { title: "Person Not Found - Waveflix" };
  const name = person.original_name || person.name;
  return {
    title: `${name} - Waveflix`,
    description: person.biography || `Profile of ${name} on Waveflix.`,
  };
}

export default async function PersonPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const person = await getPerson(id);

  if (!person) notFound();

  const castCredits = person.combined_credits?.cast ?? [];

  const seen = new Set<string>();
  const filmography = castCredits
    .filter((c) => {
      const uniqueId = `${c.media_type}-${c.id}`;
      if (!c.poster_path || seen.has(uniqueId)) return false;
      seen.add(uniqueId);
      return true;
    })
    .sort((a, b) => (b.popularity ?? 0) - (a.popularity ?? 0))
    .slice(0, 20);

  return (
    <main className="min-h-screen bg-bg">
      <Navbar />

      <div className="pt-24 px-[4%] max-w-7xl mx-auto pb-20">
        <div className="flex flex-col md:flex-row gap-6 md:gap-12 mb-12">
          <div className="shrink-0 w-full md:w-[280px]">
            <img
              src={profileUrl(person.profile_path, "h632")}
              alt={person.original_name || person.name}
              className="w-full rounded-xl object-cover shadow-lg"
            />
          </div>

          <div className="flex-1 min-w-0">
            <h1 className="text-3xl md:text-5xl font-bold text-white mb-6">
              {person.original_name || person.name}
            </h1>

            <PersonDetailsTabs
              biography={person.biography || ""}
              knownForDept={person.known_for_department || ""}
              birthday={person.birthday || null}
              placeOfBirth={person.place_of_birth || null}
              filmography={filmography}
            />
          </div>
        </div>
      </div>
    </main>
  );
}
