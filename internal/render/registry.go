package render

import "strings"

type Renderer func(pageKey string) (html []byte, tags []string, err error)

type prefixEntry struct {
	prefix   string
	renderer Renderer
}

type Registry struct {
	exact    map[string]Renderer
	prefixes []prefixEntry
}

func NewRegistry() *Registry {
	return &Registry{exact: make(map[string]Renderer)}
}

func (r *Registry) Register(pageKey string, renderer Renderer) {
	r.exact[pageKey] = renderer
}

func (r *Registry) RegisterPrefix(prefix string, renderer Renderer) {
	r.prefixes = append(r.prefixes, prefixEntry{prefix: prefix, renderer: renderer})
}

func (r *Registry) Resolve(pageKey string) (Renderer, bool) {
	if renderer, ok := r.exact[pageKey]; ok {
		return renderer, true
	}
	var best prefixEntry
	found := false
	for _, entry := range r.prefixes {
		if strings.HasPrefix(pageKey, entry.prefix) && len(entry.prefix) > len(best.prefix) {
			best = entry
			found = true
		}
	}
	if found {
		return best.renderer, true
	}
	return nil, false
}
