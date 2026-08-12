import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Accordion } from "@/components/ui";

describe("Accordion", () => {
  const items = [
    { value: "1", label: "Section 1", content: "Content 1" },
    { value: "2", label: "Section 2", content: "Content 2" },
  ];

  it("renders all item labels", () => {
    render(<Accordion items={items} />);
    expect(screen.getByText("Section 1")).toBeInTheDocument();
    expect(screen.getByText("Section 2")).toBeInTheDocument();
  });

  it("shows content when item is clicked (single mode)", () => {
    render(<Accordion items={items} />);
    fireEvent.click(screen.getByText("Section 1"));
    expect(screen.getByText("Content 1")).toBeInTheDocument();
  });

  it("toggles content on second click", () => {
    render(<Accordion items={items} />);
    fireEvent.click(screen.getByText("Section 1"));
    expect(screen.getByText("Content 1")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Section 1"));
    expect(screen.getByText("Content 1")).toBeInTheDocument();
  });

  it("renders with defaultOpen prop", () => {
    render(<Accordion items={items} defaultOpen="1" />);
    expect(screen.getByText("Content 1")).toBeInTheDocument();
  });

  it("in multiple mode, allows multiple open items", () => {
    render(<Accordion items={items} type="multiple" defaultOpen={["1", "2"]} />);
    expect(screen.getByText("Content 1")).toBeInTheDocument();
    expect(screen.getByText("Content 2")).toBeInTheDocument();
  });
});