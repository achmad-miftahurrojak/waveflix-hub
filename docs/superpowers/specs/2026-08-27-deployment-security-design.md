# Deployment Security Design

**Goal:** Make Waveflix safe and functional for a production deployment without changing its core product behavior.

**Decisions:** Browser authentication will use an HttpOnly cookie, with a temporary Bearer fallback for existing clients. User input will be bounded at HTTP boundaries, profile images will be decoded and re-encoded as real images, and proxy trust will be restricted to configured proxy addresses. Frontend security headers and backend URLs will be environment-driven.

**Acceptance:** Backend tests, frontend type/build checks, dependency audit, and E2E tests pass; no production URL points at localhost; uploads cannot store arbitrary non-image bytes or grow without a per-user quota.