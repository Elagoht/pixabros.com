import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { Button } from "@/components/ui";

describe("Button", () => {
  it("renders children", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText("Click me")).toBeInTheDocument();
  });

  it("applies default variant classes", () => {
    render(<Button>Default</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-primary-600");
  });

  it("applies destructive variant classes", () => {
    render(<Button variant="destructive">Delete</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-red-600");
  });

  it("applies secondary variant classes", () => {
    render(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-secondary-600");
  });

  it("applies ghost variant classes", () => {
    render(<Button variant="ghost">Ghost</Button>);
    expect(screen.getByRole("button")).toHaveClass("bg-transparent");
  });

  it("applies outline variant classes", () => {
    render(<Button variant="outline">Outline</Button>);
    expect(screen.getByRole("button")).toHaveClass("border");
  });

  it("applies size classes", () => {
    render(<Button size="sm">Small</Button>);
    expect(screen.getByRole("button")).toHaveClass("h-8");
  });

  it("applies lg size classes", () => {
    render(<Button size="lg">Large</Button>);
    expect(screen.getByRole("button")).toHaveClass("h-10");
  });

  it("applies align classes", () => {
    render(<Button align="left">Left</Button>);
    expect(screen.getByRole("button")).toHaveClass("justify-start");
  });

  it("calls onClick handler", () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    fireEvent.click(screen.getByText("Click"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("is disabled when disabled prop is set", () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("defaults to type button", () => {
    render(<Button>Default</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("type", "button");
  });

  it("supports type submit", () => {
    render(<Button type="submit">Submit</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("type", "submit");
  });

  it("renders as a button element", () => {
    render(<Button>Test</Button>);
    expect(screen.getByRole("button").tagName).toBe("BUTTON");
  });

  // `to` was documented in COMPONENTS.md but never implemented: the prop fell
  // through to the DOM as <button to="/x">, so the control rendered fine and
  // simply did not navigate.
  describe("as a link", () => {
    const renderLink = (props: Record<string, unknown> = {}) =>
      render(
        <MemoryRouter>
          <Button to="/games/new" {...props}>
            New game
          </Button>
        </MemoryRouter>,
      );

    it("renders an anchor when `to` is given", () => {
      renderLink();
      expect(screen.getByRole("link")).toHaveAttribute("href", "/games/new");
    });

    it("does not leak `to` onto the element as an attribute", () => {
      renderLink();
      expect(screen.getByRole("link")).not.toHaveAttribute("to");
    });

    it("keeps its variant styling as a link", () => {
      renderLink({ variant: "destructive" });
      expect(screen.getByRole("link")).toHaveClass("bg-red-600");
    });

    // An anchor has no disabled state, so a disabled link would still navigate.
    it("falls back to a real button when disabled", () => {
      renderLink({ disabled: true });
      expect(screen.queryByRole("link")).not.toBeInTheDocument();
      expect(screen.getByRole("button")).toBeDisabled();
    });

    it("still renders a button when `to` is absent", () => {
      render(<Button>Save</Button>);
      expect(screen.getByRole("button")).toBeInTheDocument();
    });
  });
});
