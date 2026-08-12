import { render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { Select } from "@/components/ui";

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
  { label: "Option A", value: "a" },
  { label: "Option B", value: "b" },
  { label: "Option C", value: "c" },
];

describe("Select", () => {
  it("renders with label", () => {
    renderWithFormik(
      <Select name="choice" label="Choose" options={options} />,
      {
        choice: "",
      },
    );
    expect(screen.getByText("Choose")).toBeInTheDocument();
  });

  it("renders all options in single mode", () => {
    renderWithFormik(<Select name="choice" options={options} />, {
      choice: "",
    });
    expect(screen.getByText("Option A")).toBeInTheDocument();
    expect(screen.getByText("Option B")).toBeInTheDocument();
    expect(screen.getByText("Option C")).toBeInTheDocument();
  });

  it("renders placeholder option", () => {
    renderWithFormik(
      <Select name="choice" options={options} placeholder="Pick one" />,
      { choice: "" },
    );
    expect(screen.getByText("Pick one")).toBeInTheDocument();
  });

  it("has a select element with correct name", () => {
    renderWithFormik(<Select name="choice" options={options} />, {
      choice: "",
    });
    const select = screen.getByRole("combobox");
    expect(select).toHaveAttribute("name", "choice");
  });

  it("renders select options in dropdown", () => {
    renderWithFormik(<Select name="choice" options={options} />, {
      choice: "",
    });
    const select = screen.getByRole("combobox") as HTMLSelectElement;
    expect(select.options.length).toBe(3);
  });
});
