import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";

vi.mock("@/lib/stores/i18n", () => ({
  useI18n: {
    getState: vi.fn(() => ({ locale: "tr" })),
  },
}));

import {
  formatMoney,
  formatDate,
  formatNumber,
  renderChip,
  renderBoolean,
  renderProgress,
  renderCellContent,
} from "@/utilities/format";

describe("formatMoney", () => {
  it("formats number as TRY currency by default", () => {
    const result = formatMoney(1234.56);
    expect(result).toContain("1.234,56");
  });

  it("formats 0", () => {
    const result = formatMoney(0);
    expect(result).toContain("0,00");
  });

  it("returns string for NaN value", () => {
    expect(formatMoney("abc")).toBe("abc");
  });

  it("returns formatted zero for null (coerced to 0)", () => {
    expect(typeof formatMoney(null)).toBe("string");
  });

  it("returns formatted string for undefined (coerced to 0)", () => {
    expect(typeof formatMoney(undefined)).toBe("string");
  });

  it("supports custom currency option", () => {
    const result = formatMoney(100, { currency: "USD" });
    expect(result).toContain("100");
  });
});

describe("formatDate", () => {
  it("returns empty string when value is falsy", () => {
    expect(formatDate("")).toBe("");
    expect(formatDate(null)).toBe("");
  });

  it("formats date string as date by default", () => {
    const result = formatDate("2024-01-15");
    expect(result).toContain("2024");
  });

  it("formats date as datetime", () => {
    const result = formatDate("2024-01-15T10:30:00", { format: "datetime" });
    expect(result).toContain("2024");
  });

  it("formats date as time", () => {
    const result = formatDate("2024-01-15T10:30:00", { format: "time" });
    expect(result).toBeTruthy();
  });

  it("returns string for invalid date", () => {
    expect(formatDate("not-a-date")).toBe("not-a-date");
  });
});

describe("formatNumber", () => {
  it("formats number with default precision", () => {
    const result = formatNumber(1234);
    expect(result).toContain("1.234");
  });

  it("formats as percent when format is percent", () => {
    expect(formatNumber(75, { format: "percent" })).toBe("%75");
  });

  it("formats with custom precision", () => {
    const result = formatNumber(12.345, { precision: 2 });
    expect(result).toContain("12,35");
  });

  it("returns string for NaN value", () => {
    expect(formatNumber("abc")).toBe("abc");
  });

  it("returns formatted string for null (coerced to 0)", () => {
    expect(typeof formatNumber(null)).toBe("string");
  });
});

describe("renderChip", () => {
  it("renders chip with value as text", () => {
    const { container } = render(renderChip("active"));
    expect(container.textContent).toContain("active");
  });

  it("renders chip with label from labels map", () => {
    const { container } = render(renderChip("active", undefined, { active: "Aktif" }));
    expect(container.textContent).toContain("Aktif");
  });

  it("renders chip with default gray color", () => {
    const { container } = render(renderChip("test"));
    expect(container.querySelector("span")).toBeTruthy();
  });

  it("renders chip with specified color", () => {
    const { container } = render(renderChip("test", { colors: { test: "green" } }));
    expect(container.querySelector("span")).toBeTruthy();
  });
});

describe("renderBoolean", () => {
  it("renders true value", () => {
    const { container } = render(renderBoolean(true));
    expect(container.textContent).toContain("True");
  });

  it("renders false value", () => {
    const { container } = render(renderBoolean(false));
    expect(container.textContent).toContain("False");
  });

  it("renders string 'true' as True", () => {
    const { container } = render(renderBoolean("true"));
    expect(container.textContent).toContain("True");
  });

  it("renders number 1 as True", () => {
    const { container } = render(renderBoolean(1));
    expect(container.textContent).toContain("True");
  });

  it("renders with custom trueLabel", () => {
    const { container } = render(renderBoolean(true, { trueLabel: "Yes" }));
    expect(container.textContent).toContain("Yes");
  });

  it("renders with custom falseLabel", () => {
    const { container } = render(renderBoolean(false, { falseLabel: "No" }));
    expect(container.textContent).toContain("No");
  });
});

describe("renderProgress", () => {
  it("renders progress bar", () => {
    const { container } = render(renderProgress(50));
    expect(container.querySelector("div")).toBeTruthy();
  });

  it("renders percentage text", () => {
    const { container } = render(renderProgress(75));
    expect(container.textContent).toContain("75%");
  });

  it("clamps value to 0-100", () => {
    const { container } = render(renderProgress(150));
    expect(container.textContent).toContain("100%");
  });

  it("clamps negative values to 0", () => {
    const { container } = render(renderProgress(-10));
    expect(container.textContent).toContain("0%");
  });

  it("returns string for NaN value", () => {
    expect(renderProgress("abc")).toBe("abc");
  });

  it("respects custom max option", () => {
    const { container } = render(renderProgress(50, { max: 200 }));
    expect(container.textContent).toContain("25%");
  });
});

describe("renderCellContent", () => {
  it("calls custom cell renderer when provided", () => {
    const result = renderCellContent("hello", {} as Record<string, unknown>, { cell: (v: unknown) => `custom: ${v}`, id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBe("custom: hello");
  });

  it("formats money type", () => {
    const result = renderCellContent(100, {} as Record<string, unknown>, { type: "money", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(typeof result).toBe("string");
  });

  it("formats date type", () => {
    const result = renderCellContent("2024-01-15", {} as Record<string, unknown>, { type: "date", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toContain("2024");
  });

  it("renders boolean type", () => {
    const result = renderCellContent(true, {} as Record<string, unknown>, { type: "boolean", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBeTruthy();
  });

  it("renders chip type", () => {
    const result = renderCellContent("active", {} as Record<string, unknown>, { type: "chip", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBeTruthy();
  });

  it("renders number type", () => {
    const result = renderCellContent(42, {} as Record<string, unknown>, { type: "number", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(typeof result).toBe("string");
  });

  it("renders progress type", () => {
    const result = renderCellContent(50, {} as Record<string, unknown>, { type: "progress", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBeTruthy();
  });

  it("renders actions type", () => {
    const result = renderCellContent({}, { id: 1 }, { type: "actions", actions: [], id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBeNull();
  });

  it("defaults to string rendering", () => {
    const result = renderCellContent("hello", {} as Record<string, unknown>, { type: "string", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBe("hello");
  });

  it("renders null/undefined as empty string by default", () => {
    const result = renderCellContent(null, {} as Record<string, unknown>, { type: "string", id: "test", header: "Test", accessor: "test" } as DataTableColumn<Record<string, unknown>>);
    expect(result).toBe("");
  });
});