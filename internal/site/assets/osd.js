// The channel banner.
//
// A television shows its channel banner when you change channel and takes it
// away once you are watching. This does the same: the banner is open at the top
// of a page, retracts to the lit channel as you read on, and returns the moment
// you scroll back up or tab into it.
//
// Everything here is an upgrade on a banner that simply stays open, which is
// what a visitor without JavaScript gets and which has every link in it.
(function () {
  var banner = document.querySelector("[data-osd]");
  if (!banner) {
    return;
  }

  // Far enough down that the banner does not flinch at the first flick of a
  // trackpad, and short enough that it is out of the way while reading.
  var RETRACT_AFTER = 160;

  var lastY = window.scrollY;
  var pending = false;

  function update() {
    pending = false;
    var y = window.scrollY;
    var scrollingDown = y > lastY;
    lastY = y;

    // Keyboard focus wins over any scroll position: a banner that hid the link
    // someone just tabbed to would be a trap.
    if (banner.contains(document.activeElement)) {
      banner.classList.remove("osd--retracted");
      return;
    }
    banner.classList.toggle("osd--retracted", scrollingDown && y > RETRACT_AFTER);
  }

  window.addEventListener(
    "scroll",
    function () {
      if (!pending) {
        pending = true;
        window.requestAnimationFrame(update);
      }
    },
    { passive: true },
  );

  banner.addEventListener("focusin", function () {
    banner.classList.remove("osd--retracted");
  });

  // The set has power: the banner draws itself once, the way one does when a
  // channel changes. Only after the script is running, so a page without it
  // never sits waiting for an animation that will not come.
  banner.classList.add("osd--live");
})();
