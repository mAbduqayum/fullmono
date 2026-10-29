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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(path, init);
    const body = await res.json().catch(() => null);
    if (!res.ok) throw new Error(body?.error ?? `${res.status} ${res.statusText}`);
    return body as T;
}
