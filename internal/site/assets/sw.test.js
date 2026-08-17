import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// sw.js is a classic service worker, not a module: it runs top-level
// self.addEventListener calls and cannot be imported. Evaluating it against a
// stub global gives the pure logic without a browser, and keeps the worker one
// file rather than splitting it just to be testable.
//
// The registered handlers are captured too, so the parts of the worker that are
// not pure functions -- the fetch handler's decision to answer at all -- can be
// driven directly. caches and fetch are passed in as parameters so a test can
// see what the worker asked for.
function loadWorker(env = {}) {
  const source = readFileSync("internal/site/assets/sw.js", "utf8");
  const handlers = {};
  const self = {
    addEventListener(name, handler) {
      handlers[name] = handler;
    },
    skipWaiting() {},
    clients: { claim() {} },
    location: { origin: "https://pixabros.test" },
  };
  const caches = env.caches || { match: () => Promise.resolve(undefined) };
  const fetch = env.fetch || (() => Promise.resolve({ ok: true }));
  new Function("self", "caches", "fetch", source)(self, caches, fetch);
  return { logic: self.SWLogic, handlers };
}

const SW = loadWorker().logic;

describe("classify", () => {
  it("sends a page to the network first", () => {
    expect(SW.classify("/")).toBe("page");
    expect(SW.classify("/games")).toBe("page");
    expect(SW.classify("/games/tetrabros")).toBe("page");
  });

  it("sends a hashed asset to the cache first", () => {
    expect(SW.classify("/assets/build/site.f0c84817.css")).toBe("asset");
    expect(SW.classify("/assets/build/fonts/space-grotesk.woff2")).toBe("asset");
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

describe("cacheKeyFor", () => {
  // The console points the iframe at the directory, the manifest names
  // index.html, and caches.match is an exact-URL lookup with none of
  // http.FileServer's directory resolution. Without this mapping a fully
  // downloaded game does not start offline at all.
  it("resolves a directory to the index the download stored", () => {
    expect(SW.cacheKeyFor("/play/x/")).toBe("/play/x/index.html");
    expect(SW.cacheKeyFor("/play/dungrid-tactics/")).toBe("/play/dungrid-tactics/index.html");
  });

  it("leaves a file's own path alone", () => {
    expect(SW.cacheKeyFor("/play/x/y.wasm")).toBe("/play/x/y.wasm");
    expect(SW.cacheKeyFor("/play/x/index.html")).toBe("/play/x/index.html");
  });
});

// The shell version is the worker's one piece of real state, and both of the
// events that set it run once. Everything below is what happens when it is
// missing anyway -- a failed shell fetch, or the cold start after a browser
// terminated an idle worker.
describe("the shell version", () => {
  const shellList = { version: "abc123def456", urls: ["/offline"] };
  const servesShell = () =>
    Promise.resolve({ ok: true, json: () => Promise.resolve(shellList) });

  function activate(env) {
    const { handlers } = loadWorker(env);
    let running = null;
    handlers.activate({
      waitUntil(promise) {
        running = promise;
      },
    });
    return running;
  }

  // The window is a network drop between fetching /sw.js and activating, which
  // is exactly the flaky connection the worker exists for. With no version to
  // compare against, isKeepable calls every shell- cache droppable -- and the
  // cleanup would delete the stylesheet, the fonts, the icons and the /offline
  // page out from under an installed worker.
  it("deletes nothing when activate could not learn it", async () => {
    const deleted = [];
    await activate({
      fetch: () => Promise.reject(new Error("offline")),
      caches: {
        keys: () => Promise.resolve(["shell-abc123def456", "pages", "leftovers"]),
        delete(name) {
          deleted.push(name);
          return Promise.resolve(true);
        },
      },
    });
    expect(deleted).toEqual([]);
  });

  it("cleans up as usual once activate knows it", async () => {
    const deleted = [];
    await activate({
      fetch: servesShell,
      caches: {
        open: () => Promise.resolve({ addAll: () => Promise.resolve() }),
        keys: () =>
          Promise.resolve(["shell-abc123def456", "shell-000000000000", "pages", "leftovers"]),
        delete(name) {
          deleted.push(name);
          return Promise.resolve(true);
        },
      },
    });
    expect(deleted).toEqual(["shell-000000000000", "leftovers"]);
  });

  // A cold start runs neither install nor activate, so the in-memory value is
  // empty during the worker's ordinary work. Recovered from the cache names,
  // an asset lands where the rest of the shell is; not recovered, it lands in
  // a cache literally called "shell-".
  function serveAsset(env) {
    const { handlers } = loadWorker(env);
    let answer = null;
    const writes = [];
    handlers.fetch({
      request: {
        method: "GET",
        url: "https://pixabros.test/assets/build/site.f0c84817.css",
        mode: "no-cors",
        headers: { get: () => null },
      },
      respondWith(promise) {
        answer = promise;
      },
      waitUntil(promise) {
        writes.push(promise);
      },
    });
    return answer.then(() => Promise.all(writes));
  }

  it("is recovered from the cache names after a restart", async () => {
    const opened = [];
    await serveAsset({
      fetch: () => Promise.resolve({ ok: true, clone: () => ({}) }),
      caches: {
        keys: () => Promise.resolve(["pages", "shell-abc123def456", "game-x-0011223344556677"]),
        match: () => Promise.resolve(undefined),
        open(name) {
          opened.push(name);
          return Promise.resolve({ put: () => Promise.resolve() });
        },
      },
    });
    expect(opened).toEqual(["shell-abc123def456"]);
  });

  it("writes nowhere at all when there is no shell to recover it from", async () => {
    const opened = [];
    await serveAsset({
      fetch: () => Promise.resolve({ ok: true, clone: () => ({}) }),
      caches: {
        keys: () => Promise.resolve(["pages"]),
        match: () => Promise.resolve(undefined),
        open(name) {
          opened.push(name);
          return Promise.resolve({ put: () => Promise.resolve() });
        },
      },
    });
    expect(opened).toEqual([]);
  });
});

// playFirst is where the mapping above has to actually be used. A game that
// downloaded every byte still shows the browser's error page offline if the
// lookup asks for the URL the frame requested rather than the key the download
// wrote, so what the worker asks the cache for is the thing worth asserting.
describe("serving a downloaded game", () => {
  function play(pathname) {
    const asked = [];
    const caches = {
      match(key) {
        asked.push(typeof key === "string" ? key : key.url);
        return Promise.resolve(undefined);
      },
    };
    const { handlers } = loadWorker({
      caches,
      fetch: () => Promise.reject(new Error("offline")),
    });

    let answer = null;
    handlers.fetch({
      request: {
        method: "GET",
        url: "https://pixabros.test" + pathname,
        mode: "navigate",
        headers: { get: () => null },
      },
      respondWith(promise) {
        answer = promise;
      },
      waitUntil() {},
    });
    // Nothing has yielded yet, so asked holds the first lookup alone -- the
    // offline fallback's own lookup comes later.
    return { answer, asked: asked.slice() };
  }

  it("looks a directory up as the index the download stored", () => {
    expect(play("/play/tetrabros/").asked).toEqual(["/play/tetrabros/index.html"]);
  });

  it("looks a file up as the request the game made", () => {
    expect(play("/play/tetrabros/tetrabros.wasm").asked).toEqual([
      "https://pixabros.test/play/tetrabros/tetrabros.wasm",
    ]);
  });

  // A game nobody downloaded, opened with no network. A plain sentence beats
  // the browser's error page inside the console's screen.
  it("falls back to the offline page rather than a dead frame", async () => {
    const response = await play("/play/tetrabros/").answer;
    expect(response.status).toBe(503);
  });
});

// A GET the page marks as a copy request must go straight to the network. The
// worker answering it from cache is what silently turns an update into a
// re-download of the build the visitor already has.
describe("the fetch handler and the download's copy requests", () => {
  function drive(headers) {
    const { handlers } = loadWorker();
    let answered = false;
    handlers.fetch({
      request: {
        method: "GET",
        url: "https://pixabros.test/play/tetrabros/tetrabros.wasm",
        mode: "no-cors",
        headers: { get: (name) => headers[name] ?? null },
      },
      respondWith() {
        answered = true;
      },
      waitUntil() {},
    });
    return answered;
  }

  it("stands aside for a marked copy request", () => {
    expect(drive({ "X-Offline-Copy": "1" })).toBe(false);
  });

  it("still answers the same URL when it is the game asking", () => {
    expect(drive({})).toBe(true);
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
