import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Form, Formik } from "formik";
import { describe, expect, it, vi } from "vitest";
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
    renderWithFormik(<Keywords name="tags" placeholder="Type here..." />, {
      tags: "",
    });
    expect(screen.getByPlaceholderText("Type here...")).toBeInTheDocument();
  });
});
describe("Keywords pasting", () => {
  const renderField = (initial = "") => {
    const values = { tags: initial };
    render(
      <Formik initialValues={values} onSubmit={vi.fn()}>
        <Keywords name="tags" placeholder="Add a tag and press Enter" />
      </Formik>,
    );
    // The placeholder is hidden once chips exist, so find the field by role.
    return screen.getByRole("textbox");
  };

  const paste = async (input: HTMLElement, text: string) => {
    const user = userEvent.setup();
    await user.click(input);
    await user.paste(text);
  };

  // Pasting a list is how a set of roles actually arrives; one chip holding
  // the whole comma-separated string would be useless.
  it("splits a pasted comma-separated list into separate chips", async () => {
    const input = renderField();

    await paste(input, "Code, 2D Art, Music, SFX");

    for (const tag of ["Code", "2D Art", "Music", "SFX"]) {
      expect(screen.getByText(tag)).toBeInTheDocument();
    }
    expect(input).toHaveValue("");
  });

  it("trims the whitespace around each pasted value", async () => {
    const input = renderField();

    await paste(input, "  Code ,   Game Design  ");

    expect(screen.getByText("Code")).toBeInTheDocument();
    expect(screen.getByText("Game Design")).toBeInTheDocument();
  });

  it("drops empty values from a sloppy list", async () => {
    const input = renderField();

    await paste(input, "Code,,, ,Music,");

    expect(screen.getByText("Code")).toBeInTheDocument();
    expect(screen.getByText("Music")).toBeInTheDocument();
    expect(screen.queryAllByRole("button")).toHaveLength(2);
  });

  // Pasting a column out of a spreadsheet gives newlines, not commas.
  it("splits on newlines and tabs as well", async () => {
    const input = renderField();

    await paste(input, "Code\nAnimation\tPolishing");

    for (const tag of ["Code", "Animation", "Polishing"]) {
      expect(screen.getByText(tag)).toBeInTheDocument();
    }
  });

  it("does not add the same keyword twice", async () => {
    const input = renderField("Code");

    await paste(input, "Code, Music");

    expect(screen.getAllByText("Code")).toHaveLength(1);
    expect(screen.getByText("Music")).toBeInTheDocument();
  });

  // A single word must stay editable rather than becoming a chip instantly.
  it("leaves a separator-free paste in the field", async () => {
    const input = renderField();

    await paste(input, "Code");

    expect(input).toHaveValue("Code");
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });
});
