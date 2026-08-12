import { describe, it, expect, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: {
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("sessionService", () => {
  it("create calls Http.post with skipAuthRefresh", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce(undefined as never);

    const { sessionService } = await import("@/services/session");
    await sessionService.create({ email: "test@test.com", password: "pass" } as never);

    expect(Http.post).toHaveBeenCalledWith(
      "/user/token/",
      { email: "test@test.com", password: "pass" },
      { skipAuthRefresh: true },
    );
  });

  it("refresh calls Http.post with skipAuthRefresh", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce({} as never);

    const { sessionService } = await import("@/services/session");
    await sessionService.refresh();

    expect(Http.post).toHaveBeenCalledWith(
      "/user/token/refresh/",
      undefined,
      { skipAuthRefresh: true },
    );
  });

  it("delete calls Http.post to logout", async () => {
    const { Http } = await import("@/utilities/http");
    vi.mocked(Http.post).mockResolvedValueOnce(undefined as never);

    const { sessionService } = await import("@/services/session");
    await sessionService.delete();

    expect(Http.post).toHaveBeenCalledWith("/user/logout/");
  });
});
