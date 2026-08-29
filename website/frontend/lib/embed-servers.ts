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
  {
    id: "vidsrc",
    name: "VidSrc",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidsrc.to/embed/tv/${m.id}/${s ?? 1}/${e ?? 1}`
        : `https://vidsrc.to/embed/movie/${m.id}`,
  },
  {
    id: "superembed",
    name: "SuperEmbed",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://multiembed.mov/directstream.php?video_id=${m.id}&tmdb=1&s=${s ?? 1}&e=${e ?? 1}`
        : `https://multiembed.mov/directstream.php?video_id=${m.id}&tmdb=1`,
  },
  {
    id: "2embed",
    name: "2Embed",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://2embed.cc/embedtv/${m.id}&s=${s ?? 1}&e=${e ?? 1}`
        : `https://2embed.cc/embed/${m.id}`,
  },
];
