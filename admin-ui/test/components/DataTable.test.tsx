import { describe, it, expect } from "vitest";
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
    expect(screen.getByText("Kayıt bulunamadı")).toBeInTheDocument();
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