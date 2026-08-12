import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/utilities/api-error";

vi.mock("@/utilities/environment", () => ({
  Environment: {
    apiBase: "http://localhost:3000",
    mediaBase: "",
    pageSize: 12,
  },
}));

vi.mock("@/utilities/navigation", () => ({
  Navigation: {
    redirectToLogin: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

import { toast } from "sonner";
import { Http } from "@/utilities/http";
import { Navigation } from "@/utilities/navigation";

const jsonResponse = (body: unknown, status = 200) =>
  ({
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers({ "content-type": "application/json" }),
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(JSON.stringify(body)),
  }) as Response;

describe("Http", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

  describe("Http.get", () => {
    it("makes a GET request", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        jsonResponse({ id: 1 }),
      );

      await Http.get("/api/test");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:3000/api/test",
        expect.objectContaining({ method: "GET" }),
      );
    });

    it("throws ApiError on non-ok response", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        jsonResponse({ error: { code: "not_found", message: "no" } }, 404),
      );

      await expect(Http.get("/api/test")).rejects.toThrow(ApiError);
    });
  });

  describe("Http.post", () => {
    it("makes a POST request with body", async () => {
      const body = { name: "test" };
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(jsonResponse({}));

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
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(jsonResponse({}));

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
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(jsonResponse({}));

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
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(jsonResponse({}));

      await Http.post("/api/upload", formData);
      const callArgs = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(callArgs[1].body).toBe(formData);
    });
  });

  describe("error reporting", () => {
    it("toasts the message from the Go error envelope", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        jsonResponse(
          { error: { code: "weak_password", message: "password too short" } },
          400,
        ),
      );

      await expect(Http.post("/api/test", {})).rejects.toThrow(ApiError);
      expect(toast.error).toHaveBeenCalledWith("password too short");
    });

    it("stays quiet when silent is set", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        jsonResponse({ error: { code: "boom", message: "nope" } }, 400),
      );

      await expect(
        Http.post("/api/test", {}, { silent: true }),
      ).rejects.toThrow(ApiError);
      expect(toast.error).not.toHaveBeenCalled();
    });
  });

  describe("401 handling", () => {
    it("redirects to login and does not retry", async () => {
      const fetchSpy = vi
        .spyOn(globalThis, "fetch")
        .mockResolvedValueOnce(
          jsonResponse(
            { error: { code: "unauthorized", message: "not logged in" } },
            401,
          ),
        );

      await expect(Http.get("/api/test")).rejects.toThrow(ApiError);

      expect(fetchSpy).toHaveBeenCalledTimes(1);
      expect(Navigation.redirectToLogin).toHaveBeenCalled();
    });

    it("does not redirect when skipAuthRedirect is set", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        jsonResponse(
          { error: { code: "unauthorized", message: "not logged in" } },
          401,
        ),
      );

      await expect(
        Http.get("/api/test", { skipAuthRedirect: true, silent: true }),
      ).rejects.toThrow(ApiError);

      expect(Navigation.redirectToLogin).not.toHaveBeenCalled();
    });
  });
});
