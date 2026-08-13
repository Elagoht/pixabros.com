package site

import "pixabros/internal/contactapi"

const (
	PageContact     = "contact"
	PageContactSent = "contact/sent"
)

type contactPage struct {
	MinMessageLength int
	// HoneypotField is named here rather than in the template so it can never
	// drift from the field the API actually checks.
	HoneypotField string
}

// renderContact builds /contact.
//
// The form posts to /api/contact. It works without JavaScript -- a plain form
// post is answered with a redirect -- and the page's script upgrades that to
// inline feedback when it runs.
func (s *Site) renderContact(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}

	html, err := s.renderer.render("contact.html", pageData{
		Title:       "Contact the studio about a game, a job or the press",
		Description: "Get in touch with " + chrome.Name + ".",
		Path:        "/" + PageContact,
		Canonical:   canonicalURL(chrome.URL, PageContact),
		Schema:      contactSchema(chrome),
		Scripts:     []string{s.renderer.bundle.URL("contact.js")},
		Site:        chrome,
		Data: contactPage{
			MinMessageLength: contactapi.MinMessageLength,
			HoneypotField:    contactapi.HoneypotField,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{siteSettingsTag}, nil
}

// renderContactSent builds /contact/sent, where a form post lands when there is
// no JavaScript to keep the visitor on the page.
func (s *Site) renderContactSent(pageKey string) ([]byte, []string, error) {
	chrome, err := s.chrome()
	if err != nil {
		return nil, nil, err
	}

	html, err := s.renderer.render("contact-sent.html", pageData{
		Title:       "Message sent, and we will get back to you shortly",
		Description: "Your message has been sent.",
		Path:        "/" + PageContact,
		Canonical:   canonicalURL(chrome.URL, PageContactSent),
		Site:        chrome,
	})
	if err != nil {
		return nil, nil, err
	}
	return html, []string{siteSettingsTag}, nil
}
