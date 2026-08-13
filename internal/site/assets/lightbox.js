// Opens an award's picture full size.
//
// The pictures are buttons that raise one shared dialog. Without JavaScript
// they are still ordinary images on the page, just not enlargeable.
(function () {
  var dialog = document.querySelector("[data-zoom-dialog]");
  var image = document.querySelector("[data-zoom-image]");
  var triggers = document.querySelectorAll("[data-zoom-src]");

  if (!(dialog && image && triggers.length && dialog.showModal)) {
    return;
  }

  function close() {
    dialog.classList.remove("lightbox--open");
    window.setTimeout(function () {
      if (dialog.open) {
        dialog.close();
      }
      // Dropped so a reopened dialog never flashes the previous picture.
      image.removeAttribute("src");
    }, 200);
  }

  Array.prototype.forEach.call(triggers, function (trigger) {
    trigger.addEventListener("click", function () {
      image.src = trigger.getAttribute("data-zoom-src");
      image.alt = trigger.getAttribute("data-zoom-alt") || "";
      dialog.showModal();
      requestAnimationFrame(function () {
        dialog.classList.add("lightbox--open");
      });
    });
  });

  var closer = dialog.querySelector("[data-zoom-close]");
  if (closer) {
    closer.addEventListener("click", close);
  }

  // Anywhere outside the picture closes it.
  dialog.addEventListener("click", function (event) {
    if (event.target !== image) {
      close();
    }
  });

  dialog.addEventListener("cancel", function (event) {
    event.preventDefault();
    close();
  });
})();
