// Drives the carousel's arrows.
//
// The arrows are the script's own: they sit at the sides of the carousel
// rather than on each card, so they need to know where the track is scrolled
// to. Without JavaScript they stay hidden and the track is still an ordinary
// scroller, with the dots as anchors.
//
// The dots are driven by intent, not by the scroll position: a click marks
// its target the moment it lands, so the pager answers while the smooth
// scroll is still on its way, and a swipe hands control back to the track
// once it settles.
(function (window, document) {
  // The pager on its own, without the page: the tests load this file and
  // drive it with no carousel in sight. createPager(count, initial) keeps the
  // index navigation is aimed at and wraps at both ends. go(step) steps from
  // the current target and answers the new one at once; goTo(index) takes one
  // directly, which is how a settled scroll hands control back to where the
  // track really is.
  window.CarouselLogic = {
    createPager: function (count, initial) {
      var current = initial || 0;
      var wrap = function (index) {
        return ((index % count) + count) % count;
      };
      return {
        current: function () {
          return current;
        },
        go: function (step) {
          current = wrap(current + step);
          return current;
        },
        goTo: function (index) {
          current = wrap(index);
          return current;
        },
      };
    },
  };

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

  var pager = window.CarouselLogic.createPager(slides.length, currentIndex());

  // The dots say which slide you are on. aria-current carries it, so the
  // styling and what a screen reader announces come from the same attribute.
  function markIndex(index) {
    Array.prototype.forEach.call(dots, function (dot, i) {
      if (i === index) {
        dot.setAttribute("aria-current", "true");
      } else {
        dot.removeAttribute("aria-current");
      }
    });
  }

  function go(step) {
    var index = pager.go(step);
    markIndex(index);
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

  markIndex(pager.current());

  // Following the scroll rather than only the clicks, so dragging or swiping
  // the track updates the dots too. The debounce keeps a settling scroll from
  // repainting the pager once per frame; by the time it fires, a click's own
  // scroll has long been marked by hand.
  var pending = null;
  track.addEventListener("scroll", function () {
    if (pending) {
      window.clearTimeout(pending);
    }
    pending = window.setTimeout(function () {
      markIndex(pager.goTo(currentIndex()));
    }, 90);
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
    var index = pager.goTo(Array.prototype.indexOf.call(slides, slide));
    markIndex(index);
    track.scrollTo({ left: centreOf(slide), behavior: "smooth" });
  });
})(window, document);
