# Waveflix Hub

Platform streaming katalog film & serial berbasis web — jelajah katalog TMDB, watchlist, riwayat tontonan, dan akun pengguna dengan autentikasi JWT.

![CI](https://github.com/hamin-baek/waveflix-hub/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)

## Fitur

- **Katalog lengkap** — trending Indonesia, per-platform (Netflix, Disney+ Hotstar, Apple TV+, HBO Max), recently added
- **Halaman detail** movie/TV — sinopsis, genre, cast, episode + season selector
- **Browse & filter** — media, genre, tahun, negara, platform, urutan
- **Pencarian** instan
- **Akun** — register/login (JWT), email verification opsional via SMTP
- **Watchlist & riwayat** — sinkron ke server saat login, fallback localStorage saat belum login
- **Multi-bahasa** konten (id-ID dengan fallback en-US)
- ISR revalidate 15 menit → konten baru cepat masuk

## Tech Stack

| Bagian | Teknologi |
|---|---|
| Frontend | Next.js 15 (App Router), React 19, TypeScript 5, Tailwind CSS 3 |
| Backend | Go 1.25, chi/net-http, SQLite (`modernc.org/sqlite`), JWT |
| Testing | Playwright (E2E), `go test` |
| CI | GitHub Actions |

## Struktur

```
waveflix-hub/
├── website/
│   ├── backend/     # API Go (proxy TMDB, auth, watchlist, history)
│   └── frontend/    # Next.js app
└── .github/
    └── workflows/   # CI: backend test + frontend build & Playwright
```

## Menjalankan

Prasyarat: Go ≥ 1.25, Node.js 20.

### 1. Backend (:8080)

```bash
cd website/backend
cp .env.example .env   # lalu isi nilai
go run .
```

Konfigurasi `website/backend/.env`:

| Var | Wajib? | Keterangan |
|---|---|---|
| `TMDB_API_KEY` | Ya | Dari [TMDB](https://www.themoviedb.org/settings/api) |
| `JWT_SECRET` | Ya | String acak minimal 32 karakter |
| `TRAKT_CLIENT_ID` / `TRAKT_CLIENT_SECRET` / `TRAKT_REDIRECT_URI` | Opsional | Integrasi Trakt.tv |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` | Opsional | Email verification |
| `PORT` | Tidak | Default `8080` |
| `ALLOWED_ORIGIN` | Prod | CORS origin frontend |
| `TRUSTED_PROXY_IPS` | Prod | IP/CIDR reverse proxy, comma-separated |
| `AUTH_COOKIE_SECURE` | Prod | Set `1` saat memakai HTTPS |

### 2. Frontend (:3000)

```bash
cd website/frontend
npm install
npm run dev
```

Konfigurasi `website/frontend/.env.local`:

- `BACKEND_URL` — dipakai server-side
- `NEXT_PUBLIC_BACKEND_URL` — dipakai browser (default `http://localhost:8080`)

Di production, isi `BACKEND_URL` dan `NEXT_PUBLIC_BACKEND_URL` dengan URL HTTPS backend yang sebenarnya.

Buka http://localhost:3000.

## Rute Utama

- `/` — homepage: hero, trending Indonesia, per-platform
- `/browse` — jelajah + filter
- `/search?q=` — pencarian
- `/[media]/[id]` — detail movie/tv
- `/daftar-saya`, `/riwayat` — watchlist & tontonan
- `/masuk`, `/daftar` — login/register

## Testing

```bash
# Backend
cd website/backend && go test ./...

# Frontend E2E (perlu backend jalan)
cd website/frontend && npm run test:e2e
```

CI menjalankan keduanya pada push/PR ke `main`/`master`.

## Kontribusi

1. Fork & buat branch fitur
2. Pastikan `go test` dan Playwright lolos
3. Buka PR ke `main`

## Lisensi

[MIT](LICENSE)
