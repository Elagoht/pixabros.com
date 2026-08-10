package render

import (
	"strings"
	"sync"
)

type Renderer func(pageKey string) (html []byte, tags []string, err error)

type prefixEntry struct {
	prefix   string
	renderer Renderer
}

// Registry maps page keys to renderers. Registration is expected to happen
// during startup, but the mutex makes concurrent registration and resolution
// safe regardless — Resolve runs on the render worker's goroutine, which is
// live while the rest of main() is still wiring handlers up.
type Registry struct {
	mu       sync.RWMutex
	exact    map[string]Renderer
	prefixes []prefixEntry
}

func NewRegistry() *Registry {
	return &Registry{exact: make(map[string]Renderer)}
}

func (r *Registry) Register(pageKey string, renderer Renderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exact[pageKey] = renderer
}

func (r *Registry) RegisterPrefix(prefix string, renderer Renderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prefixes = append(r.prefixes, prefixEntry{prefix: prefix, renderer: renderer})
}

func (r *Registry) Resolve(pageKey string) (Renderer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
