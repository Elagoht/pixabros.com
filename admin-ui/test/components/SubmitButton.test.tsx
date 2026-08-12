import { render, screen } from "@testing-library/react";
import { FormikProvider, useFormik } from "formik";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { SubmitButton } from "@/components/ui";

vi.mock("@/lib/stores/i18n", () => ({
  useI18n: vi.fn(() => ({
    t: (key: string) => key,
    locale: "en",
    setLocale: vi.fn(),
  })),
}));

function TestWrapper({ isSubmitting = false }: { isSubmitting?: boolean }) {
  const formik = useFormik({
    initialValues: { name: "" },
    onSubmit: () => {},
  });

  // Override isSubmitting for test
  Object.defineProperty(formik, "isSubmitting", {
    value: isSubmitting,
    writable: false,
  });

  return (
    <MemoryRouter>
      <FormikProvider value={formik}>
        <SubmitButton>Save</SubmitButton>
      </FormikProvider>
    </MemoryRouter>
  );
}

describe("SubmitButton", () => {
  it("renders children text", () => {
    render(<TestWrapper />);
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("is disabled when form is submitting", () => {
    render(<TestWrapper isSubmitting />);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("shows loading icon class when submitting", () => {
    render(<TestWrapper isSubmitting />);
    const button = screen.getByRole("button");
    // The button should be disabled during submission
    expect(button).toBeDisabled();
    // The loading icon with animate-spin should be present
    const svg = button.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.classList.contains("animate-spin")).toBe(true);
  });
});
