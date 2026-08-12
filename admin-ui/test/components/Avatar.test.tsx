import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Avatar } from "@/components/ui";

describe("Avatar", () => {
  it("renders initials when name is provided without src", () => {
    render(<Avatar name="John Doe" />);
    expect(screen.getByText("JD")).toBeInTheDocument();
  });

  it("renders single initial for single name", () => {
    render(<Avatar name="John" />);
    expect(screen.getByText("J")).toBeInTheDocument();
  });

  it("renders with different sizes", () => {
    const { container } = render(<Avatar name="Test" size="xs" />);
    expect(container.querySelector(".h-6")).toBeTruthy();
  });

  it("renders lg size", () => {
    const { container } = render(<Avatar name="Test" size="lg" />);
    expect(container.querySelector(".h-12")).toBeTruthy();
  });

  it("renders online status indicator", () => {
    render(<Avatar name="Test" status="online" />);
    expect(document.querySelector(".bg-green-500")).toBeTruthy();
  });

  it("renders offline status indicator", () => {
    render(<Avatar name="Test" status="offline" />);
    expect(document.querySelector(".bg-gray-400")).toBeTruthy();
  });
});