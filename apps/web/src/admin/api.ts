import { api, unwrap } from "../api";
import type { components } from "../api.gen";

// The login is an httpOnly cookie the api sets: this code never sees the
// token, and fetch sends the cookie with its same-origin requests by itself.

export type Session = components["schemas"]["Session"];

/** One page of a search, with what paging through it needs. */
export type SearchPage = components["schemas"]["SearchPage"];

export type Stats = components["schemas"]["Stats"];

/** Who's logged in: an ApiError with status 401 when nobody is. */
export const currentAdmin = () => unwrap(api.GET("/api/admin/session"));

export const logIn = (username: string, password: string) =>
    unwrap(api.POST("/api/admin/session", { body: { username, password } }));

export const logOut = () => unwrap(api.DELETE("/api/admin/session"));

export const searchLinks = (query: string, page: number) =>
    unwrap(api.GET("/api/admin/links", { params: { query: { q: query, page } } }));

export const deleteLink = (code: string) =>
    unwrap(api.DELETE("/api/admin/links/{code}", { params: { path: { code } } }));

export const linkStats = () => unwrap(api.GET("/api/admin/stats"));
