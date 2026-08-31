# Waveflix Hub

Web-based streaming catalog platform for movies and TV series. Browse TMDB catalog, manage watchlists, track viewing history, and user accounts with JWT authentication.

**Tags:** `streaming` `catalog` `fullstack`

![CI](https://github.com/hamin-baek/waveflix-hub/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)

## Features

- **Complete catalog** with trending Indonesian content, platform-specific sections (Netflix, Disney+ Hotstar, Apple TV+, HBO Max), and recently added titles
- **Detailed pages** for movies/TV shows including synopsis, genres, cast, and episode/season navigation
- **Browse and filter** by media type, genre, year, country, platform, and sorting options
- **Instant search** functionality
- **User accounts** with registration/login (JWT), optional email verification via SMTP
- **Watchlist and history** synchronized to server when logged in, localStorage fallback when offline
- **Multi-language content** support (id-ID with en-US fallback)
- **ISR revalidation** every 15 minutes for fresh content delivery

## Tech Stack

| Component | Technology |
|---|---|
| Frontend | Next.js 15 (App Router), React 19, TypeScript 5, Tailwind CSS 3 |
| Backend | Go 1.25, chi/net-http, PostgreSQL (pgx), JWT |
| Testing | Playwright (E2E), go test |
| CI/CD | GitHub Actions |

## Project Structure

```
waveflix-hub/
├── website/
│   ├── backend/     # Go API (TMDB proxy, auth, watchlist, history)
│   └── frontend/    # Next.js application
└── .github/
    └── workflows/   # CI: backend tests + frontend build & Playwright
```

## Getting Started

Prerequisites: Go ≥ 1.25, Node.js 20.

### 1. Backend Server (:8080)

```bash
cd website/backend
cp .env.example .env   # Configure environment variables
go run .
```

Configure `website/backend/.env`:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection URL |
| `TMDB_API_KEY` | Yes | API key from [TMDB](https://www.themoviedb.org/settings/api) |
| `JWT_SECRET` | Yes | Random string minimum 32 characters |
| `TRAKT_CLIENT_ID` / `TRAKT_CLIENT_SECRET` / `TRAKT_REDIRECT_URI` | Optional | Trakt.tv integration |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` | Optional | Email verification setup |
| `SMTP_FROM` | Production | Sender address for verification emails |
| `PORT` | No | Default `8080` |
| `ALLOWED_ORIGIN` | Production | CORS origin for frontend |
| `TRUSTED_PROXY_IPS` | Production | Reverse proxy IPs/CIDRs, comma-separated |
| `AUTH_COOKIE_SECURE` | Production | Set `1` when using HTTPS |

### 2. Frontend Application (:3000)

```bash
cd website/frontend
npm install
npm run dev
```

Configure `website/frontend/.env.local`:

- `BACKEND_URL` for server-side requests
- `NEXT_PUBLIC_BACKEND_URL` for browser requests (default `http://localhost:8080`)

For production, configure both `BACKEND_URL` and `NEXT_PUBLIC_BACKEND_URL` with actual HTTPS backend URLs.

For Gmail integration, use App Passwords instead of main account password. After configuring SMTP variables in `website/backend/.env`, restart backend and use the resend code button.

Open http://localhost:3000.

## Main Routes

- `/` — Homepage with hero section, trending Indonesian content, platform sections
- `/browse` — Browse and filter interface
- `/search?q=` — Search results
- `/[media]/[id]` — Movie/TV show details
- `/daftar-saya`, `/riwayat` — Watchlist and viewing history
- `/masuk`, `/daftar` — Login and registration

## Testing

```bash
# Backend tests
cd website/backend && go test ./...

# Frontend E2E tests (requires running backend)
cd website/frontend && npm run test:e2e
```

CI runs both test suites on push/PR to `main`/`master` branches.

## Contributing

1. Fork the repository and create a feature branch
2. Ensure `go test` and Playwright tests pass
3. Open a pull request to `main`

## License

[MIT](LICENSE)
