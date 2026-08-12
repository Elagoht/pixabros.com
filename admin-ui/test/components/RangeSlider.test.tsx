import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RangeSlider } from "@/components/ui";

describe("RangeSlider", () => {
  it("renders with min/max values", () => {
    render(<RangeSlider minValue={10} maxValue={80} onChange={() => {}} />);
    expect(screen.getAllByRole("slider")).toHaveLength(2);
  });

  it("displays aria attributes for min slider", () => {
    render(
      <RangeSlider
        minValue={10}
        maxValue={80}
        onChange={() => {}}
        min={0}
        max={100}
      />,
    );
    const sliders = screen.getAllByRole("slider");
    expect(sliders[0]).toHaveAttribute("aria-valuenow", "10");
  });

  it("displays aria attributes for max slider", () => {
    render(
      <RangeSlider
        minValue={10}
        maxValue={80}
        onChange={() => {}}
        min={0}
        max={100}
      />,
    );
    const sliders = screen.getAllByRole("slider");
    expect(sliders[1]).toHaveAttribute("aria-valuenow", "80");
  });

  it("shows value labels when showValue is true", () => {
    render(
      <RangeSlider minValue={20} maxValue={60} onChange={() => {}} showValue />,
    );
    expect(screen.getByText("20 - 60")).toBeInTheDocument();
  });

  it("does not show value labels when showValue is false", () => {
    render(<RangeSlider minValue={20} maxValue={60} onChange={() => {}} />);
    expect(screen.queryByText("20 - 60")).not.toBeInTheDocument();
  });
});
