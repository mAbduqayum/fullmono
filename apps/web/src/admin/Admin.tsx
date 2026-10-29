import { type FormEvent, useEffect, useState } from "react";
import { ApiError, type Link } from "../api";
import {
    currentAdmin,
    deleteLink,
    linkStats,
    logIn,
    logOut,
    type SearchPage,
    searchLinks,
    type Stats,
} from "./api";

/** A 401: no login, or one that has expired. */
const loggedOut = (error: unknown) => error instanceof ApiError && error.status === 401;

const count = (n: number, noun: string) => `${n} ${noun}${n === 1 ? "" : "s"}`;

/** /admin: the login form, or the panel once the api knows who's here. */
export default function Admin() {
    // undefined while the api is asked, null when nobody is logged in.
    const [admin, setAdmin] = useState<string | null>();

    useEffect(() => {
        currentAdmin().then(
            (session) => setAdmin(session.username),
            () => setAdmin(null),
        );
    }, []);

    if (admin === undefined) return null;
    return (
        <main className="admin">
            {admin === null ? (
                <LogInForm onLogIn={setAdmin} />
            ) : (
                <Panel admin={admin} onLogOut={() => setAdmin(null)} />
            )}
        </main>
    );
}

function LogInForm({ onLogIn }: { onLogIn: (username: string) => void }) {
    const [error, setError] = useState<string | null>(null);
    const [sending, setSending] = useState(false);

    const submit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        setSending(true);
        setError(null);
        try {
            const session = await logIn(String(form.get("username")), String(form.get("password")));
            onLogIn(session.username);
        } catch (e) {
            setError((e as Error).message);
            setSending(false);
        }
    };

    return (
        <>
            <h1>fullmono admin</h1>
            <form className="login" onSubmit={submit}>
                <label>
                    Username
                    <input name="username" autoComplete="username" required />
                </label>
                <label>
                    Password
                    <input
                        name="password"
                        type="password"
                        autoComplete="current-password"
                        required
                    />
                </label>
                <button type="submit" disabled={sending}>
                    Log in
                </button>
            </form>
            {error && <p className="error">{error}</p>}
        </>
    );
}

function Panel({ admin, onLogOut }: { admin: string; onLogOut: () => void }) {
    const [typed, setTyped] = useState("");
    const [query, setQuery] = useState("");
    const [page, setPage] = useState(1);
    const [found, setFound] = useState<SearchPage | null>(null);
    const [stats, setStats] = useState<Stats | null>(null);
    const [error, setError] = useState<string | null>(null);
    // Bumped after a delete, to load the page and the stats again.
    const [version, setVersion] = useState(0);

    useEffect(() => {
        let current = true;
        Promise.all([searchLinks(query, page), linkStats()]).then(
            ([links, totals]) => {
                if (!current) return;
                // The last page's last link was deleted: show the page before.
                if (links.links.length === 0 && page > 1) setPage(page - 1);
                setFound(links);
                setStats(totals);
            },
            (e: Error) => {
                if (!current) return;
                if (loggedOut(e)) onLogOut();
                else setError(e.message);
            },
        );
        return () => {
            current = false;
        };
    }, [query, page, version, onLogOut]);

    const search = (event: FormEvent) => {
        event.preventDefault();
        setQuery(typed.trim());
        setPage(1);
    };

    const remove = async (link: Link) => {
        if (!confirm(`Delete ${link.code}? It will stop redirecting to ${link.url}.`)) return;
        try {
            await deleteLink(link.code);
            setVersion((v) => v + 1);
        } catch (e) {
            if (loggedOut(e)) onLogOut();
            else setError((e as Error).message);
        }
    };

    const leave = async () => {
        try {
            await logOut();
            onLogOut();
        } catch (e) {
            setError((e as Error).message);
        }
    };

    const pages = found ? Math.max(1, Math.ceil(found.total / found.pageSize)) : 1;
    return (
        <>
            <header>
                <h1>fullmono admin</h1>
                <p>
                    Logged in as <strong>{admin}</strong>{" "}
                    <button type="button" onClick={leave}>
                        Log out
                    </button>
                </p>
            </header>
            {error && <p className="error">{error}</p>}

            {stats && (
                <section aria-label="Stats">
                    <p className="totals">
                        {count(stats.links, "link")}, {count(stats.clicks, "click")}
                    </p>
                    {stats.top.length > 0 && (
                        <>
                            <h2>Most clicked</h2>
                            <ul>
                                {stats.top.map((link) => (
                                    <li key={link.code}>
                                        <span>{link.code}</span>
                                        <span className="target">{link.url}</span>
                                        <span className="clicks">
                                            {count(link.clicks, "click")}
                                        </span>
                                    </li>
                                ))}
                            </ul>
                        </>
                    )}
                </section>
            )}

            <h2>Links</h2>
            <form role="search" onSubmit={search}>
                <input
                    type="search"
                    aria-label="Search links"
                    placeholder="A code, or part of a URL"
                    value={typed}
                    onChange={(event) => setTyped(event.target.value)}
                />
                <button type="submit">Search</button>
            </form>
            {found && found.total === 0 && (
                <p className="lead">{query ? `No link matches "${query}".` : "No links yet."}</p>
            )}
            {found && found.total > 0 && (
                <>
                    <div className="table">
                        <table>
                            <thead>
                                <tr>
                                    <th>Code</th>
                                    <th>URL</th>
                                    <th>Clicks</th>
                                    <th>Created</th>
                                    <th aria-label="Actions" />
                                </tr>
                            </thead>
                            <tbody>
                                {found.links.map((link) => (
                                    <tr key={link.code}>
                                        <td>{link.code}</td>
                                        <td className="url" title={link.url}>
                                            {link.url}
                                        </td>
                                        <td>{link.clicks}</td>
                                        <td>{new Date(link.createdAt).toLocaleDateString()}</td>
                                        <td>
                                            <button
                                                type="button"
                                                aria-label={`Delete ${link.code}`}
                                                onClick={() => remove(link)}
                                            >
                                                Delete
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                    <nav className="pager" aria-label="Pages">
                        <button
                            type="button"
                            disabled={page === 1}
                            onClick={() => setPage(page - 1)}
                        >
                            Newer
                        </button>
                        <span>
                            Page {page} of {pages}
                        </span>
                        <button
                            type="button"
                            disabled={page >= pages}
                            onClick={() => setPage(page + 1)}
                        >
                            Older
                        </button>
                    </nav>
                </>
            )}
        </>
    );
}
