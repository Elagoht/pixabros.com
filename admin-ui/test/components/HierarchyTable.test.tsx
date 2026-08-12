import { render, screen } from "@testing-library/react";
import { Formik } from "formik";
import { describe, expect, it } from "vitest";
import HierarchyTable from "@/components/ui/HierarchyTable";

const items: HierarchyNode<{ label: string }>[] = [
  { id: "1", parentId: null, data: { label: "Item 1" } },
  { id: "2", parentId: "1", data: { label: "Item 2" } },
  { id: "3", parentId: null, data: { label: "Item 3" } },
];

const renderTable = () =>
  render(
    <Formik initialValues={{ nodes: items }} onSubmit={() => {}}>
      <HierarchyTable
        name="nodes"
        renderRow={(node) => <>{(node.data as { label: string }).label}</>}
      />
    </Formik>,
  );

describe("HierarchyTable", () => {
  it("renders tree items", () => {
    renderTable();
    expect(screen.getByText("Item 1")).toBeInTheDocument();
    expect(screen.getByText("Item 2")).toBeInTheDocument();
    expect(screen.getByText("Item 3")).toBeInTheDocument();
  });

  it("renders save button", () => {
    renderTable();
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("renders hierarchy header", () => {
    renderTable();
    expect(screen.getByText("Hierarchy")).toBeInTheDocument();
  });
});
