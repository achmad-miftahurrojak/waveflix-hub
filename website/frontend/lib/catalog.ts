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

export const COLLECTIONS: { id: number; name: string }[] = [
  { id: 10194, name: "Toy Story Collection" },
  { id: 86311, name: "The Avengers Collection" },
  { id: 119, name: "The Lord of the Rings Collection" },
  { id: 1241, name: "Harry Potter Collection" },
  { id: 10, name: "Star Wars Collection" },
  { id: 9485, name: "Fast and Furious Collection" },
  { id: 645, name: "James Bond Collection" },
  { id: 263, name: "The Dark Knight Collection" },
  { id: 1703, name: "X-Men Collection" },
  { id: 230, name: "Star Trek Collection" },
  { id: 328, name: "Jurassic Park Collection" },
  { id: 115, name: "The Matrix Collection" },
  { id: 556, name: "Spider-Man (Sam Raimi) Collection" },
  { id: 8945, name: "Mad Max Collection" },
  { id: 121938, name: "The Hobbit Collection" },
  { id: 404609, name: "John Wick Collection" },
  { id: 1575, name: "Rocky Collection" },
  { id: 87359, name: "Mission: Impossible Collection" },
  { id: 33514, name: "The Godfather Collection" },
];
