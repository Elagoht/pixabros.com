import { fireEvent, render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { Input } from "@/components/ui";

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

describe("Input", () => {
  it("renders with label", () => {
    renderWithFormik(<Input name="email" label="Email" />, { email: "" });
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<Input name="email" />, { email: "" });
    expect(screen.queryByLabelText("Email")).not.toBeInTheDocument();
  });

  it("applies initial value from formik", () => {
    renderWithFormik(<Input name="email" />, { email: "test@example.com" });
    expect(screen.getByDisplayValue("test@example.com")).toBeInTheDocument();
  });

  it("updates value on change", () => {
    renderWithFormik(<Input name="email" />, { email: "" });
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "hello@test.com" } });
    expect(screen.getByDisplayValue("hello@test.com")).toBeInTheDocument();
  });

  it("shows error message when touched and error exists", () => {
    render(
      <Formik
        initialValues={{ email: "" }}
        initialErrors={{ email: "Required" }}
        initialTouched={{ email: true }}
        onSubmit={() => {}}
      >
        <Form>
          <Input name="email" label="Email" />
        </Form>
      </Formik>,
    );
    expect(screen.getByText("Required")).toBeInTheDocument();
  });

  it("does not show error when not touched", () => {
    render(
      <Formik
        initialValues={{ email: "" }}
        initialErrors={{ email: "Required" }}
        onSubmit={() => {}}
      >
        <Form>
          <Input name="email" />
        </Form>
      </Formik>,
    );
    expect(screen.queryByText("Required")).not.toBeInTheDocument();
  });

  it("renders password type input", () => {
    renderWithFormik(<Input name="pw" type="password" />, { pw: "" });
    const input = screen.getByDisplayValue("") as HTMLInputElement;
    expect(input.type).toBe("password");
  });

  it("toggles password visibility", () => {
    renderWithFormik(<Input name="pw" type="password" />, { pw: "secret" });
    const toggleBtn = screen.getByRole("button");
    const input = screen.getByDisplayValue("secret") as HTMLInputElement;
    expect(input.type).toBe("password");
    fireEvent.click(toggleBtn);
    expect(input.type).toBe("text");
    fireEvent.click(toggleBtn);
    expect(input.type).toBe("password");
  });

  it("does not show password toggle for non-password types", () => {
    renderWithFormik(<Input name="email" type="email" />, { email: "" });
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
