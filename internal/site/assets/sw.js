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

self.addEventListener("install", function () {
  self.skipWaiting();
});
