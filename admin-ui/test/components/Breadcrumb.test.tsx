import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { Breadcrumb } from "@/components/ui";

const renderBreadcrumb = (ui: React.ReactElement) =>
  render(ui, { wrapper: MemoryRouter });

describe("Breadcrumb", () => {
  it("renders breadcrumb items", () => {
    renderBreadcrumb(
      <Breadcrumb>
        <Breadcrumb.Item to="/">Home</Breadcrumb.Item>
        <Breadcrumb.Item>Products</Breadcrumb.Item>
      </Breadcrumb>,
    );
    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.getByText("Products")).toBeInTheDocument();
  });

  it("renders last item as text (not link)", () => {
    renderBreadcrumb(
      <Breadcrumb>
        <Breadcrumb.Item to="/">Home</Breadcrumb.Item>
        <Breadcrumb.Item>Current</Breadcrumb.Item>
      </Breadcrumb>,
    );
    expect(screen.getByText("Current")).toHaveClass("font-medium");
  });

  it("renders link items", () => {
    renderBreadcrumb(
      <Breadcrumb>
        <Breadcrumb.Item to="/">Home</Breadcrumb.Item>
      </Breadcrumb>,
    );
    const link = screen.getByText("Home");
    expect(link.closest("a")).toBeTruthy();
  });

  it("renders custom separator", () => {
    renderBreadcrumb(
      <Breadcrumb separator="/">
        <Breadcrumb.Item to="/">Home</Breadcrumb.Item>
        <Breadcrumb.Item>Products</Breadcrumb.Item>
      </Breadcrumb>,
    );
    expect(screen.getByText("/")).toBeInTheDocument();
  });
});
