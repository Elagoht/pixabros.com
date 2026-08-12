import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Formik, Form } from "formik";
import { DateTimePicker } from "@/components/ui";

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

describe("DateTimePicker", () => {
  it("renders input with label", () => {
    renderWithFormik(
      <DateTimePicker name="datetime" label="Start Date" />,
      { datetime: "" },
    );
    expect(screen.getByText("Start Date")).toBeInTheDocument();
  });

  it("renders button when no value", () => {
    renderWithFormik(<DateTimePicker name="datetime" />, { datetime: "" });
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<DateTimePicker name="datetime" />, { datetime: "" });
    expect(screen.queryByText("Start Date")).not.toBeInTheDocument();
  });
});