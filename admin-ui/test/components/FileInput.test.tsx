import { render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it, vi } from "vitest";
import { FileInput } from "@/components/ui";

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

describe("FileInput", () => {
  it("renders with label", () => {
    renderWithFormik(<FileInput name="file" label="Upload" />, { file: null });
    expect(screen.getByText("Upload")).toBeInTheDocument();
  });

  it("renders file input element", () => {
    renderWithFormik(<FileInput name="file" />, { file: null });
    expect(screen.getByRole("button", { name: /browse/i })).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<FileInput name="file" />, { file: null });
    expect(screen.queryByText("Upload")).not.toBeInTheDocument();
  });

  it("applies accept prop to hidden input", () => {
    renderWithFormik(<FileInput name="file" accept=".pdf,.doc" />, {
      file: null,
    });
    const hiddenInput = document.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    expect(hiddenInput).toHaveAttribute("accept", ".pdf,.doc");
  });

  it("shows placeholder when no file is selected", () => {
    renderWithFormik(<FileInput name="file" />, { file: null });
    expect(screen.getByText("common.noFileChosen")).toBeInTheDocument();
  });
});
