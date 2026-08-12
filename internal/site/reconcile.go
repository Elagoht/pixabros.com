package site

import (
	"fmt"

	"pixabros/internal/render"
)

// DesiredPages is every page key the site should currently have.
//
// It is the definition of "what the site consists of". The reconciler renders
// anything here that is missing and forgets anything rendered that is no
// longer here. As dynamic pages land -- one per game, one per devlog post --
// they are appended here from the database.
func (s *Site) DesiredPages() ([]string, error) {
	defs := s.pages()
	keys := make([]string, 0, len(defs))
	for _, page := range defs {
		keys = append(keys, page.Key)
	}

	// One page per published game. Unpublishing a game drops it from this list,
	// which is what makes the reconciler stop serving its page.
	published, err := s.publishedGames()
	if err != nil {
		return nil, err
	}
	for _, game := range published {
		keys = append(keys, GamePagePrefix+game.Slug)
	}

	// And one per published devlog post.
	posts, err := s.publishedPosts()
	if err != nil {
		return nil, err
	}
	for _, post := range posts {
		keys = append(keys, DevlogPagePrefix+post.Slug)
	}

	return keys, nil
}

// Reconciler brings the set of rendered pages in line with the set that should
// exist.
//
// This is what makes a page exist in the first place. The regen worker only
// resolves a tag to pages that already declared it in page_tags, so a page key
// that has never been rendered has no rows and no job can ever produce it.
// Without reconciliation the whole site stays 404 forever.
type Reconciler struct {
	// desired is injected rather than read off Site so the reconciler has no
	// opinion about where the page list comes from -- and so a test can hand
	// it a page whose renderer fails without adding a hook to production code.
	desired  func() ([]string, error)
	store    *render.Store
	registry *render.Registry
	onError  func(error)
}

type ReconcilerOption func(*Reconciler)

// WithReconcileErrorLogger reports a page that failed to render. A failure is
// not fatal: one broken page must not take the rest of the site down.
func WithReconcileErrorLogger(onError func(error)) ReconcilerOption {
	return func(r *Reconciler) { r.onError = onError }
}

func NewReconciler(desired func() ([]string, error), store *render.Store, registry *render.Registry, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		desired:  desired,
		store:    store,
		registry: registry,
		onError:  func(error) {},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Reconcile renders every desired page that has no rendered_pages row, and
// forgets every rendered page that is no longer desired -- a deleted game must
// stop being served, not linger until someone notices.
//
// Removing a page deletes its rendered_pages row; its page_tags go with it by
// cascade. The file in the rendered store is deliberately left behind: files
// are keyed by content hash and may be shared between pages, and the store has
// no reference counting. That is known, bounded garbage; if it ever matters it
// wants a sweeper like internal/media's, not an inline delete.
func (r *Reconciler) Reconcile() (rendered int, removed int, err error) {
	desired, err := r.desired()
	if err != nil {
		return 0, 0, err
	}

	wanted := make(map[string]bool, len(desired))
	for _, key := range desired {
		wanted[key] = true
	}

	for _, key := range desired {
		exists, err := r.store.HasPage(key)
		if err != nil {
			return rendered, removed, err
		}
		if exists {
			continue
		}

		renderer, ok := r.registry.Resolve(key)
		if !ok {
			r.onError(fmt.Errorf("no renderer registered for page %q", key))
			continue
		}
		if _, err := r.store.RenderAndPersist(key, renderer); err != nil {
			r.onError(fmt.Errorf("render %q: %w", key, err))
			continue
		}
		rendered++
	}

	existing, err := r.store.PageKeys()
	if err != nil {
		return rendered, removed, err
	}
	for _, key := range existing {
		if wanted[key] {
			continue
		}
		if err := r.store.Forget(key); err != nil {
			r.onError(fmt.Errorf("forget %q: %w", key, err))
			continue
		}
		removed++
	}

	return rendered, removed, nil
}
