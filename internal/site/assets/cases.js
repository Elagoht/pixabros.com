// Opens a game's case.
//
// The case on the shelf is a button that raises a <dialog>; the lid swings
// aside and the two pages inside are revealed. Without JavaScript no dialog
// opens, which is why every case also lists a link to the game's own page --
// that page says everything this one does.
(function () {
  var openers = document.querySelectorAll("[data-case-open]");
  if (!openers.length) {
    return;
  }

  // How long the lid takes to swing, kept in step with the stylesheet.
  var OPEN_MS = 560;

  function open(dialog) {
    if (!dialog.showModal) {
      return;
    }
    dialog.showModal();
    // On the next frame, so the browser has a closed state to animate from.
    requestAnimationFrame(function () {
      dialog.classList.add("jewel--open");
    });
  }

  function close(dialog) {
    dialog.classList.remove("jewel--open");
    // Let the lid swing shut before the dialog disappears.
    window.setTimeout(function () {
      if (dialog.open) {
        dialog.close();
      }
    }, OPEN_MS);
  }

  Array.prototype.forEach.call(openers, function (opener) {
    opener.addEventListener("click", function () {
      var dialog = document.getElementById(opener.getAttribute("data-case-open"));
      if (dialog) {
        open(dialog);
      }
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll(".jewel"), function (dialog) {
    var closer = dialog.querySelector("[data-case-close]");
    if (closer) {
      closer.addEventListener("click", function () {
        close(dialog);
      });
    }

    // Clicking the backdrop closes it. The dialog fills the viewport, so the
    // test is whether the click landed outside the case itself.
    dialog.addEventListener("click", function (event) {
      if (!event.target.closest(".jewel__box, [data-case-close]")) {
        close(dialog);
      }
    });

    // Escape closes a dialog natively, which would skip the animation.
    dialog.addEventListener("cancel", function (event) {
      event.preventDefault();
      close(dialog);
    });
  });
})();
