// Runs before every test file (vite.config.ts's setupFiles).
import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Unmount what each test rendered, so the next one starts from an empty page.
afterEach(cleanup);
