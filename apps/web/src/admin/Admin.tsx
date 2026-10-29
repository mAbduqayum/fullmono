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

const plural = (n: number, noun: string) => (n === 1 ? noun : `${noun}s`);

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
        <section className="card login">
            <h1>fullmono admin</h1>
            <p className="lead">Log in to manage the links.</p>
            <form onSubmit={submit}>
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
                <button type="submit" className="primary" disabled={sending}>
                    Log in
                </button>
            </form>
            {error && <p className="error">{error}</p>}
        </section>
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
            <header className="bar">
                <h1>fullmono admin</h1>
                <p>
                    <span>
                        Logged in as <strong>{admin}</strong>
                    </span>
                    <button type="button" onClick={leave}>
                        Log out
                    </button>
                </p>
            </header>
            {error && <p className="error">{error}</p>}

            {stats && (
                <section className="stats" aria-label="Stats">
                    <ul className="totals" aria-label="Totals">
                        <Total n={stats.links} noun="link" />
                        <Total n={stats.clicks} noun="click" />
                    </ul>
                    <div className="card top">
                        <h2>Most clicked</h2>
                        {stats.top.length === 0 ? (
                            <p>No clicks yet.</p>
                        ) : (
                            <ol>
                                {stats.top.map((link) => (
                                    <li key={link.code}>
                                        <span className="code">{link.code}</span>
                                        <span className="url">{link.url}</span>
                                        <span className="num">
                                            {link.clicks.toLocaleString()}{" "}
                                            {plural(link.clicks, "click")}
                                        </span>
                                    </li>
                                ))}
                            </ol>
                        )}
                    </div>
                </section>
            )}

            <section className="card" aria-labelledby="links-heading">
                <div className="card-head">
                    <h2 id="links-heading">Links</h2>
                    <form className="inline" role="search" onSubmit={search}>
                        <input
                            type="search"
                            aria-label="Search links"
                            placeholder="A code, or part of a URL"
                            value={typed}
                            onChange={(event) => setTyped(event.target.value)}
                        />
                        <button type="submit">Search</button>
                    </form>
                </div>
                {found && found.total === 0 && (
                    <p className="empty">
                        {query ? `No link matches "${query}".` : "No links yet."}
                    </p>
                )}
                {found && found.total > 0 && (
                    <>
                        <div className="table">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Code</th>
                                        <th>URL</th>
                                        <th className="num">Clicks</th>
                                        <th>Created</th>
                                        <th aria-label="Actions" />
                                    </tr>
                                </thead>
                                <tbody>
                                    {found.links.map((link) => (
                                        <tr key={link.code}>
                                            <td className="code">{link.code}</td>
                                            <td className="url" title={link.url}>
                                                {link.url}
                                            </td>
                                            <td className="num">{link.clicks.toLocaleString()}</td>
                                            <td>{new Date(link.createdAt).toLocaleDateString()}</td>
                                            <td className="num">
                                                <button
                                                    type="button"
                                                    className="danger"
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
            </section>
        </>
    );
}

/** One of the totals: the figure large, what it counts under it. */
function Total({ n, noun }: { n: number; noun: string }) {
    return (
        <li className="card">
            <strong>{n.toLocaleString()}</strong> {plural(n, noun)}
        </li>
    );
}
