import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// sw.js is a classic service worker, not a module: it runs top-level
// self.addEventListener calls and cannot be imported. Evaluating it against a
// stub global gives the pure logic without a browser, and keeps the worker one
// file rather than splitting it just to be testable.
function loadWorker() {
  const source = readFileSync("internal/site/assets/sw.js", "utf8");
  const self = { addEventListener() {}, skipWaiting() {}, clients: { claim() {} } };
  new Function("self", source)(self);
  return self.SWLogic;
}

const SW = loadWorker();

describe("classify", () => {
  it("sends a page to the network first", () => {
    expect(SW.classify("/")).toBe("page");
    expect(SW.classify("/games")).toBe("page");
    expect(SW.classify("/games/tetrabros")).toBe("page");
  });

  it("sends a hashed asset to the cache first", () => {
    expect(SW.classify("/assets/build/site.f0c84817.css")).toBe("asset");
    expect(SW.classify("/assets/build/fonts/vt323.woff2")).toBe("asset");
  });

  // Media keys are not content-hashed, so a cache-first copy would outlive a
  // deleted image forever.
  it("sends an upload to the network first", () => {
    expect(SW.classify("/media/cd_cover_art/2026-abc.webp")).toBe("media");
  });

  // A game's iframe is a navigation too, so the prefix has to be checked
  // before the request type.
  it("sends a game build to its own cache", () => {
    expect(SW.classify("/play/tetrabros/")).toBe("play");
    expect(SW.classify("/play/tetrabros/tetrabros.wasm")).toBe("play");
  });

  it("never touches the API or the admin panel", () => {
    expect(SW.classify("/api/shell")).toBe("bypass");
    expect(SW.classify("/api/games/tetrabros/build")).toBe("bypass");
    expect(SW.classify("/I-am-a-pixabro/")).toBe("bypass");
    expect(SW.classify("/I-am-a-pixabro/games")).toBe("bypass");
  });
});

describe("game caches", () => {
  it("round-trips a name", () => {
    const name = SW.gameCacheName("tetrabros", "a1b2c3d4e5f60718");
    expect(SW.parseGameCache(name)).toEqual({
      slug: "tetrabros",
      version: "a1b2c3d4e5f60718",
    });
  });

  // Slugs carry hyphens, so a naive split on "-" would read the wrong slug.
  it("round-trips a hyphenated slug", () => {
    const name = SW.gameCacheName("dungrid-tactics", "0011223344556677");
    expect(SW.parseGameCache(name)).toEqual({
      slug: "dungrid-tactics",
      version: "0011223344556677",
    });
  });

  it("refuses a name from another family", () => {
    expect(SW.parseGameCache("pages")).toBeNull();
    expect(SW.parseGameCache("shell-abc123")).toBeNull();
  });

  it("reads the slug out of a play path", () => {
    expect(SW.playSlug("/play/tetrabros/tetrabros.wasm")).toBe("tetrabros");
    expect(SW.playSlug("/play/tetrabros/")).toBe("tetrabros");
    expect(SW.playSlug("/games/tetrabros")).toBeNull();
  });

  it("names the completion marker inside the build's own scope", () => {
    expect(SW.completeKey("tetrabros")).toBe("/play/tetrabros/__offline-complete");
  });
});

describe("isKeepable", () => {
  // Activate drops anything outside the four families, so a renamed cache from
  // an older worker cannot linger and hold storage the visitor cannot see.
  it("keeps the current shell and drops an older one", () => {
    expect(SW.isKeepable("shell-abc123def456", "abc123def456")).toBe(true);
    expect(SW.isKeepable("shell-000000000000", "abc123def456")).toBe(false);
  });

  it("keeps the bounded runtime caches", () => {
    expect(SW.isKeepable("pages", "abc123def456")).toBe(true);
    expect(SW.isKeepable("media", "abc123def456")).toBe(true);
  });

  // A downloaded game outlives shell versions on purpose: the visitor decides
  // when to spend 90 MB again, not a deploy.
  it("keeps every game cache whatever its version", () => {
    expect(SW.isKeepable("game-tetrabros-0011223344556677", "abc123def456")).toBe(true);
  });

  it("drops anything else", () => {
    expect(SW.isKeepable("leftovers", "abc123def456")).toBe(false);
  });
});
