import { describe, expect, it } from "vitest";
import { Navigation } from "@/utilities/navigation";

describe("Navigation", () => {
  it("basePath has no trailing slash", () => {
    expect(Navigation.basePath.endsWith("/")).toBe(false);
  });

  it("toAppPath leaves an already-relative path alone", () => {
    expect(Navigation.toAppPath("/login")).toBe("/login");
  });

  it("toBrowserPath and toAppPath round-trip", () => {
    expect(Navigation.toAppPath(Navigation.toBrowserPath("/change-password"))).toBe(
      "/change-password",
    );
  });

  it("toAppPath maps the bare base path to the app root", () => {
    const base = Navigation.basePath || "/";
    expect(Navigation.toAppPath(base)).toBe("/");
  });
});
