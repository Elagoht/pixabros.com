import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const OFFLINE_SOURCE = readFileSync("internal/site/assets/offline.js", "utf8");
const WORKER_SOURCE = readFileSync("internal/site/assets/sw.js", "utf8");

// offline.js is a plain script guarded by an IIFE. Evaluating it against a
// stub window exposes its pure helpers without a browser. fetch is a parameter
// rather than the global so nothing here can reach the network by accident.
function loadOffline() {
  const window = { addEventListener() {} };
  const navigator = {};
  const document = { querySelector: () => null, querySelectorAll: () => [] };
  const fetch = () => Promise.reject(new Error("no network in a test"));
  new Function("window", "navigator", "document", "fetch", OFFLINE_SOURCE)(
    window,
    navigator,
    document,
    fetch
  );
  return window.OfflineLogic;
}

// loadWorker mirrors sw.test.js: sw.js is a classic service worker, evaluated
// against a stub self to reach SWLogic without a browser.
function loadWorker() {
  const self = { addEventListener() {}, skipWaiting() {}, clients: { claim() {} } };
  new Function("self", WORKER_SOURCE)(self);
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
  // system, so each keeps its own copy of the completion-marker key, the
  // game-cache prefix and the rule for reading a cache name back apart.
  // Nothing else stops those copies from drifting apart -- if they ever
  // disagree, a downloaded game stops being recognised as held, or worse, the
  // page deletes a cache the worker considers someone else's. This test is the
  // tripwire.
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

  // The "har" / "har-2" pair is the one that matters: a prefix test would read
  // har-2's cache as har's, find no marker for har in it, and delete another
  // game's finished 90 MB download. Both sides have to split at the last
  // hyphen, and both sides have to agree that they do.
  it("agrees with SWLogic on reading a cache name apart", () => {
    const names = [
      "game-har-0011223344556677",
      "game-har-2-0011223344556677",
      "game-tetrabros-a1b2c3d4e5f60718",
      "game-dungrid-tactics-0011223344556677",
      "game-",
      "game-nover-",
      "pages",
      "media",
      "shell-abc123def456",
    ];
    for (const name of names) {
      expect(Offline.parseGameCache(name)).toEqual(SW.parseGameCache(name));
    }

    expect(Offline.parseGameCache("game-har-2-0011223344556677").slug).toBe("har-2");
    expect(Offline.parseGameCache("game-har-0011223344556677").slug).toBe("har");
  });
});

// ---------------------------------------------------------------------------
// The download itself.
//
// Everything above tests pure functions, and every defect found in download()
// so far lived outside them -- in the wiring between a cache, a fetch and the
// control. A worker serving the old build back into the new version's cache
// passed the pure tests, the Go tests, the asset pipeline and a curl by hand.
// So the wiring gets a fake browser: small enough to read, complete enough to
// fail.
// ---------------------------------------------------------------------------

// fakeElement is as much DOM as offline.js touches. Setting textContent to ""
// drops the children, because that is what a real node does and it is how the
// control clears itself -- a fake that kept them would let a test pass while
// looking at the previous state.
function fakeElement(tag) {
  const el = {
    tagName: tag,
    className: "",
    type: "",
    children: [],
    attributes: {},
    listeners: {},
    appendChild(child) {
      el.children.push(child);
      return child;
    },
    addEventListener(name, handler) {
      el.listeners[name] = el.listeners[name] || [];
      el.listeners[name].push(handler);
    },
    setAttribute(name, value) {
      el.attributes[name] = String(value);
    },
    getAttribute(name) {
      return name in el.attributes ? el.attributes[name] : null;
    },
    click() {
      (el.listeners.click || []).forEach((handler) => handler());
    },
  };
  let own = "";
  Object.defineProperty(el, "textContent", {
    get: () => own,
    set(value) {
      own = String(value);
      el.children.length = 0;
    },
  });
  return el;
}

// fakeCaches is Cache Storage with a log. Order is what most of these tests are
// about -- the completion marker landing last, the previous version going only
// after it has -- so every write and every deletion joins one sequence.
function fakeCaches(log) {
  const stores = new Map();

  function handle(name) {
    const entries = stores.get(name);
    return {
      put(key, response) {
        entries.set(String(key), response);
        log.push({ op: "put", cache: name, key: String(key) });
        return Promise.resolve();
      },
      add(key) {
        entries.set(String(key), new Response("copy of " + key));
        log.push({ op: "add", cache: name, key: String(key) });
        return Promise.resolve();
      },
      match(key) {
        return Promise.resolve(entries.get(String(key)));
      },
      keys() {
        return Promise.resolve([...entries.keys()]);
      },
      delete(key) {
        return Promise.resolve(entries.delete(String(key)));
      },
    };
  }

  return {
    stores,
    open(name) {
      if (!stores.has(name)) {
        stores.set(name, new Map());
      }
      return Promise.resolve(handle(name));
    },
    keys() {
      return Promise.resolve([...stores.keys()]);
    },
    delete(name) {
      log.push({ op: "delete", cache: name });
      return Promise.resolve(stores.delete(name));
    },
    match(key) {
      for (const entries of stores.values()) {
        if (entries.has(String(key))) {
          return Promise.resolve(entries.get(String(key)));
        }
      }
      return Promise.resolve(undefined);
    },
  };
}

const SLUG = "tetrabros";
const VERSION = "a1b2c3d4e5f60718";
const OLD_VERSION = "0011223344556677";
const MANIFEST = {
  version: VERSION,
  bytes: 46137344,
  files: [
    { path: "index.html", bytes: 1048576 },
    { path: "tetrabros.wasm", bytes: 45088768 },
  ],
};

// start evaluates offline.js against the fake browser and hands back the parts
// a test needs to drive and inspect it. options.file decides how each build
// file's request behaves: "ok", "fail" (the network drops), or "hang" (a
// request that only ever ends by being aborted).
function start(options = {}) {
  const manifest = options.manifest || MANIFEST;
  const log = [];
  const caches = fakeCaches(log);
  const fetches = [];

  const seed = options.seed || {};
  // Seeded caches go in behind the log: they are the state the visit starts
  // from, not something this download did.
  for (const name of Object.keys(seed)) {
    const entries = new Map();
    for (const key of Object.keys(seed[name])) {
      entries.set(key, new Response(seed[name][key]));
    }
    caches.stores.set(name, entries);
  }

  function fetch(url, init) {
    const target = String(url);
    fetches.push({ url: target, init: init || null });

    if (target.indexOf("/api/games/") === 0) {
      return Promise.resolve(new Response(JSON.stringify(manifest)));
    }

    const plan = options.file ? options.file(target) : "ok";
    if (plan === "fail") {
      return Promise.reject(new Error("the network went away"));
    }
    if (plan === "hang") {
      return new Promise((resolve, reject) => {
        const signal = init && init.signal;
        if (!signal) {
          return;
        }
        signal.addEventListener("abort", () => {
          const aborted = new Error("The operation was aborted.");
          aborted.name = "AbortError";
          reject(aborted);
        });
      });
    }
    return Promise.resolve(new Response("bytes of " + target));
  }

  const mount = fakeElement("div");
  mount.setAttribute("data-offline-game", SLUG);

  const document = {
    querySelector: (selector) => (selector === "[data-offline-game]" ? mount : null),
    querySelectorAll: () => [],
    createElement: fakeElement,
  };
  const window = {
    addEventListener() {},
    caches,
    location: { pathname: "/games/" + SLUG },
  };
  const navigator = {
    serviceWorker: { register: () => Promise.resolve() },
    storage: {
      estimate: () => Promise.resolve({ quota: 1e12, usage: 0 }),
      persist: () => Promise.resolve(true),
    },
  };

  new Function("window", "navigator", "document", "fetch", OFFLINE_SOURCE)(
    window,
    navigator,
    document,
    fetch
  );

  return { control: mount.children[0], caches, log, fetches, manifest };
}

// settle waits for the script's own promise chains. There are no timers in
// them, so one macrotask turn drains any number of microtasks; the loop is
// there because a chain can span several fetches.
async function settle(condition, what) {
  for (let turn = 0; turn < 200; turn += 1) {
    if (condition()) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  throw new Error("timed out waiting for " + what);
}

function labels(control) {
  return control.children.map((child) => child.textContent);
}

function shows(control, pattern) {
  return labels(control).some((label) => pattern.test(label));
}

function press(control, pattern) {
  const target = control.children.find((child) => pattern.test(child.textContent));
  if (!target) {
    throw new Error("nothing matching " + pattern + " in " + JSON.stringify(labels(control)));
  }
  target.click();
}

function gameCaches(caches) {
  return [...caches.stores.keys()].filter((name) => name.indexOf("game-") === 0);
}

function copyRequests(fetches) {
  return fetches.filter((call) => call.url.indexOf("/play/") === 0);
}

describe("download", () => {
  it("writes every file of the build, then the marker, and says so", async () => {
    const harness = start();
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(() => labels(harness.control).includes("Playable offline"), "the download");

    const cache = "game-" + SLUG + "-" + VERSION;
    const written = harness.log
      .filter((entry) => entry.op === "put" && entry.cache === cache)
      .map((entry) => entry.key);

    // Every manifest path, and the marker last: a cache carrying the marker is
    // a promise that the whole build is behind it.
    expect(written).toEqual([
      "/play/tetrabros/index.html",
      "/play/tetrabros/tetrabros.wasm",
      "/play/tetrabros/__offline-complete",
    ]);
  });

  // The regression test for the update that never downloaded anything: without
  // this header the worker answers each copy request out of the *old* game
  // cache, so the new version's cache fills with the old build's bytes and the
  // marker then lies about which build is on the device.
  it("marks its copy requests so the worker stands aside", async () => {
    const harness = start();
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(() => labels(harness.control).includes("Playable offline"), "the download");

    const copies = copyRequests(harness.fetches);
    expect(copies.map((call) => call.url)).toEqual([
      "/play/tetrabros/index.html",
      "/play/tetrabros/tetrabros.wasm",
    ]);
    for (const call of copies) {
      expect(call.init && call.init.headers).toEqual({ "X-Offline-Copy": "1" });
    }
  });

  it("encodes a file name the URL would otherwise cut short", async () => {
    const harness = start({
      manifest: {
        version: VERSION,
        bytes: 2048,
        files: [{ path: "data/level #1.pck", bytes: 2048 }],
      },
    });
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(() => labels(harness.control).includes("Playable offline"), "the download");

    expect(copyRequests(harness.fetches).map((call) => call.url)).toEqual([
      "/play/tetrabros/data/level%20%231.pck",
    ]);
  });

  it("fetches the new build afresh and drops the old copy only once the marker lands", async () => {
    const old = "game-" + SLUG + "-" + OLD_VERSION;
    const harness = start({
      seed: { [old]: { "/play/tetrabros/__offline-complete": OLD_VERSION } },
    });
    await settle(() => shows(harness.control, /^Update/), "the update offer");

    press(harness.control, /^Update/);
    await settle(() => labels(harness.control).includes("Playable offline"), "the update");

    // Every byte came off the network, into the new version's own cache.
    expect(copyRequests(harness.fetches)).toHaveLength(MANIFEST.files.length);
    expect(gameCaches(harness.caches)).toEqual(["game-" + SLUG + "-" + VERSION]);

    // An update interrupted between the two must never cost the visitor the
    // game they already had, so the order is not incidental.
    const marker = harness.log.findIndex(
      (entry) => entry.op === "put" && entry.key.endsWith("__offline-complete")
    );
    const dropped = harness.log.findIndex(
      (entry) => entry.op === "delete" && entry.cache === old
    );
    expect(marker).toBeGreaterThanOrEqual(0);
    expect(dropped).toBeGreaterThan(marker);
  });

  it("leaves nothing behind when the network drops mid-build", async () => {
    const harness = start({
      file: (url) => (url.endsWith(".wasm") ? "fail" : "ok"),
    });
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(() => shows(harness.control, /Could not save/), "the failure");

    // A half-built game does not start, so there must be no cache left for
    // heldVersion to find -- with or without its marker.
    expect(gameCaches(harness.caches)).toEqual([]);
    expect(shows(harness.control, /^Try again/)).toBe(true);
  });

  it("stops when asked, keeps nothing, and does not call it a failure", async () => {
    const harness = start({
      file: (url) => (url.endsWith(".wasm") ? "hang" : "ok"),
    });
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(
      () => harness.fetches.some((call) => call.url.endsWith(".wasm")),
      "the large file to start"
    );

    press(harness.control, /^Cancel$/);
    await settle(() => shows(harness.control, /Make playable offline/), "the control to return");

    expect(gameCaches(harness.caches)).toEqual([]);
    // Stopping a download you started is an ordinary thing to do. It must not
    // read like something went wrong.
    expect(shows(harness.control, /Could not save/)).toBe(false);
    expect(labels(harness.control)).toEqual(["Make playable offline — 44 MB"]);
  });

  it("announces its progress to a screen reader", async () => {
    const harness = start({ file: (url) => (url.endsWith(".wasm") ? "hang" : "ok") });
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    press(harness.control, /Make playable offline/);
    await settle(() => shows(harness.control, /Downloading… 2%/), "the first file to land");

    const status = harness.control.children[0];
    expect(status.getAttribute("aria-live")).toBe("polite");
    expect(status.getAttribute("aria-busy")).toBe("true");
  });

  // "game-tetrabros-" also prefixes "game-tetrabros-2-<version>". Selecting
  // this game's caches by prefix would find the neighbour, fail to find *this*
  // game's marker in it, and delete a finished download that belongs to another
  // game entirely.
  it("never touches a neighbouring game's download", async () => {
    const neighbour = "game-" + SLUG + "-2-" + OLD_VERSION;
    const harness = start({
      seed: { [neighbour]: { "/play/tetrabros-2/__offline-complete": OLD_VERSION } },
    });
    await settle(() => shows(harness.control, /Make playable offline/), "the offer");

    expect(harness.caches.stores.has(neighbour)).toBe(true);
    expect(harness.log.some((entry) => entry.op === "delete")).toBe(false);
  });
});
