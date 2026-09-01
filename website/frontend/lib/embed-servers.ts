import { TmdbItem } from "./types";
import { isTv } from "./helpers";

export interface EmbedServer {
  id: string;
  name: string;
  getUrl: (m: TmdbItem, season?: number, episode?: number) => string;
}

export const EMBED_SERVERS: EmbedServer[] = [
  {
    id: "vidlink",
    name: "VidLink",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidlink.pro/tv/${m.id}/${s ?? 1}/${e ?? 1}`
        : `https://vidlink.pro/movie/${m.id}`,
  },
];
