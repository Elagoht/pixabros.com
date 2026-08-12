import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { CommandPalette } from "@/components/ui";

const groups = [
  {
    heading: "Actions",
    items: [
      { id: "new", label: "New File", onSelect: vi.fn() },
      { id: "open", label: "Open File", onSelect: vi.fn() },
    ],
  },
];

beforeEach(() => {
  Element.prototype.scrollIntoView = () => {};
});

describe("CommandPalette", () => {
  it("does not render when closed", () => {
    const { container } = render(
      <CommandPalette open={false} onClose={() => {}} groups={groups} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders when open", () => {
    render(<CommandPalette open onClose={() => {}} groups={groups} />);
    expect(
      screen.getByPlaceholderText("Type a command or search..."),
    ).toBeInTheDocument();
  });

  it("renders group headings and items when open", () => {
    render(<CommandPalette open onClose={() => {}} groups={groups} />);
    expect(screen.getByText("Actions")).toBeInTheDocument();
    expect(screen.getByText("New File")).toBeInTheDocument();
    expect(screen.getByText("Open File")).toBeInTheDocument();
  });
});
