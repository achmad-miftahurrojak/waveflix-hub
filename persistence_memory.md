# Persistence Memory & Architectural Decisions

Dokumen ini mencatat perubahan penting dan keputusan arsitektur agar tidak terulang kesalahan yang sama:

## 1. TMDB API Search Fix (Bug 400)
- **Masalah:** API Search `/api/search` dari frontend mengirim parameter `q` (contoh: `?q=nawilla`), namun backend `parallel_handlers.go` mencari parameter `query`. Hal ini menyebabkan error `400 Bad Request`.
- **Solusi:** Diperbarui `handleSearchParallel` di backend untuk membaca parameter `q`. Jika tidak ada, baru membaca parameter `query`.

## 2. Provider Filtering (Section 3 Home)
- **Masalah:** Fitur filter provider (Netflix, Disney+, dll) di halaman utama tidak berfungsi sebagaimana mestinya karena backend melakukan hardcode `with_watch_providers` untuk semua request discover.
- **Solusi:** `handleDiscoverParallel` diubah sehingga jika menerima query parameter `provider`, backend akan memetakannya ke `with_watch_providers`. Jika tidak ada, barulah menggunakan default hardcode providers.

## 3. Primary Content Provider
- **Keputusan:** Menetapkan **vidlink.pro** sebagai provider konten (embed) satu-satunya dan utama, demi menghindari kebingungan pengguna dengan terlalu banyak opsi provider.
- **Implementasi:** Provider lain (vidsrc, superembed, 2embed) dihapus dari daftar `EMBED_SERVERS` di frontend.

## 4. Navbar & Navigation
- **Keputusan:** Tab kategori "Asian" dan "Anime" dihapus dari navigasi utama (Navbar) untuk menyederhanakan tampilan.

## 5. Hero Section Home
- **Keputusan:** Hero section diatur agar menampilkan mix antara Top 10 film/series populer (khususnya untuk region Indonesia) ditambah dengan K-Drama.
- **Data yang Ditampilkan:** Backdrop image, trailer (jika ada), logo judul (fallback menggunakan teks yang bagus), durasi, dan status tayang. Semua sudah terimplementasi pada `HeroCarousel.tsx`.

## 6. Country Filter Bug Fix — Korean/Chinese Content Empty (Bug)
- **Masalah:** Saat filter `country=KR` atau `country=CN`, hasil yang ditampilkan kosong/sangat sedikit. Padahal K-Drama dan C-Drama ada banyak.
- **Root Cause:** `handleDiscoverParallel` di `parallel_handlers.go` selalu set `watch_region=ID` dan `with_watch_providers` ke provider-provider yang tersedia di Indonesia. Akibatnya, konten Korea/China yang hanya ada di provider regional Asia tidak muncul.
- **Solusi:** Ketika ada parameter `country`, backend tidak menerapkan `watch_region=ID` dan tidak memaksa `with_watch_providers`. Filter provider hanya aktif saat tidak ada filter negara, atau jika user memang memilih provider secara eksplisit.
- **File:** `website/backend/parallel_handlers.go` → `handleDiscoverParallel`

## 7. Poster Language — Pakai Poster Global/English
- **Masalah:** Poster film/series Korea, China, Jepang ditampilkan dalam huruf lokal (Hangul, Hanzi, Hiragana) yang sulit dibaca pengguna Indonesia.
- **Solusi:** `getImageLangs()` di `tmdb_stubs.go` diubah default-nya dari `"id,null"` menjadi `"en,null"`. Dengan ini TMDB akan mengembalikan poster berbahasa Inggris (global) sebagai prioritas utama.
- **File:** `website/backend/tmdb_stubs.go`

## 8. Curated Provider List — Hanya Platform Besar
- **Keputusan:** `MAJOR_PROVIDERS` dikurangi dari 13 provider menjadi 6 provider utama terpercaya saja: **Netflix (8), Disney+ (122), HBO Max (384), Apple TV+ (350), Prime Video (119), Viu (158)**.
- **Alasan:** Terlalu banyak provider mengakibatkan konten bodong/tidak dikenal masuk ke katalog. Dengan hanya 6 platform major, konten yang muncul lebih terjamin kualitas dan ketersediaannya.
- **File:** `website/frontend/lib/tmdb.ts` → `MAJOR_PROVIDERS`; `website/frontend/lib/catalog.ts` → `PROVIDERS`

## 9. Home Page Sections Order
- **Keputusan final urutan home page:**
  1. Hero Carousel (mix Indonesia populer + K-Drama terbaru)
  2. Continue Watching (kalau user sudah login dan punya history)
  3. Trending Now (TOP 20 Global)
  4. Trending KDrama (hanya drakor, tanpa anime/variety show)
  5. Platforms (Netflix, Disney+, HBO Max, Apple TV+, Prime Video, Viu) — switchable
  6. Asian Drama (Korean, Chinese, Japanese, Thai) — switchable, **tanpa Anime**
  7. Latest Movies
  8. Latest Series
  9. New Episodes (episode terbaru dari series yang sedang tayang)
- **Catatan:** Sections Asian Drama TIDAK menggunakan provider filter agar konten negara tersebut tetap muncul (mengikuti fix #6 di atas).
