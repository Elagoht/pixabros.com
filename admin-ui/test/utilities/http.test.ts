import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApiError } from "@/utilities/api-error";

vi.mock("@/utilities/environment", () => ({
  Environment: {
    apiBase: "http://localhost:3000",
    mediaBase: "http://localhost:3000/media",
    pageSize: 12,
  },
}));

import { Http } from "@/utilities/http";

describe("Http", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  describe("Http.get", () => {
    it("makes a GET request", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: () => Promise.resolve({ id: 1 }),
        text: () => Promise.resolve(JSON.stringify({ id: 1 })),
      } as Response);

      await Http.get("/api/test");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:3000/api/test",
        expect.objectContaining({ method: "GET" }),
      );
    });

    it("throws ApiError on non-ok response", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: false,
        status: 404,
        headers: new Headers(),
        json: () => Promise.resolve({ error: "Not found" }),
      } as Response);

      await expect(Http.get("/api/test")).rejects.toThrow(ApiError);
    });
  });

  describe("Http.post", () => {
    it("makes a POST request with body", async () => {
      const body = { name: "test" };
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(JSON.stringify({})),
      } as Response);

      await Http.post("/api/test", body);
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:3000/api/test",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(body),
        }),
      );
    });
  });

  describe("Http.put", () => {
    it("makes a PUT request with body", async () => {
      const body = { name: "updated" };
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(JSON.stringify({})),
      } as Response);

      await Http.put("/api/test", body);
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:3000/api/test",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify(body),
        }),
      );
    });
  });

  describe("Http.delete", () => {
    it("makes a DELETE request", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers(),
        json: () => Promise.resolve(undefined),
        text: () => Promise.resolve(""),
      } as Response);

      await Http.delete("/api/test");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:3000/api/test",
        expect.objectContaining({ method: "DELETE" }),
      );
    });
  });

  describe("FormData handling", () => {
    it("sends FormData without Content-Type header", async () => {
      const formData = new FormData();
      formData.append("file", new Blob(["test"]), "test.txt");
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers(),
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(""),
      } as Response);

      await Http.post("/api/upload", formData);
      const callArgs = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(callArgs[1].body).toBe(formData);
    });
  });

  describe("401 handling", () => {
    it("attempts refresh on 401 and retries", async () => {
      const refreshMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(""),
      });
      const successMock = {
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: () => Promise.resolve({ id: 1 }),
        text: () => Promise.resolve(JSON.stringify({ id: 1 })),
      };

      let callCount = 0;
      vi.spyOn(globalThis, "fetch").mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.resolve({
            ok: false,
            status: 401,
            headers: new Headers(),
            json: () => Promise.resolve({}),
          });
        }
        if (callCount === 2) {
          return refreshMock();
        }
        return Promise.resolve(successMock);
      });

      await Http.get("/api/test");
      expect(callCount).toBeGreaterThanOrEqual(2);
    });
  });
});