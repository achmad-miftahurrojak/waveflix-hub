import { test, expect } from "@playwright/test";

test.describe("Waveflix-Web E2E Tests", () => {
  test("should load home page", async ({ page }) => {

    await page.goto("/");
    await expect(page).toHaveTitle(/Waveflix/i);
  });

  test("should check navigation to search page", async ({ page }) => {
    await page.goto("/search", { waitUntil: "domcontentloaded" });
    await expect(page).toHaveURL(/\/search/);
  });
});
