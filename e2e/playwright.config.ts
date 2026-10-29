import { defineConfig, devices } from "@playwright/test";

const ci = Boolean(process.env.CI);

export default defineConfig({
    testDir: "tests",
    // Every test makes its own data (see uniqueUrl), so they can all run at once.
    fullyParallel: true,
    // A test.only left in would quietly skip the others: fail CI instead.
    forbidOnly: ci,
    // In CI, a test that fails and then passes on a retry is reported as flaky.
    retries: ci ? 2 : 0,
    reporter: [["list"], ["html", { open: "never" }]],
    use: {
        // compose's e2e service points it at the web container; on the host, at `just up`.
        baseURL: process.env.BASE_URL ?? "http://localhost:8000",
        // Each failed test's trace (DOM snapshots, network, console) goes in the report.
        trace: "retain-on-failure",
    },
    projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
