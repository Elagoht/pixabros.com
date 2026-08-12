import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { EmptyState } from "@/components/ui";
import { IconPlus } from "@tabler/icons-react";

describe("EmptyState", () => {
  it("renders title", () => {
    render(<EmptyState title="No items" />);
    expect(screen.getByText("No items")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<EmptyState title="No items" description="Add some items to get started" />);
    expect(screen.getByText("Add some items to get started")).toBeInTheDocument();
  });

  it("renders action when provided", () => {
    render(<EmptyState title="No items" action={<button>Add item</button>} />);
    expect(screen.getByText("Add item")).toBeInTheDocument();
  });

  it("renders custom icon when provided", () => {
    render(<EmptyState title="No items" icon={IconPlus} />);
    expect(screen.getByText("No items")).toBeInTheDocument();
  });

  it("renders default icon when icon not provided", () => {
    render(<EmptyState title="No items" />);
    expect(screen.getByText("No items")).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(<EmptyState title="Test" className="mt-4" />);
    expect(container.firstChild).toHaveClass("mt-4");
  });
});