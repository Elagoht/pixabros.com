import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useLoginRedirect } from "@/hooks/useLoginRedirect";

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useSearchParams: () => [new URLSearchParams()],
}));

describe("useLoginRedirect", () => {
  it("returns a function", () => {
    const { result } = renderHook(() => useLoginRedirect());
    expect(typeof result.current).toBe("function");
  });
});