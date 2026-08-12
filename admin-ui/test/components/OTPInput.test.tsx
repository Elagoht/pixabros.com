import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OTPInput } from "@/components/ui";

describe("OTPInput", () => {
  it("renders correct number of inputs by default", () => {
    render(<OTPInput value="" onChange={() => {}} />);
    const inputs = screen.getAllByRole("textbox");
    expect(inputs).toHaveLength(6);
  });

  it("renders correct number of inputs with custom digits prop", () => {
    render(<OTPInput value="" onChange={() => {}} digits={4} />);
    const inputs = screen.getAllByRole("textbox");
    expect(inputs).toHaveLength(4);
  });

  it("displays value across inputs", () => {
    render(<OTPInput value="123456" onChange={() => {}} />);
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    expect(inputs[0]).toHaveValue("1");
    expect(inputs[1]).toHaveValue("2");
    expect(inputs[2]).toHaveValue("3");
    expect(inputs[3]).toHaveValue("4");
    expect(inputs[4]).toHaveValue("5");
    expect(inputs[5]).toHaveValue("6");
  });

  it("calls onChange when typing in an input", () => {
    const onChange = vi.fn();
    render(<OTPInput value="" onChange={onChange} />);
    const inputs = screen.getAllByRole("textbox");
    fireEvent.change(inputs[0], { target: { value: "5" } });
    expect(onChange).toHaveBeenCalledWith("5");
  });

  it("has inputMode numeric on inputs", () => {
    render(<OTPInput value="" onChange={() => {}} />);
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    for (const input of inputs) {
      expect(input).toHaveAttribute("inputmode", "numeric");
    }
  });

  it("disables inputs when disabled prop is set", () => {
    render(<OTPInput value="" onChange={() => {}} disabled />);
    const inputs = screen.getAllByRole("textbox") as HTMLInputElement[];
    for (const input of inputs) {
      expect(input).toBeDisabled();
    }
  });
});
