import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { router } from "@/lib/routes";
import QuickActions from "@/pages/(panel)/dashboard/QuickActions";

const renderActions = () =>
  render(
    <MemoryRouter>
      <QuickActions />
    </MemoryRouter>,
  );

// Collect every concrete path the router knows, so a shortcut cannot point at
// a route that does not exist.
const declaredPaths = (): string[] => {
  const paths: string[] = [];
  const walk = (entries: unknown[]): void => {
    for (const entry of entries) {
      const route = entry as { path?: string; children?: unknown[] };
      if (route.path) {
        paths.push(route.path);
      }
      if (route.children) {
        walk(route.children);
      }
    }
  };
  walk(router.routes as unknown[]);
  return paths;
};

describe("QuickActions", () => {
  it("renders a shortcut for each action", () => {
    renderActions();
    expect(screen.getAllByRole("link").length).toBeGreaterThanOrEqual(5);
  });

  it("shows the create shortcuts", () => {
    renderActions();
    for (const label of [
      "New game",
      "New devlog post",
      "New award",
      "New member",
      "Media library",
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });

  // A quick action landing on a 404 is worse than no shortcut at all, and a
  // renamed route would otherwise break this silently.
  it("only links to routes the router actually declares", () => {
    renderActions();
    const known = declaredPaths();

    for (const link of screen.getAllByRole("link")) {
      const href = link.getAttribute("href") ?? "";
      expect(known, `no route declares ${href}`).toContain(href);
    }
  });
});
