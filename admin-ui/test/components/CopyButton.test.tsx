import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CopyButton } from "@/components/ui";

Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn().mockResolvedValue(undefined),
  },
});

describe("CopyButton", () => {
  it("renders button with default text", () => {
    render(<CopyButton value="test" />);
    expect(screen.getByText("Copy")).toBeInTheDocument();
  });

  it("renders custom children", () => {
    render(<CopyButton value="test">Copy this</CopyButton>);
    expect(screen.getByText("Copy this")).toBeInTheDocument();
  });

  it("calls clipboard.writeText on click", () => {
    render(<CopyButton value="hello world" />);
    fireEvent.click(screen.getByRole("button"));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("hello world");
  });

  it("clears timeout on unmount to prevent state update after unmount", () => {
    vi.useFakeTimers();
    const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout");
    const { unmount } = render(<CopyButton value="test" />);

    fireEvent.click(screen.getByRole("button"));
    unmount();

    // After unmount, the pending timeout should have been cleared
    expect(clearTimeoutSpy).toHaveBeenCalled();
    vi.useRealTimers();
    clearTimeoutSpy.mockRestore();
  });
});