import { describe, expect, it } from "vitest";
import {
  DAYS_EN,
  DAYS_TR,
  formatDateDisplay,
  getDaysInMonth,
  getFirstDay,
  localeData,
  MONTHS_EN,
  MONTHS_TR,
} from "@/utilities/date-locale";

describe("DAYS constants", () => {
  it("DAYS_EN has 7 entries", () => {
    expect(DAYS_EN).toHaveLength(7);
  });

  it("DAYS_TR has 7 entries", () => {
    expect(DAYS_TR).toHaveLength(7);
  });
});

describe("MONTHS constants", () => {
  it("MONTHS_EN has 12 entries", () => {
    expect(MONTHS_EN).toHaveLength(12);
  });

  it("MONTHS_TR has 12 entries", () => {
    expect(MONTHS_TR).toHaveLength(12);
  });
});

describe("localeData", () => {
  it("has en, tr keys", () => {
    expect(Object.keys(localeData)).toEqual(["en", "tr"]);
  });

  it("each locale has days and months", () => {
    for (const key of Object.keys(localeData)) {
      expect(localeData[key]).toHaveProperty("days");
      expect(localeData[key]).toHaveProperty("months");
      expect(localeData[key].days).toHaveLength(7);
      expect(localeData[key].months).toHaveLength(12);
    }
  });
});

describe("getDaysInMonth", () => {
  it("returns 31 for January", () => {
    expect(getDaysInMonth(2024, 0)).toBe(31);
  });

  it("returns 28 for February in non-leap year", () => {
    expect(getDaysInMonth(2023, 1)).toBe(28);
  });

  it("returns 29 for February in leap year", () => {
    expect(getDaysInMonth(2024, 1)).toBe(29);
  });

  it("returns 30 for April", () => {
    expect(getDaysInMonth(2024, 3)).toBe(30);
  });

  it("returns 31 for December", () => {
    expect(getDaysInMonth(2024, 11)).toBe(31);
  });
});

describe("getFirstDay", () => {
  it("returns a day index 0-6", () => {
    const day = getFirstDay(2024, 0);
    expect(day).toBeGreaterThanOrEqual(0);
    expect(day).toBeLessThanOrEqual(6);
  });

  it("returns 1 for January 2024 (Monday)", () => {
    expect(getFirstDay(2024, 0)).toBe(1);
  });

  it("returns correct day index for March 2024", () => {
    expect(getFirstDay(2024, 2)).toBe(new Date(2024, 2, 1).getDay());
  });
});

describe("formatDateDisplay", () => {
  it("formats date with en locale", () => {
    const result = formatDateDisplay("2024-01-15", "en");
    expect(result).toContain("2024");
  });

  it("formats date with tr locale", () => {
    const result = formatDateDisplay("2024-01-15", "tr");
    expect(result).toBeTruthy();
  });
});
