import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Separator } from "@/components/ui";

describe("Separator", () => {
  it("renders horizontal separator by default", () => {
    const { container } = render(<Separator />);
    expect(container.firstChild).toHaveClass("h-px", "w-full");
  });

  it("renders vertical separator", () => {
    const { container } = render(<Separator orientation="vertical" />);
    expect(container.firstChild).toHaveClass("h-full", "w-px");
  });

  it("has role separator", () => {
    render(<Separator />);
    expect(screen.getByRole("separator")).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(<Separator className="my-4" />);
    expect(container.firstChild).toHaveClass("my-4");
  });
});