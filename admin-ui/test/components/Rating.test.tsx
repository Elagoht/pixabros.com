import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Rating } from "@/components/ui";

describe("Rating", () => {
  it("renders correct number of stars by default", () => {
    const onChange = vi.fn();
    render(<Rating value={0} onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(5);
  });

  it("renders custom number of stars", () => {
    const onChange = vi.fn();
    render(<Rating value={0} count={10} onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(10);
  });

  it("calls onChange when clicking a star", () => {
    const onChange = vi.fn();
    render(<Rating value={0} onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    fireEvent.click(buttons[2]);
    expect(onChange).toHaveBeenCalledWith(3);
  });

  it("renders more buttons with allowHalf", () => {
    const onChange = vi.fn();
    render(<Rating value={0} allowHalf onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    expect(buttons.length).toBeGreaterThan(5);
  });

  it("does not call onChange when disabled", () => {
    const onChange = vi.fn();
    render(<Rating value={3} onChange={onChange} disabled />);
    const buttons = screen.getAllByRole("button");
    for (const btn of buttons) {
      expect(btn).toBeDisabled();
    }
    fireEvent.click(buttons[0]);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("does not call onChange when onChange is not provided", () => {
    render(<Rating value={3} />);
    const buttons = screen.getAllByRole("button");
    expect(() => fireEvent.click(buttons[0])).not.toThrow();
  });
});
