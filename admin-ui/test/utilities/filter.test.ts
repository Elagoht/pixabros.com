import { describe, expect, it } from "vitest";
import {
  booleanFilterFn,
  chipFilterFn,
  dateFilterFn,
  filterFnMap,
  numberFilterFn,
} from "@/utilities/filter";

const makeRow = (value: unknown) => ({ getValue: (_id: string) => value });

describe("numberFilterFn", () => {
  it("returns true when filterValue is undefined", () => {
    expect(numberFilterFn(makeRow(10), "col", undefined)).toBe(true);
  });

  it("returns true when filterValue is not an object", () => {
    expect(numberFilterFn(makeRow(10), "col", "string")).toBe(true);
  });

  it("returns true when op is missing", () => {
    expect(numberFilterFn(makeRow(10), "col", { value: 10 })).toBe(true);
  });

  it("returns true when value is undefined", () => {
    expect(
      numberFilterFn(makeRow(10), "col", { op: undefined, value: 10 }),
    ).toBe(true);
  });

  it("filters with = operator", () => {
    expect(numberFilterFn(makeRow(10), "col", { op: "=", value: 10 })).toBe(
      true,
    );
    expect(numberFilterFn(makeRow(10), "col", { op: "=", value: 5 })).toBe(
      false,
    );
  });

  it("filters with > operator", () => {
    expect(numberFilterFn(makeRow(10), "col", { op: ">", value: 5 })).toBe(
      true,
    );
    expect(numberFilterFn(makeRow(10), "col", { op: ">", value: 10 })).toBe(
      false,
    );
  });

  it("filters with >= operator", () => {
    expect(numberFilterFn(makeRow(10), "col", { op: ">=", value: 10 })).toBe(
      true,
    );
    expect(numberFilterFn(makeRow(10), "col", { op: ">=", value: 11 })).toBe(
      false,
    );
  });

  it("filters with < operator", () => {
    expect(numberFilterFn(makeRow(5), "col", { op: "<", value: 10 })).toBe(
      true,
    );
    expect(numberFilterFn(makeRow(10), "col", { op: "<", value: 10 })).toBe(
      false,
    );
  });

  it("filters with <= operator", () => {
    expect(numberFilterFn(makeRow(10), "col", { op: "<=", value: 10 })).toBe(
      true,
    );
    expect(numberFilterFn(makeRow(11), "col", { op: "<=", value: 10 })).toBe(
      false,
    );
  });

  it("returns true when cell value is NaN", () => {
    expect(numberFilterFn(makeRow("abc"), "col", { op: "=", value: 10 })).toBe(
      true,
    );
  });
});

describe("dateFilterFn", () => {
  it("returns true when filterValue is undefined", () => {
    expect(dateFilterFn(makeRow("2024-01-01"), "col", undefined)).toBe(true);
  });

  it("returns true when no start or end", () => {
    expect(dateFilterFn(makeRow("2024-06-15"), "col", {})).toBe(true);
  });

  it("filters with start date", () => {
    const row = makeRow("2024-06-15T00:00:00");
    expect(dateFilterFn(row, "col", { start: "2024-06-01" })).toBe(true);
    expect(dateFilterFn(row, "col", { start: "2024-07-01" })).toBe(false);
  });

  it("filters with end date", () => {
    const row = makeRow("2024-06-15T00:00:00");
    expect(dateFilterFn(row, "col", { end: "2024-07-01" })).toBe(true);
    expect(dateFilterFn(row, "col", { end: "2024-06-01" })).toBe(false);
  });

  it("filters with start and end date range", () => {
    const row = makeRow("2024-06-15T00:00:00");
    expect(
      dateFilterFn(row, "col", { start: "2024-06-01", end: "2024-07-01" }),
    ).toBe(true);
    expect(
      dateFilterFn(row, "col", { start: "2024-07-01", end: "2024-08-01" }),
    ).toBe(false);
  });

  it("returns true for NaN date value", () => {
    expect(
      dateFilterFn(makeRow("not-a-date"), "col", { start: "2024-01-01" }),
    ).toBe(true);
  });
});

describe("booleanFilterFn", () => {
  it("returns true when filterValue is undefined", () => {
    expect(booleanFilterFn(makeRow(true), "col", undefined)).toBe(true);
  });

  it("returns true when filterValue is 'all'", () => {
    expect(booleanFilterFn(makeRow(true), "col", "all")).toBe(true);
  });

  it("filters true values", () => {
    expect(booleanFilterFn(makeRow(true), "col", "true")).toBe(true);
    expect(booleanFilterFn(makeRow(false), "col", "true")).toBe(false);
  });

  it("filters false values", () => {
    expect(booleanFilterFn(makeRow(false), "col", "false")).toBe(true);
    expect(booleanFilterFn(makeRow(true), "col", "false")).toBe(false);
  });
});

describe("chipFilterFn", () => {
  it("returns true when filterValue is undefined", () => {
    expect(chipFilterFn(makeRow("active"), "col", undefined)).toBe(true);
  });

  it("returns true when filterValue is empty array", () => {
    expect(chipFilterFn(makeRow("active"), "col", [])).toBe(true);
  });

  it("returns true when value is in the filter array", () => {
    expect(chipFilterFn(makeRow("active"), "col", ["active", "pending"])).toBe(
      true,
    );
  });

  it("returns false when value is not in the filter array", () => {
    expect(chipFilterFn(makeRow("closed"), "col", ["active", "pending"])).toBe(
      false,
    );
  });

  it("handles null cell value", () => {
    expect(chipFilterFn(makeRow(null), "col", ["active"])).toBe(false);
  });
});

describe("filterFnMap", () => {
  it("maps number type to numberFilterFn", () => {
    expect(filterFnMap.number).toBe(numberFilterFn);
  });

  it("maps money type to numberFilterFn", () => {
    expect(filterFnMap.money).toBe(numberFilterFn);
  });

  it("maps progress type to numberFilterFn", () => {
    expect(filterFnMap.progress).toBe(numberFilterFn);
  });

  it("maps date type to dateFilterFn", () => {
    expect(filterFnMap.date).toBe(dateFilterFn);
  });

  it("maps boolean type to booleanFilterFn", () => {
    expect(filterFnMap.boolean).toBe(booleanFilterFn);
  });

  it("maps chip type to chipFilterFn", () => {
    expect(filterFnMap.chip).toBe(chipFilterFn);
  });
});
