import { test, expect } from "@playwright/test";

test.describe("Waveflix Watchlist E2E Tests", () => {
  test("should register, add a movie to watchlist, verify, and remove it", async ({ page }) => {
    const email = `test-${Date.now()}@example.com`;
    const username = `TestUser-${Date.now()}`;
    const password = `supersecret123`;

    // 1. Go to register page
    await page.goto("/daftar");
    await expect(page).toHaveTitle(/Waveflix/i);

    // 2. Fill registration details
    await page.fill('input[type="email"]', email);
    await page.fill('input[placeholder="Your name"]', username);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');

    // 3. Wait for redirect to /home
    await page.waitForURL(/\/home/);

    // 4. Navigate to a movie detail (ID 550 - Fight Club)
    await page.goto("/movie/550");
    
    // Wait for watchlist button and verify it's visible
    const watchlistBtn = page.locator('button:has-text("My List")');
    await expect(watchlistBtn).toBeVisible();

    // 5. Add to watchlist
    await watchlistBtn.click();

    // Verify button changes to "Saved"
    const savedBtn = page.locator('button:has-text("Saved")');
    await expect(savedBtn).toBeVisible();

    // 6. Go to /daftar-saya and verify Fight Club is listed
    await page.goto("/daftar-saya");
    await expect(page.locator("h4:has-text('Fight Club')")).toBeVisible();

    // 7. Go back to movie detail and remove
    await page.goto("/movie/550");
    await expect(savedBtn).toBeVisible();
    await savedBtn.click();
    await expect(watchlistBtn).toBeVisible();

    // 8. Go to /daftar-saya and verify it's gone
    await page.goto("/daftar-saya");
    await expect(page.locator("h4:has-text('Fight Club')")).not.toBeVisible();
  });
});
