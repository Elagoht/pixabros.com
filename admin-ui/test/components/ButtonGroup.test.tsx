import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ButtonGroup, Button } from "@/components/ui";

describe("ButtonGroup", () => {
  it("renders children", () => {
    render(
      <ButtonGroup>
        <Button>One</Button>
        <Button>Two</Button>
      </ButtonGroup>,
    );
    expect(screen.getByText("One")).toBeInTheDocument();
    expect(screen.getByText("Two")).toBeInTheDocument();
  });

  it("applies group styling classes", () => {
    render(
      <ButtonGroup>
        <Button>One</Button>
        <Button>Two</Button>
      </ButtonGroup>,
    );
    const group = screen.getByText("One").closest("div");
    expect(group).toHaveClass("inline-flex");
  });
});