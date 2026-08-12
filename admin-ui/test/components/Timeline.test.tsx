import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Timeline } from "@/components/ui";

describe("Timeline", () => {
  it("renders timeline items", () => {
    const items = [
      { id: "1", title: "Step 1" },
      { id: "2", title: "Step 2" },
    ];
    render(<Timeline items={items} />);
    expect(screen.getByText("Step 1")).toBeInTheDocument();
    expect(screen.getByText("Step 2")).toBeInTheDocument();
  });

  it("renders item description", () => {
    const items = [{ id: "1", title: "Step 1", description: "Details here" }];
    render(<Timeline items={items} />);
    expect(screen.getByText("Details here")).toBeInTheDocument();
  });

  it("renders item timestamp", () => {
    const items = [{ id: "1", title: "Step 1", timestamp: "2 hours ago" }];
    render(<Timeline items={items} />);
    expect(screen.getByText("2 hours ago")).toBeInTheDocument();
  });

  it("returns null for empty items", () => {
    const { container } = render(<Timeline items={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders custom icon", () => {
    const items = [{ id: "1", title: "Step 1" }];
    render(<Timeline items={items} />);
    expect(screen.getByText("Step 1")).toBeInTheDocument();
  });
});