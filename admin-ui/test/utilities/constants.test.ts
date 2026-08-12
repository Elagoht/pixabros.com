import { describe, it, expect } from "vitest";
import { chipColors, checkboxBase, filterInputBase, filterSelectBase } from "@/utilities/constants";

describe("chipColors", () => {
  it("has expected color keys", () => {
    const expectedKeys = [
      "green", "red", "yellow", "blue", "gray",
      "indigo", "purple", "pink", "orange", "teal",
    ];
    for (const key of expectedKeys) {
      expect(chipColors).toHaveProperty(key);
    }
  });

  it("all values are non-empty strings", () => {
    for (const value of Object.values(chipColors)) {
      expect(typeof value).toBe("string");
      expect(value.length).toBeGreaterThan(0);
    }
  });

  it("returns gray as fallback", () => {
    expect(chipColors.gray).toBeTruthy();
  });
});

describe("checkboxBase", () => {
  it("is a non-empty string", () => {
    expect(typeof checkboxBase).toBe("string");
    expect(checkboxBase.length).toBeGreaterThan(0);
  });

  it("contains appearance-none class", () => {
    expect(checkboxBase).toContain("appearance-none");
  });
});

describe("filterInputBase", () => {
  it("is a non-empty string", () => {
    expect(typeof filterInputBase).toBe("string");
    expect(filterInputBase.length).toBeGreaterThan(0);
  });

  it("contains rounded class", () => {
    expect(filterInputBase).toContain("rounded");
  });
});

describe("filterSelectBase", () => {
  it("is a non-empty string", () => {
    expect(typeof filterSelectBase).toBe("string");
    expect(filterSelectBase.length).toBeGreaterThan(0);
  });

  it("contains rounded class", () => {
    expect(filterSelectBase).toContain("rounded");
  });
});