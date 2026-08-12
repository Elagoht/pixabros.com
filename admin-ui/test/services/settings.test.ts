import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), put: vi.fn() },
}));

import { SettingsService } from "@/services/settings";
import { Http } from "@/utilities/http";

describe("SettingsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads each group from its own path", async () => {
    vi.mocked(Http.get).mockResolvedValue({});

    await SettingsService.get("site");
    expect(Http.get).toHaveBeenCalledWith("/api/admin/settings/site", {
      silent: true,
    });

    await SettingsService.get("homepage");
    expect(Http.get).toHaveBeenCalledWith("/api/admin/settings/homepage", {
      silent: true,
    });
  });

  it("saves values under a values envelope", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce({});
    await SettingsService.update("homepage", {
      values: { hero_slogan: "Play" },
    });

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/settings/homepage",
      { values: { hero_slogan: "Play" } },
      { silent: true },
    );
  });
});
