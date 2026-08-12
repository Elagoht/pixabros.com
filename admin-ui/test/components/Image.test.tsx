import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Image } from "@/components/ui";

describe("Image", () => {
  it("renders image with src and alt", () => {
    render(<Image src="/test.jpg" alt="Test image" width={200} height={200} />);
    const img = screen.getByRole("img");
    expect(img).toHaveAttribute("src", "/test.jpg");
    expect(img).toHaveAttribute("alt", "Test image");
  });

  it("shows fallback on error", () => {
    render(<Image src="/broken.jpg" alt="Test image" width={200} height={200} />);
    const img = screen.getByRole("img");
    fireEvent.error(img);
    expect(screen.getByText("Failed to load")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    const { container } = render(
      <Image src="/test.jpg" alt="Test" width={200} height={200} />,
    );
    expect(container.querySelector(".animate-pulse")).toBeInTheDocument();
  });

  it("removes loading state after load", () => {
    const { container } = render(
      <Image src="/test.jpg" alt="Test" width={200} height={200} />,
    );
    const img = screen.getByRole("img");
    fireEvent.load(img);
    expect(container.querySelector(".animate-pulse")).not.toBeInTheDocument();
  });
});