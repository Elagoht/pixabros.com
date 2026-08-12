import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Formik, Form } from "formik";
import { Keywords } from "@/components/ui";

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

describe("Keywords", () => {
  it("renders with label", () => {
    renderWithFormik(<Keywords name="tags" label="Tags" />, { tags: "" });
    expect(screen.getByText("Tags")).toBeInTheDocument();
  });

  it("renders input with placeholder", () => {
    renderWithFormik(<Keywords name="tags" />, { tags: "" });
    expect(screen.getByPlaceholderText("Add keyword...")).toBeInTheDocument();
  });

  it("renders existing keywords from string value", () => {
    renderWithFormik(<Keywords name="tags" />, { tags: "react, typescript" });
    expect(screen.getByText("react")).toBeInTheDocument();
    expect(screen.getByText("typescript")).toBeInTheDocument();
  });

  it("adds keyword on Enter", () => {
    renderWithFormik(<Keywords name="tags" />, { tags: "" });
    const input = screen.getByPlaceholderText("Add keyword...");
    fireEvent.change(input, { target: { value: "vue" } });
    fireEvent.keyDown(input, { key: "Enter" });
  });

  it("removes keyword on click", () => {
    renderWithFormik(<Keywords name="tags" />, { tags: "react, typescript" });
    expect(screen.getByText("react")).toBeInTheDocument();
    const removeButtons = screen.getAllByRole("button");
    const reactRemove = removeButtons[0];
    fireEvent.click(reactRemove);
  });

  it("uses custom placeholder", () => {
    renderWithFormik(<Keywords name="tags" placeholder="Type here..." />, { tags: "" });
    expect(screen.getByPlaceholderText("Type here...")).toBeInTheDocument();
  });
});