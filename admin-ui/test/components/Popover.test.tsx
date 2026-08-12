import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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
