import { describe, it, expect } from "vitest";
import {
  detectCycle,
  getAllDescendantIds,
  applyDrop,
  getDepth,
  hasChildren,
} from "@/utilities/hierarchy-table";

describe("detectCycle", () => {
  const items = [
    { id: "1", parentId: null },
    { id: "2", parentId: "1" },
    { id: "3", parentId: "2" },
  ];

  it("returns false when moving to root (null parent)", () => {
    expect(detectCycle(items, "1", null)).toBe(false);
  });

  it("returns false when no cycle would be created", () => {
    expect(detectCycle(items, "3", "1")).toBe(false);
  });

  it("returns true when moving item under its own descendant creates cycle", () => {
    expect(detectCycle(items, "1", "3")).toBe(true);
  });

  it("returns false when moving item to sibling", () => {
    expect(detectCycle(items, "2", "1")).toBe(false);
  });

  it("returns true for direct self-reference", () => {
    expect(detectCycle(items, "1", "1")).toBe(true);
  });
});

describe("getAllDescendantIds", () => {
  const items = [
    { id: "1", parentId: null },
    { id: "2", parentId: "1" },
    { id: "3", parentId: "1" },
    { id: "4", parentId: "2" },
    { id: "5", parentId: "4" },
  ];

  it("returns all descendants of root item", () => {
    expect(getAllDescendantIds(items, "1")).toEqual(new Set(["2", "3", "4", "5"]));
  });

  it("returns only direct children for leaf-like item", () => {
    expect(getAllDescendantIds(items, "3")).toEqual(new Set());
  });

  it("returns nested descendants", () => {
    expect(getAllDescendantIds(items, "2")).toEqual(new Set(["4", "5"]));
  });

  it("returns empty set for nonexistent parent", () => {
    expect(getAllDescendantIds(items, "999")).toEqual(new Set());
  });
});

describe("applyDrop", () => {
  const items = [
    { id: "1", parentId: null },
    { id: "2", parentId: "1" },
    { id: "3", parentId: "1" },
    { id: "4", parentId: null },
  ];

  it("moves item to root (null target)", () => {
    const result = applyDrop(items, "2", null, "child");
    const moved = result.find((i) => i.id === "2");
    expect(moved?.parentId).toBeNull();
  });

  it("moves item as child of target (position: child)", () => {
    const result = applyDrop(items, "4", "1", "child");
    const moved = result.find((i) => i.id === "4");
    expect(moved?.parentId).toBe("1");
  });

  it("moves item before target (position: before)", () => {
    const result = applyDrop(items, "4", "1", "before");
    const moved = result.find((i) => i.id === "4");
    expect(moved?.parentId).toBeNull();
    const idx4 = result.findIndex((i) => i.id === "4");
    const idx1 = result.findIndex((i) => i.id === "1");
    expect(idx4).toBeLessThan(idx1);
  });

  it("moves item after target (position: after)", () => {
    const result = applyDrop(items, "4", "1", "after");
    const moved = result.find((i) => i.id === "4");
    expect(moved?.parentId).toBeNull();
    const idx4 = result.findIndex((i) => i.id === "4");
    const idx1 = result.findIndex((i) => i.id === "1");
    expect(idx4).toBeGreaterThan(idx1);
  });

  it("does not move item into its own descendant", () => {
    const result = applyDrop(items, "1", "2", "child");
    expect(result).toEqual(items);
  });

  it("returns same items if target not found", () => {
    const result = applyDrop(items, "4", "999", "child");
    expect(result).toEqual(items);
  });
});

describe("getDepth", () => {
  const items = [
    { id: "1", parentId: null },
    { id: "2", parentId: "1" },
    { id: "3", parentId: "2" },
  ];

  it("returns 0 for root item", () => {
    expect(getDepth(items, "1")).toBe(0);
  });

  it("returns 1 for child of root", () => {
    expect(getDepth(items, "2")).toBe(1);
  });

  it("returns 2 for grandchild", () => {
    expect(getDepth(items, "3")).toBe(2);
  });

  it("returns 0 for item not in the map (root)", () => {
    expect(getDepth([...items, { id: "4", parentId: "3" }], "4")).toBe(3);
  });
});

describe("hasChildren", () => {
  const items = [
    { id: "1", parentId: null },
    { id: "2", parentId: "1" },
    { id: "3", parentId: null },
  ];

  it("returns true for item with children", () => {
    expect(hasChildren(items, "1")).toBe(true);
  });

  it("returns false for item without children", () => {
    expect(hasChildren(items, "2")).toBe(false);
  });

  it("returns false for item with no children", () => {
    expect(hasChildren(items, "3")).toBe(false);
  });
});