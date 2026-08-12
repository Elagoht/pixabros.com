import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Tooltip } from "@/components/ui";

describe("Tooltip", () => {
  it("renders children", () => {
    render(<Tooltip content="Tip text">Hover me</Tooltip>);
    expect(screen.getByText("Hover me")).toBeInTheDocument();
  });

  it("shows content on mouse enter", () => {
    render(<Tooltip content="Tip text">Hover me</Tooltip>);
    fireEvent.mouseEnter(screen.getByText("Hover me"));
    expect(screen.getByText("Tip text")).toBeInTheDocument();
  });

  it("hides content on mouse leave", () => {
    render(<Tooltip content="Tip text">Hover me</Tooltip>);
    fireEvent.mouseEnter(screen.getByText("Hover me"));
    expect(screen.getByText("Tip text")).toBeInTheDocument();

    fireEvent.mouseLeave(screen.getByText("Hover me"));
    expect(screen.queryByText("Tip text")).not.toBeInTheDocument();
  });
});