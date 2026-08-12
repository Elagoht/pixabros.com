import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/stores/auth";

vi.mock("@/services/session", () => ({
  SessionService: {
    me: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/utilities/navigation", () => ({
  Navigation: {
    redirectToLogin: vi.fn(),
  },
}));

describe("useAuthStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      user: null,
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

  it("setSession seeds the admin without hitting the network", async () => {
    const { SessionService } = await import("@/services/session");

    useAuthStore.getState().setSession({ username: "furkan" });

    const state = useAuthStore.getState();
    expect(state.user).toEqual({ username: "furkan" });
    expect(state.isAuthenticated).toBe(true);
    expect(state.isLoading).toBe(false);
    expect(SessionService.me).not.toHaveBeenCalled();
  });

  it("checkAuth stores the admin returned by whoami", async () => {
    const { SessionService } = await import("@/services/session");
    vi.mocked(SessionService.me).mockResolvedValueOnce({ username: "furkan" });

    await useAuthStore.getState().checkAuth();

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.isLoading).toBe(false);
    expect(state.user).toEqual({ username: "furkan" });
  });

  it("checkAuth sets unauthenticated when whoami rejects", async () => {
    const { SessionService } = await import("@/services/session");
    vi.mocked(SessionService.me).mockRejectedValueOnce(new Error("401"));

    await useAuthStore.getState().checkAuth();

    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isLoading).toBe(false);
    expect(state.user).toBeNull();
  });

  it("checkAuth can run again after a failure", async () => {
    const { SessionService } = await import("@/services/session");
    vi.mocked(SessionService.me).mockRejectedValueOnce(new Error("401"));
    await useAuthStore.getState().checkAuth();

    vi.mocked(SessionService.me).mockResolvedValueOnce({ username: "furkan" });
    await useAuthStore.getState().checkAuth();

    expect(SessionService.me).toHaveBeenCalledTimes(2);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
  });

  it("logout clears the session and redirects to login", async () => {
    useAuthStore.setState({ isAuthenticated: true, user: { username: "f" } });

    const { SessionService } = await import("@/services/session");
    const { Navigation } = await import("@/utilities/navigation");
    vi.mocked(SessionService.delete).mockResolvedValueOnce(undefined);

    await useAuthStore.getState().logout();

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
    expect(Navigation.redirectToLogin).toHaveBeenCalled();
  });

  it("logout still clears state when the API call fails", async () => {
    useAuthStore.setState({ isAuthenticated: true, user: { username: "f" } });

    const { SessionService } = await import("@/services/session");
    vi.mocked(SessionService.delete).mockRejectedValueOnce(new Error("boom"));

    await expect(useAuthStore.getState().logout()).rejects.toThrow("boom");
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
