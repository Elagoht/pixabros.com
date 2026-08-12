import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HoverCard } from "@/components/ui";

describe("HoverCard", () => {
  it("renders trigger content", () => {
    render(
      <HoverCard>
        <HoverCard.Trigger>Hover me</HoverCard.Trigger>
        <HoverCard.Content>Card content</HoverCard.Content>
      </HoverCard>,
    );
    expect(screen.getByText("Hover me")).toBeInTheDocument();
  });

  it("clears pending timers on unmount", () => {
    vi.useFakeTimers();
    const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout");

    const { unmount } = render(
      <HoverCard>
        <HoverCard.Trigger>Hover me</HoverCard.Trigger>
        <HoverCard.Content>Card content</HoverCard.Content>
      </HoverCard>,
    );

    // Trigger mouseEnter to start open timer
    fireEvent.mouseEnter(screen.getByText("Hover me"));
    // Trigger mouseLeave to start close timer
    fireEvent.mouseLeave(screen.getByText("Hover me"));

    unmount();

    // After unmount, pending timers should be cleared
    expect(clearTimeoutSpy).toHaveBeenCalled();
    vi.useRealTimers();
    clearTimeoutSpy.mockRestore();
  });
});
