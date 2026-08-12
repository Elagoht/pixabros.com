import { render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it, vi } from "vitest";
import { DatePicker } from "@/components/ui";

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

describe("DatePicker", () => {
  it("renders input with label", () => {
    renderWithFormik(<DatePicker name="date" label="Birth Date" />, {
      date: "",
    });
    expect(screen.getByText("Birth Date")).toBeInTheDocument();
  });

  it("renders button with placeholder when no value", () => {
    renderWithFormik(<DatePicker name="date" />, { date: "" });
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<DatePicker name="date" />, { date: "" });
    expect(screen.queryByLabelText("Birth Date")).not.toBeInTheDocument();
  });
});
