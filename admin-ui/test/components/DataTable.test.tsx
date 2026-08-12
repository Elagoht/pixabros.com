import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import DataTable from "@/components/ui/DataTable";

const columns: DataTableColumn<{ id: string; name: string; age: number }>[] = [
  { id: "name", header: "Name", accessor: "name" },
  { id: "age", header: "Age", accessor: "age" },
];

const data = [
  { id: "1", name: "Alice", age: 30 },
  { id: "2", name: "Bob", age: 25 },
];

describe("DataTable", () => {
  it("renders with columns and rows", () => {
    render(
      <DataTable
        columns={columns}
        data={data}
        getRowId={(row) => row.id}
        completeData
      />,
    );
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
    expect(screen.getByText("30")).toBeInTheDocument();
    expect(screen.getAllByText("25").length).toBeGreaterThan(0);
  });

  it("renders headers", () => {
    render(
      <DataTable
        columns={columns}
        data={data}
        getRowId={(row) => row.id}
        completeData
      />,
    );
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Age")).toBeInTheDocument();
  });

  it("shows empty state when data is empty", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        getRowId={(row) => row.id}
        completeData
      />,
    );
    expect(screen.getByText("No results found")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        getRowId={(row) => row.id}
        completeData
        isLoading
      />,
    );
    expect(screen.getByText("Name")).toBeInTheDocument();
  });
});
describe("DataTable text is translated", () => {
  // The empty and error states shipped with hardcoded Turkish strings; every
  // other component routes its text through t().
  it("takes the empty state text from the translations", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        getRowId={(row) => row.id}
        isEmpty
      />,
    );

    expect(screen.getByText("No results found")).toBeInTheDocument();
    expect(screen.getByText("No data to display")).toBeInTheDocument();
  });

  it("takes the error title from the translations", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        getRowId={(row) => row.id}
        error="boom"
      />,
    );

    expect(screen.getByText("An error occurred")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();
  });
});

describe("DataTable sorting is opt-in", () => {
  const onSortChange = vi.fn();

  const renderTable = (cols: DataTableColumn<{ id: string; name: string }>[]) =>
    render(
      <DataTable
        columns={cols}
        data={[{ id: "1", name: "Alice" }]}
        getRowId={(row) => row.id}
        onSortChange={onSortChange}
      />,
    );

  const sortControls = () =>
    document.querySelectorAll("thead .inline-flex.flex-col").length;

  // Sorting runs against a server-side whitelist, so a column that has not
  // said it is sortable must not offer a control that would ask the API to
  // order by something it rejects.
  it("offers no sort control on a column that did not opt in", () => {
    renderTable([{ id: "name", header: "Name", accessor: "name" }]);

    expect(sortControls()).toBe(0);
  });

  it("offers a sort control on a column that opted in", () => {
    renderTable([
      { id: "name", header: "Name", accessor: "name", sortable: true },
    ]);

    expect(sortControls()).toBe(1);
  });

  // The actions column holds buttons; ordering by it is meaningless.
  it("never offers a sort control on the actions column", () => {
    renderTable([
      {
        id: "actions",
        header: "",
        accessor: () => "",
        type: "actions",
        sortable: true,
        actions: [],
      },
    ]);

    expect(sortControls()).toBe(0);
  });
});
