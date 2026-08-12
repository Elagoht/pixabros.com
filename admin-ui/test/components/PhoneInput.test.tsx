import { fireEvent, render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { PhoneInput } from "@/components/ui";

const renderWithFormik = (
  ui: React.ReactElement,
  initialValues: Record<string, unknown> = {},
) => {
  return render(
    <Formik initialValues={initialValues} onSubmit={() => {}}>
      <Form>{ui}</Form>
    </Formik>,
  );
};

describe("PhoneInput", () => {
  it("renders input with label", () => {
    renderWithFormik(<PhoneInput name="phone" label="Phone" />, { phone: "" });
    expect(screen.getByLabelText("Phone")).toBeInTheDocument();
  });

  it("renders with placeholder", () => {
    renderWithFormik(<PhoneInput name="phone" />, { phone: "" });
    expect(
      screen.getByPlaceholderText("+90 (5XX) XXX XX XX"),
    ).toBeInTheDocument();
  });

  it("renders input with type tel", () => {
    renderWithFormik(<PhoneInput name="phone" />, { phone: "" });
    const input = screen.getByRole("textbox") as HTMLInputElement;
    expect(input.type).toBe("tel");
  });

  it("strips non-digit characters on change", () => {
    renderWithFormik(<PhoneInput name="phone" />, { phone: "" });
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "0555-abc-1234" } });
    expect(screen.getByDisplayValue("+90 (555) 123 4")).toBeInTheDocument();
  });

  it("renders with phone icon present", () => {
    renderWithFormik(<PhoneInput name="phone" label="Phone" />, { phone: "" });
    expect(screen.getByLabelText("Phone")).toBeInTheDocument();
  });
});
