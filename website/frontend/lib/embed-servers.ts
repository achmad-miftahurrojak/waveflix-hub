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
    id: "vidsrcpro",
    name: "VidSrc PRO",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidsrc.pro/embed/tv/${m.id}/${s ?? 1}/${e ?? 1}`
        : `https://vidsrc.pro/embed/movie/${m.id}`,
  },
  {
    id: "autoembed",
    name: "AutoEmbed",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://autoembed.to/tv/tmdb/${m.id}-${s ?? 1}-${e ?? 1}`
        : `https://autoembed.to/movie/tmdb/${m.id}`,
  },
  {
    id: "multiembed",
    name: "MultiEmbed",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://multiembed.mov/?video_id=${m.id}&tmdb=1&s=${s ?? 1}&e=${e ?? 1}`
        : `https://multiembed.mov/?video_id=${m.id}&tmdb=1`,
  },
  {
    id: "smashy",
    name: "SmashyStream",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://player.smashy.stream/tv/${m.id}?s=${s ?? 1}&e=${e ?? 1}`
        : `https://player.smashy.stream/movie/${m.id}`,
  },
  {
    id: "vidsrcme",
    name: "VidSrc.ME",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidsrc.me/embed/tv?tmdb=${m.id}&season=${s ?? 1}&episode=${e ?? 1}`
        : `https://vidsrc.me/embed/movie?tmdb=${m.id}`,
  },
  {
    id: "vidsrcnet",
    name: "VidSrc.NET",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidsrc.net/embed/tv?tmdb=${m.id}&season=${s ?? 1}&episode=${e ?? 1}`
        : `https://vidsrc.net/embed/movie?tmdb=${m.id}`,
  },
  {
    id: "vidsrcxyz",
    name: "VidSrc.XYZ",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://vidsrc.xyz/embed/tv?tmdb=${m.id}&season=${s ?? 1}&episode=${e ?? 1}`
        : `https://vidsrc.xyz/embed/movie?tmdb=${m.id}`,
  },
  {
    id: "2embed",
    name: "2Embed",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://www.2embed.cc/embedtv/${m.id}&s=${s ?? 1}&e=${e ?? 1}`
        : `https://www.2embed.cc/embed/${m.id}`,
  },
  {
    id: "moviesapi",
    name: "MoviesAPI",
    getUrl: (m, s, e) =>
      isTv(m)
        ? `https://moviesapi.club/tv/${m.id}-${s ?? 1}-${e ?? 1}`
        : `https://moviesapi.club/movie/${m.id}`,
  },
];
