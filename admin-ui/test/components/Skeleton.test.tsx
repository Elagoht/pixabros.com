import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { Skeleton } from "@/components/ui";

describe("Skeleton", () => {
  it("renders rect variant by default", () => {
    const { container } = render(<Skeleton />);
    expect(container.firstChild).toHaveClass("rounded-lg");
  });

  it("renders circle variant", () => {
    const { container } = render(<Skeleton variant="circle" />);
    expect(container.firstChild).toHaveClass("rounded-full");
  });

  it("renders text variant", () => {
    const { container } = render(<Skeleton variant="text" />);
    expect(container.firstChild).toHaveClass("rounded");
  });

  it("applies animate-pulse class", () => {
    const { container } = render(<Skeleton />);
    expect(container.firstChild).toHaveClass("animate-pulse");
  });

  it("applies width and height style", () => {
    const { container } = render(<Skeleton width={100} height={20} />);
    const el = container.firstChild as HTMLElement;
    expect(el.style.width).toBe("100px");
    expect(el.style.height).toBe("20px");
  });

  it("applies custom className", () => {
    const { container } = render(<Skeleton className="mb-2" />);
    expect(container.firstChild).toHaveClass("mb-2");
  });
});