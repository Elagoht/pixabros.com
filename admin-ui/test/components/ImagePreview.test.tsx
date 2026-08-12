import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ImagePreview } from "@/components/ui";

const openPreview = async () => {
  const user = userEvent.setup();
  render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);
  await user.click(screen.getByRole("button"));
  return user;
};

const fullSizeImage = () =>
  screen
    .getAllByAltText("Screenshot 1")
    .find((img) => img.className.includes("max-h-[90vh]"));

describe("ImagePreview", () => {
  it("renders the thumbnail inside a button so it is keyboard reachable", () => {
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    const trigger = screen.getByRole("button");
    expect(trigger.querySelector("img")).toHaveAttribute(
      "src",
      "/media/a.webp",
    );
  });

  // A grid of thumbnails must not leave one hidden full-size image per
  // thumbnail in the DOM.
  it("renders nothing but the thumbnail while closed", () => {
    render(<ImagePreview src="/media/a.webp" alt="Screenshot 1" />);

    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(screen.getAllByAltText("Screenshot 1")).toHaveLength(1);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("opens a lightbox showing the image at full size", async () => {
    await openPreview();

    expect(screen.getByRole("dialog")).toBeInTheDocument();

    const image = fullSizeImage();
    expect(image).toBeDefined();
    expect(image).toHaveAttribute("src", "/media/a.webp");
    // Bounded by the viewport on both axes, with the aspect ratio kept.
    expect(image?.className).toContain("max-w-[90vw]");
    expect(image?.className).toContain("object-contain");
  });

  it("dims the page behind the image", async () => {
    await openPreview();

    expect(screen.getByRole("dialog").className).toContain("bg-black/80");
  });

  it("closes when the X button is clicked", async () => {
    const user = await openPreview();

    const closeButton = screen
      .getAllByRole("button")
      .find((button) => button.getAttribute("aria-label") === "Close");
    expect(closeButton).toBeDefined();

    await user.click(closeButton as HTMLElement);

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("closes when the backdrop outside the image is clicked", async () => {
    const user = await openPreview();

    await user.click(screen.getByRole("dialog"));

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  // Clicking the picture is how you look at it; it must not dismiss it.
  it("stays open when the image itself is clicked", async () => {
    const user = await openPreview();

    await user.click(fullSizeImage() as HTMLElement);

    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  // The lightbox mounts hidden and is revealed on the next frame, so the
  // entrance actually transitions instead of snapping to its end state.
  it("animates in rather than appearing at full opacity", async () => {
    await openPreview();

    await waitFor(() =>
      expect(screen.getByRole("dialog").className).toContain("opacity-100"),
    );
  });

  it("animates out before leaving the DOM", async () => {
    const user = await openPreview();
    await waitFor(() =>
      expect(screen.getByRole("dialog").className).toContain("opacity-100"),
    );

    await user.keyboard("{Escape}");

    // Still present, already fading.
    expect(screen.getByRole("dialog").className).toContain("opacity-0");
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("closes on Escape", async () => {
    const user = await openPreview();

    await user.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("restores page scrolling after closing", async () => {
    const user = await openPreview();
    expect(document.body.style.overflow).toBe("hidden");

    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(document.body.style.overflow).not.toBe("hidden"),
    );
  });
});
