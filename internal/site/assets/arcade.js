// Loads a cartridge into the console without leaving the page.
//
// Without this the cartridges are ordinary links to each game's own page, so
// the shelf still works with scripting switched off; this only upgrades it to
// playing in place.
(function () {
  var console_ = document.querySelector("[data-console]");
  var screen = document.querySelector("[data-console-screen]");
  var idle = document.querySelector("[data-console-idle]");
  var title = document.querySelector("[data-console-title]");
  var reset = document.querySelector("[data-console-reset]");
  var cartridges = document.querySelectorAll("[data-play-url]");

  if (!console_ || !screen || cartridges.length === 0) {
    return;
  }

  var current = "";

  function load(url, name) {
    current = url;
    screen.src = url;
    screen.hidden = false;
    if (idle) {
      idle.hidden = true;
    }
    if (title) {
      title.textContent = name;
    }
    if (reset) {
      reset.hidden = false;
    }
    console_.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  Array.prototype.forEach.call(cartridges, function (cartridge) {
    cartridge.addEventListener("click", function (event) {
      event.preventDefault();
      load(cartridge.getAttribute("data-play-url"), cartridge.getAttribute("data-play-title") || "");
    });
  });

  if (reset) {
    reset.addEventListener("click", function () {
      if (!current) {
        return;
      }
      // Reassigning the same src is what restarts the build: there is no other
      // way in from outside the frame.
      screen.src = "about:blank";
      screen.src = current;
    });
  }
})();
