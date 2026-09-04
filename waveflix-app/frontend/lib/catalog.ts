import type { MediaType } from "./types";

export const MOVIE_GENRES: { id: number; name: string }[] = [
  { id: 28, name: "Action" },
  { id: 12, name: "Adventure" },
  { id: 16, name: "Animation" },
  { id: 35, name: "Comedy" },
  { id: 80, name: "Crime" },
  { id: 99, name: "Documentary" },
  { id: 18, name: "Drama" },
  { id: 10751, name: "Family" },
  { id: 14, name: "Fantasy" },
  { id: 27, name: "Horror" },
  { id: 9648, name: "Mystery" },
  { id: 10749, name: "Romance" },
  { id: 878, name: "Science Fiction" },
  { id: 53, name: "Thriller" },
  { id: 10752, name: "War" },
];

export const TV_GENRES: { id: number; name: string }[] = [
  { id: 10759, name: "Action & Adventure" },
  { id: 16, name: "Animation" },
  { id: 35, name: "Comedy" },
  { id: 80, name: "Crime" },
  { id: 99, name: "Documentary" },
  { id: 18, name: "Drama" },
  { id: 10751, name: "Family" },
  { id: 9648, name: "Mystery" },
  { id: 10765, name: "Sci-Fi & Fantasy" },
  { id: 10768, name: "War & Politics" },
];

export function genresFor(media: MediaType) {
  return media === "tv" ? TV_GENRES : MOVIE_GENRES;
}

export const COUNTRIES: { code: string; name: string }[] = [
  { code: "ID", name: "Indonesia" },
  { code: "US", name: "United States" },
  { code: "KR", name: "Korea" },
  { code: "JP", name: "Japan" },
  { code: "CN", name: "China" },
  { code: "GB", name: "United Kingdom" },
  { code: "IN", name: "India" },
  { code: "TH", name: "Thailand" },
];

export const PROVIDERS: { id: number; name: string }[] = [
  { id: 8, name: "Netflix" },
  { id: 122, name: "Disney+" },
  { id: 384, name: "HBO Max" },
  { id: 350, name: "Apple TV+" },
  { id: 119, name: "Prime Video" },
  { id: 158, name: "Viu" },
];

export const SORTS: { value: string; label: string }[] = [
  { value: "popularity.desc", label: "Most Popular" },
  { value: "vote_average.desc", label: "Top Rated" },
  { value: "primary_release_date.desc", label: "Newest" },
];

export const YEARS: string[] = Array.from({ length: 30 }, (_, i) =>
  String(new Date().getFullYear() - i)
);

export type CollectionType = "movie" | "tv" | "animation";

export interface CollectionData {
  id: string;
  name: string;
  type: CollectionType[];
  country: string[];
}

export const COLLECTIONS: CollectionData[] = [
  { id: "10194", name: "Toy Story Collection", type: ["movie", "animation"], country: ["US"] },
  { id: "86311,131292,131295,131296,284433,422834,618529,529892,582496", name: "Marvel Cinematic Universe", type: ["movie"], country: ["US"] },
  { id: "119,121938", name: "Middle-earth Collection", type: ["movie"], country: ["US", "GB"] },
  { id: "1241,435259", name: "Wizarding World Collection", type: ["movie"], country: ["GB", "US"] },
  { id: "10", name: "Star Wars Collection", type: ["movie"], country: ["US"] },
  { id: "9485", name: "Fast and Furious Collection", type: ["movie"], country: ["US"] },
  { id: "645", name: "James Bond Collection", type: ["movie"], country: ["GB"] },
  { id: "263,120794,948485", name: "Batman Collection", type: ["movie"], country: ["US", "GB"] },
  { id: "1703,453993,448150", name: "X-Men Collection", type: ["movie"], country: ["US"] },
  { id: "230", name: "Star Trek Collection", type: ["movie"], country: ["US"] },
  { id: "328", name: "Jurassic Park Collection", type: ["movie"], country: ["US"] },
  { id: "115", name: "The Matrix Collection", type: ["movie"], country: ["US"] },
  { id: "556,125574,531241,573436", name: "Spider-Man Collection", type: ["movie"], country: ["US"] },
  { id: "8945", name: "Mad Max Collection", type: ["movie"], country: ["US"] },
  { id: "404609", name: "John Wick Collection", type: ["movie"], country: ["US"] },
  { id: "1575", name: "Rocky Collection", type: ["movie"], country: ["US"] },
  { id: "87359", name: "Mission: Impossible Collection", type: ["movie"], country: ["US"] },
  { id: "33514", name: "The Godfather Collection", type: ["movie"], country: ["US"] },
];
