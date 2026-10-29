import { expect, type Page, test } from "@playwright/test";

// End-to-end tests: Chromium against the whole stack (Caddy, the api and
// Postgres), used as a visitor would. They're the slowest kind and the only
// one that sees the pieces work together, so there are a few, each a journey.

// A URL no other run has shortened, so the tests can run in any order, at
// once, against a database that already holds links.
const uniqueUrl = () => `https://example.com/e2e/${crypto.randomUUID()}`;

async function shorten(page: Page, url: string) {
    await page.getByLabel("URL to shorten").fill(url);
    await page.getByRole("button", { name: "Shorten" }).click();
    return page.getByRole("listitem").filter({ hasText: url });
}

test.beforeEach(async ({ page }) => {
    await page.goto("/");
});

test("shortens a URL and keeps it", async ({ page }) => {
    const url = uniqueUrl();
    const item = await shorten(page, url);

    await expect(item.locator(".clicks")).toHaveText("0 clicks");
    await expect(page.getByLabel("URL to shorten")).toHaveValue("");
    // Still listed after a reload: the api saved it, the page didn't only draw it.
    await page.reload();
    await expect(item).toBeVisible();
});

test("the short URL redirects and counts the visit", async ({ page, request }) => {
    const url = uniqueUrl();
    const item = await shorten(page, url);
    const shortUrl = (await item.getByRole("link").getAttribute("href")) ?? "";
    expect(shortUrl).toMatch(/\/r\/[0-9A-Za-z]{7}$/);

    // Not followed: the test is about the redirect, not about example.com.
    const res = await request.get(shortUrl, { maxRedirects: 0 });
    expect(res.status()).toBe(302);
    expect(res.headers().location).toBe(url);

    await page.reload();
    await expect(item.locator(".clicks")).toHaveText("1 click");
});

test("shows why the api rejected a URL", async ({ page }) => {
    // The browser's own check accepts ftp://, so it's the api that says no.
    await shorten(page, "ftp://example.com/file");

    await expect(page.getByText("the URL must start with http:// or https://")).toBeVisible();
});
