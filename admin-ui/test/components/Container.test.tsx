import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Container } from "@/components/ui";

describe("Container", () => {
  it("renders children", () => {
    render(<Container>Content</Container>);
    expect(screen.getByText("Content")).toBeInTheDocument();
  });

  it("applies sm max width by default", () => {
    render(<Container size="sm">Content</Container>);
    expect(screen.getByText("Content").closest("div")).toHaveClass("max-w-2xl");
  });

  it("applies md max width", () => {
    render(<Container size="md">Content</Container>);
    expect(screen.getByText("Content").closest("div")).toHaveClass("max-w-4xl");
  });

  it("applies lg max width (default)", () => {
    render(<Container>Content</Container>);
    expect(screen.getByText("Content").closest("div")).toHaveClass("max-w-6xl");
  });

  it("applies xl max width", () => {
    render(<Container size="xl">Content</Container>);
    expect(screen.getByText("Content").closest("div")).toHaveClass("max-w-7xl");
  });

  it("applies full max width", () => {
    render(<Container size="full">Content</Container>);
    expect(screen.getByText("Content").closest("div")).toHaveClass(
      "max-w-full",
    );
  });
});
