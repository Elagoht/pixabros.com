import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Formik, Form } from "formik";
import { Checkbox } from "@/components/ui";

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

describe("Checkbox", () => {
  it("renders with label", () => {
    renderWithFormik(<Checkbox name="agree" label="I agree" />, { agree: false });
    expect(screen.getByText("I agree")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<Checkbox name="agree" />, { agree: false });
    expect(screen.queryByText("I agree")).not.toBeInTheDocument();
  });

  it("toggles checked state on click", () => {
    renderWithFormik(<Checkbox name="agree" label="I agree" />, { agree: false });
    const checkbox = screen.getByRole("checkbox");
    expect(checkbox).not.toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).toBeChecked();
  });

  it("unchecks when clicking a checked checkbox", () => {
    renderWithFormik(<Checkbox name="agree" label="I agree" />, { agree: true });
    const checkbox = screen.getByRole("checkbox");
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    expect(checkbox).not.toBeChecked();
  });

  it("uses id based on name and value when value is provided", () => {
    renderWithFormik(
      <Checkbox name="colors" value="red" label="Red" />,
      { colors: [] },
    );
    expect(screen.getByLabelText("Red")).toHaveAttribute("id", "colors-red");
  });

  it("uses id based on name only when value is not provided", () => {
    renderWithFormik(<Checkbox name="agree" label="I agree" />, { agree: false });
    expect(screen.getByLabelText("I agree")).toHaveAttribute("id", "agree");
  });
});