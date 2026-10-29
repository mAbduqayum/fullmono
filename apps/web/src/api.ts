/** A shortened URL, as the api returns it. */
export interface Link {
    code: string;
    url: string;
    clicks: number;
    createdAt: string;
}

/** The newest links first. */
export function listLinks(): Promise<Link[]> {
    return request("/api/links");
}

export function createLink(url: string): Promise<Link> {
    return request("/api/links", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
    });
}

/** Where a link's code redirects from: this page's origin, which Caddy (or Vite) sends on to the api. */
export const shortUrl = (link: Link) => `${location.origin}/r/${link.code}`;

/** What the api said no with: its message, and the HTTP status (401: log in). */
export class ApiError extends Error {
    status: number;

    constructor(message: string, status: number) {
        super(message);
        this.status = status;
    }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(path, init);
    // A 204 has no body, and a proxy's error page isn't JSON: both read as null.
    const body = await res.json().catch(() => null);
    if (!res.ok) throw new ApiError(body?.error ?? `${res.status} ${res.statusText}`, res.status);
    return body as T;
}
