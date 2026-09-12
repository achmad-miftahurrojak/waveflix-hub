"use client";

import React, { useState } from "react";
import { CoverflowCarousel, CoverflowSlide } from "@/components/ui/coverflow-carousel";
import { TmdbItem } from "@/lib/types";
import Link from "next/link";
import { PlayIcon, StarIcon } from "lucide-react";
import { itemYear, ratingText, genreNames } from "@/lib/helpers";

interface CollectionViewerProps {
  slides: CoverflowSlide[];
  movies: TmdbItem[];
  isTv: boolean;
  currentId: string;
}

export default function CollectionViewer({ slides, movies, isTv, currentId }: CollectionViewerProps) {
  const [activeIndex, setActiveIndex] = useState(0);
  
  const activeMovie = movies[activeIndex];
  if (!activeMovie) return null;

  const genres = genreNames(activeMovie);
  const overview = activeMovie.overview || "No description available for this title.";

  
  
  const href = isTv ? `/tv/${currentId}` : `/movie/${activeMovie.id}`;

  return (
    <div className="flex flex-col items-center w-full">
      <CoverflowCarousel
        slides={slides}
        loop={true}
        showCaption={false}
        showPagination={true}
        showNavigation={true}
        onIndexChange={setActiveIndex}
      />
      
      <div className="w-full max-w-3xl mt-8 px-4 flex flex-col items-center text-center">
        <h2 className="text-3xl font-bold mb-2">
          {activeMovie.title || activeMovie.name || "Unknown Title"}
        </h2>
        
        <div className="flex items-center gap-3 text-sm text-white/70 mb-4">
          <span className="flex items-center text-yellow-500">
            <StarIcon className="w-4 h-4 mr-1 fill-current" />
            {ratingText(activeMovie)}
          </span>
          {itemYear(activeMovie) && (
            <>
              <span>&bull;</span>
              <span>{itemYear(activeMovie)}</span>
            </>
          )}
          {genres.length > 0 && (
            <>
              <span>&bull;</span>
              <span>{genres.join(", ")}</span>
            </>
          )}
        </div>
        
        <p className="text-white/80 line-clamp-3 mb-6">
          {overview}
        </p>
        
        <Link 
          href={href}
          className="flex items-center gap-2 bg-white text-black px-6 py-2.5 rounded-full font-bold hover:bg-white/90 transition-colors"
        >
          <PlayIcon className="w-5 h-5 fill-current" />
          Watch Now
        </Link>
      </div>
    </div>
  );
}
