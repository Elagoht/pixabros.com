// Runs the console, on the games page and on a game's own page.
//
// Two ways in, one machine. On the shelf you slot a cartridge in; on a game's
// own page the cartridge is already there and you press start. Either way the
// frame gets its src here and nowhere else: a web build is tens of megabytes,
// so nothing downloads until a visitor asks for it.
//
// Without this the cartridges are ordinary links to each game's own page and
// the start button is replaced by the "open the game on its own" link beneath
// the console, so the shelf still works with scripting switched off.
(function () {
  var consoleEl = document.querySelector("[data-console]");
  var screen = document.querySelector("[data-console-screen]");
  var stage = document.querySelector("[data-console-stage]");
  var idle = document.querySelector("[data-console-idle]");
  var title = document.querySelector("[data-console-title]");
  var slot = document.querySelector("[data-console-cartridge]");
  var controls = document.querySelector("[data-console-controls]");
  var resetButton = document.querySelector("[data-console-reset]");
  var ejectButton = document.querySelector("[data-console-eject]");
  var led = document.querySelector("[data-console-led]");
  var crtButton = document.querySelector("[data-console-crt]");
  var fullscreenButton = document.querySelector("[data-console-fullscreen]");
  var cartridges = document.querySelectorAll("[data-play-url]");
  var startButton = document.querySelector("[data-console-start]");

  if (!(consoleEl && screen) || !(cartridges.length || startButton)) {
    return;
  }

  var current = "";

  function show(element, visible) {
    if (element) {
      element.hidden = !visible;
    }
  }

  // A cartridge is in the machine, so its place on the shelf is empty: the
  // gap is kept rather than closed, so the grid does not reflow underneath.
  function setShelfGap(cartridge, empty) {
    var slotOnShelf = cartridge.closest("li");
    if (slotOnShelf) {
      slotOnShelf.classList.toggle("cartridges__slot--empty", empty);
    }
  }

  function clearShelf() {
    Array.prototype.forEach.call(cartridges, function (cartridge) {
      setShelfGap(cartridge, false);
    });
  }

  function insert(cartridge) {
    if (!slot) {
      return;
    }
    // The cartridge in the slot is the one from the shelf, so the two can
    // never drift apart visually. The clone is decoration: it must not be a
    // second link to the same page, nor be read out again.
    var copy = cartridge.cloneNode(true);
    copy.removeAttribute("href");
    copy.removeAttribute("data-play-url");
    copy.setAttribute("aria-hidden", "true");
    copy.classList.add("cartridge--slotted");
    slot.replaceChildren(copy);
    slot.classList.add("console__cartridge--loaded");
  }

  function run(url) {
    current = url;
    screen.src = url;

    show(screen, true);
    show(idle, false);
    show(controls, true);
    show(resetButton, true);
    if (led) {
      led.classList.add("nes__led--on");
    }
  }

  function load(cartridge) {
    clearShelf();
    setShelfGap(cartridge, true);
    run(cartridge.getAttribute("data-play-url"));

    // Eject only makes sense where there is a shelf to put the cartridge back
    // on. A game's own page has one cartridge and it is this game's.
    show(ejectButton, true);
    if (title) {
      title.textContent = cartridge.getAttribute("data-play-title") || "";
    }
    insert(cartridge);

    consoleEl.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function eject() {
    clearShelf();
    current = "";
    // about:blank first so the running game is torn down rather than left
    // playing audio behind a hidden frame.
    screen.src = "about:blank";

    show(screen, false);
    show(idle, true);
    show(controls, false);
    show(resetButton, false);
    show(ejectButton, false);
    if (title) {
      title.textContent = "";
    }
    if (slot) {
      slot.replaceChildren();
      slot.classList.remove("console__cartridge--loaded");
    }
    if (led) {
      led.classList.remove("nes__led--on");
    }
  }

  Array.prototype.forEach.call(cartridges, function (cartridge) {
    cartridge.addEventListener("click", function (event) {
      event.preventDefault();
      load(cartridge);
    });
  });

  if (startButton) {
    startButton.addEventListener("click", function () {
      run(startButton.getAttribute("data-console-start"));
    });
  }

  if (resetButton) {
    resetButton.addEventListener("click", function () {
      if (!current) {
        return;
      }
      // Reassigning the same src is what restarts the build: there is no other
      // way in from outside the frame.
      screen.src = "about:blank";
      screen.src = current;
    });
  }

  if (ejectButton) {
    ejectButton.addEventListener("click", eject);
  }

  if (crtButton) {
    crtButton.addEventListener("click", function () {
      var on = consoleEl.classList.toggle("console--flat");
      // The class turns the effect off, so pressed means the opposite of it.
      crtButton.setAttribute("aria-pressed", on ? "false" : "true");
    });
  }

  if (fullscreenButton && stage) {
    fullscreenButton.addEventListener("click", function () {
      if (document.fullscreenElement) {
        document.exitFullscreen();
        return;
      }
      // The stage rather than the frame, so the scanlines and the bezel's
      // glow go fullscreen with the game instead of being left behind.
      if (stage.requestFullscreen) {
        stage.requestFullscreen();
      }
    });

    document.addEventListener("fullscreenchange", function () {
      var label = document.fullscreenElement ? "Exit fullscreen" : "Fullscreen";
      fullscreenButton.setAttribute("aria-label", label);
      fullscreenButton.setAttribute("title", label);
    });
  }
})();
