import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CodeBlock } from "@/components/ui";

describe("CodeBlock", () => {
  it("renders code content", () => {
    render(<CodeBlock code="const x = 1;" />);
    expect(screen.getByText("const x = 1;")).toBeInTheDocument();
  });

  it("renders filename when provided", () => {
    render(<CodeBlock code="x" filename="index.ts" />);
    expect(screen.getByText("index.ts")).toBeInTheDocument();
  });

  it("renders language when provided", () => {
    render(<CodeBlock code="x" language="typescript" />);
    expect(screen.getByText("typescript")).toBeInTheDocument();
  });

  it("renders copy button", () => {
    render(<CodeBlock code="x" />);
    expect(screen.getByText("Copy")).toBeInTheDocument();
  });
});
