import { describe, it, expect, vi, beforeEach } from "vitest";
import { useAuthStore } from "@/lib/stores/auth";

vi.mock("@/services/session", () => ({
  sessionService: {
    refresh: vi.fn(),
    me: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("useAuthStore", () => {
  beforeEach(() => {
    useAuthStore.setState({
      isAuthenticated: false,
      isLoading: true,
    });
  });

  it("has correct initial state", () => {
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isLoading).toBe(true);
  });

  it("setAuthenticated updates isAuthenticated", () => {
    useAuthStore.getState().setAuthenticated(true);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);

    useAuthStore.getState().setAuthenticated(false);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it("checkAuth sets authenticated on success", async () => {
    const { sessionService } = await import("@/services/session");
    vi.mocked(sessionService.refresh).mockResolvedValueOnce({} as never);
    vi.mocked(sessionService.me).mockResolvedValueOnce({} as never);

    await useAuthStore.getState().checkAuth();

    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it("checkAuth sets unauthenticated on failure", async () => {
    const { sessionService } = await import("@/services/session");
    vi.mocked(sessionService.refresh).mockRejectedValueOnce(new Error("fail"));

    await useAuthStore.getState().checkAuth();

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it("logout sets isAuthenticated to false", async () => {
    useAuthStore.setState({ isAuthenticated: true });

    const { sessionService } = await import("@/services/session");
    vi.mocked(sessionService.delete).mockResolvedValueOnce(undefined as never);

    delete (window as unknown as Record<string, unknown>).location;
    (window as unknown as Record<string, unknown>).location = { href: "", pathname: "/dashboard", search: "", hash: "" };

    await useAuthStore.getState().logout();

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});