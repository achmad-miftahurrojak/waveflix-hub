# Deployment Security Implementation Plan

> **For agentic workers:** Execute inline task-by-task with focused verification after each slice.

**Goal:** Harden authentication, uploads, request limits, proxying, production configuration, dependencies, and E2E coverage.

**Architecture:** Keep the existing Go HTTP API and Next.js App Router. Add small boundary helpers rather than introducing a validation framework; use an HttpOnly auth cookie while retaining Bearer compatibility during migration.

**Tech Stack:** Go 1.25, SQLite, JWT, Next.js 15, React 19, npm, Playwright.

**Spec:** `docs/superpowers/specs/2026-08-27-deployment-security-design.md`

## Global Constraints

- Preserve existing user flows and API paths.
- Never commit `.env`, `.env.local`, database files, or secrets.
- Run focused tests after each implementation slice.

### Task 1: Backend boundaries

- [ ] Add failing tests for cookie auth, bounded proxy trust, image validation, and request limits.
- [ ] Implement the smallest helpers and update handlers.
- [ ] Run `go test ./...` and `go vet ./...`.

### Task 2: Production frontend configuration

- [ ] Make browser API URL, upload rewrite, and CSP connect sources environment-driven.
- [ ] Update E2E registration flow to select a profile.
- [ ] Run `npm run build` and `npm run test:e2e`.

### Task 3: Dependency and release gate

- [ ] Update vulnerable dependency constraints and lockfile.
- [ ] Run `npm audit --audit-level=high`, backend tests, frontend build, and E2E.
- [ ] Review staged diff for secrets, commit, and push `main`.