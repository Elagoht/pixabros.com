import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import CategoryTable from "@/components/ui/CategoryTable";

const categories: HierarchyNode<{ name: string }>[] = [
  { id: "1", parentId: null, data: { name: "Electronics" } },
  { id: "2", parentId: "1", data: { name: "Phones" } },
  { id: "3", parentId: null, data: { name: "Books" } },
];

describe("CategoryTable", () => {
  it("renders categories", () => {
    render(
      <CategoryTable
        categories={categories}
        getCategoryLabel={(c) => c.data.name}
        onSave={vi.fn()}
        onCreate={vi.fn()}
        onDelete={vi.fn()}
        renderFormFields={() => <div>Form fields</div>}
        initialFormData={{ name: "" }}
      />,
    );
    expect(screen.getAllByText("Electronics").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Phones").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Books").length).toBeGreaterThanOrEqual(1);
  });

  it("renders create button", () => {
    render(
      <CategoryTable
        categories={categories}
        getCategoryLabel={(c) => c.data.name}
        onSave={vi.fn()}
        onCreate={vi.fn()}
        onDelete={vi.fn()}
        renderFormFields={() => <div>Form fields</div>}
        initialFormData={{ name: "" }}
      />,
    );
    expect(screen.getByText("Categories")).toBeInTheDocument();
  });

  it("shows empty state when no categories", () => {
    render(
      <CategoryTable
        categories={[]}
        getCategoryLabel={(c) => c.data.name}
        onSave={vi.fn()}
        onCreate={vi.fn()}
        onDelete={vi.fn()}
        renderFormFields={() => <div>Form fields</div>}
        initialFormData={{ name: "" }}
      />,
    );
    expect(screen.getByText("No categories")).toBeInTheDocument();
  });
});