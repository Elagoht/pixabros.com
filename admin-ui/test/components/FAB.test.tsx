import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { FAB } from "@/components/ui";

describe("FAB", () => {
  it("renders button element", () => {
    render(<FAB />);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("calls onClick when clicked", () => {
    const onClick = vi.fn();
    render(<FAB onClick={onClick} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("renders default icon (IconPlus)", () => {
    render(<FAB />);
    const button = screen.getByRole("button");
    expect(button).toBeInTheDocument();
  });

  it("renders label when children provided", () => {
    render(<FAB>Add</FAB>);
    expect(screen.getByText("Add")).toBeInTheDocument();
  });

  it("applies default variant classes", () => {
    render(<FAB />);
    expect(screen.getByRole("button")).toHaveClass("bg-primary-500");
  });

  it("applies secondary variant classes", () => {
    render(<FAB variant="secondary" />);
    expect(screen.getByRole("button")).toHaveClass("bg-secondary-700");
  });

  it("applies bottom-right position by default", () => {
    render(<FAB />);
    expect(screen.getByRole("button")).toHaveClass("bottom-6", "right-6");
  });

  it("applies bottom-left position", () => {
    render(<FAB position="bottom-left" />);
    expect(screen.getByRole("button")).toHaveClass("bottom-6", "left-6");
  });
});
