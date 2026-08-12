import { describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe("SessionService", () => {
  it("create posts credentials to the admin login endpoint", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce({ username: "furkan" });

    const { SessionService } = await import("@/services/session");
    await SessionService.create({ username: "furkan", password: "pass" });

    expect(Http.post).toHaveBeenCalledWith(
      "/api/admin/login",
      { username: "furkan", password: "pass" },
      { silent: true, skipAuthRedirect: true },
    );
  });

  it("me reads whoami without redirecting on 401", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.get).mockResolvedValueOnce({ username: "furkan" });

    const { SessionService } = await import("@/services/session");
    await SessionService.me();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/whoami", {
      silent: true,
      skipAuthRedirect: true,
    });
  });

  it("delete posts to the admin logout endpoint", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce(undefined);

    const { SessionService } = await import("@/services/session");
    await SessionService.delete();

    expect(Http.post).toHaveBeenCalledWith("/api/admin/logout", undefined, {
      silent: true,
      skipAuthRedirect: true,
    });
  });

  it("changePassword posts snake_case fields the Go API expects", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce(undefined);

    const { SessionService } = await import("@/services/session");
    await SessionService.changePassword({
      current_password: "old-password",
      new_password: "new-password",
    });

    expect(Http.post).toHaveBeenCalledWith(
      "/api/admin/change-password",
      { current_password: "old-password", new_password: "new-password" },
      { silent: true },
    );
  });
});
