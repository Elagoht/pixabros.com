import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { MediaLibraryService } from "@/services/mediaLibrary";
import { Http } from "@/utilities/http";

describe("MediaLibraryService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("lists the library", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({ items: [], orphaned: 0 });
    await MediaLibraryService.list();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/media", {
      silent: true,
    });
  });

  it("saves alt text", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce(undefined);
    await MediaLibraryService.setAltText("m1", "A falling block");

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/media/m1",
      { alt_text: "A falling block" },
      { silent: true },
    );
  });

  // Clearing alt text has to be sendable, which is why the API field is
  // nullable rather than omitted-when-empty.
  it("can clear alt text", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce(undefined);
    await MediaLibraryService.setAltText("m1", "");

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/media/m1",
      { alt_text: "" },
      { silent: true },
    );
  });

  it("deletes by id", async () => {
    vi.mocked(Http.delete).mockResolvedValueOnce(undefined);
    await MediaLibraryService.delete("m1");

    expect(Http.delete).toHaveBeenCalledWith("/api/admin/media/m1", {
      silent: true,
    });
  });

  // Uploading belongs to the module that needs the image, because the upload
  // target decides the stored dimensions.
  it("exposes no upload path", () => {
    expect("upload" in MediaLibraryService).toBe(false);
  });
});
