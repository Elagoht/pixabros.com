import { IconPencil, IconTrash } from "@tabler/icons-react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderActions } from "@/utilities/format";

interface Row {
  id: string;
}

const row: Row = { id: "g1" };

describe("renderActions", () => {
  it("renders nothing when there are no actions", () => {
    const { container } = render(renderActions(row, []));
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the action label as a tooltip on hover", async () => {
    const user = userEvent.setup();
    render(
      renderActions(row, [
        { icon: IconPencil, label: "Edit", onClick: vi.fn() },
      ]),
    );

    expect(screen.queryByText("Edit")).toBeNull();

    await user.hover(screen.getByRole("button"));

    expect(await screen.findByText("Edit")).toBeInTheDocument();
  });

  it("calls the action when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(renderActions(row, [{ icon: IconTrash, label: "Delete", onClick }]));

    await user.click(screen.getByRole("button"));

    expect(onClick).toHaveBeenCalledWith(row);
  });

  it("disables an action whose predicate says so", () => {
    render(
      renderActions(row, [
        {
          icon: IconPencil,
          label: "Edit",
          disabled: (r: Row) => r.id === "g1",
          onClick: vi.fn(),
        },
      ]),
    );

    expect(screen.getByRole("button")).toBeDisabled();
  });

  // A disabled Button has pointer-events: none, so the tooltip has to be
  // driven by a wrapper -- the label explaining why it is unavailable is
  // exactly what a user needs at that moment.
  it("still shows the tooltip for a disabled action", async () => {
    const user = userEvent.setup();
    render(
      renderActions(row, [
        {
          icon: IconPencil,
          label: "Not available",
          disabled: true,
          onClick: vi.fn(),
        },
      ]),
    );

    await user.hover(screen.getByRole("button").parentElement as HTMLElement);

    expect(await screen.findByText("Not available")).toBeInTheDocument();
  });
});
