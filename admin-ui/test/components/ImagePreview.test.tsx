import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ImagePreview } from "@/components/ui";

describe("ImagePreview", () => {
  it("renders the thumbnail inside a button so it is keyboard reachable", () => {
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    const trigger = screen.getByRole("button");
    expect(trigger.querySelector("img")).toHaveAttribute(
      "src",
      "/media/a.webp",
    );
  });

  // A grid of thumbnails must not put one hidden dialog per thumbnail into
  // the accessibility tree.
  it("renders nothing but the thumbnail while closed", () => {
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(screen.getAllByAltText("Screenshot 1")).toHaveLength(1);
  });

  it("opens the preview when the thumbnail is clicked", async () => {
    const user = userEvent.setup();
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    await user.click(screen.getByRole("button"));

    // Both the thumbnail and the full-size image now show the same source,
    // so opening a preview costs no extra download.
    const images = screen.getAllByAltText("Screenshot 1");
    expect(images.length).toBeGreaterThan(1);
    for (const image of images) {
      expect(image).toHaveAttribute("src", "/media/a.webp");
    }
  });

  it("shows the caption in the preview header when given one", async () => {
    const user = userEvent.setup();
    render(
      <ImagePreview src="/media/a.webp" alt="alt text" caption="Screenshot 3" />,
    );

    await user.click(screen.getAllByRole("button")[0]);

    expect(screen.getByText("Screenshot 3")).toBeInTheDocument();
  });

  it("falls back to the alt text when no caption is given", async () => {
    const user = userEvent.setup();
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    await user.click(screen.getAllByRole("button")[0]);

    expect(screen.getByText("Screenshot 1")).toBeInTheDocument();
  });

  it("closes on Escape", async () => {
    const user = userEvent.setup();
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    await user.click(screen.getByRole("button"));
    expect(screen.getByText("Screenshot 1")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    // The heading only exists while the preview is open.
    expect(screen.queryByText("Screenshot 1")).toBeNull();
    expect(screen.getAllByRole("button")).toHaveLength(1);
  });
});
