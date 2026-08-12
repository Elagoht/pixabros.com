import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Sidebar } from "@/components/ui";
import { IconHome, IconSettings } from "@tabler/icons-react";

describe("Sidebar", () => {
  it("renders groups and items", () => {
    render(
      <MemoryRouter>
        <Sidebar
          groups={[
            {
              items: [
                { id: "home", icon: IconHome, labelKey: "menu.dashboard" },
                {
                  id: "settings",
                  icon: IconSettings,
                  labelKey: "menu.settings",
                },
              ],
            },
          ]}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });
});
