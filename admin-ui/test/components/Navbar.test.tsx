import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Navbar } from "@/components/ui";

describe("Navbar", () => {
  it("renders navbar with brand and items", () => {
    render(
      <Navbar>
        <Navbar.Brand>MyApp</Navbar.Brand>
        <Navbar.Content>
          <Navbar.Item>Home</Navbar.Item>
          <Navbar.Item active>Dashboard</Navbar.Item>
        </Navbar.Content>
      </Navbar>,
    );
    expect(screen.getByText("MyApp")).toBeInTheDocument();
    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("applies active styles to active item", () => {
    render(
      <Navbar>
        <Navbar.Content>
          <Navbar.Item active>Active</Navbar.Item>
        </Navbar.Content>
      </Navbar>,
    );
    expect(screen.getByText("Active")).toHaveClass("bg-primary-50");
  });
});
