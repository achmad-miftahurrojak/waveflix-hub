export type MediaType = "movie" | "tv" | "all";

export interface TmdbItem {
  id: number;
  title?: string;
  name?: string;
  original_name?: string;
  overview?: string;
  poster_path?: string | null;
  logo_path?: string | null;
  backdrop_path?: string | null;
  vote_average?: number;
  popularity?: number;
  release_date?: string;
  first_air_date?: string;
  media_type?: string;
  genre_ids?: number[];
  original_language?: string;
  vidlink_available?: boolean;
  number_of_seasons?: number;
}

export interface TmdbListResponse {
  page: number;
  results: TmdbItem[];
  total_pages?: number;
}

export interface TmdbLogo {
  file_path: string;
  iso_639_1: string | null;
}

export interface TmdbImagesResponse {
  logos?: TmdbLogo[];
}

export interface Genre {
  id: number;
  name: string;
}

export interface CastMember {
  id: number;
  name: string;
  original_name?: string;
  character?: string;
  profile_path?: string | null;
  order?: number;
}

export interface CrewMember {
  id: number;
  name: string;
  original_name?: string;
  job?: string;
}

export interface TmdbVideo {
  key: string;
  site: string;
  type: string;
  official?: boolean;
}

export interface ProductionCompany {
  id: number;
  name: string;
  logo_path?: string | null;
  origin_country?: string;
}

export interface SeasonSummary {
  season_number: number;
  name?: string;
  episode_count?: number;
  poster_path?: string | null;
  overview?: string;
  vote_average?: number;
}

export interface Episode {
  episode_number: number;
  name?: string;
  overview?: string;
  still_path?: string | null;
  runtime?: number | null;
  air_date?: string;
  vote_average?: number;
}

export interface TmdbDetail extends TmdbItem {
  tagline?: string;
  runtime?: number;
  episode_run_time?: number[];
  number_of_seasons?: number;
  number_of_episodes?: number;
  status?: string;
  original_language?: string;
  origin_country?: string[];
  production_countries?: { iso_3166_1: string; name: string }[];
  production_companies?: ProductionCompany[];
  created_by?: { id: number; name: string }[];
  last_episode_to_air?: {
    season_number?: number;
    episode_number?: number;
    name?: string;
    overview?: string;
    still_path?: string | null;
    air_date?: string;
    runtime?: number | null;
  };
  genres?: Genre[];
  seasons?: SeasonSummary[];
  credits?: { cast?: CastMember[]; crew?: CrewMember[] };
  videos?: { results?: TmdbVideo[] };
  recommendations?: { results?: TmdbItem[] };
  similar?: { results?: TmdbItem[] };
  images?: { logos?: TmdbLogo[]; backdrops?: { file_path: string }[]; posters?: { file_path: string }[] };
}

export interface PersonCredit extends TmdbItem {
  character?: string;
  job?: string;
  department?: string;
}

export interface TmdbPerson {
  id: number;
  name: string;
  original_name?: string;
  biography?: string;
  profile_path?: string | null;
  birthday?: string | null;
  deathday?: string | null;
  place_of_birth?: string | null;
  known_for_department?: string;
  combined_credits?: {
    cast?: PersonCredit[];
    crew?: PersonCredit[];
  };
}

export interface HistoryItem extends TmdbItem {
  season?: number;
  episode?: number;
  runtime?: number;
  progress?: number;
}
