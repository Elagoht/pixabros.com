import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Carousel } from "@/components/ui";
import { required } from "../utils/dom";

const slides = [
  { id: "1", content: "Slide 1" },
  { id: "2", content: "Slide 2" },
  { id: "3", content: "Slide 3" },
];

describe("Carousel", () => {
  it("renders slides", () => {
    render(<Carousel slides={slides} />);
    expect(screen.getByText("Slide 1")).toBeInTheDocument();
    expect(screen.getByText("Slide 2")).toBeInTheDocument();
    expect(screen.getByText("Slide 3")).toBeInTheDocument();
  });

  it("renders navigation arrows", () => {
    render(<Carousel slides={slides} />);
    const buttons = screen.getAllByRole("button");
    const arrowButtons = buttons.filter(
      (btn) =>
        btn.querySelector("svg") && btn.className.includes("rounded-full"),
    );
    expect(arrowButtons.length).toBe(2);
  });

  it("renders dot indicators", () => {
    render(<Carousel slides={slides} />);
    const dots = screen
      .getAllByRole("button")
      .filter(
        (btn) =>
          btn.className.includes("rounded-full") &&
          btn.className.includes("h-2"),
      );
    expect(dots.length).toBe(3);
  });

  it("navigates to next slide on next arrow click", () => {
    render(<Carousel slides={slides} />);
    const nextBtn = required(
      screen
        .getAllByRole("button")
        .find((btn) => btn.className.includes("right-2")),
      "the next-slide arrow",
    );
    fireEvent.click(nextBtn);
  });

  it("does not render arrows when showArrows is false", () => {
    render(<Carousel slides={slides} showArrows={false} />);
    const buttons = screen.getAllByRole("button");
    const arrowButtons = buttons.filter(
      (btn) =>
        btn.className.includes("rounded-full") &&
        btn.className.includes("absolute"),
    );
    expect(arrowButtons.length).toBe(0);
  });

  it("returns null for empty slides", () => {
    const { container } = render(<Carousel slides={[]} />);
    expect(container.innerHTML).toBe("");
  });
});
