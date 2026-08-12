import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MenuBar } from "@/components/ui";

const items = [
  {
    id: "file",
    label: "File",
    children: [
      { id: "new", label: "New" },
      { id: "open", label: "Open" },
    ],
  },
  {
    id: "edit",
    label: "Edit",
    children: [{ id: "undo", label: "Undo" }],
  },
];

describe("MenuBar", () => {
  it("renders menu bar items", () => {
    render(<MenuBar items={items} />);
    expect(screen.getByText("File")).toBeInTheDocument();
    expect(screen.getByText("Edit")).toBeInTheDocument();
  });

  it("opens dropdown on item click", () => {
    render(<MenuBar items={items} />);
    fireEvent.click(screen.getByText("File"));
    expect(screen.getByText("New")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();
  });

  it("calls onClick on child item click", () => {
    const onClick = vi.fn();
    const itemsWithAction = [
      {
        id: "file",
        label: "File",
        children: [{ id: "new", label: "New", onClick }],
      },
    ];
    render(<MenuBar items={itemsWithAction} />);
    fireEvent.click(screen.getByText("File"));
    fireEvent.click(screen.getByText("New"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
