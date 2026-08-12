import { describe, expect, it } from "vitest";
import { queryClient } from "@/lib/query/client";

describe("queryClient", () => {
  it("has staleTime of 60 seconds", () => {
    expect(queryClient.getDefaultOptions().queries?.staleTime).toBe(60 * 1000);
  });

  it("has retry set to 1", () => {
    expect(queryClient.getDefaultOptions().queries?.retry).toBe(1);
  });

  it("has refetchOnWindowFocus set to true", () => {
    expect(queryClient.getDefaultOptions().queries?.refetchOnWindowFocus).toBe(
      true,
    );
  });
});
