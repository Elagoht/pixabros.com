import { describe, expect, it } from "vitest";
import { Environment } from "@/utilities/environment";

describe("Environment", () => {
  it("has apiBase property", () => {
    expect(Environment).toHaveProperty("apiBase");
  });

  it("has mediaBase property", () => {
    expect(Environment).toHaveProperty("mediaBase");
  });

  it("pageSize is a number", () => {
    expect(typeof Environment.pageSize).toBe("number");
  });

  // The Go server serves the SPA and the API from one origin, so the default
  // base must stay relative -- an absolute default would break the
  // SameSite=Strict session cookie.
  it("apiBase defaults to same-origin", () => {
    expect(Environment.apiBase).toBe(import.meta.env.VITE_API_BASE ?? "");
  });

  it("mediaBase defaults to same-origin", () => {
    expect(Environment.mediaBase).toBe(import.meta.env.VITE_MEDIA_BASE ?? "");
  });

  it("pageSize defaults to 12 when no env var", () => {
    expect(Environment.pageSize).toBe(
      Number(import.meta.env.VITE_PAGE_SIZE) || 12,
    );
  });
});
