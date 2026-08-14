import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// offline.js is a plain script guarded by an IIFE. Evaluating it against a
// stub window exposes its pure helpers without a browser.
function loadOffline() {
  const source = readFileSync("internal/site/assets/offline.js", "utf8");
  const window = { addEventListener() {} };
  const navigator = {};
  const document = { querySelector: () => null, querySelectorAll: () => [] };
  new Function("window", "navigator", "document", source)(window, navigator, document);
  return window.OfflineLogic;
}

// loadWorker mirrors sw.test.js: sw.js is a classic service worker, evaluated
// against a stub self to reach SWLogic without a browser.
function loadWorker() {
  const source = readFileSync("internal/site/assets/sw.js", "utf8");
  const self = { addEventListener() {}, skipWaiting() {}, clients: { claim() {} } };
  new Function("self", source)(self);
  return self.SWLogic;
}

const Offline = loadOffline();
const SW = loadWorker();

describe("stateFor", () => {
  it("offers the download when nothing is held", () => {
    expect(Offline.stateFor(null, "a1b2c3d4e5f60718")).toBe("absent");
  });

  it("reports ready when the held version is current", () => {
    expect(Offline.stateFor("a1b2c3d4e5f60718", "a1b2c3d4e5f60718")).toBe("ready");
  });

  it("reports stale when a newer build exists", () => {
    expect(Offline.stateFor("0011223344556677", "a1b2c3d4e5f60718")).toBe("stale");
  });

  // Offline the build endpoint is unreachable, so there is no current version
  // to compare against. Claiming staleness that cannot be verified would be a
  // lie; a held copy is simply ready.
  it("never claims staleness it cannot verify", () => {
    expect(Offline.stateFor("0011223344556677", null)).toBe("ready");
    expect(Offline.stateFor(null, null)).toBe("unavailable");
  });
});

describe("formatBytes", () => {
  it("reads as a download size, not as a number", () => {
    expect(Offline.formatBytes(46137344)).toBe("44 MB");
    expect(Offline.formatBytes(1024)).toBe("1 KB");
    expect(Offline.formatBytes(0)).toBe("0 KB");
  });
});

describe("hasRoomFor", () => {
  // Refusing up front beats failing halfway through a 90 MB download.
  it("leaves headroom above the download", () => {
    expect(Offline.hasRoomFor({ quota: 1000, usage: 0 }, 100)).toBe(true);
    expect(Offline.hasRoomFor({ quota: 1000, usage: 950 }, 100)).toBe(false);
  });

  // A browser that will not estimate is not a browser that has no room.
  it("allows the attempt when the browser will not say", () => {
    expect(Offline.hasRoomFor(null, 100)).toBe(true);
    expect(Offline.hasRoomFor({}, 100)).toBe(true);
  });
});

describe("binds to sw.js", () => {
  // offline.js and sw.js run in separate global scopes with no shared module
  // system, so each keeps its own copy of the completion-marker key and the
  // game-cache prefix. Nothing else stops those copies from drifting apart --
  // if they ever disagree, a downloaded game stops being recognised as held.
  // This test is the tripwire.
  it("agrees with SWLogic on the completion marker key", () => {
    const slug = "tetrabros";
    expect(Offline.completeKey(slug)).toBe(SW.completeKey(slug));
    expect(Offline.completeKey("dungrid-tactics")).toBe(SW.completeKey("dungrid-tactics"));
  });

  it("agrees with SWLogic on the game cache name", () => {
    const slug = "tetrabros";
    const version = "a1b2c3d4e5f60718";
    expect(Offline.gameCacheName(slug, version)).toBe(SW.gameCacheName(slug, version));

    const hyphenated = Offline.gameCacheName("dungrid-tactics", "0011223344556677");
    expect(hyphenated).toBe(SW.gameCacheName("dungrid-tactics", "0011223344556677"));
    expect(SW.parseGameCache(hyphenated)).toEqual({
      slug: "dungrid-tactics",
      version: "0011223344556677",
    });
  });
});
