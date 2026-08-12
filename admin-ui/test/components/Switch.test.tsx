import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Formik, Form } from "formik";
import { Switch } from "@/components/ui";

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

describe("Switch", () => {
  it("renders label", () => {
    renderWithFormik(<Switch name="darkMode" label="Dark Mode" />, {
      darkMode: false,
    });
    expect(screen.getByText("Dark Mode")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<Switch name="darkMode" />, { darkMode: false });
    expect(screen.queryByText("Dark Mode")).not.toBeInTheDocument();
  });

  it("has role=switch", () => {
    renderWithFormik(<Switch name="darkMode" label="Dark Mode" />, {
      darkMode: false,
    });
    expect(screen.getByRole("switch")).toBeInTheDocument();
  });

  it("has aria-checked=false when off", () => {
    renderWithFormik(<Switch name="darkMode" />, { darkMode: false });
    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "false");
  });

  it("has aria-checked=true when on", () => {
    renderWithFormik(<Switch name="darkMode" />, { darkMode: true });
    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "true");
  });

  it("toggles value on click", () => {
    renderWithFormik(<Switch name="darkMode" label="Dark Mode" />, {
      darkMode: false,
    });
    const btn = screen.getByRole("switch");
    expect(btn).toHaveAttribute("aria-checked", "false");
    fireEvent.click(btn);
    expect(btn).toHaveAttribute("aria-checked", "true");
  });
});