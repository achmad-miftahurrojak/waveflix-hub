
export const TOKEN_KEY = "waveflix_token";
export const LIST_KEY = "waveflix_mylist";
export const FAV_KEY = "waveflix_favorites";

const LEGACY: Record<string, string> = {
  [TOKEN_KEY]: "summertide_token",
  [LIST_KEY]: "summertide_mylist",
};

let done = false;

export function migrateStorage(): void {
  if (done || typeof window === "undefined") return;
  done = true;
  for (const [next, legacy] of Object.entries(LEGACY)) {
    try {
      if (localStorage.getItem(next) === null) {
        const val = localStorage.getItem(legacy);
        if (val !== null) localStorage.setItem(next, val);
      }
    } catch {

    }
  }
}
