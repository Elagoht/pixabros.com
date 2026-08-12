// Keeps a carousel click inside the carousel.
//
// The arrows and dots are anchors to a slide's id, which is what makes the
// whole thing work without scripting. The cost is that following one is a
// navigation: the browser scrolls the page vertically to bring the target into
// view, so choosing a game yanks the page around.
//
// This intercepts the click and scrolls the track horizontally instead. With
// no JavaScript the anchors still work exactly as before.
(function () {
  var track = document.querySelector(".carousel__track");
  if (!track) {
    return;
  }

  track.closest(".carousel").addEventListener("click", function (event) {
    var control = event.target.closest("[data-carousel-target]");
    if (!control) {
      return;
    }

    var slide = document.getElementById(control.getAttribute("data-carousel-target"));
    if (!slide) {
      return;
    }

    event.preventDefault();

    // Centre the slide in the track the same way scroll-snap would, measured
    // from the track's own scroll origin rather than the page's.
    var left = slide.offsetLeft - track.offsetLeft - (track.clientWidth - slide.clientWidth) / 2;
    track.scrollTo({ left: left, behavior: "smooth" });
  });
})();
