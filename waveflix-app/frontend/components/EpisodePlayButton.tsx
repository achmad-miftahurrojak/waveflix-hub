"use client";

import type { TmdbItem } from "@/lib/types";
import { useUI } from "./UIProvider";
import { PlayIcon } from "./Icons";

export default function EpisodePlayButton({
  show,
  season,
  episode,
}: {
  show: TmdbItem;
  season: number;
  episode: number;
}) {
  const { play } = useUI();
  return (
    <button
      onClick={() => play(show, season, episode)}
      className="flex items-center gap-2 rounded-md bg-accent px-8 py-3.5 text-lg font-semibold text-black transition hover:scale-105 hover:bg-accent-dark"
    >
      <PlayIcon className="text-black" /> Play
    </button>
  );
}
