import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import ColorPicker from "@/components/ui/ColorPicker";
import { required } from "../utils/dom";

describe("ColorPicker", () => {
  it("renders with color value", () => {
    const onChange = vi.fn();
    render(<ColorPicker value="#ff0000" onChange={onChange} />);
    const input = screen.getByDisplayValue("#FF0000");
    expect(input).toBeInTheDocument();
  });

  it("displays color swatch with background", () => {
    const onChange = vi.fn();
    const { container } = render(
      <ColorPicker value="#00ff00" onChange={onChange} />,
    );
    const swatch = container.querySelector('[style*="background"]');
    expect(swatch).toBeTruthy();
    expect((swatch as HTMLElement).style.background).toBe("rgb(0, 255, 0)");
  });

  it("calls onChange when valid hex is entered", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<ColorPicker value="#ff0000" onChange={onChange} />);
    const input = screen.getByDisplayValue("#FF0000");
    await user.clear(input);
    await user.type(input, "#0000FF");
    expect(onChange).toHaveBeenCalled();
  });

  it("toggles picker panel on button click", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    const { container } = render(
      <ColorPicker value="#ff0000" onChange={onChange} />,
    );
    const button = required(
      container.querySelector("button"),
      "the colour swatch button",
    );
    await user.click(button);
    expect(screen.getByText("H")).toBeInTheDocument();
  });
});
