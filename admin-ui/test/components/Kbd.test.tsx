import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Kbd } from "@/components/ui";

describe("Kbd", () => {
  it("renders children", () => {
    render(<Kbd>Ctrl</Kbd>);
    expect(screen.getByText("Ctrl")).toBeInTheDocument();
  });

  it("renders kbd element", () => {
    render(<Kbd>⌘</Kbd>);
    const el = screen.getByText("⌘");
    expect(el.tagName).toBe("KBD");
  });

  it("applies kbd styling classes", () => {
    render(<Kbd>K</Kbd>);
    expect(screen.getByText("K")).toHaveClass("rounded-md");
  });

  it("applies custom className", () => {
    render(<Kbd className="ml-1">K</Kbd>);
    expect(screen.getByText("K")).toHaveClass("ml-1");
  });
});