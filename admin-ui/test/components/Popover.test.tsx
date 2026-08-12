import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Popover } from "@/components/ui";

describe("Popover", () => {
  it("renders trigger content", () => {
    render(
      <Popover>
        <Popover.Trigger>Click me</Popover.Trigger>
        <Popover.Content>Popover content</Popover.Content>
      </Popover>,
    );
    expect(screen.getByText("Click me")).toBeInTheDocument();
  });
});