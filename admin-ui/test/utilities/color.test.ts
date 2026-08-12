import { describe, expect, it } from "vitest";
import {
  hexToRgb,
  hsvToRgb,
  isValidHex,
  rgbToHex,
  rgbToHsv,
} from "@/utilities/color";

describe("hexToRgb", () => {
  it("converts #ff0000 to [255, 0, 0]", () => {
    expect(hexToRgb("#ff0000")).toEqual([255, 0, 0]);
  });

  it("converts #0000ff to [0, 0, 255]", () => {
    expect(hexToRgb("#0000ff")).toEqual([0, 0, 255]);
  });

  it("converts #ffffff to [255, 255, 255]", () => {
    expect(hexToRgb("#ffffff")).toEqual([255, 255, 255]);
  });

  it("handles hex without # prefix", () => {
    expect(hexToRgb("00ff00")).toEqual([0, 255, 0]);
  });

  it("handles #000000", () => {
    expect(hexToRgb("#000000")).toEqual([0, 0, 0]);
  });
});

describe("rgbToHex", () => {
  it("converts [255, 0, 0] to #ff0000", () => {
    expect(rgbToHex(255, 0, 0)).toBe("#ff0000");
  });

  it("converts [0, 255, 0] to #00ff00", () => {
    expect(rgbToHex(0, 255, 0)).toBe("#00ff00");
  });

  it("converts [0, 0, 0] to #000000", () => {
    expect(rgbToHex(0, 0, 0)).toBe("#000000");
  });

  it("converts [255, 255, 255] to #ffffff", () => {
    expect(rgbToHex(255, 255, 255)).toBe("#ffffff");
  });

  it("rounds fractional values", () => {
    expect(rgbToHex(127.5, 0, 0)).toBe("#800000");
  });
});

describe("rgbToHsv", () => {
  it("converts pure red", () => {
    const [r, g, b] = rgbToHsv(255, 0, 0);
    expect(r).toBeCloseTo(0, 0);
    expect(g).toBeCloseTo(100, 0);
    expect(b).toBeCloseTo(100, 0);
  });

  it("converts pure green", () => {
    const [r, g, b] = rgbToHsv(0, 255, 0);
    expect(r).toBeCloseTo(120, 0);
    expect(g).toBeCloseTo(100, 0);
    expect(b).toBeCloseTo(100, 0);
  });

  it("converts pure blue", () => {
    const [r, g, b] = rgbToHsv(0, 0, 255);
    expect(r).toBeCloseTo(240, 0);
    expect(g).toBeCloseTo(100, 0);
    expect(b).toBeCloseTo(100, 0);
  });

  it("converts black", () => {
    const [_r, g, b] = rgbToHsv(0, 0, 0);
    expect(g).toBe(0);
    expect(b).toBe(0);
  });

  it("converts white", () => {
    const result = rgbToHsv(255, 255, 255);
    expect(result[1]).toBeCloseTo(0, 0);
    expect(result[2]).toBeCloseTo(100, 0);
  });

  it("converts pure green", () => {
    const [h, s, v] = rgbToHsv(0, 255, 0);
    expect(h).toBeCloseTo(120, 0);
    expect(s).toBeCloseTo(100, 0);
    expect(v).toBeCloseTo(100, 0);
  });

  it("converts pure blue", () => {
    const [h, s, v] = rgbToHsv(0, 0, 255);
    expect(h).toBeCloseTo(240, 0);
    expect(s).toBeCloseTo(100, 0);
    expect(v).toBeCloseTo(100, 0);
  });

  it("converts black", () => {
    const [_h, s, v] = rgbToHsv(0, 0, 0);
    expect(s).toBe(0);
    expect(v).toBe(0);
  });

  it("converts white", () => {
    const [_h, s, v] = rgbToHsv(255, 255, 255);
    expect(s).toBeCloseTo(0, 0);
    expect(v).toBeCloseTo(100, 0);
  });
});

describe("hsvToRgb", () => {
  it("converts pure red (0, 100, 100)", () => {
    const [r, g, b] = hsvToRgb(0, 100, 100);
    expect(r).toBeCloseTo(255, 0);
    expect(g).toBeCloseTo(0, 0);
    expect(b).toBeCloseTo(0, 0);
  });

  it("converts white (any hue, 0, 100)", () => {
    const [r, g, b] = hsvToRgb(0, 0, 100);
    expect(r).toBeCloseTo(255, 0);
    expect(g).toBeCloseTo(255, 0);
    expect(b).toBeCloseTo(255, 0);
  });

  it("converts black (any hue, any sat, 0)", () => {
    const [r, g, b] = hsvToRgb(0, 50, 0);
    expect(r).toBeCloseTo(0, 0);
    expect(g).toBeCloseTo(0, 0);
    expect(b).toBeCloseTo(0, 0);
  });

  it("roundtrip: hex → rgb → hsv → rgb → hex", () => {
    const original = "#3b82f6";
    const rgb = hexToRgb(original);
    const hsv = rgbToHsv(...rgb);
    const backToRgb = hsvToRgb(...hsv);
    const backToHex = rgbToHex(...backToRgb);
    expect(backToHex).toBe(original);
  });
});

describe("isValidHex", () => {
  it("returns true for valid hex colors", () => {
    expect(isValidHex("#ff0000")).toBe(true);
    expect(isValidHex("#000000")).toBe(true);
    expect(isValidHex("#abcdef")).toBe(true);
    expect(isValidHex("#ABCDEF")).toBe(true);
    expect(isValidHex("#123456")).toBe(true);
  });

  it("returns false for invalid hex colors", () => {
    expect(isValidHex("ff0000")).toBe(false);
    expect(isValidHex("#fff")).toBe(false);
    expect(isValidHex("#ff")).toBe(false);
    expect(isValidHex("#ff00000")).toBe(false);
    expect(isValidHex("")).toBe(false);
    expect(isValidHex("#gggggg")).toBe(false);
  });
});
