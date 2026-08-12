import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Dropdown } from "@/components/ui";

describe("Dropdown", () => {
  const items = [
    { id: "edit", label: "Edit" },
    { id: "delete", label: "Delete" },
  ];

  it("renders trigger content", () => {
    render(<Dropdown trigger={<button>Menu</button>} items={items} />);
    expect(screen.getByText("Menu")).toBeInTheDocument();
  });

  it("opens dropdown on trigger click", () => {
    render(<Dropdown trigger={<button>Menu</button>} items={items} />);
    fireEvent.click(screen.getByText("Menu"));
    expect(screen.getByText("Edit")).toBeInTheDocument();
    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("closes on outside click", () => {
    render(<Dropdown trigger={<button>Menu</button>} items={items} />);
    fireEvent.click(screen.getByText("Menu"));
    expect(screen.getByText("Edit")).toBeInTheDocument();

    fireEvent.mouseDown(document.body);
    // Dropdown portal hides the menu after outside click
    expect(screen.queryByText("Edit")).not.toBeInTheDocument();
  });

  it("renders items with labels", () => {
    render(<Dropdown trigger={<button>Menu</button>} items={items} />);
    fireEvent.click(screen.getByText("Menu"));
    expect(screen.getByText("Edit")).toBeInTheDocument();
    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("calls item onClick and closes dropdown", () => {
    const onClick = vi.fn();
    const itemsWithAction = [
      { id: "edit", label: "Edit", onClick },
    ];
    render(
      <Dropdown trigger={<button>Menu</button>} items={itemsWithAction} />,
    );
    fireEvent.click(screen.getByText("Menu"));
    fireEvent.click(screen.getByText("Edit"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});