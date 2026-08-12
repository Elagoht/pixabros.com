import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { AwardService } from "@/services/award";
import { Http } from "@/utilities/http";

describe("AwardService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("addresses a single award by id", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({});
    await AwardService.get("tb0kp5nnhaw9kntcmmrmilna");

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/awards/tb0kp5nnhaw9kntcmmrmilna",
      { silent: true },
    );
  });

  it("sends the sort field and direction as query parameters", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await AwardService.list({ field: "date", direction: "desc" });

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/awards?sort=date&dir=desc",
      { silent: true },
    );
  });

  // With no sort the server decides, which for awards means newest first.
  it("asks for no particular sort when none is chosen", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await AwardService.list();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/awards", {
      silent: true,
    });
  });
});
