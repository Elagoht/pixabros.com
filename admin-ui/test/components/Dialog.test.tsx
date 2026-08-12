import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Dialog } from "@/components/ui";
import { required } from "../utils/dom";

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
    const confirmBtn = required(
      screen.getAllByRole("button").find((btn) => btn.textContent === "Delete"),
      "the confirm button",
    );
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
