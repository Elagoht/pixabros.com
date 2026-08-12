import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), post: vi.fn() },
}));

import { Http } from "@/utilities/http";
import { MediaService } from "@/services/media";

describe("MediaService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The target decides the stored dimensions server-side, so it must reach
  // the API as a query parameter rather than being dropped.
  it("uploads with the target as a query parameter", async () => {
    vi.mocked(Http.post).mockResolvedValueOnce({});
    const file = new File(["img"], "art.png");
    await MediaService.upload(file, "cartridge_art");

    const [path, body] = vi.mocked(Http.post).mock.calls[0];
    expect(path).toBe("/api/admin/media/upload?target=cartridge_art");
    expect((body as FormData).get("file")).toBe(file);
  });

  it("reads a single media record by id", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({});
    await MediaService.get("m1");

    expect(Http.get).toHaveBeenCalledWith("/api/admin/media/m1", {
      silent: true,
    });
  });
});
