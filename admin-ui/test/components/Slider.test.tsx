import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Slider } from "@/components/ui";

describe("Slider", () => {
  it("renders with value prop", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("aria-valuenow", "50");
  });

  it("renders with default min and max", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("aria-valuemin", "0");
    expect(slider).toHaveAttribute("aria-valuemax", "100");
  });

  it("renders with custom min and max", () => {
    const onChange = vi.fn();
    render(<Slider value={5} min={0} max={10} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("aria-valuemin", "0");
    expect(slider).toHaveAttribute("aria-valuemax", "10");
    expect(slider).toHaveAttribute("aria-valuenow", "5");
  });

  it("increments value on ArrowRight key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith(51);
  });

  it("decrements value on ArrowLeft key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowLeft" });
    expect(onChange).toHaveBeenCalledWith(49);
  });

  it("increments value on ArrowUp key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowUp" });
    expect(onChange).toHaveBeenCalledWith(51);
  });

  it("decrements value on ArrowDown key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowDown" });
    expect(onChange).toHaveBeenCalledWith(49);
  });

  it("sets value to min on Home key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "Home" });
    expect(onChange).toHaveBeenCalledWith(0);
  });

  it("sets value to max on End key", () => {
    const onChange = vi.fn();
    render(<Slider value={50} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "End" });
    expect(onChange).toHaveBeenCalledWith(100);
  });

  it("respects step prop for keyboard interactions", () => {
    const onChange = vi.fn();
    render(<Slider value={50} step={5} onChange={onChange} />);
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith(55);
  });

  it("does not call onChange when disabled on keyboard", () => {
    const onChange = vi.fn();
    render(<Slider value={50} disabled onChange={onChange} />);
    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("aria-disabled", "true");
    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("shows value when showValue is true", () => {
    const onChange = vi.fn();
    render(<Slider value={42} onChange={onChange} showValue />);
    expect(screen.getByText("42")).toBeInTheDocument();
  });
});