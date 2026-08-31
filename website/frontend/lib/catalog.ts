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
  { id: 337, name: "Disney+" },
  { id: 119, name: "Prime Video" },
  { id: 350, name: "Apple TV+" },
  { id: 384, name: "HBO Max" },
];

export const SORTS: { value: string; label: string }[] = [
  { value: "popularity.desc", label: "Most Popular" },
  { value: "vote_average.desc", label: "Top Rated" },
  { value: "primary_release_date.desc", label: "Newest" },
];

export const YEARS: string[] = Array.from({ length: 30 }, (_, i) =>
  String(new Date().getFullYear() - i)
);
