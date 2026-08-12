import { describe, expect, it } from "vitest";
import { ApiError } from "@/utilities/api-error";

describe("ApiError", () => {
  it("sets status and body from constructor", () => {
    const body = { message: "Not found" };
    const error = new ApiError(404, body);
    expect(error.status).toBe(404);
    expect(error.body).toEqual(body);
  });

  it("sets name to ApiError", () => {
    const error = new ApiError(500, {});
    expect(error.name).toBe("ApiError");
  });

  it("sets message from status code", () => {
    const error = new ApiError(401, {});
    expect(error.message).toBe("API error 401");
  });

  it("is an instance of Error", () => {
    const error = new ApiError(400, {});
    expect(error).toBeInstanceOf(Error);
  });

  it("is an instance of ApiError", () => {
    const error = new ApiError(400, {});
    expect(error).toBeInstanceOf(ApiError);
  });

  it("preserves body as Record<string, unknown>", () => {
    const body = { errors: ["a", "b"], count: 2 };
    const error = new ApiError(422, body);
    expect(error.body).toEqual(body);
  });
});
