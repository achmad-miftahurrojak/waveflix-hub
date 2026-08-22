import { notFound } from "next/navigation";
import type { MediaType, TmdbItem } from "@/lib/types";
import { getDetail, getHeroLogo, discover } from "@/lib/tmdb";
import {
  backdropUrl,
  itemTitle,
  itemYear,
  ratingText,
  durationText,
  genreNames,
  statusLabel,
  isTv,
} from "@/lib/helpers";
import { StarIcon } from "@/components/Icons";
import DetailActions from "@/components/DetailActions";
import CastRow from "@/components/CastRow";
import EpisodesSection from "@/components/EpisodesSection";
import MovieRow from "@/components/MovieRow";
import InlineHeroVideo from "@/components/InlineHeroVideo";

export const dynamicParams = true;

export default async function DetailPage({
  params,
}: {
  params: Promise<{ media: string; id: string }>;
}) {
  const { media, id } = await params;
  if (media !== "movie" && media !== "tv") notFound();

  const detail = await getDetail(media as MediaType, id);
  if (!detail) notFound();

  const logo = await getHeroLogo(detail);
  const tv = isTv(detail);
  const genres = genreNames(detail);
  const cast = detail.credits?.cast ?? [];

  // Meta ala IDLIX
  const country = detail.production_countries?.[0]?.name ?? "";
  const lang = detail.original_language
    ? detail.original_language.toUpperCase()
    : "";
  const status = statusLabel(detail);
  const creators = tv
    ? (detail.created_by ?? []).map((c) => c.name)
    : (detail.credits?.crew ?? [])
        .filter((c) => c.job === "Director")
        .map((c) => c.name);
  const creatorLabel = tv ? "Creator" : "Director";
  const studios = (detail.production_companies ?? []).filter(
    (s) => s.logo_path
  );

  // "More Like This": tipe sama (rekomendasi TMDB memang setipe) + negara sama
  // (didekati lewat bahasa asli) supaya nonton Korea tidak dikasih film barat.
  // Animasi/anime dikecualikan dari batasan negara.
  const isAnimation = (detail.genres ?? []).some((g) => g.id === 16);
  const originCountry = detail.production_countries?.[0]?.iso_3166_1;
  const similarSeen = new Set<number>([detail.id]);
  
  let similar: TmdbItem[] = [];

  if (!isAnimation && originCountry) {
    const discoverData = await discover({ media: media as MediaType, country: originCountry, sort_by: "popularity.desc" });
    similar = (discoverData.results || [])
      .filter((m) => {
        if (!m.poster_path || similarSeen.has(m.id)) return false;
        similarSeen.add(m.id);
        return true;
      })
      .map((m) => ({ ...m, media_type: media }))
      .slice(0, 14);
  }

  // Fallback if not enough similar items or it's animation
  if (similar.length < 14) {
    const detailLang = detail.original_language;
    const fallbackList = [
      ...(detail.recommendations?.results ?? []),
      ...(detail.similar?.results ?? []),
    ]
      .filter((m) => {
        if (!m.poster_path || similarSeen.has(m.id)) return false;
        if (
          !isAnimation &&
          detailLang &&
          m.original_language &&
          m.original_language !== detailLang
        )
          return false;
        similarSeen.add(m.id);
        return true;
      })
      .map((m) => ({ ...m, media_type: media }));
    
    similar = [...similar, ...fallbackList].slice(0, 14);
  }

  const vids = detail.videos?.results ?? [];
  const yt =
    vids.find((v) => v.site === "YouTube" && v.type === "Trailer") ??
    vids.find((v) => v.site === "YouTube" && v.type === "Teaser") ??
    vids.find((v) => v.site === "YouTube");
  const trailerKey = yt?.key ?? null;

  const Dot = () => <span className="text-white/40">&bull;</span>;

  return (
    <main className="min-h-screen">
      {/* ===== BAGIAN 1: HERO (judul · genre · tombol · meta) ala IDLIX ===== */}
      <InlineHeroVideo
        id={detail.id}
        backdrop={backdropUrl(detail)}
        heightClass="h-screen min-h-[600px]"
        trailer={trailerKey}
      >
        <div className="relative z-[2] w-full max-w-3xl px-[4%] pb-24">
          {logo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={logo}
              alt={itemTitle(detail)}
              className="mb-4 max-h-20 w-auto max-w-[260px] object-contain object-left drop-shadow-[0_2px_10px_rgba(0,0,0,0.8)] md:max-h-28 md:max-w-[340px]"
            />
          ) : (
            <h1 className="mb-4 text-4xl font-extrabold drop-shadow-[0_2px_10px_rgba(0,0,0,0.9)] md:text-5xl">
              {itemTitle(detail)}
            </h1>
          )}

          {/* Genre inline (dot-separated) */}
          {genres.length > 0 && (
            <p className="mb-5 text-base font-medium text-white/85 drop-shadow md:text-lg">
              {genres.join("  ·  ")}
            </p>
          )}

          {/* Tombol aksi */}
          <DetailActions item={detail} />

          {/* Meta: rating · tahun · durasi/musim · negara · bahasa · status */}
          <div className="mt-6 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-white/85 drop-shadow-[0_1px_5px_rgba(0,0,0,0.9)]">
            <span className="flex items-center gap-1 font-semibold text-[#f5c518]">
              <StarIcon /> {ratingText(detail)}
            </span>
            <Dot />
            <span>{itemYear(detail)}</span>
            {durationText(detail) && (
              <>
                <Dot />
                <span>{durationText(detail)}</span>
              </>
            )}
            {country && (
              <>
                <Dot />
                <span>{country}</span>
              </>
            )}
            {lang && (
              <>
                <Dot />
                <span>{lang}</span>
              </>
            )}
            {status && (
              <span className="rounded border border-emerald-500/70 px-2 py-0.5 text-xs font-bold text-emerald-400">
                {status}
              </span>
            )}
          </div>
        </div>
      </InlineHeroVideo>

      {/* ===== BAGIAN 2: DESKRIPSI (creator · tagline · overview · studio) ===== */}
      <div className="px-[4%] py-10">
        <section className="max-w-3xl">
          {creators.length > 0 && (
            <p className="mb-3 text-sm text-white/55">
              {creatorLabel}:{" "}
              <span className="font-semibold text-white/90">
                {creators.join(", ")}
              </span>
            </p>
          )}

          {detail.tagline && (
            <p className="mb-4 text-base italic text-white/60">
              {detail.tagline}
            </p>
          )}

          <p className="max-w-2xl text-[15px] leading-7 text-white/75">
            {detail.overview || "No description available for this title."}
          </p>

          {studios.length > 0 && (
            <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-3">
              {studios.map((s) => (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  key={s.id}
                  src={`https://image.tmdb.org/t/p/w200${s.logo_path}`}
                  alt={s.name}
                  title={s.name}
                  className="h-5 w-auto object-contain opacity-60 transition-all duration-300 hover:scale-110 hover:opacity-100 hover:drop-shadow-[0_0_8px_rgba(255,255,255,0.5)] [filter:brightness(0)_invert(1)] md:h-6 cursor-pointer"
                />
              ))}
            </div>
          )}
        </section>

        {tv && detail.seasons && (
          <EpisodesSection show={detail} seasons={detail.seasons} />
        )}

        <CastRow cast={cast} />
      </div>

      {similar.length > 0 && (
        <div className="pb-4">
          <MovieRow title="More Like This" items={similar} />
        </div>
      )}
    </main>
  );
}
