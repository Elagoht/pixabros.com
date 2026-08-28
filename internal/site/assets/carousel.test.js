import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

let source = "";
try {
  source = readFileSync("internal/site/assets/carousel.js", "utf8");
} catch {}

const loadPager = (...args) => {
  const window = {};
  const document = { querySelector: () => null, querySelectorAll: () => [] };
  new Function("window", "document", source)(window, document);
  return window.CarouselLogic.createPager(...args);
};

describe("carousel pager", () => {
  // The whole point of the pager: a click answers with its target at once, so
  // the dots move on the click rather than when the scroll finishes arriving.
  // It also means rapid clicks accumulate -- each steps from the last target,
  // not from wherever the mid-flight track happens to be.
  it("steps one slide per click and answers immediately", () => {
    const pager = loadPager(3);
    const answers = [pager.go(1), pager.go(1), pager.go(1)];
    expect(answers).toEqual([1, 2, 0]);
  });

  it("wraps backwards past the first slide", () => {
    const pager = loadPager(3);
    expect(pager.go(-1)).toBe(2);
  });

  it("starts where the track starts", () => {
    expect(loadPager(4, 2).current()).toBe(2);
  });

  // A drag or swipe moves the track without going through go(), so once the
  // track settles the pager hands control back to the scroll position.
  it("follows the track when it settles somewhere else", () => {
    const pager = loadPager(3);
    pager.go(1);
    expect(pager.goTo(0)).toBe(0);
    expect(pager.go(1)).toBe(1);
  });
});
