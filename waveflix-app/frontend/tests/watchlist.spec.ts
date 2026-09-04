import { test, expect } from "@playwright/test";

test.describe("Waveflix Watchlist E2E Tests", () => {
  test("should register, add a movie to watchlist, verify, and remove it", async ({ page }) => {
    const email = `test-${Date.now()}@example.com`;
    const username = `TestUser-${Date.now()}`;
    const password = `supersecret123`;

    await page.goto("/daftar");
    await expect(page).toHaveTitle(/Waveflix/i);

    await page.fill('input[type="email"]', email);
    await page.fill('input[placeholder="Your name"]', username);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');
    await expect(page.getByRole("textbox", { name: "Verification code" })).toHaveValue(/\d{6}/);
    await page.getByRole("button", { name: "Verify & Create Account" }).click();

    await page.waitForURL(/\/profiles/);
    await page.locator('button.group').first().click();
    await page.waitForURL(/\/home/);

    await page.goto("/movie/550");

    const watchlistBtn = page.getByRole("button", { name: "Daftar Saya" });
    await expect(watchlistBtn).toBeVisible();

    await watchlistBtn.click();

    const savedBtn = page.getByRole("button", { name: "Disimpan" });
    await expect(savedBtn).toBeVisible();

    await page.goto("/daftar-saya");
    await expect(page.getByRole("link", { name: "Fight Club" })).toBeVisible();

    await page.goto("/movie/550");
    await expect(savedBtn).toBeVisible();
    await savedBtn.click();
    await expect(watchlistBtn).toBeVisible();

    await page.goto("/daftar-saya");
    await expect(page.getByRole("link", { name: "Fight Club" })).not.toBeVisible();
  });
});
