import { type FormEvent, useEffect, useState } from "react";
import { createLink, type Link, listLinks, shortUrl } from "./api";

export default function App() {
    const [links, setLinks] = useState<Link[]>([]);
    const [url, setUrl] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        listLinks().then(setLinks, (e: Error) => setError(e.message));
    }, []);

    const shorten = async (event: FormEvent) => {
        event.preventDefault();
        setSaving(true);
        setError(null);
        try {
            const link = await createLink(url.trim());
            setLinks((all) => [link, ...all]);
            setUrl("");
        } catch (e) {
            setError((e as Error).message);
        } finally {
            setSaving(false);
        }
    };

    return (
        <main>
            <h1>fullmono</h1>
            <p className="lead">Paste a long URL, get a short one.</p>
            <form className="inline" onSubmit={shorten}>
                <input
                    type="url"
                    required
                    placeholder="https://example.com/a/very/long/path"
                    value={url}
                    onChange={(event) => setUrl(event.target.value)}
                    aria-label="URL to shorten"
                />
                <button type="submit" className="primary" disabled={saving}>
                    Shorten
                </button>
            </form>
            {error && <p className="error">{error}</p>}
            <ul className="links card">
                {links.map((link) => (
                    <li key={link.code}>
                        <a href={shortUrl(link)}>{shortUrl(link)}</a>
                        <span className="target">{link.url}</span>
                        <span className="clicks">
                            {link.clicks} {link.clicks === 1 ? "click" : "clicks"}
                        </span>
                    </li>
                ))}
            </ul>
        </main>
    );
}
