import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Modal } from "@/components/ui";

describe("Modal", () => {
  it("renders content when open", () => {
    render(
      <Modal open onClose={() => {}}>
        <Modal.Body>Modal content</Modal.Body>
      </Modal>,
    );
    expect(screen.getByText("Modal content")).toBeInTheDocument();
  });

  it("renders content with hidden styles when closed", () => {
    render(
      <Modal open={false} onClose={() => {}}>
        <Modal.Body>Hidden content</Modal.Body>
      </Modal>,
    );
    expect(screen.getByText("Hidden content")).toBeInTheDocument();
    const wrapper = screen.getByText("Hidden content").parentElement!;
    expect(wrapper).toHaveClass("opacity-0");
  });

  it("calls onClose on backdrop click", () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose}>
        <Modal.Body>Content</Modal.Body>
      </Modal>,
    );
    const backdrop = screen.getByText("Content").parentElement!.parentElement!;
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onClose on backdrop click when persistent", () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} persistent>
        <Modal.Body>Content</Modal.Body>
      </Modal>,
    );
    fireEvent.click(screen.getByText("Content").parentElement!);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("calls onClose on Escape key", () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose}>
        <Modal.Body>Content</Modal.Body>
      </Modal>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onClose on Escape when persistent", () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose} persistent>
        <Modal.Body>Content</Modal.Body>
      </Modal>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("renders Header, Body, and Footer sub-components", () => {
    const onClose = vi.fn();
    render(
      <Modal open onClose={onClose}>
        <Modal.Header onClose={onClose}>Title</Modal.Header>
        <Modal.Body>Body text</Modal.Body>
        <Modal.Footer>Footer text</Modal.Footer>
      </Modal>,
    );
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.getByText("Body text")).toBeInTheDocument();
    expect(screen.getByText("Footer text")).toBeInTheDocument();
  });
});