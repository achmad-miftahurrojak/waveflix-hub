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
