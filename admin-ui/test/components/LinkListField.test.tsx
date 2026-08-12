import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Formik } from "formik";
import { describe, expect, it, vi } from "vitest";
import { LinkListField } from "@/components/ui";

const renderField = (initial: { label: string; url: string }[]) => {
  const onSubmit = vi.fn();
  render(
    <Formik initialValues={{ links: initial }} onSubmit={onSubmit}>
      <LinkListField
        name="links"
        labelPlaceholder="itch.io"
        urlPlaceholder="https://example.com"
        addLabel="Add link"
        emptyLabel="No links yet."
        removeLabel="Remove"
      />
    </Formik>,
  );
};

describe("LinkListField", () => {
  it("shows the empty message when there are no links", () => {
    renderField([]);
    expect(screen.getByText("No links yet.")).toBeInTheDocument();
  });

  it("renders a label and url input per link", () => {
    renderField([
      { label: "Itch", url: "https://a.dev" },
      { label: "Steam", url: "https://b.dev" },
    ]);

    expect(screen.queryByText("No links yet.")).toBeNull();
    expect(screen.getByDisplayValue("Itch")).toBeInTheDocument();
    expect(screen.getByDisplayValue("https://a.dev")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Steam")).toBeInTheDocument();
    expect(screen.getByDisplayValue("https://b.dev")).toBeInTheDocument();
  });

  it("appends an empty row when adding", async () => {
    const user = userEvent.setup();
    renderField([{ label: "Itch", url: "https://a.dev" }]);

    await user.click(screen.getByRole("button", { name: "Add link" }));

    expect(screen.getAllByPlaceholderText("itch.io")).toHaveLength(2);
  });

  it("removes the row that was clicked, not the last one", async () => {
    const user = userEvent.setup();
    renderField([
      { label: "Itch", url: "https://a.dev" },
      { label: "Steam", url: "https://b.dev" },
    ]);

    const [firstRemove] = screen.getAllByTitle("Remove");
    await user.click(firstRemove);

    expect(screen.queryByDisplayValue("Itch")).toBeNull();
    expect(screen.getByDisplayValue("Steam")).toBeInTheDocument();
  });

  // A single row has nowhere to move, so the grip would be noise.
  it("only offers drag handles once there is more than one link", () => {
    renderField([{ label: "Itch", url: "https://a.dev" }]);
    const single = document.querySelectorAll(".cursor-grab").length;
    expect(single).toBe(0);
  });

  it("offers a drag handle per row when several links exist", () => {
    renderField([
      { label: "Itch", url: "https://a.dev" },
      { label: "Steam", url: "https://b.dev" },
    ]);
    expect(document.querySelectorAll(".cursor-grab")).toHaveLength(2);
  });
});
