import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import ProgressBar from "@/components/ui/Progress/ProgressBar";

describe("ProgressBar", () => {
  it("renders with value and applies width style", () => {
    const { container } = render(<ProgressBar value={60} />);
    const bar = container.querySelector(".bg-primary-500");
    expect(bar).toBeTruthy();
    expect((bar as HTMLElement).style.width).toBe("60%");
  });

  it("clamps value above 100 to 100", () => {
    const { container } = render(<ProgressBar value={150} />);
    const bar = container.querySelector(".bg-primary-500") as HTMLElement;
    expect(bar.style.width).toBe("100%");
  });

  it("clamps value below 0 to 0", () => {
    const { container } = render(<ProgressBar value={-10} />);
    const bar = container.querySelector(".bg-primary-500") as HTMLElement;
    expect(bar.style.width).toBe("0%");
  });

  it("shows percentage when showValue is true", () => {
    render(<ProgressBar value={75} showValue />);
    expect(screen.getByText("75%")).toBeInTheDocument();
  });

  it("does not show percentage when showValue is false", () => {
    render(<ProgressBar value={75} />);
    expect(screen.queryByText("75%")).not.toBeInTheDocument();
  });

  it("applies size classes", () => {
    const { container } = render(<ProgressBar value={50} size="lg" />);
    const track = container.querySelector(".h-4");
    expect(track).toBeTruthy();
  });
});