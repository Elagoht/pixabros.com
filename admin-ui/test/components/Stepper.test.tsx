import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Stepper } from "@/components/ui";

describe("Stepper", () => {
  const steps = ["Details", "Review", "Complete"];

  it("renders step labels", () => {
    render(<Stepper steps={steps} activeStep={0} />);
    expect(screen.getByText("Details")).toBeInTheDocument();
    expect(screen.getByText("Review")).toBeInTheDocument();
    expect(screen.getByText("Complete")).toBeInTheDocument();
  });

  it("highlights active step", () => {
    render(<Stepper steps={steps} activeStep={1} />);
    expect(screen.getByText("Review")).toHaveClass("text-primary-700");
  });

  it("shows step numbers for incomplete steps", () => {
    render(<Stepper steps={steps} activeStep={0} />);
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("shows checkmark for completed steps", () => {
    const { container } = render(
      <Stepper steps={steps} activeStep={2} />,
    );
    const iconElements = container.querySelectorAll(
      'svg[stroke-width="3"]',
    );
    expect(iconElements.length).toBeGreaterThanOrEqual(1);
  });

  it("calls onStepClick when step is clicked", () => {
    const onStepClick = vi.fn();
    render(<Stepper steps={steps} activeStep={0} onStepClick={onStepClick} />);
    fireEvent.click(screen.getByText("2"));
    expect(onStepClick).toHaveBeenCalledWith(1);
  });
});