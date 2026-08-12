import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { DevlogService } from "@/services/devlog";
import { Http } from "@/utilities/http";

describe("DevlogService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The API accepts an id or a slug, so a link built from a public URL works.
  it("addresses a post by whatever key it is given", async () => {
    vi.mocked(Http.get).mockResolvedValue({});

    await DevlogService.get("tb0kp5nnhaw9kntcmmrmilna");
    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/devlog/tb0kp5nnhaw9kntcmmrmilna",
      { silent: true },
    );

    await DevlogService.get("kartus-sistemi");
    expect(Http.get).toHaveBeenCalledWith("/api/admin/devlog/kartus-sistemi", {
      silent: true,
    });
  });

  it("sends the sort field and direction as query parameters", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await DevlogService.list({ field: "published_at", direction: "desc" });

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/devlog?sort=published_at&dir=desc",
      { silent: true },
    );
  });

  it("asks for no particular sort when none is chosen", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await DevlogService.list();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/devlog", {
      silent: true,
    });
  });
});
