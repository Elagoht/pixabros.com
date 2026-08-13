// Drives the carousel's arrows.
//
// The arrows are the script's own: they sit at the sides of the carousel
// rather than on each card, so they need to know where the track is scrolled
// to. Without JavaScript they stay hidden and the track is still an ordinary
// scroller, with the dots as anchors.
(function () {
  var carousel = document.querySelector(".carousel");
  if (!carousel) {
    return;
  }

  var track = carousel.querySelector(".carousel__track");
  var prev = carousel.querySelector("[data-carousel-prev]");
  var next = carousel.querySelector("[data-carousel-next]");
  var slides = track ? track.querySelectorAll(".slide") : [];
  var dots = carousel.querySelectorAll(".carousel__dot");

  if (!(track && slides.length > 1)) {
    return;
  }

  if (prev) {
    prev.hidden = false;
  }
  if (next) {
    next.hidden = false;
  }

  function centreOf(slide) {
    return (
      slide.offsetLeft -
      track.offsetLeft -
      (track.clientWidth - slide.clientWidth) / 2
    );
  }

  // The slide nearest the middle of the track is the one on screen.
  function currentIndex() {
    var middle = track.scrollLeft + track.clientWidth / 2;
    var best = 0;
    var bestDistance = Infinity;
    Array.prototype.forEach.call(slides, function (slide, index) {
      var slideMiddle = slide.offsetLeft - track.offsetLeft + slide.clientWidth / 2;
      var distance = Math.abs(slideMiddle - middle);
      if (distance < bestDistance) {
        bestDistance = distance;
        best = index;
      }
    });
    return best;
  }

  // The dots say which slide you are on. aria-current carries it, so the
  // styling and what a screen reader announces come from the same attribute.
  function markCurrent() {
    var index = currentIndex();
    Array.prototype.forEach.call(dots, function (dot, i) {
      if (i === index) {
        dot.setAttribute("aria-current", "true");
      } else {
        dot.removeAttribute("aria-current");
      }
    });
  }

  function go(step) {
    // Wraps, so the last card's next arrow returns to the first.
    var index = (currentIndex() + step + slides.length) % slides.length;
    track.scrollTo({ left: centreOf(slides[index]), behavior: "smooth" });
  }

  if (prev) {
    prev.addEventListener("click", function () {
      go(-1);
    });
  }
  if (next) {
    next.addEventListener("click", function () {
      go(1);
    });
  }

  markCurrent();

  // Following the scroll rather than only the clicks, so dragging or swiping
  // the track updates the dots too.
  var pending = null;
  track.addEventListener("scroll", function () {
    if (pending) {
      window.clearTimeout(pending);
    }
    pending = window.setTimeout(markCurrent, 90);
  });

  // The dots are anchors, so following one would scroll the page vertically to
  // bring the slide into view. Scrolling the track instead keeps the page put.
  carousel.addEventListener("click", function (event) {
    var dot = event.target.closest("[data-carousel-target]");
    if (!dot) {
      return;
    }
    var slide = document.getElementById(dot.getAttribute("data-carousel-target"));
    if (!slide) {
      return;
    }
    event.preventDefault();
    track.scrollTo({ left: centreOf(slide), behavior: "smooth" });
  });
})();
