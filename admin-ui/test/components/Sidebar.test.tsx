import { IconDeviceGamepad2, IconHome } from "@tabler/icons-react";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { Sidebar } from "@/components/ui";
import { sidebarGroups } from "@/data/sidebar";

const renderSidebar = (groups: SidebarGroupData[]) =>
  render(
    <MemoryRouter>
      <Sidebar groups={groups} />
    </MemoryRouter>,
  );

describe("Sidebar", () => {
  it("renders group titles and items", () => {
    renderSidebar([
      {
        titleKey: "menu.groups.content",
        items: [
          { id: "home", icon: IconHome, labelKey: "menu.dashboard", path: "/" },
          {
            id: "games",
            icon: IconDeviceGamepad2,
            labelKey: "menu.games",
            path: "/games",
          },
        ],
      },
    ]);

    expect(screen.getByText("Content")).toBeInTheDocument();
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Games")).toBeInTheDocument();
  });

  it("renders an item with a path as a link", () => {
    renderSidebar([
      {
        items: [{ id: "games", labelKey: "menu.games", path: "/games" }],
      },
    ]);

    expect(screen.getByRole("link", { name: "Games" })).toHaveAttribute(
      "href",
      "/games",
    );
  });

  // Modules whose backend does not exist yet are listed but must not look or
  // behave like working navigation.
  it("renders an item with no path as a non-clickable, disabled entry", () => {
    renderSidebar([
      {
        items: [{ id: "devlog", labelKey: "menu.devlog" }],
      },
    ]);

    expect(screen.queryByRole("link", { name: "Devlog" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Devlog" })).toBeNull();

    const label = screen.getByText("Devlog");
    expect(label.closest("span.cursor-not-allowed")).not.toBeNull();
  });
});

describe("sidebarGroups data", () => {
  const allItems = sidebarGroups.flatMap((group) => group.items);

  it("only exposes a path for modules that are actually built", () => {
    const navigable = allItems
      .filter((item) => item.path)
      .map((item) => item.path);

    expect(navigable).toEqual(["/", "/games", "/awards", "/members"]);
  });

  it("lists every module from the architecture spec", () => {
    expect(allItems.map((item) => item.id)).toEqual([
      "dashboard",
      "games",
      "devlog",
      "awards",
      "members",
      "homepage",
      "site-settings",
      "media",
      "contact",
      "regen-jobs",
    ]);
  });

  it("has no user-management entry", () => {
    expect(allItems.some((item) => item.id === "users")).toBe(false);
  });
});
