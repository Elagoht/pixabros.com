package site

import (
	"strings"

	"pixabros/internal/devlog"
	"pixabros/internal/games"
)

// The structured data each kind of page publishes.
//
// Every page carries the studio and the site itself, so a crawler that lands
// anywhere learns who made this and what it is. On top of that a page describes
// its own subject: a game is a VideoGame, a post is a BlogPosting, a listing is
// a collection with the things it lists.
//
// A property whose value is unknown is left out rather than published empty,
// which is what schema.set does. That is why so much of this is unconditional:
// the emptiness check lives in one place.

// baseNodes are the studio and the site, published on every page.
//
// These do not depend on the site's address being configured: an Organization
// with a name, a logo and a description is worth publishing whether or not it
// can also state its own URL.
func baseNodes(chrome SiteChrome) []schema {
	return []schema{chrome.organization(), chrome.website()}
}

// page is the WebPage node the rest of a page's data hangs off.
func (c SiteChrome) page(path, name, description string) schema {
	url := canonicalURL(c.URL, path)
	node := schema{
		"@type":    "WebPage",
		"@id":      fallbackID(url, "#webpage"),
		"name":     name,
		"isPartOf": schema{"@id": c.nodeID("#website")},
	}
	node.set("url", url)
	node.set("description", description)
	return node
}

func landingSchema(chrome SiteChrome, page landingPage) jsonLD {
	nodes := baseNodes(chrome)

	// The team is the studio's own people, so they are attached to the
	// organization rather than floating as unrelated Person nodes.
	members := make([]any, 0, len(page.Members))
	for _, member := range page.Members {
		person := schema{"@type": "Person", "name": member.Name}
		person.set("jobTitle", strings.Join(member.Tags, ", "))
		person.set("description", member.Bio)
		person.set("image", absoluteURL(chrome.URL, member.Avatar.URL))

		urls := make([]string, 0, len(member.Links))
		for _, link := range member.Links {
			urls = append(urls, link.URL)
		}
		person.set("sameAs", urls)
		members = append(members, person)
	}

	if len(members) > 0 {
		nodes[0].set("member", members)
		// employee as well as member: search engines read the two differently,
		// and a studio's people are both.
		nodes[0].set("employee", members)
	}

	return graph(append(nodes,
		chrome.page("", chrome.Name, chrome.Description),
	)...)
}

// videoGame describes one game, in the detail a listing has room for.
func videoGame(chrome SiteChrome, game games.Game, imageURL string) schema {
	url := canonicalURL(chrome.URL, GamePagePrefix+game.Slug)
	node := schema{
		"@type":     "VideoGame",
		"@id":       fallbackID(url, "#game"),
		"name":      game.Title,
		"publisher": schema{"@id": chrome.nodeID("#organization")},
		"author":    schema{"@id": chrome.nodeID("#organization")},
	}
	node.set("url", url)
	node.set("description", game.ShortDescription)
	node.set("genre", strings.TrimSpace(game.Genre))
	node.set("datePublished", game.ReleaseDate)
	node.set("keywords", splitTags(game.Tags))
	node.set("image", absoluteURL(chrome.URL, imageURL))

	if game.IsBrowserPlayable {
		// The two properties say different things: one is where it runs, the
		// other is that it runs without being installed.
		node.set("gamePlatform", "Web browser")
		node.set("playMode", "SinglePlayer")
	}
	if videoID, ok := trailerID(game.VideoURL); ok {
		node.set("trailer", schema{
			"@type":       "VideoObject",
			"name":        game.Title + " trailer",
			"embedUrl":    videoID,
			"uploadDate":  game.ReleaseDate,
			"description": game.ShortDescription,
		})
	}
	if game.IsForSale && game.PriceDisplay != "" {
		// The stored price is a label an admin typed ("$4.99", "Free"), not a
		// number, so it is published as one rather than guessed into a
		// currency and an amount that might both be wrong.
		offer := schema{
			"@type":         "Offer",
			"availability":  "https://schema.org/InStock",
			"priceCurrency": "USD",
			"description":   game.PriceDisplay,
		}
		offer.set("url", url)
		node.set("offers", offer)
	}
	return node
}

// trailerID turns a stored trailer link into the player's address, or reports
// that there is nothing to publish.
func trailerID(rawURL string) (string, bool) {
	embed := videoEmbedURL(rawURL)
	return embed, embed != ""
}

func arcadeSchema(chrome SiteChrome, page arcadePage) jsonLD {
	nodes := baseNodes(chrome)

	items := make([]any, 0, len(page.Cases))
	for i, item := range page.Cases {
		items = append(items, schema{
			"@type":    "ListItem",
			"position": i + 1,
			"url":      canonicalURL(chrome.URL, GamePagePrefix+item.Slug),
			"name":     item.Title,
		})
	}

	collection := chrome.page(PageGames, "Games", "Every game the studio has made.")
	collection["@type"] = "CollectionPage"
	collection.set("mainEntity", schema{
		"@type":           "ItemList",
		"numberOfItems":   len(items),
		"itemListElement": items,
	})

	return graph(append(nodes,
		collection,
		chrome.breadcrumbs("Home", "", "Games", PageGames),
	)...)
}

func gameSchema(chrome SiteChrome, game games.Game, page gamePage) jsonLD {
	nodes := baseNodes(chrome)

	node := videoGame(chrome, game, page.Cover.URL)

	shots := make([]any, 0, len(page.Shots))
	for _, shot := range page.Shots {
		shots = append(shots, absoluteURL(chrome.URL, shot.URL))
	}
	node.set("screenshot", shots)

	// The stores a game is actually on. sameAs is the right property for
	// "the same thing, elsewhere", which is what a store page is.
	stores := make([]string, 0, len(page.Links))
	for _, link := range page.Links {
		stores = append(stores, link.URL)
	}
	node.set("sameAs", stores)

	return graph(append(nodes,
		node,
		chrome.breadcrumbs("Home", "", "Games", PageGames, game.Title, GamePagePrefix+game.Slug),
	)...)
}

func (s *Site) devlogIndexSchema(chrome SiteChrome, posts []postSummary) jsonLD {
	nodes := baseNodes(chrome)

	items := make([]any, 0, len(posts))
	for i, post := range posts {
		items = append(items, schema{
			"@type":    "ListItem",
			"position": i + 1,
			"url":      canonicalURL(chrome.URL, DevlogPagePrefix+post.Slug),
			"name":     post.Title,
		})
	}

	blog := chrome.page(PageDevlog, "Devlog", "Notes on what the studio is building.")
	blog["@type"] = "Blog"
	blog.set("publisher", schema{"@id": chrome.nodeID("#organization")})
	blog.set("mainEntity", schema{
		"@type":           "ItemList",
		"numberOfItems":   len(items),
		"itemListElement": items,
	})

	return graph(append(nodes,
		blog,
		chrome.breadcrumbs("Home", "", "Devlog", PageDevlog),
	)...)
}

func (s *Site) devlogPostSchema(chrome SiteChrome, post devlog.Post, page devlogPostPage) jsonLD {
	nodes := baseNodes(chrome)

	url := canonicalURL(chrome.URL, DevlogPagePrefix+post.Slug)
	article := schema{
		"@type":            "BlogPosting",
		"@id":              fallbackID(url, "#post"),
		"headline":         post.Title,
		"mainEntityOfPage": schema{"@type": "WebPage", "@id": fallbackID(url, "#webpage")},
		"publisher":        schema{"@id": chrome.nodeID("#organization")},
		"author":           schema{"@id": chrome.nodeID("#organization")},
	}
	article.set("url", url)
	if index := canonicalURL(chrome.URL, PageDevlog); index != "" {
		// An @id of nothing is worse than no link between the two: it points
		// the graph at a node that cannot exist.
		article.set("isPartOf", schema{"@id": index})
	}
	article.set("datePublished", post.PublishedAt)
	article.set("dateModified", post.UpdatedAt.UTC().Format("2006-01-02"))
	article.set("description", excerpt(post.ContentMarkdown, 160))
	article.set("image", absoluteURL(chrome.URL, page.Image.URL))
	if page.GameSlug != "" {
		article.set("about", schema{
			"@id": canonicalURL(chrome.URL, GamePagePrefix+page.GameSlug),
		})
	}

	return graph(append(nodes,
		article,
		chrome.breadcrumbs("Home", "", "Devlog", PageDevlog, post.Title, DevlogPagePrefix+post.Slug),
	)...)
}

func awardsSchema(chrome SiteChrome, awards []awardView) jsonLD {
	nodes := baseNodes(chrome)

	// An award belongs to whoever won it, so the studio carries them rather
	// than the page listing them as unattached things.
	names := make([]string, 0, len(awards))
	items := make([]any, 0, len(awards))
	for i, award := range awards {
		names = append(names, award.Title+" ("+award.Issuer+")")

		item := schema{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     award.Title,
		}
		item.set("url", award.Link)
		items = append(items, item)
	}
	nodes[0].set("award", names)

	page := chrome.page(PageAwards, "Awards", "Recognition the studio's games have picked up.")
	page["@type"] = "CollectionPage"
	page.set("mainEntity", schema{
		"@type":           "ItemList",
		"numberOfItems":   len(items),
		"itemListElement": items,
	})

	return graph(append(nodes,
		page,
		chrome.breadcrumbs("Home", "", "Awards", PageAwards),
	)...)
}

func contactSchema(chrome SiteChrome) jsonLD {
	nodes := baseNodes(chrome)

	page := chrome.page(PageContact, "Contact", "Get in touch with the studio.")
	page["@type"] = "ContactPage"

	return graph(append(nodes,
		page,
		chrome.breadcrumbs("Home", "", "Contact", PageContact),
	)...)
}
