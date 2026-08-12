import { fireEvent, render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { CheckboxGroup } from "@/components/ui";

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

const options = [
  { label: "Red", value: "red" },
  { label: "Green", value: "green" },
  { label: "Blue", value: "blue" },
];

describe("CheckboxGroup", () => {
  it("renders all options", () => {
    renderWithFormik(<CheckboxGroup name="colors" options={options} />, {
      colors: [],
    });
    expect(screen.getByText("Red")).toBeInTheDocument();
    expect(screen.getByText("Green")).toBeInTheDocument();
    expect(screen.getByText("Blue")).toBeInTheDocument();
  });

  it("renders with label", () => {
    renderWithFormik(
      <CheckboxGroup name="colors" label="Pick colors" options={options} />,
      { colors: [] },
    );
    expect(screen.getByText("Pick colors")).toBeInTheDocument();
  });

  it("toggles selection on click", () => {
    renderWithFormik(<CheckboxGroup name="colors" options={options} />, {
      colors: [],
    });
    const redCheckbox = screen.getByLabelText("Red");
    fireEvent.click(redCheckbox);
    expect(redCheckbox).toBeChecked();
  });

  it("deselects on second click", () => {
    renderWithFormik(<CheckboxGroup name="colors" options={options} />, {
      colors: ["red"],
    });
    const redCheckbox = screen.getByLabelText("Red");
    expect(redCheckbox).toBeChecked();
    fireEvent.click(redCheckbox);
    expect(redCheckbox).not.toBeChecked();
  });

  it("allows multiple selections", () => {
    renderWithFormik(<CheckboxGroup name="colors" options={options} />, {
      colors: [],
    });
    fireEvent.click(screen.getByLabelText("Red"));
    fireEvent.click(screen.getByLabelText("Blue"));
    expect(screen.getByLabelText("Red")).toBeChecked();
    expect(screen.getByLabelText("Blue")).toBeChecked();
    expect(screen.getByLabelText("Green")).not.toBeChecked();
  });
});
