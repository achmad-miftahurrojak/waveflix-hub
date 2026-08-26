# AGENTS.md

## Workflow

- Setiap kali selesai menulis/mengubah code, langsung commit dan push ke GitHub (branch `main`). Jangan tunggu diminta.
- Commit style: Conventional Commits (`feat:`, `fix:`, `chore:`).
- Jangan commit secrets (`.env`, `.env.local`).

## Commands

- Backend test: `cd website/backend && go test ./...`
- Frontend dev: `cd website/frontend && npm run dev`
- Frontend build: `cd website/frontend && npm run build`
- Frontend E2E: `cd website/frontend && npm run test:e2e`
