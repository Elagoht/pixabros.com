// The site's offline worker.
//
// One worker owns the whole origin, including the uploaded game builds under
// /play. An engine's own worker would otherwise claim that scope and shadow
// this one, which is why extraction drops any it finds in an archive.
//
// The decisions live in SWLogic as plain functions so they can be tested
// without a browser; the event handlers below are the thin part.

var SHELL_PATH = "/api/shell";
var PAGES_CACHE = "pages";
var MEDIA_CACHE = "media";
var PAGES_LIMIT = 60;
var MEDIA_LIMIT = 120;
var GAME_PREFIX = "game-";
var SHELL_PREFIX = "shell-";

self.SWLogic = {
  // classify decides which strategy a request gets. The URL prefix is checked
  // before the request type on purpose: a game's iframe is a navigation, but
  // it must be served from the game's cache rather than from the page cache.
  classify: function (pathname) {
    if (pathname.indexOf("/api/") === 0 || pathname.indexOf("/I-am-a-pixabro/") === 0) {
      return "bypass";
    }
    if (pathname.indexOf("/play/") === 0) {
      return "play";
    }
    if (pathname.indexOf("/assets/") === 0) {
      return "asset";
    }
    if (pathname.indexOf("/media/") === 0) {
      return "media";
    }
    return "page";
  },

  gameCacheName: function (slug, version) {
    return GAME_PREFIX + slug + "-" + version;
  },

  // parseGameCache splits a game cache name back apart. The version is taken
  // from the last hyphen rather than the first, because slugs carry hyphens
  // ("dungrid-tactics") and the version never does.
  parseGameCache: function (name) {
    if (name.indexOf(GAME_PREFIX) !== 0) {
      return null;
    }
    var rest = name.slice(GAME_PREFIX.length);
    var split = rest.lastIndexOf("-");
    if (split <= 0 || split === rest.length - 1) {
      return null;
    }
    return { slug: rest.slice(0, split), version: rest.slice(split + 1) };
  },

  // cacheKeyFor maps a requested path to the key the download actually stored.
  //
  // The console points the game's iframe at /play/{slug}/, but the manifest
  // names the same file "index.html", so that is what the cache holds.
  // caches.match is an exact-URL lookup: it does not do the directory-index
  // resolution http.FileServer does on the server. Without this the whole
  // download misses offline and the frame shows the browser's error page.
  cacheKeyFor: function (pathname) {
    if (pathname.charAt(pathname.length - 1) === "/") {
      return pathname + "index.html";
    }
    return pathname;
  },

  playSlug: function (pathname) {
    if (pathname.indexOf("/play/") !== 0) {
      return null;
    }
    var rest = pathname.slice("/play/".length);
    var end = rest.indexOf("/");
    var slug = end === -1 ? rest : rest.slice(0, end);
    return slug === "" ? null : slug;
  },

  // completeKey is written last, after every file of a build has landed. A
  // cache without it is a download that was interrupted, and treating it as
  // ready would mean a game that does not start on a plane.
  //
  // It sits inside the build's own path so it cannot collide with a real file
  // that the manifest lists.
  completeKey: function (slug) {
    return "/play/" + slug + "/__offline-complete";
  },

  // isKeepable reports whether a cache survives activation. Anything outside
  // the four families is from an older worker and holds storage the visitor
  // has no way to see or free.
  isKeepable: function (name, shellVersion) {
    if (name === PAGES_CACHE || name === MEDIA_CACHE) {
      return true;
    }
    if (name.indexOf(GAME_PREFIX) === 0) {
      // A downloaded game outlives shell versions deliberately: the visitor
      // decides when to spend 90 MB again, not a deploy.
      return self.SWLogic.parseGameCache(name) !== null;
    }
    return name === SHELL_PREFIX + shellVersion;
  },
};

// fetchShell asks the server what a page needs offline. The worker cannot hold
// these URLs itself: they are content-hashed, so they move whenever a script
// or the stylesheet changes.
function fetchShell() {
  return fetch(SHELL_PATH, { cache: "no-store" }).then(function (response) {
    if (!response.ok) {
      throw new Error("shell list unavailable");
    }
    return response.json();
  });
}

// precache stores the shell under a name carrying its stamp, so an older shell
// is a different cache and activation can drop it whole.
function precache(shell) {
  return caches.open(SHELL_PREFIX + shell.version).then(function (cache) {
    return cache.addAll(shell.urls);
  });
}

// currentShellVersion is remembered between events so activate knows which
// shell cache to keep without asking the network again.
var currentShellVersion = "";

// shellCheckInFlight keeps a burst of navigations from starting a burst of
// shell fetches. The list is a few hundred bytes, but one request per
// navigation would still be one request nobody asked for.
var shellCheckInFlight = false;

// shellVersion answers with the stamp of the shell this worker is serving,
// recovering it from the cache names when the in-memory copy is empty.
//
// Browsers terminate an idle worker aggressively, and neither install nor
// activate runs again on the cold start that follows -- so the variable above
// is empty during exactly the ordinary work the worker exists to do. Without
// the recovery, hashed assets land in a cache literally named "shell-" and
// refreshShell's comparison can never match, re-precaching the entire shell on
// the first online navigation after every restart.
//
// The recovery lives here rather than at each use site so there is one place
// that knows the value can be missing.
function shellVersion() {
  if (currentShellVersion) {
    return Promise.resolve(currentShellVersion);
  }
  return caches.keys().then(function (names) {
    for (var i = 0; i < names.length; i++) {
      if (names[i].indexOf(SHELL_PREFIX) === 0) {
        currentShellVersion = names[i].slice(SHELL_PREFIX.length);
        break;
      }
    }
    return currentShellVersion;
  });
}

// refreshShell re-reads the shell list and re-precaches when the stamp moved.
//
// Install alone is not enough. A visitor who installs the worker, then a
// deploy moves the stylesheet, then the visitor goes offline, would open a
// page asking for a stylesheet URL that was never cached -- an unstyled page.
// Checking after a navigation that reached the network closes that.
function refreshShell() {
  if (shellCheckInFlight) {
    return;
  }
  shellCheckInFlight = true;
  Promise.all([fetchShell(), shellVersion()])
    .then(function (results) {
      var shell = results[0];
      var previous = results[1];
      if (shell.version === previous) {
        return null;
      }
      currentShellVersion = shell.version;
      return precache(shell).then(function () {
        if (previous) {
          return caches.delete(SHELL_PREFIX + previous);
        }
        return null;
      });
    })
    .catch(function () {})
    .then(function () {
      shellCheckInFlight = false;
    });
}

self.addEventListener("install", function (event) {
  event.waitUntil(
    fetchShell()
      .then(function (shell) {
        currentShellVersion = shell.version;
        return precache(shell);
      })
      // A failed install would leave the visitor with no worker at all, which
      // is worse than a worker whose shell fills in as they browse.
      .catch(function () {})
      .then(function () {
        return self.skipWaiting();
      })
  );
});

self.addEventListener("activate", function (event) {
  event.waitUntil(
    fetchShell()
      .then(function (shell) {
        currentShellVersion = shell.version;
        return precache(shell);
      })
      .catch(function () {})
      .then(function () {
        // A shell fetch that failed -- a network drop between fetching /sw.js
        // and activating, which is the flaky connection this worker is for --
        // leaves no version to compare against. isKeepable would then call
        // every shell- cache droppable and the cleanup would delete the shell
        // just precached: stylesheet, fonts, icons and the /offline page.
        // There is nothing safe to decide without the version, so decide
        // nothing.
        if (currentShellVersion === "") {
          return null;
        }
        return caches.keys().then(function (names) {
          return Promise.all(
            names.map(function (name) {
              if (self.SWLogic.isKeepable(name, currentShellVersion)) {
                return null;
              }
              return caches.delete(name);
            })
          );
        });
      })
      .then(function () {
        return self.clients.claim();
      })
  );
});

// trim keeps a runtime cache bounded by dropping its oldest entries. Cache
// API returns keys in insertion order, which is what makes this possible
// without keeping a second ledger; a true LRU would mean writing a timestamp
// on every read, and these two caches are not worth that.
function trim(cache, limit) {
  return cache.keys().then(function (keys) {
    if (keys.length <= limit) {
      return null;
    }
    return Promise.all(
      keys.slice(0, keys.length - limit).map(function (key) {
        return cache.delete(key);
      })
    );
  });
}

// offlinePage is the last thing a navigation gets when neither the network nor
// the cache could answer it. The offline page is precached on install, but
// install swallows a failed shell fetch rather than leaving the visitor with no
// worker at all -- so it can genuinely be missing, and precisely when it is
// most needed. respondWith(undefined) throws, which would replace a plain
// sentence with the browser's own error page.
function offlinePage() {
  return caches.match("/offline").then(function (offline) {
    return (
      offline ||
      new Response("You are offline, and this page has not been opened on this device before.", {
        status: 503,
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      })
    );
  });
}

// networkFirst is what freshness-sensitive things get. The whole site is
// pre-rendered and served with an ETag; serving a cached copy in preference to
// the network would undo that.
function networkFirst(event, request, cacheName, limit) {
  return fetch(request)
    .then(function (response) {
      if (!response.ok) {
        return response;
      }
      var copy = response.clone();
      // The page gets the response the moment it arrives, which ends the
      // event's extended lifetime -- and a browser may reclaim the worker
      // right then, dropping the write. waitUntil keeps the worker alive
      // until the copy has actually landed, without making the page wait.
      event.waitUntil(
        caches.open(cacheName).then(function (cache) {
          return cache.put(request, copy).then(function () {
            return trim(cache, limit);
          });
        })
      );
      return response;
    })
    .catch(function () {
      return caches.match(request).then(function (cached) {
        if (cached) {
          return cached;
        }
        if (request.mode === "navigate") {
          return offlinePage();
        }
        return Response.error();
      });
    });
}

// cacheFirst is for content-hashed assets: their URL changes when their bytes
// do, so a hit is never stale.
function cacheFirst(event, request, cacheName) {
  return caches.match(request).then(function (cached) {
    if (cached) {
      return cached;
    }
    return fetch(request).then(function (response) {
      // An empty cache name means no shell cache exists to recover a version
      // from, so there is nowhere this asset belongs; writing it to a cache
      // called "shell-" would only leave litter for a later activate.
      if (response.ok && cacheName) {
        var copy = response.clone();
        // Same reason as networkFirst: the response resolves and ends the
        // event's extended lifetime before an un-awaited write finishes,
        // so the worker may be torn down mid-write. waitUntil keeps it
        // alive for the write without delaying the response.
        event.waitUntil(
          caches.open(cacheName).then(function (cache) {
            return cache.put(request, copy);
          })
        );
      }
      return response;
    });
  });
}

// playFirst serves a game build out of whichever version the visitor holds.
// The completion marker is never handed out as a file: it is bookkeeping, not
// part of the build.
function playFirst(request, pathname) {
  var slug = self.SWLogic.playSlug(pathname);
  if (!slug || pathname === self.SWLogic.completeKey(slug)) {
    return fetch(request);
  }
  // The requested path and the stored key are not always the same string; see
  // SWLogic.cacheKeyFor. Matching on the request object where they do agree
  // keeps the ordinary case going through the same Request the page made.
  var key = self.SWLogic.cacheKeyFor(pathname);
  return caches.match(key === pathname ? request : key).then(function (cached) {
    if (cached) {
      return cached;
    }
    return fetch(request).catch(function (err) {
      // A game that was never downloaded, opened with no network. The frame is
      // a navigation like any other, so it gets the same plain sentence a page
      // would rather than the browser's error page.
      if (request.mode === "navigate") {
        return offlinePage();
      }
      throw err;
    });
  });
}

self.addEventListener("fetch", function (event) {
  var request = event.request;
  if (request.method !== "GET") {
    return;
  }

  // The page's download loop marks the requests it makes to fill a game cache,
  // and they must reach the network. Served from cache instead, an update would
  // find every file of the *old* build still held under /play/{slug}/, copy
  // those bytes into the new version's cache, write a marker claiming the new
  // version, and delete the old cache -- leaving the visitor told they are
  // current, out no bandwidth, and running the old build forever. Nothing would
  // ever reconcile it, because the marker is what the page trusts.
  //
  // Same-origin and not a forbidden header, so this costs no preflight.
  if (request.headers.get("X-Offline-Copy")) {
    return;
  }

  var url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  var kind = self.SWLogic.classify(url.pathname);
  if (kind === "bypass") {
    return;
  }
  if (kind === "asset") {
    event.respondWith(
      shellVersion().then(function (version) {
        return cacheFirst(event, request, version ? SHELL_PREFIX + version : "");
      })
    );
    return;
  }
  if (kind === "media") {
    event.respondWith(networkFirst(event, request, MEDIA_CACHE, MEDIA_LIMIT));
    return;
  }
  if (kind === "play") {
    event.respondWith(playFirst(request, url.pathname));
    return;
  }
  event.respondWith(
    networkFirst(event, request, PAGES_CACHE, PAGES_LIMIT).then(function (response) {
      // Only a navigation that actually reached the network proves we are
      // online, which is the moment worth spending a shell check on.
      if (request.mode === "navigate" && response && response.ok) {
        refreshShell();
      }
      return response;
    })
  );
});
