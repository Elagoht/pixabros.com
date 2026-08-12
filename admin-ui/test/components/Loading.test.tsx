import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { Loading } from "@/components/ui";

describe("Loading", () => {
  it("renders spinner element", () => {
    const { container } = render(<Loading />);
    expect(container.querySelector(".animate-spin")).toBeTruthy();
  });

  it("has full height container", () => {
    const { container } = render(<Loading />);
    expect(container.firstChild).toHaveClass("h-screen");
  });
});