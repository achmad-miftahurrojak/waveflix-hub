// Daftar Variety Show yang akan menggunakan scraper khusus (Dramacool)
// Key adalah TMDB ID, Value adalah nama slug yang dipakai di Dramacool.

export const VERIFIED_ASIAN_SHOWS: Record<number, string> = {
  33238: "running-man",
  43222: "running-man", // Kadang ada 2 ID TMDB
  70672: "knowing-bros",
  30801: "2-days-1-night",
  64356: "the-return-of-superman",
  214250: "jinnys-kitchen",
  65282: "i-live-alone",
  72896: "my-little-old-boy",
  80736: "the-manager",
  78648: "amazing-saturday",
  81978: "you-quiz-on-the-block",
  236536: "apartment-404",
  210905: "physical-100",
  139798: "singles-inferno",
  13648: "sixth-sense",
  203508: "earth-arcade",
  70123: "new-journey-to-the-west",
  91121: "hangout-with-yoo",
  65270: "radio-star",
  66046: "king-of-mask-singer",
  52823: "weekly-idol",
  76033: "master-in-the-house",
  52910: "law-of-the-jungle",
  119645: "unexpected-business",
  68157: "three-meals-a-day",
  70910: "youns-kitchen",
  78798: "busted",
  2129: "the-genius",
  89861: "great-escape",
  1431: "crime-scene",
  116013: "girls-high-school-mystery-class",
  226878: "transit-love",
  103147: "heart-signal",
  129236: "i-am-solo",
  7677: "we-got-married",
  18618: "family-outing",
  5092: "infinity-challenge",
  218266: "happy-together",
  79449: "idol-room",
  130347: "street-woman-fighter",
  68884: "show-me-the-money",
  65552: "produce-101",
  217361: "boys-planet",
  91678: "queendom",
  118895: "kingdom-legendary-war",
  90755: "i-land",
};

/**
 * Cek apakah sebuah acara adalah variety show terverifikasi
 */
export function getAsianShowSlug(item: any): string | null {
  // 1. Prioritaskan dari whitelist manual (jika ada)
  if (VERIFIED_ASIAN_SHOWS[item.id]) {
    return VERIFIED_ASIAN_SHOWS[item.id];
  }

  // 2. Deteksi dinamis: Apakah ini Variety Show / Reality Show Korea?
  const isKorean =
    item.original_language === "ko" ||
    (item.origin_country && item.origin_country.includes("KR"));

  // 10764 = Reality, 10767 = Talk
  const hasVarietyGenre =
    item.genre_ids?.includes(10764) ||
    item.genre_ids?.includes(10767) ||
    item.genres?.some((g: any) => g.id === 10764 || g.id === 10767);

  if (isKorean && hasVarietyGenre) {
    const title = item.name || item.original_name || item.title;
    if (title) {
      // Ubah judul menjadi slug (contoh: "Jinny's Kitchen" -> "jinnys-kitchen")
      return title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-") // ubah spasi & karakter aneh jadi strip
        .replace(/(^-|-$)/g, "");    // hapus strip di awal/akhir
    }
  }

  return null;
}
