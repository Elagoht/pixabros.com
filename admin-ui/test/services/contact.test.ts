import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/utilities/http", () => ({
  Http: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { ContactService } from "@/services/contact";
import { Http } from "@/utilities/http";

describe("ContactService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The inbox is newest-first by default, decided server side.
  it("asks for no particular sort when none is chosen", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({ submissions: [], unread: 0 });
    await ContactService.list();

    expect(Http.get).toHaveBeenCalledWith("/api/admin/contact", {
      silent: true,
    });
  });

  it("sends the sort field and direction as query parameters", async () => {
    vi.mocked(Http.get).mockResolvedValueOnce({ submissions: [], unread: 0 });
    await ContactService.list({ field: "created_at", direction: "asc" });

    expect(Http.get).toHaveBeenCalledWith(
      "/api/admin/contact?sort=created_at&dir=asc",
      { silent: true },
    );
  });

  // Read state is a sub-resource, not a general update: everything else about
  // a submission is what the sender wrote.
  it("marks a submission read through its read sub-resource", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce({});
    await ContactService.setRead("s1", true);

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/contact/s1/read",
      { is_read: true },
      { silent: true },
    );
  });

  it("can mark a submission unread again", async () => {
    vi.mocked(Http.put).mockResolvedValueOnce({});
    await ContactService.setRead("s1", false);

    expect(Http.put).toHaveBeenCalledWith(
      "/api/admin/contact/s1/read",
      { is_read: false },
      { silent: true },
    );
  });

  it("deletes a submission by id", async () => {
    vi.mocked(Http.delete).mockResolvedValueOnce(undefined);
    await ContactService.delete("s1");

    expect(Http.delete).toHaveBeenCalledWith("/api/admin/contact/s1", {
      silent: true,
    });
  });

  // There is deliberately no create or update: submissions come from the
  // public form and are otherwise immutable.
  it("exposes no way to create or edit a submission", () => {
    expect("create" in ContactService).toBe(false);
    expect("update" in ContactService).toBe(false);
  });
});
