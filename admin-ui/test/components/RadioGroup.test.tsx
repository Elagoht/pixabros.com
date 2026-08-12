import { fireEvent, render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { RadioGroup } from "@/components/ui";

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
  { label: "Small", value: "sm" },
  { label: "Medium", value: "md" },
  { label: "Large", value: "lg" },
];

describe("RadioGroup", () => {
  it("renders all options", () => {
    renderWithFormik(<RadioGroup name="size" options={options} />, {
      size: "",
    });
    expect(screen.getByText("Small")).toBeInTheDocument();
    expect(screen.getByText("Medium")).toBeInTheDocument();
    expect(screen.getByText("Large")).toBeInTheDocument();
  });

  it("renders with label", () => {
    renderWithFormik(
      <RadioGroup name="size" label="Select size" options={options} />,
      { size: "" },
    );
    expect(screen.getByText("Select size")).toBeInTheDocument();
  });

  it("selects option on click", () => {
    renderWithFormik(<RadioGroup name="size" options={options} />, {
      size: "",
    });
    fireEvent.click(screen.getByLabelText("Medium"));
    const mediumRadio = screen.getByLabelText("Medium") as HTMLInputElement;
    expect(mediumRadio).toBeChecked();
  });

  it("renders radio inputs with correct name", () => {
    renderWithFormik(<RadioGroup name="size" options={options} />, {
      size: "",
    });
    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(3);
    for (const radio of radios) {
      expect(radio).toHaveAttribute("name", "size");
    }
  });
});
