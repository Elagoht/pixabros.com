import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Dialog } from "@/components/ui";

describe("Dialog", () => {
  it("renders when open", () => {
    render(
      <Dialog open onClose={() => {}} title="Confirm" onConfirm={() => {}} />,
    );
    expect(screen.getByText("Confirm")).toBeInTheDocument();
  });

  it("renders cancel button that calls onClose", () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} title="Confirm" onConfirm={() => {}} />,
    );
    fireEvent.click(screen.getByText("No"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders confirm button that calls onConfirm", () => {
    const onConfirm = vi.fn();
    render(
      <Dialog
        open
        onClose={() => {}}
        title="Confirm Delete"
        onConfirm={onConfirm}
        confirmLabel="Delete"
      />,
    );
    const confirmBtn = screen.getAllByRole("button").find(
      (btn) => btn.textContent === "Delete",
    )!;
    fireEvent.click(confirmBtn);
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it("renders description when provided", () => {
    render(
      <Dialog
        open
        onClose={() => {}}
        title="Confirm"
        onConfirm={() => {}}
        description="Are you sure?"
      />,
    );
    expect(screen.getByText("Are you sure?")).toBeInTheDocument();
  });

  it("does not render description when not provided", () => {
    render(
      <Dialog open onClose={() => {}} title="Confirm" onConfirm={() => {}} />,
    );
    expect(screen.queryByText("Are you sure?")).not.toBeInTheDocument();
  });

  it("calls onCancel when cancel button clicked", () => {
    const onCancel = vi.fn();
    render(
      <Dialog
        open
        onClose={() => {}}
        title="Confirm"
        onConfirm={() => {}}
        onCancel={onCancel}
      />,
    );
    fireEvent.click(screen.getByText("No"));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});