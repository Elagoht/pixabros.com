import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Pagination } from "@/components/ui";

describe("Pagination", () => {
  it("renders page buttons", () => {
    render(<Pagination page={1} totalPages={5} onChange={() => {}} />);
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
  });

  it("calls onChange when page is clicked", () => {
    const onChange = vi.fn();
    render(<Pagination page={1} totalPages={5} onChange={onChange} />);
    fireEvent.click(screen.getByText("3"));
    expect(onChange).toHaveBeenCalledWith(3);
  });

  it("calls onChange on prev click", () => {
    const onChange = vi.fn();
    render(<Pagination page={3} totalPages={5} onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    fireEvent.click(buttons[0]);
    expect(onChange).toHaveBeenCalledWith(2);
  });

  it("calls onChange on next click", () => {
    const onChange = vi.fn();
    render(<Pagination page={3} totalPages={5} onChange={onChange} />);
    const buttons = screen.getAllByRole("button");
    fireEvent.click(buttons[buttons.length - 1]);
    expect(onChange).toHaveBeenCalledWith(4);
  });

  it("returns null when totalPages is 1", () => {
    const { container } = render(
      <Pagination page={1} totalPages={1} onChange={() => {}} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("disables prev button on first page", () => {
    render(<Pagination page={1} totalPages={5} onChange={() => {}} />);
    const buttons = screen.getAllByRole("button");
    const prevBtn = buttons[0];
    expect(prevBtn).toBeDisabled();
  });

  it("disables next button on last page", () => {
    render(<Pagination page={5} totalPages={5} onChange={() => {}} />);
    const buttons = screen.getAllByRole("button");
    const nextBtn = buttons[buttons.length - 1];
    expect(nextBtn).toBeDisabled();
  });
});