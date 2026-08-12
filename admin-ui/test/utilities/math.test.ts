import { describe, it, expect } from "vitest";
import { clamp, roundStep } from "@/utilities/math";

describe("clamp", () => {
  it("returns value when within range", () => {
    expect(clamp(5, 0, 10)).toBe(5);
  });

  it("returns min when value is below range", () => {
    expect(clamp(-3, 0, 10)).toBe(0);
  });

  it("returns max when value is above range", () => {
    expect(clamp(15, 0, 10)).toBe(10);
  });

  it("returns value at min boundary", () => {
    expect(clamp(0, 0, 10)).toBe(0);
  });

  it("returns value at max boundary", () => {
    expect(clamp(10, 0, 10)).toBe(10);
  });

  it("works with negative ranges", () => {
    expect(clamp(0, -10, -5)).toBe(-5);
  });

  it("works with floating point numbers", () => {
    expect(clamp(2.5, 0, 5)).toBe(2.5);
  });
});

describe("roundStep", () => {
  it("rounds to nearest 1 by default", () => {
    expect(roundStep(2.4, 1)).toBeCloseTo(2, 5);
    expect(roundStep(2.6, 1)).toBeCloseTo(3, 5);
  });

  it("rounds to nearest 0.5", () => {
    expect(roundStep(2.3, 0.5)).toBeCloseTo(2.5, 5);
    expect(roundStep(2.7, 0.5)).toBeCloseTo(2.5, 5);
  });

  it("rounds to nearest 10", () => {
    expect(roundStep(24, 10)).toBeCloseTo(20, 5);
    expect(roundStep(26, 10)).toBeCloseTo(30, 5);
  });

  it("rounds 0 to 0", () => {
    expect(roundStep(0, 5)).toBe(0);
  });

  it("handles step of 0.1", () => {
    expect(roundStep(1.23, 0.1)).toBeCloseTo(1.2, 5);
  });
});