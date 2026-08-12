import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import ProgressCircle from "@/components/ui/Progress/ProgressCircle";

describe("ProgressCircle", () => {
  it("renders with value and SVG elements", () => {
    const { container } = render(<ProgressCircle value={50} />);
    const circles = container.querySelectorAll("circle");
    expect(circles.length).toBe(2);
  });

  it("shows percentage when showValue is true", () => {
    render(<ProgressCircle value={75} showValue />);
    expect(screen.getByText("75%")).toBeInTheDocument();
  });

  it("does not show percentage when showValue is false", () => {
    render(<ProgressCircle value={75} />);
    expect(screen.queryByText("75%")).not.toBeInTheDocument();
  });

  it("clamps value above 100 to 100", () => {
    render(<ProgressCircle value={150} showValue />);
    expect(screen.getByText("100%")).toBeInTheDocument();
  });

  it("clamps value below 0 to 0", () => {
    render(<ProgressCircle value={-5} showValue />);
    expect(screen.getByText("0%")).toBeInTheDocument();
  });

  it("applies custom size", () => {
    const { container } = render(<ProgressCircle value={50} size={100} />);
    const svg = container.querySelector("svg");
    expect(svg?.getAttribute("width")).toBe("100");
    expect(svg?.getAttribute("height")).toBe("100");
  });
});
