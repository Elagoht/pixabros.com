// Offline support on the page's side: registers the worker, and (in the next
// task) drives the per-game download control.
(function () {
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
})();
