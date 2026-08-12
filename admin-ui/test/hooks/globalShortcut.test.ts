import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("useGlobalShortcut", () => {
  let listeners: Record<string, Array<(e: Event) => void>> = {};

  beforeEach(() => {
    listeners = {};
    vi.spyOn(document, "addEventListener").mockImplementation(
      (event, handler) => {
        if (!listeners[event]) {
          listeners[event] = [];
        }
        listeners[event].push(handler as (e: Event) => void);
      },
    );
    vi.spyOn(document, "removeEventListener").mockImplementation(
      (event, handler) => {
        if (listeners[event]) {
          listeners[event] = listeners[event].filter((h) => h !== handler);
        }
      },
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("fires callback on key match", async () => {
    const { useGlobalShortcut } = await import("@/hooks/useGlobalShortcut");
    const callback = vi.fn();

    renderHook(() => useGlobalShortcut({ key: "k", metaKey: true }, callback));

    const handler = listeners.keydown?.[0];
    expect(handler).toBeDefined();

    handler?.(new KeyboardEvent("keydown", { key: "k", metaKey: true }));
    expect(callback).toHaveBeenCalled();
  });

  it("does not fire on wrong key", async () => {
    const { useGlobalShortcut } = await import("@/hooks/useGlobalShortcut");
    const callback = vi.fn();

    renderHook(() => useGlobalShortcut({ key: "k", metaKey: true }, callback));

    const handler = listeners.keydown?.[0];
    handler?.(new KeyboardEvent("keydown", { key: "j", metaKey: true }));
    expect(callback).not.toHaveBeenCalled();
  });

  it("removes listener on unmount", async () => {
    const { useGlobalShortcut } = await import("@/hooks/useGlobalShortcut");
    const callback = vi.fn();

    const { unmount } = renderHook(() =>
      useGlobalShortcut({ key: "k", metaKey: true }, callback),
    );

    expect(document.addEventListener).toHaveBeenCalledWith(
      "keydown",
      expect.any(Function),
    );

    unmount();

    expect(document.removeEventListener).toHaveBeenCalledWith(
      "keydown",
      expect.any(Function),
    );
  });

  it("respects ctrlKey", async () => {
    const { useGlobalShortcut } = await import("@/hooks/useGlobalShortcut");
    const callback = vi.fn();

    renderHook(() => useGlobalShortcut({ key: "k", ctrlKey: true }, callback));

    const handler = listeners.keydown?.[0];
    handler?.(new KeyboardEvent("keydown", { key: "k", ctrlKey: true }));
    expect(callback).toHaveBeenCalled();
  });

  it("calls preventDefault on matching event", async () => {
    const { useGlobalShortcut } = await import("@/hooks/useGlobalShortcut");
    const callback = vi.fn();

    renderHook(() => useGlobalShortcut({ key: "k", metaKey: true }, callback));

    const handler = listeners.keydown?.[0];
    const event = new KeyboardEvent("keydown", { key: "k", metaKey: true });
    const spy = vi.spyOn(event, "preventDefault");

    handler?.(event);
    expect(spy).toHaveBeenCalled();
  });
});
