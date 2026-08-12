import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Calendar } from "@/components/ui";

describe("Calendar", () => {
  it("renders month/year header", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    expect(screen.getByText(/June 2025/)).toBeInTheDocument();
  });

  it("renders day headers", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    expect(screen.getByText("Mo")).toBeInTheDocument();
    expect(screen.getByText("Su")).toBeInTheDocument();
  });

  it("renders day grid with the selected day", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    expect(screen.getByText("15")).toBeInTheDocument();
  });

  it("highlights selected day", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    const dayBtn = screen.getByText("15");
    expect(dayBtn).toHaveClass("bg-primary-500");
  });

  it("calls onChange when a day is selected", () => {
    const onChange = vi.fn();
    render(<Calendar value={new Date(2025, 5, 15)} onChange={onChange} />);
    fireEvent.click(screen.getByText("10"));
    expect(onChange).toHaveBeenCalledOnce();
  });

  it("navigates to previous month", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    const buttons = screen.getAllByRole("button");
    fireEvent.click(buttons[0]);
    expect(screen.getByText(/May 2025/)).toBeInTheDocument();
  });

  it("navigates to next month", () => {
    render(<Calendar value={new Date(2025, 5, 15)} onChange={() => {}} />);
    const buttons = screen.getAllByRole("button");
    fireEvent.click(buttons[2]);
    expect(screen.getByText(/July 2025/)).toBeInTheDocument();
  });
});
