import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Tabs } from "@/components/ui";

const items = [
  { value: "tab1", label: "Tab 1", content: "Content 1" },
  { value: "tab2", label: "Tab 2", content: "Content 2" },
  { value: "tab3", label: "Tab 3", content: "Content 3" },
];

describe("Tabs", () => {
  it("renders tab labels", () => {
    render(<Tabs items={items} />);
    expect(screen.getByText("Tab 1")).toBeInTheDocument();
    expect(screen.getByText("Tab 2")).toBeInTheDocument();
    expect(screen.getByText("Tab 3")).toBeInTheDocument();
  });

  it("defaults to first tab content", () => {
    render(<Tabs items={items} />);
    expect(screen.getByText("Content 1")).toBeInTheDocument();
  });

  it("switches active tab on click", () => {
    render(<Tabs items={items} />);
    fireEvent.click(screen.getByText("Tab 2"));
    expect(screen.getByText("Content 2")).toBeInTheDocument();
    expect(screen.queryByText("Content 1")).not.toBeInTheDocument();
  });

  it("calls onChange when tab is clicked", () => {
    const onChange = vi.fn();
    render(<Tabs items={items} onChange={onChange} />);
    fireEvent.click(screen.getByText("Tab 2"));
    expect(onChange).toHaveBeenCalledWith("tab2");
  });

  it("respects defaultValue prop", () => {
    render(<Tabs items={items} defaultValue="tab2" />);
    expect(screen.getByText("Content 2")).toBeInTheDocument();
  });
});
