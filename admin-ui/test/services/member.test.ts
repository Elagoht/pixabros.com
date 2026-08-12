import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { MemberService } from "@/services/member";
import { Http } from "@/utilities/http";

describe("MemberService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("addresses a single member by id", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({});
    await MemberService.get("tb0kp5nnhaw9kntcmmrmilna");

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/members/tb0kp5nnhaw9kntcmmrmilna",
      { silent: true },
    );
  });

  // Sorting is the database's job, so it has to reach the API.
  it("sends the sort field and direction as query parameters", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await MemberService.list({ field: "name", direction: "desc" });

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/members?sort=name&dir=desc",
      { silent: true },
    );
  });

  it("asks for no particular sort when none is chosen", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce([]);
    await MemberService.list();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/members", {
      silent: true,
    });
  });

  it("sends the whole ordered id list to reorder", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce(undefined);
    await MemberService.reorder(["c", "a", "b"]);

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/members/reorder",
      { ids: ["c", "a", "b"] },
      { silent: true },
    );
  });
});
