import { type Link, request } from "../api";

// The login is an httpOnly cookie the api sets: this code never sees the
// token, and fetch sends the cookie with its same-origin requests by itself.

export interface Session {
    username: string;
}

/** One page of a search, with what paging through it needs. */
export interface SearchPage {
    links: Link[];
    total: number;
    pageSize: number;
}

export interface Stats {
    links: number;
    clicks: number;
    /** The most clicked links. */
    top: Link[];
}

/** Who's logged in: an ApiError with status 401 when nobody is. */
export const currentAdmin = () => request<Session>("/api/admin/session");

export const logIn = (username: string, password: string) =>
    request<Session>("/api/admin/session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
    });

export const logOut = () => request<null>("/api/admin/session", { method: "DELETE" });

export const searchLinks = (query: string, page: number) =>
    request<SearchPage>(
        `/api/admin/links?${new URLSearchParams({ q: query, page: String(page) })}`,
    );

export const deleteLink = (code: string) =>
    request<null>(`/api/admin/links/${encodeURIComponent(code)}`, { method: "DELETE" });

export const linkStats = () => request<Stats>("/api/admin/stats");
