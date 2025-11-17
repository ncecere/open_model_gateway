import { test, expect } from '@playwright/test';

const basePath = process.env.E2E_BASE_PATH || '/';

test('landing page renders', async ({ page }) => {
  await page.goto(basePath);
  await expect(page).toHaveTitle(/Open Model Gateway/i);
  await expect(page.locator('#app')).toBeVisible();
});
