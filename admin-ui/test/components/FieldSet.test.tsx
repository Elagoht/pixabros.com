import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { FieldSet } from "@/components/ui";
import { IconUser } from "@tabler/icons-react";

describe("FieldSet", () => {
  it("renders legend and children", () => {
    render(<FieldSet legend="Personal Info">Content</FieldSet>);
    expect(screen.getByText("Personal Info")).toBeInTheDocument();
    expect(screen.getByText("Content")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(<FieldSet legend="Info" description="Fill in details">Content</FieldSet>);
    expect(screen.getByText("Fill in details")).toBeInTheDocument();
  });

  it("renders icon when provided", () => {
    render(<FieldSet legend="Info" icon={IconUser}>Content</FieldSet>);
    expect(screen.getByText("Info")).toBeInTheDocument();
  });

  it("renders error message", () => {
    render(<FieldSet legend="Info" error="Something went wrong">Content</FieldSet>);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("applies disabled attribute", () => {
    const { container } = render(<FieldSet legend="Info" disabled>Content</FieldSet>);
    expect(container.querySelector("fieldset")).toHaveAttribute("disabled");
  });

  it("applies error border class", () => {
    render(<FieldSet legend="Info" error="Error">Content</FieldSet>);
    expect(screen.getByText("Content").closest("fieldset")).toHaveClass("border-red-300");
  });
});