import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Vitest runs without globalThis hooks, so RTL's automatic cleanup never
// fires — do it explicitly so each test starts with a clean document.
afterEach(() => {
  cleanup();
});

