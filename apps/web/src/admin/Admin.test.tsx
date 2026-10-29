import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import type { Link } from "../api";
import Admin from "./Admin";

// Component tests: the panel with a fake api that answers by method and path,
// since the panel asks for several things at once.

const link = (code: string, url: string, clicks = 0): Link => ({
    code,
    url,
    clicks,
    createdAt: "2026-10-01T12:00:00Z",
});

type Route = (url: URL, init?: RequestInit) => [status: number, body?: unknown];

function fakeApi(routes: Record<string, Route>) {
    const fetch = vi.fn<typeof globalThis.fetch>(async (input, init) => {
        const url = new URL(String(input), location.origin);
        const route = routes[`${init?.method ?? "GET"} ${url.pathname}`];
        const [status, body] = route ? route(url, init) : [404, { error: "no such route" }];
        return body === undefined
            ? new Response(null, { status })
            : Response.json(body, { status });
    });
    vi.stubGlobal("fetch", fetch);
    return fetch;
}

/** The panel's own routes, over a list of links that a delete changes. */
function linksApi(links: Link[]): Record<string, Route> {
    return {
        "GET /api/admin/links": (url) => {
            const q = url.searchParams.get("q") ?? "";
            const found = links.filter((l) => l.code.includes(q) || l.url.includes(q));
            return [200, { links: found, total: found.length, pageSize: 25 }];
        },
        "GET /api/admin/stats": () => [
            200,
            {
                links: links.length,
                clicks: links.reduce((sum, l) => sum + l.clicks, 0),
                top: links.filter((l) => l.clicks > 0),
            },
        ],
        "DELETE /api/admin/links/go00000": () => {
            links.splice(
                links.findIndex((l) => l.code === "go00000"),
                1,
            );
            return [204];
        },
    };
}

/** The totals as they read, once the panel has them. */
async function totals() {
    const list = await screen.findByRole("list", { name: "Totals" });
    return within(list)
        .getAllByRole("listitem")
        .map((item) => item.textContent);
}

it("asks for a login, then shows the panel", async () => {
    const user = userEvent.setup();
    const fetch = fakeApi({
        "GET /api/admin/session": () => [401, { error: "log in first" }],
        "POST /api/admin/session": () => [200, { username: "alice" }],
        ...linksApi([link("go00000", "https://go.dev", 2), link("vite000", "https://vite.dev")]),
    });
    render(<Admin />);

    await user.type(await screen.findByLabelText("Username"), "alice");
    await user.type(screen.getByLabelText("Password"), "correct horse battery");
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(await totals()).toEqual(["2 links", "2 clicks"]);
    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(3); // the header and two links
    expect(fetch).toHaveBeenCalledWith(
        "/api/admin/session",
        expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ username: "alice", password: "correct horse battery" }),
        }),
    );
});

it("shows why a login failed", async () => {
    const user = userEvent.setup();
    fakeApi({
        "GET /api/admin/session": () => [401, { error: "log in first" }],
        "POST /api/admin/session": () => [401, { error: "wrong username or password" }],
    });
    render(<Admin />);

    await user.type(await screen.findByLabelText("Username"), "alice");
    await user.type(screen.getByLabelText("Password"), "a wrong password");
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(await screen.findByText("wrong username or password")).toBeInTheDocument();
});

it("searches, and deletes a link once confirmed", async () => {
    const user = userEvent.setup();
    const fetch = fakeApi({
        "GET /api/admin/session": () => [200, { username: "alice" }],
        ...linksApi([link("go00000", "https://go.dev"), link("vite000", "https://vite.dev")]),
    });
    const confirm = vi.fn<typeof globalThis.confirm>(() => true);
    vi.stubGlobal("confirm", confirm);
    render(<Admin />);

    await user.type(await screen.findByLabelText("Search links"), "go.dev");
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(fetch).toHaveBeenCalledWith("/api/admin/links?q=go.dev&page=1", undefined);
    await user.click(await screen.findByRole("button", { name: "Delete go00000" }));

    expect(confirm).toHaveBeenCalled();
    expect(await screen.findByText('No link matches "go.dev".')).toBeInTheDocument();
    expect(await totals()).toEqual(["1 link", "0 clicks"]);
});

it("goes back to the login form when the login expires", async () => {
    fakeApi({
        "GET /api/admin/session": () => [200, { username: "alice" }],
        "GET /api/admin/links": () => [401, { error: "log in again" }],
        "GET /api/admin/stats": () => [401, { error: "log in again" }],
    });
    render(<Admin />);

    expect(await screen.findByRole("button", { name: "Log in" })).toBeInTheDocument();
});
