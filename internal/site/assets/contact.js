// Progressive enhancement for the contact form.
//
// Without this the form posts normally and the browser follows a redirect to
// /contact/sent. With it, the submission goes over fetch so the visitor keeps
// what they typed when something is wrong.
(function () {
  var form = document.querySelector('form[action="/api/contact"]');
  if (!form) {
    return;
  }

  var status = form.querySelector("[data-contact-status]");
  // Selected by data attribute, not by type: the HTML minifier drops
  // type="submit" because it is the default inside a form, which left this
  // lookup returning null and the whole script throwing.
  var button = form.querySelector("[data-contact-submit]");

  function show(message, kind) {
    if (!status) {
      return;
    }
    status.textContent = message;
    status.className = "form__status form__status--" + kind;
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();

    var data = new FormData(form);
    var body = {
      name: data.get("name") || "",
      subject: data.get("subject") || "",
      phone: data.get("phone") || "",
      email: data.get("email") || "",
      message: data.get("message") || "",
      wants_callback: data.get("wants_callback") !== null,
      website: data.get("website") || "",
    };

    if (button) {
      button.disabled = true;
    }
    show("Sending…", "ok");

    fetch(form.action, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
      .then(function (response) {
        return response.json().then(function (payload) {
          return { ok: response.ok, payload: payload };
        });
      })
      .then(function (result) {
        if (result.ok) {
          form.reset();
          show("Thank you. Your message is on its way.", "ok");
          return;
        }
        var error = result.payload && result.payload.error;
        show(error && error.message ? error.message : "Something went wrong.", "error");
      })
      .catch(function () {
        show("Could not reach the server. Please try again.", "error");
      })
      .finally(function () {
        if (button) {
          button.disabled = false;
        }
      });
  });
})();
