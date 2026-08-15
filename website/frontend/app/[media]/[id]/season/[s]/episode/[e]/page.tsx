import { notFound } from "next/navigation";
import Link from "next/link";
import { getDetail, getHeroLogo, getSeasonEpisodes } from "@/lib/tmdb";
import { IMG, backdropUrl, itemTitle, runtimeText } from "@/lib/helpers";
import { ChevronLeft, ChevronRight } from "@/components/Icons";
import EpisodePlayButton from "@/components/EpisodePlayButton";
import EpisodesSection from "@/components/EpisodesSection";
import InlineHeroVideo from "@/components/InlineHeroVideo";

export const dynamicParams = true;

export default async function EpisodePage({
  params,
}: {
  params: Promise<{ media: string; id: string; s: string; e: string }>;
}) {
  const { media, id, s, e } = await params;
  if (media !== "tv") notFound();

  const season = Number(s);
  const episode = Number(e);

  const detail = await getDetail("tv", id);
  if (!detail) notFound();

  const [logo, episodes] = await Promise.all([
    getHeroLogo(detail),
    getSeasonEpisodes(id, season),
  ]);
  const ep = episodes.find((x) => x.episode_number === episode);
  if (!ep) notFound();

  // Navigasi antar-episode (hanya yang sudah rilis).
  const aired = episodes.filter(
    (x) => !x.air_date || new Date(x.air_date).getTime() <= Date.now()
  );
  const idx = aired.findIndex((x) => x.episode_number === episode);
  const prev = idx > 0 ? aired[idx - 1] : null;
  const next = idx >= 0 && idx < aired.length - 1 ? aired[idx + 1] : null;
  const epHref = (n: number) => `/tv/${id}/season/${season}/episode/${n}`;

  const bg = ep.still_path
    ? `${IMG}/original${ep.still_path}`
    : backdropUrl(detail);
  const country = detail.production_countries?.[0]?.name ?? "";
  const lang = detail.original_language
    ? detail.original_language.toUpperCase()
    : "";

  return (
    <main className="min-h-screen">
      <InlineHeroVideo
        id={detail.id}
        season={season}
        episode={episode}
        backdrop={bg}
        heightClass="h-[86vh] min-h-[560px]"
      >
        <div className="relative z-[2] w-full max-w-3xl px-[4%] pb-24 pt-28">
          <Link
            href={`/tv/${id}`}
            className="mb-5 inline-flex items-center gap-1 text-sm font-medium text-white/70 transition hover:text-accent"
          >
            <ChevronLeft className="h-4 w-4" /> {itemTitle(detail)}
          </Link>

          {logo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={logo}
              alt={itemTitle(detail)}
              className="mb-4 max-h-16 w-auto max-w-[220px] object-contain object-left drop-shadow-[0_2px_10px_rgba(0,0,0,0.8)] md:max-h-20 md:max-w-[260px]"
            />
          ) : null}

          <h1 className="text-3xl font-extrabold drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)] md:text-4xl">
            Season {season} Episode {episode}
          </h1>
          <p className="mb-5 mt-1 text-lg text-white/80 drop-shadow">
            {ep.name || `Episode ${episode}`}
          </p>

          <div className="mb-6">
            <EpisodePlayButton show={detail} season={season} episode={episode} />
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-white/85 drop-shadow-[0_1px_5px_rgba(0,0,0,0.9)]">
            {ep.runtime ? (
              <>
                <span>{runtimeText(ep.runtime)}</span>
                <span className="text-white/40">&bull;</span>
              </>
            ) : null}
            {country && (
              <>
                <span>{country}</span>
                <span className="text-white/40">&bull;</span>
              </>
            )}
            {lang && <span>{lang}</span>}
            {ep.air_date && (
              <>
                <span className="text-white/40">&bull;</span>
                <span>{ep.air_date}</span>
              </>
            )}
          </div>
        </div>
      </InlineHeroVideo>

      <div className="px-[4%] py-10">
        {/* Navigasi Sebelumnya / Berikutnya */}
        {(prev || next) && (
          <div className="mb-8 grid grid-cols-2 gap-3">
            {prev ? (
              <Link
                href={epHref(prev.episode_number)}
                className="flex items-center gap-3 rounded-xl bg-white/5 p-4 ring-1 ring-white/10 transition hover:bg-white/10"
              >
                <ChevronLeft className="h-6 w-6 shrink-0 text-white/50" />
                <div className="min-w-0">
                  <div className="text-xs font-semibold uppercase tracking-wide text-white/50">
                    Previous
                  </div>
                  <div className="truncate font-semibold">
                    Ep {prev.episode_number}:{" "}
                    {prev.name || `Episode ${prev.episode_number}`}
                  </div>
                </div>
              </Link>
            ) : (
              <div />
            )}
            {next ? (
              <Link
                href={epHref(next.episode_number)}
                className="col-start-2 flex items-center justify-end gap-3 rounded-xl bg-white/5 p-4 text-right ring-1 ring-white/10 transition hover:bg-white/10"
              >
                <div className="min-w-0">
                  <div className="text-xs font-semibold uppercase tracking-wide text-accent">
                    Next
                  </div>
                  <div className="truncate font-semibold">
                    Ep {next.episode_number}:{" "}
                    {next.name || `Episode ${next.episode_number}`}
                  </div>
                </div>
                <ChevronRight className="h-6 w-6 shrink-0 text-accent" />
              </Link>
            ) : (
              <div />
            )}
          </div>
        )}

        <section className="max-w-3xl">
          <h2 className="mb-3 text-xl font-bold">Overview</h2>
          <p className="text-[15px] leading-7 text-white/75">
            {ep.overview ||
              detail.overview ||
              "No description available for this episode."}
          </p>
        </section>

        {detail.seasons && (
          <EpisodesSection
            show={detail}
            seasons={detail.seasons}
            initialSeason={season}
            currentEpisode={episode}
          />
        )}
      </div>
    </main>
  );
}
