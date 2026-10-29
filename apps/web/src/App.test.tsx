import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import type { Link } from "./api";
import App from "./App";

// Component tests: App rendered into jsdom and used the way a person would, by
// its labels and text (Testing Library's queries), never its state or markup.
// The api is a fake fetch answering in turn, so each test sets what it returns.

const link = (code: string, url: string, clicks = 0): Link => ({
    code,
    url,
    clicks,
    createdAt: "2026-10-01T12:00:00Z",
});

function fakeApi(...answers: Array<[status: number, body: unknown]>) {
    const fetch = vi.fn<typeof globalThis.fetch>();
    for (const [status, body] of answers) {
        fetch.mockResolvedValueOnce(Response.json(body, { status }));
    }
    vi.stubGlobal("fetch", fetch);
    return fetch;
}

it("lists the links the api has", async () => {
    fakeApi([200, [link("abc1234", "https://go.dev", 1), link("def5678", "https://react.dev")]]);
    render(<App />);

    const items = await screen.findAllByRole("listitem");
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent("https://go.dev");
    expect(items[0]).toHaveTextContent("1 click");
    expect(items[1]).toHaveTextContent("0 clicks");
});

it("shortens a URL and puts it first", async () => {
    const user = userEvent.setup();
    const fetch = fakeApi(
        [200, [link("abc1234", "https://go.dev")]],
        [201, link("new0000", "https://vite.dev")],
    );
    render(<App />);
    await screen.findByText("https://go.dev");

    const input = screen.getByLabelText("URL to shorten");
    await user.type(input, "https://vite.dev");
    await user.click(screen.getByRole("button", { name: "Shorten" }));

    const first = (await screen.findAllByRole("listitem"))[0];
    expect(within(first).getByText("https://vite.dev")).toBeInTheDocument();
    expect(input).toHaveValue("");
    expect(fetch).toHaveBeenLastCalledWith(
        "/api/links",
        expect.objectContaining({ method: "POST", body: '{"url":"https://vite.dev"}' }),
    );
});

it("shows why the api rejected a URL", async () => {
    const user = userEvent.setup();
    fakeApi([200, []], [400, { error: "the URL must start with http:// or https://" }]);
    render(<App />);

    await user.type(screen.getByLabelText("URL to shorten"), "ftp://example.com");
    await user.click(screen.getByRole("button", { name: "Shorten" }));

    expect(
        await screen.findByText("the URL must start with http:// or https://"),
    ).toBeInTheDocument();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
});
