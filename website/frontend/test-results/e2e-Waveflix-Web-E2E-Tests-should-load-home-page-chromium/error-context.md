# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: e2e.spec.ts >> Waveflix-Web E2E Tests >> should load home page
- Location: tests\e2e.spec.ts:4:7

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:3000/
Call log:
  - navigating to "http://localhost:3000/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from "@playwright/test";
  2  | 
  3  | test.describe("Waveflix-Web E2E Tests", () => {
  4  |   test("should load home page", async ({ page }) => {
  5  |     // Jalankan tes dengan asumsi server menyala
> 6  |     await page.goto("/");
     |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:3000/
  7  |     await expect(page).toHaveTitle(/Waveflix/i);
  8  |   });
  9  | 
  10 |   test("should check navigation to search page", async ({ page }) => {
  11 |     await page.goto("/search");
  12 |     await expect(page).toHaveURL(/\/search/);
  13 |   });
  14 | });
  15 | 
```