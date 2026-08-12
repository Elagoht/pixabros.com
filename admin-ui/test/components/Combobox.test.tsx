import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Formik, Form } from "formik";
import { Combobox } from "@/components/ui";

vi.mock("@/lib/stores/i18n", () => ({
  useI18n: () => ({
    locale: "en",
    t: (key: string) => key,
  }),
}));

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
  { label: "Apple", value: "apple" },
  { label: "Banana", value: "banana" },
  { label: "Cherry", value: "cherry" },
];

describe("Combobox", () => {
  it("renders with label", () => {
    renderWithFormik(<Combobox name="fruit" label="Fruit" options={options} />, {
      fruit: "",
    });
    expect(screen.getByText("Fruit")).toBeInTheDocument();
  });

it("renders with placeholder text", () => {
    renderWithFormik(<Combobox name="fruit" options={options} />, { fruit: "" });
    expect(screen.getByText("Select...")).toBeInTheDocument();
  });

  it("shows dropdown options on click", () => {
    renderWithFormik(<Combobox name="fruit" options={options} />, { fruit: "" });
    const comboboxArea = screen.getByText("Select...");
    fireEvent.click(comboboxArea);
    expect(screen.getByText("Apple")).toBeInTheDocument();
    expect(screen.getByText("Banana")).toBeInTheDocument();
    expect(screen.getByText("Cherry")).toBeInTheDocument();
  });

  it("selects an option in single mode", () => {
    renderWithFormik(<Combobox name="fruit" options={options} />, { fruit: "" });
    fireEvent.click(screen.getByText("Select..."));
    fireEvent.click(screen.getByText("Banana"));
  });

  it("filters options based on search in single mode", () => {
    renderWithFormik(<Combobox name="fruit" options={options} />, { fruit: "" });
    fireEvent.click(screen.getByText("Select..."));
    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.change(searchInput, { target: { value: "ch" } });
    expect(screen.getByText("Cherry")).toBeInTheDocument();
  });
});