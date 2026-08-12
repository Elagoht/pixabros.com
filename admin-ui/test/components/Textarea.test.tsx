import { fireEvent, render, screen } from "@testing-library/react";
import { Form, Formik } from "formik";
import { describe, expect, it } from "vitest";
import { Textarea } from "@/components/ui";

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

describe("Textarea", () => {
  it("renders with label", () => {
    renderWithFormik(<Textarea name="bio" label="Bio" />, { bio: "" });
    expect(screen.getByLabelText("Bio")).toBeInTheDocument();
  });

  it("renders without label", () => {
    renderWithFormik(<Textarea name="bio" />, { bio: "" });
    expect(screen.queryByLabelText("Bio")).not.toBeInTheDocument();
  });

  it("renders textarea element", () => {
    renderWithFormik(<Textarea name="bio" />, { bio: "" });
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("shows error message when touched and error exists", () => {
    render(
      <Formik
        initialValues={{ bio: "" }}
        initialErrors={{ bio: "Too short" }}
        initialTouched={{ bio: true }}
        onSubmit={() => {}}
      >
        <Form>
          <Textarea name="bio" label="Bio" />
        </Form>
      </Formik>,
    );
    expect(screen.getByText("Too short")).toBeInTheDocument();
  });

  it("does not show error when not touched", () => {
    render(
      <Formik
        initialValues={{ bio: "" }}
        initialErrors={{ bio: "Too short" }}
        onSubmit={() => {}}
      >
        <Form>
          <Textarea name="bio" />
        </Form>
      </Formik>,
    );
    expect(screen.queryByText("Too short")).not.toBeInTheDocument();
  });

  it("updates value on input", () => {
    renderWithFormik(<Textarea name="bio" />, { bio: "" });
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: "Hello world" } });
    expect(screen.getByDisplayValue("Hello world")).toBeInTheDocument();
  });

  it("defaults to 4 rows", () => {
    renderWithFormik(<Textarea name="bio" />, { bio: "" });
    const textarea = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(textarea.rows).toBe(4);
  });
});
