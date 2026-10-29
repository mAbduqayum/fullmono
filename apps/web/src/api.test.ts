import { describe, expect, it, vi } from "vitest";
import { createLink, shortUrl } from "./api";

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
        expect(fetch).toHaveBeenCalledWith("/api/links", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url: "https://go.dev" }),
        });
    });

    it("throws the api's error message", async () => {
        fakeFetch(400, { error: "the URL must start with http:// or https://" });

        await expect(createLink("ftp://example.com")).rejects.toThrow(
            "the URL must start with http:// or https://",
        );
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
