import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Chip } from "@/components/ui";

describe("Chip", () => {
  it("renders children text", () => {
    render(<Chip onRemove={() => {}}>React</Chip>);
    expect(screen.getByText("React")).toBeInTheDocument();
  });

  it("calls onRemove when close button is clicked", () => {
    const onRemove = vi.fn();
    render(<Chip onRemove={onRemove}>Tag</Chip>);
    const button = screen.getByRole("button");
    fireEvent.click(button);
    expect(onRemove).toHaveBeenCalledOnce();
  });

  it("renders remove icon", () => {
    render(<Chip onRemove={() => {}}>Test</Chip>);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });
});
