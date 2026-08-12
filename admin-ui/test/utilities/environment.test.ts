import { describe, it, expect } from "vitest";
import { Environment } from "@/utilities/environment";

describe("Environment", () => {
  it("has apiBase property", () => {
    expect(Environment).toHaveProperty("apiBase");
  });

  it("has mediaBase property", () => {
    expect(Environment).toHaveProperty("mediaBase");
  });

  it("has pageSize property", () => {
    expect(Environment).toHaveProperty("pageSize");
  });

  it("pageSize is a number", () => {
    expect(typeof Environment.pageSize).toBe("number");
  });

  it("apiBase defaults to localhost:3000 when no env var", () => {
    const expected = import.meta.env.VITE_API_BASE ?? "http://localhost:3000";
    expect(Environment.apiBase).toBe(expected);
  });

  it("mediaBase defaults to localhost:3001 when no env var", () => {
    const expected = import.meta.env.VITE_MEDIA_BASE ?? "http://localhost:3001";
    expect(Environment.mediaBase).toBe(expected);
  });

  it("pageSize defaults to 12 when no env var", () => {
    const expected = Number(import.meta.env.VITE_PAGE_SIZE) || 12;
    expect(Environment.pageSize).toBe(expected);
  });
});