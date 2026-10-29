import { expect, type Page, test } from "@playwright/test";

// The admin `just test-e2e` sets up for the run, with a new random password.
const admin = { username: "e2e", password: process.env.ADMIN_PASSWORD ?? "" };

async function logIn(page: Page, password = admin.password) {
    await page.goto("/admin");
    await page.getByLabel("Username").fill(admin.username);
    await page.getByLabel("Password").fill(password);
    await page.getByRole("button", { name: "Log in" }).click();
}

test("an admin finds a link and deletes it", async ({ page, request }) => {
    // A link to delete, made through the api the way the visitors' page makes one.
    const url = `https://example.com/e2e/${crypto.randomUUID()}`;
    const { code } = await (await request.post("/api/links", { data: { url } })).json();

    await logIn(page);
    await page.getByLabel("Search links").fill(url);
    await page.getByRole("button", { name: "Search" }).click();
    const row = page.getByRole("row").filter({ hasText: code });
    await expect(row).toBeVisible();

    page.once("dialog", (dialog) => dialog.accept());
    await row.getByRole("button", { name: `Delete ${code}` }).click();
    await expect(page.getByText(`No link matches "${url}".`)).toBeVisible();
    // Gone for visitors too.
    expect((await request.get(`/r/${code}`, { maxRedirects: 0 })).status()).toBe(404);
});

test("the login lasts until logging out", async ({ page }) => {
    await logIn(page);
    await expect(page.getByRole("button", { name: "Log out" })).toBeVisible();
    // The token is out of the page's scripts' reach, and other sites' requests.
    const [cookie] = await page.context().cookies();
    expect(cookie).toMatchObject({ name: "admin_token", httpOnly: true, sameSite: "Strict" });

    await page.reload();
    await page.getByRole("button", { name: "Log out" }).click();
    await page.reload();
    await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();
});

test("a wrong password doesn't log in", async ({ page }) => {
    await logIn(page, "not the password");

    await expect(page.getByText("wrong username or password")).toBeVisible();
});

test("the admin api answers 401 without a login", async ({ request }) => {
    for (const path of ["/api/admin/session", "/api/admin/links", "/api/admin/stats"]) {
        expect((await request.get(path)).status(), path).toBe(401);
    }
});
