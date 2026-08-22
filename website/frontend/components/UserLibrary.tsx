"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { TmdbItem } from "@/lib/types";
import { useAuth } from "./AuthProvider";
import PosterGrid from "./PosterGrid";
import HeroCarousel from "./HeroCarousel";
import { mediaTypeOf, durationText, genreNames, statusLabel } from "@/lib/helpers";
import { useTranslation } from "@/lib/i18n";

// Tipe data yang sama dengan HeroSlide di tmdb.ts
interface ClientHeroSlide {
  item: TmdbItem;
  logo: string | null;
  overview: string;
  tagline: string;
  genres: string[];
  duration: string;
  status: string;
  trailer: string | null;
}

function toItem(r: any): TmdbItem {
  return {
    id: r.tmdb_id ?? r.id,
    title: r.title,
    name: r.title,
    media_type: r.media_type,
    poster_path: r.poster_path,
    vote_average: r.vote_average,
    first_air_date: r.media_type === "tv" ? " " : undefined,
  };
}

interface Props {
  title: string;
  endpoint: "/api/watchlist" | "/api/favorites" | "/api/history";
  /** Untuk watchlist: tampilkan localStorage saat belum login. */
  localFallbackKey?: string;
}

export default function UserLibrary({ title, endpoint, localFallbackKey }: Props) {
  const { t } = useTranslation();
  const { user, ready, authFetch } = useAuth();
  const [items, setItems] = useState<TmdbItem[] | null>(null);
  const [heroSlides, setHeroSlides] = useState<ClientHeroSlide[]>([]);

  useEffect(() => {
    if (!ready) return;

    const fetchHeroDetails = async (listItems: TmdbItem[]) => {
      // Ambil maksimal 5 item teratas untuk hero
      const top5 = listItems.slice(0, 5);
      
      const slides = await Promise.all(
        top5.map(async (item) => {
          const media = mediaTypeOf(item);
          try {
            const [detailRes, imagesRes] = await Promise.all([
              authFetch(`/api/detail?media=${media}&id=${item.id}`),
              authFetch(`/api/images?media=${media}&id=${item.id}`)
            ]);
            
            const detail = await detailRes.json();
            const images = await imagesRes.json();

            const enLogo = images.logos?.find((l: any) => l.iso_639_1 === "en");
            const anyLogo = images.logos?.length > 0 ? images.logos[0] : null;
            const logo = enLogo || anyLogo;

            const tr = detail.videos?.results?.find(
              (v: any) => v.type === "Trailer" && v.site === "YouTube"
            );

            // Karena data list mungkin cuma punya poster, kita update backdrop-nya dari detail
            const itemWithBackdrop = {
              ...item,
              backdrop_path: detail.backdrop_path || item.backdrop_path,
            };

            return {
              item: itemWithBackdrop,
              logo: logo ? `https://image.tmdb.org/t/p/w500${logo.file_path}` : null,
              overview: detail.overview || "",
              tagline: detail.tagline || "",
              genres: genreNames(detail),
              duration: durationText(detail),
              status: statusLabel(detail),
              trailer: tr ? tr.key : null,
            } as ClientHeroSlide;
          } catch (e) {
            console.error("Gagal ambil detail hero", e);
            return null;
          }
        })
      );

      // Filter out yang gagal
      setHeroSlides(slides.filter((s): s is ClientHeroSlide => s !== null));
    };

    if (user) {
      authFetch(endpoint)
        .then((r) => r.json())
        .then((d) => {
          const list = (d.results ?? []).map(toItem);
          setItems(list);
          fetchHeroDetails(list);
        })
        .catch(() => setItems([]));
    } else if (localFallbackKey) {
      try {
        const local = JSON.parse(localStorage.getItem(localFallbackKey) || "[]");
        const list = [...local].reverse();
        setItems(list);
        fetchHeroDetails(list);
      } catch {
        setItems([]);
      }
    } else {
      setItems([]);
    }
  }, [user, ready, endpoint, localFallbackKey, authFetch]);

  const hasHero = heroSlides.length > 0;

  return (
    <div>
      {/* Jika ada hero, tampilkan di atas */}
      {hasHero && <HeroCarousel slides={heroSlides} />}

      <div className={`px-[4%] ${hasHero ? "pt-8" : "pt-28"}`}>
        <h1 className="mb-6 text-2xl font-bold">{title}</h1>

        {!ready || items === null ? (
          <p className="py-16 text-center text-white/50">{t("ui.loading")}</p>
        ) : !user && !localFallbackKey ? (
          <p className="py-16 text-center text-white/50">
            {t("ui.loginToView")}
          </p>
        ) : items.length === 0 ? (
          <p className="py-16 text-center text-white/50">{t("ui.emptyLibrary")}</p>
        ) : (
          <PosterGrid items={items} />
        )}
      </div>
    </div>
  );
}
