import { describe, expect, it, vi } from "vitest";
import { createLink, shortUrl } from "./api";
import { sent } from "./test/sent";

// Unit tests for the api client: fetch is replaced by a fake that answers what
// the test needs, so no server runs and nothing leaves the process.
function fakeFetch(status: number, body?: unknown) {
    const fetch = vi.fn<typeof globalThis.fetch>(async () =>
        body === undefined
            ? new Response("upstream down", { status, statusText: "Bad Gateway" })
            : Response.json(body, { status }),
    );
    vi.stubGlobal("fetch", fetch);
    return fetch;
}

describe("createLink", () => {
    it("posts the URL as JSON and returns the link", async () => {
        const link = { code: "abc1234", url: "https://go.dev", clicks: 0, createdAt: "" };
        const fetch = fakeFetch(201, link);

        await expect(createLink("https://go.dev")).resolves.toEqual(link);
        const request = fetch.mock.calls[0]?.[0];
        expect(await sent(request)).toEqual({
            route: "POST /api/links",
            query: "",
            body: { url: "https://go.dev" },
        });
        // The api turns away a body that doesn't say it's JSON.
        expect((request as Request).headers.get("Content-Type")).toBe("application/json");
    });

    it("throws the api's error message", async () => {
        fakeFetch(400, { error: "the URL must start with http:// or https://" });

        await expect(createLink("ftp://example.com")).rejects.toThrow(
            "the URL must start with http:// or https://",
        );
    });

    it("says the status too, so a caller can tell a 401", async () => {
        fakeFetch(401, { error: "log in first" });

        await expect(createLink("https://go.dev")).rejects.toMatchObject({
            message: "log in first",
            status: 401,
        });
    });

    it("throws the status when the answer isn't JSON", async () => {
        fakeFetch(502);

        await expect(createLink("https://go.dev")).rejects.toThrow("502 Bad Gateway");
    });
});

it("builds the short URL on this page's origin", () => {
    const link = { code: "abc1234", url: "https://go.dev", clicks: 0, createdAt: "" };
    expect(shortUrl(link)).toBe(`${location.origin}/r/abc1234`);
});
