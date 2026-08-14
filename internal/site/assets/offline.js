// Offline support on the page's side: registers the worker, and (in the next
// task) drives the per-game download control.
(function () {
  var GAME_PREFIX = "game-";

  // The decisions live here as plain functions so they can be tested without a
  // browser. Everything below them is DOM and network.
  //
  // completeKey and gameCacheName mirror SWLogic in sw.js exactly: the worker
  // and this script are separate global scopes with no shared module system,
  // so each keeps its own copy. offline.test.js loads both files and asserts
  // the two agree, which is what keeps that duplication from drifting.
  window.OfflineLogic = {
    // stateFor decides what the control says. current is null when the build
    // endpoint could not be reached, which is exactly the offline case: a held
    // copy is then reported ready rather than stale, because staleness cannot
    // be verified without the server and claiming it would be a lie.
    stateFor: function (held, current) {
      if (!held) {
        return current ? "absent" : "unavailable";
      }
      if (!current || held === current) {
        return "ready";
      }
      return "stale";
    },

    // formatBytes writes a download size the way a visitor reads one. Anything
    // under a megabyte is rounded up to a kilobyte rather than shown as 0.
    formatBytes: function (bytes) {
      var mb = bytes / (1024 * 1024);
      if (mb >= 1) {
        return Math.round(mb) + " MB";
      }
      return Math.round(bytes / 1024) + " KB";
    },

    // hasRoomFor refuses up front rather than failing halfway through 90 MB.
    // A browser that will not estimate is not a browser that has no room, so
    // an absent estimate allows the attempt.
    hasRoomFor: function (estimate, bytes) {
      if (!estimate || typeof estimate.quota !== "number" || typeof estimate.usage !== "number") {
        return true;
      }
      return estimate.quota - estimate.usage > bytes * 1.1;
    },

    gameCacheName: function (slug, version) {
      return GAME_PREFIX + slug + "-" + version;
    },

    // Mirrors SWLogic.completeKey in sw.js: the worker refuses to hand this
    // key out as a file, and the download writes it last.
    completeKey: function (slug) {
      return "/play/" + slug + "/__offline-complete";
    },
  };

  if (!("serviceWorker" in navigator)) {
    return;
  }

  // Registration waits for load so it never competes with the page's own
  // requests on a first visit, which is the visit that matters most.
  window.addEventListener("load", function () {
    navigator.serviceWorker.register("/sw.js").catch(function () {
      // A site that cannot register a worker is a site without offline
      // support, not a broken site. Nothing here is worth an error dialog.
    });
  });

  var mount = document.querySelector("[data-offline-game]");
  if (!mount || !window.caches || !navigator.storage) {
    return;
  }
  var slug = mount.getAttribute("data-offline-game");

  var control = document.createElement("div");
  control.className = "offline";
  mount.appendChild(control);

  var manifest = null; // {version, bytes, files} once fetched, null offline
  var held = null; // the version this device holds, or null

  function gameCacheName(version) {
    return window.OfflineLogic.gameCacheName(slug, version);
  }

  function completeKey() {
    return window.OfflineLogic.completeKey(slug);
  }

  // heldVersion finds the completed copy on this device. A cache without its
  // completion marker is an interrupted download: it is deleted rather than
  // reported, because a half-built game does not start.
  function heldVersion() {
    return window.caches.keys().then(function (names) {
      var mine = names.filter(function (name) {
        return name.indexOf(GAME_PREFIX + slug + "-") === 0;
      });
      return mine.reduce(function (chain, name) {
        return chain.then(function (found) {
          if (found) {
            return found;
          }
          return window.caches.open(name).then(function (cache) {
            return cache.match(completeKey()).then(function (marker) {
              if (!marker) {
                return window.caches.delete(name).then(function () {
                  return null;
                });
              }
              return marker.text();
            });
          });
        });
      }, Promise.resolve(null));
    });
  }

  function loadManifest() {
    return fetch("/api/games/" + slug + "/build")
      .then(function (response) {
        return response.ok ? response.json() : null;
      })
      .catch(function () {
        return null;
      });
  }

  function button(label, onClick) {
    var el = document.createElement("button");
    el.type = "button";
    el.className = "offline__action";
    el.textContent = label;
    el.addEventListener("click", onClick);
    return el;
  }

  function say(text) {
    var el = document.createElement("p");
    el.className = "offline__status";
    el.textContent = text;
    return el;
  }

  function render() {
    control.textContent = "";
    var state = window.OfflineLogic.stateFor(held, manifest ? manifest.version : null);

    if (state === "unavailable") {
      return;
    }
    if (state === "absent") {
      control.appendChild(
        button("Make playable offline — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
      );
      return;
    }
    if (state === "ready") {
      control.appendChild(say("Playable offline"));
      control.appendChild(button("Remove", remove));
      return;
    }
    control.appendChild(say("A new version is available"));
    control.appendChild(
      button("Update — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
    );
    control.appendChild(button("Remove", remove));
  }

  function download() {
    control.textContent = "";
    var status = say("Downloading… 0%");
    control.appendChild(status);

    navigator.storage
      .estimate()
      .catch(function () {
        return null;
      })
      .then(function (estimate) {
        if (!window.OfflineLogic.hasRoomFor(estimate, manifest.bytes)) {
          throw new Error("not enough room on this device");
        }
        // Asking to persist makes the browser less willing to evict the copy.
        // It is a request, not a guarantee, and is refused outright on iOS.
        return navigator.storage.persist ? navigator.storage.persist() : null;
      })
      .then(function () {
        return window.caches.open(gameCacheName(manifest.version));
      })
      .then(function (cache) {
        var done = 0;
        return manifest.files
          .reduce(function (chain, file) {
            return chain.then(function () {
              var url = "/play/" + slug + "/" + file.path;
              return fetch(url).then(function (response) {
                if (!response.ok) {
                  throw new Error("could not fetch " + file.path);
                }
                return cache.put(url, response).then(function () {
                  done += file.bytes;
                  status.textContent =
                    "Downloading… " + Math.round((done / manifest.bytes) * 100) + "%";
                });
              });
            });
          }, Promise.resolve())
          .then(function () {
            // Written last, and only once every file has landed: this marker is
            // what separates a finished download from an interrupted one.
            return cache.put(completeKey(), new Response(manifest.version));
          });
      })
      .then(
        function () {
          // The marker above landed: this copy is complete and, from this
          // instant, belongs to the visitor. It is not ours to revoke, so
          // finish() runs detached from this chain -- nothing it does, or
          // fails to do, is allowed to reach the failure branch below and be
          // mistaken for a download that never completed.
          finish();
        },
        function (err) {
          // Reached only when the marker never landed, i.e. everything above
          // this .then. The copy is incomplete and was never the visitor's,
          // so discarding it costs them nothing they had.
          window.caches.delete(gameCacheName(manifest.version));
          control.textContent = "";
          control.appendChild(say("Could not save this game for offline play: " + err.message));
          control.appendChild(
            button("Try again — " + window.OfflineLogic.formatBytes(manifest.bytes), download)
          );
        }
      );

    // finish runs everything that happens once the copy is already the
    // visitor's: caching the page it launches from, dropping the previous
    // version, and repainting the control. None of that is worth losing the
    // build over, so its own failures are absorbed rather than surfaced --
    // render() always runs, and always reports "Playable offline", because
    // that is the truth regardless of what finish() otherwise managed.
    function finish() {
      cacheThisPage()
        .then(function () {
          var previous = held;
          held = manifest.version;
          // The old copy goes only now, so an interrupted update never costs a
          // visitor the game they already had.
          if (previous && previous !== held) {
            return window.caches.delete(gameCacheName(previous));
          }
          return null;
        })
        .catch(function () {})
        .then(render);
    }
  }

  // cacheThisPage stores the game's own page and the images on it alongside the
  // build. Without it "Playable offline" would be a half-truth: the game would
  // run, but the page you launch it from would not open.
  //
  // Failures here are swallowed: a cached build with an uncached page is worth
  // far more than a download reported as failed.
  function cacheThisPage() {
    var images = [];
    var nodes = document.querySelectorAll("img[src]");
    for (var i = 0; i < nodes.length; i++) {
      var src = nodes[i].getAttribute("src");
      if (src && src.indexOf("/media/") === 0) {
        images.push(src);
      }
    }

    return window.caches
      .open("pages")
      .then(function (cache) {
        return cache.add(window.location.pathname);
      })
      .then(function () {
        return window.caches.open("media");
      })
      .then(function (cache) {
        return Promise.all(
          images.map(function (src) {
            return cache.add(src).catch(function () {});
          })
        );
      })
      .catch(function () {});
  }

  function remove() {
    var version = held;
    window.caches.delete(gameCacheName(version)).then(function () {
      held = null;
      render();
    });
  }

  Promise.all([loadManifest(), heldVersion()]).then(function (results) {
    manifest = results[0];
    held = results[1];
    render();
  });
})();
