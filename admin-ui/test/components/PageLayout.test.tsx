import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { PageLayout } from "@/components/ui";

describe("PageLayout", () => {
  it("renders children", () => {
    render(
      <PageLayout>
        <PageLayout.Sidebar>Sidebar</PageLayout.Sidebar>
        <PageLayout.Content>Content</PageLayout.Content>
      </PageLayout>,
    );
    expect(screen.getByText("Sidebar")).toBeInTheDocument();
    expect(screen.getByText("Content")).toBeInTheDocument();
  });

  it("renders sidebar with default md width", () => {
    render(<PageLayout.Sidebar>Sidebar</PageLayout.Sidebar>);
    expect(screen.getByText("Sidebar").closest("aside")).toHaveClass("w-64");
  });

  it("renders sidebar with sm width", () => {
    render(<PageLayout.Sidebar width="sm">Sidebar</PageLayout.Sidebar>);
    expect(screen.getByText("Sidebar").closest("aside")).toHaveClass("w-48");
  });

  it("renders sidebar with lg width", () => {
    render(<PageLayout.Sidebar width="lg">Sidebar</PageLayout.Sidebar>);
    expect(screen.getByText("Sidebar").closest("aside")).toHaveClass("w-80");
  });

  it("renders content with flex-1", () => {
    render(<PageLayout.Content>Content</PageLayout.Content>);
    expect(screen.getByText("Content").closest("main")).toHaveClass("flex-1");
  });

  it("applies right sidebar position", () => {
    render(<PageLayout sidebarPosition="right">Content</PageLayout>);
    expect(screen.getByText("Content").closest("div")).toHaveClass("flex-row-reverse");
  });
});