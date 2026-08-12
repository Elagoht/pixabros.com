import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

import { Http } from "@/utilities/http";
import { GameService } from "@/services/game";

describe("GameService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("addresses a single game by its id", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({});
    await GameService.get("tb0kp5nnhaw9kntcmmrmilna");

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/games/tb0kp5nnhaw9kntcmmrmilna",
      { silent: true },
    );
  });

  it("sends the whole ordered id list to reorder", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce(undefined);
    await GameService.reorder(["c", "a", "b"]);

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/games/reorder",
      { ids: ["c", "a", "b"] },
      { silent: true },
    );
  });

  it("scopes screenshot removal to the owning game", async () => {
    vi.mocked(Http.delete).mockResolvedValueOnce(undefined);
    await GameService.removeScreenshot("game-1", "shot-9");

    expect(Http.delete).toHaveBeenCalledWith(
      "/api/admin/games/game-1/screenshots/shot-9",
      { silent: true },
    );
  });

  // The extracted build lives at /play/{slug}/, so this one endpoint is
  // addressed by slug while everything else uses the id.
  it("uploads a build by slug, as multipart form data", async () => {
    vi.mocked(Http.post).mockResolvedValueOnce(undefined);
    const file = new File(["zip"], "build.zip");
    await GameService.uploadBuild("pixel-quest", file);

    const [path, body, options] = vi.mocked(Http.post).mock.calls[0];
    expect(path).toBe("/api/admin/games/pixel-quest/upload");
    expect(body).toBeInstanceOf(FormData);
    expect((body as FormData).get("file")).toBe(file);
    expect(options).toEqual({ silent: true });
  });
});

describe("GameService screenshot reordering", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The endpoint is scoped to the game, and takes the complete ordered list
  // rather than a moved-item delta.
  it("sends the whole ordered screenshot list, scoped to the game", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce(undefined);
    await GameService.reorderScreenshots("game-1", ["s3", "s1", "s2"]);

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/games/game-1/screenshots/reorder",
      { ids: ["s3", "s1", "s2"] },
      { silent: true },
    );
  });
});
