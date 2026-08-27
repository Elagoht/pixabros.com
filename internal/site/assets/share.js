(function (window, navigator, document) {
  "use strict";

  async function sharePost(client, payload) {
    if (typeof client.share === "function") {
      await client.share(payload);
      return "shared";
    }
    await client.clipboard.writeText(payload.url);
    return "copied";
  }

  window.ShareLogic = { sharePost: sharePost };

  window.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("[data-share-native]").forEach(function (button) {
      button.addEventListener("click", async function () {
        var status = document.querySelector("[data-share-status]");
        try {
          var result = await sharePost(navigator, {
            title: button.dataset.shareTitle,
            url: button.dataset.shareUrl,
          });
          if (status && result === "copied") status.textContent = "Link copied";
        } catch (error) {
          if (error && error.name === "AbortError") return;
          if (status) status.textContent = "Could not share this link";
        }
      });
    });
  });
})(window, navigator, document);
