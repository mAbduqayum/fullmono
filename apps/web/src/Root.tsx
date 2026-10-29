import { lazy, Suspense } from "react";
import App from "./App";

// A chunk of its own, which only /admin downloads.
const Admin = lazy(() => import("./admin/Admin"));

/** The page for the URL: the admin panel at /admin, the shortener anywhere else. */
export default function Root() {
    if (location.pathname.replace(/\/$/, "") !== "/admin") return <App />;
    return (
        <Suspense>
            <Admin />
        </Suspense>
    );
}
