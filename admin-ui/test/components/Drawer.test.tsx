import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Drawer } from "@/components/ui";
import { parentOf } from "../utils/dom";

describe("Drawer", () => {
  it("renders content when open", () => {
    render(
      <Drawer open onClose={() => {}}>
        <Drawer.Body>Drawer content</Drawer.Body>
      </Drawer>,
    );
    expect(screen.getByText("Drawer content")).toBeInTheDocument();
  });

  it("renders content with hidden styles when closed", () => {
    render(
      <Drawer open={false} onClose={() => {}}>
        <Drawer.Body>Hidden</Drawer.Body>
      </Drawer>,
    );
    const wrapper = parentOf(screen.getByText("Hidden"), "drawer panel");
    expect(wrapper).toHaveClass("translate-x-full");
  });

  it("calls onClose on backdrop click", () => {
    const onClose = vi.fn();
    render(
      <Drawer open onClose={onClose}>
        <Drawer.Body>Content</Drawer.Body>
      </Drawer>,
    );
    const backdrop = parentOf(
      parentOf(screen.getByText("Content"), "drawer panel"),
      "drawer backdrop",
    );
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onClose on backdrop click when persistent", () => {
    const onClose = vi.fn();
    render(
      <Drawer open onClose={onClose} persistent>
        <Drawer.Body>Content</Drawer.Body>
      </Drawer>,
    );
    fireEvent.click(parentOf(screen.getByText("Content"), "drawer panel"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("renders Header, Body, and Footer sub-components", () => {
    const onClose = vi.fn();
    render(
      <Drawer open onClose={onClose}>
        <Drawer.Header onClose={onClose}>Title</Drawer.Header>
        <Drawer.Body>Body text</Drawer.Body>
        <Drawer.Footer>Footer text</Drawer.Footer>
      </Drawer>,
    );
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.getByText("Body text")).toBeInTheDocument();
    expect(screen.getByText("Footer text")).toBeInTheDocument();
  });
});
