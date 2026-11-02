import createClient from "openapi-fetch";
import type { components, paths } from "./api.gen";

// api.gen.ts is the api's contract, apps/api/openapi.yaml, as TypeScript types
// (`just web gen`): a path, parameter or body the api doesn't take, or a field
// its answers don't have, fails the type check rather than the request.

/** A shortened URL, as the api returns it. */
export type Link = components["schemas"]["Link"];

/** The api's client: this page's origin, which Caddy (or Vite) sends on to the api. */
export const api = createClient<paths>({
    baseUrl: location.origin,
    // fetch looked up on each call, not once here, so that tests can stub it.
    fetch: (request) => fetch(request),
});

/** The newest links first. */
export const listLinks = () => unwrap(api.GET("/api/links"));

export const createLink = (url: string) => unwrap(api.POST("/api/links", { body: { url } }));

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

/** A call's data, or an ApiError when the answer isn't a 2xx. */
export async function unwrap<T>(
    call: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
    const { data, error, response } = await call;
    if (!response.ok) {
        // A proxy's error page isn't the api's JSON: its status says what went wrong.
        const message = (error as { error?: string } | undefined)?.error;
        throw new ApiError(message ?? `${response.status} ${response.statusText}`, response.status);
    }
    return data as T;
}
