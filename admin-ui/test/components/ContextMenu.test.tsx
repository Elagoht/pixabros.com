import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ContextMenu } from "@/components/ui";

const items = [
  { id: "copy", label: "Copy" },
  { id: "paste", label: "Paste" },
];

describe("ContextMenu", () => {
  it("renders children", () => {
    render(<ContextMenu items={items}>Right click me</ContextMenu>);
    expect(screen.getByText("Right click me")).toBeInTheDocument();
  });

  it("shows menu on right-click", () => {
    render(<ContextMenu items={items}>Right click me</ContextMenu>);
    fireEvent.contextMenu(screen.getByText("Right click me"));
    expect(screen.getByText("Copy")).toBeInTheDocument();
    expect(screen.getByText("Paste")).toBeInTheDocument();
  });

  it("calls item onClick when menu item is clicked", () => {
    const onClick = vi.fn();
    const itemsWithAction = [{ id: "copy", label: "Copy", onClick }];
    render(<ContextMenu items={itemsWithAction}>Target</ContextMenu>);
    fireEvent.contextMenu(screen.getByText("Target"));
    fireEvent.click(screen.getByText("Copy"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});