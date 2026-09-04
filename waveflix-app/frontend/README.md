# Summer Tide — Frontend (Next.js)

Stack: **Next.js 15 (App Router) + TypeScript + Tailwind 3**. Backend: Golang di `../backend` (proxy TMDB). Struktur `website/` sekarang hanya: `backend` + `frontend`.

## Menjalankan

```bash
cd ../backend && go run .     # API di :8080
```
```bash
npm install && npm run dev    # frontend di :3000
```

`.env.local`: `BACKEND_URL` (server) + `NEXT_PUBLIC_BACKEND_URL` (browser, mis. season selector).

## Rute

- `/` — homepage: Hero (full-screen) → Trending in Indonesia (ranking + toggle) → Netflix → Disney+ → Apple TV+ → HBO Max → Recently Added Movies/Series.
- `/[media]/[id]` — halaman detail (movie/tv): backdrop+logo, genre/tahun/durasi/rating, sinopsis (fallback EN), **episode + season selector** (tv), **cast berfoto**, tombol Putar & Daftar Saya.
- `/browse` — jelajah dengan filter: media, genre, tahun, negara, platform, urutan (query-driven).
- `/search?q=` — hasil pencarian.
- `/daftar-saya` — watchlist (localStorage; nanti pindah ke akun).

## Data (TMDB via backend)

- Trending = **Indonesia** (discover `watch_region=ID`, gabung film+tv by popularity), bukan global.
- Provider ID (region ID): Netflix `8`, Disney+ Hotstar `122`, Apple TV+ `350`, HBO Max `1899`.
- Deskripsi kosong id-ID otomatis fallback ke en-US di backend (`/api/detail`).
- ISR revalidate 15 menit → konten baru cepat masuk.

## Akun / Login (SUDAH ADA)

Golang + JWT + SQLite (`backend/waveflix.db`, pure-Go driver `modernc.org/sqlite`).
- `/masuk`, `/daftar` — form login/register. Token JWT di `localStorage` (`summertide_token`).
- Watchlist & history tersinkron ke DB saat login (fallback localStorage saat belum login).
- Halaman `/riwayat` (tontonan) + `/daftar-saya`. Navbar: avatar+dropdown (Keluar) kalau login.
- Endpoint: `/api/auth/{register,login,me}`, `/api/watchlist` (GET/POST/DELETE), `/api/history` (GET/POST) — semua kecuali register/login butuh `Authorization: Bearer`.
- `JWT_SECRET` dari env (ada default dev — WAJIB ganti di produksi).

## Belum (fase berikutnya)

- Trailer (video key sudah diambil di `/api/detail` → tinggal render).
- Scraper stream asli (`/api/stream` masih mock).
- Pindahkan API key TMDB hardcoded di `../backend/main.go` ke env var.
