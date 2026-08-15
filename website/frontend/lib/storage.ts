// Kunci localStorage (brand baru: Waveflix).
export const TOKEN_KEY = "waveflix_token";
export const LIST_KEY = "waveflix_mylist";
export const FAV_KEY = "waveflix_favorites";

// Pemetaan kunci lama → baru untuk migrasi (jangan sampai user lama ke-reset).
const LEGACY: Record<string, string> = {
  [TOKEN_KEY]: "summertide_token",
  [LIST_KEY]: "summertide_mylist",
};

let done = false;

/**
 * Salin nilai dari kunci lama ke kunci baru sekali saja, kalau kunci baru
 * belum ada. Idempotent & aman dipanggil dari mana pun (client only).
 */
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
      /* localStorage tidak tersedia — abaikan */
    }
  }
}
