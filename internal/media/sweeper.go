package media

import (
	"context"
	"time"

	"pixabros/internal/storage"
)

// OrphanGrace is how long an unreferenced image is left alone before the sweep
// takes it. An upload is stored before the form that will point at it is
// saved, so without this delay the sweep would delete artwork out from under
// someone in the middle of filling in a page.
const OrphanGrace = 24 * time.Hour

// Sweeper runs SweepOrphans on a schedule. Without it nothing ever reclaims
// the images left behind by deleted games, replaced artwork or abandoned
// uploads: the rows and their files simply accumulate.
type Sweeper struct {
	repo       *Repo
	files      storage.Storage
	referenced ReferenceLookup
	interval   time.Duration
	grace      time.Duration
	onSweep    func(deleted int)
	onError    func(error)
}

type SweeperOption func(*Sweeper)

// WithSweepLogger reports each completed sweep, so an operator can see that it
// is running and what it removed.
func WithSweepLogger(onSweep func(deleted int)) SweeperOption {
	return func(s *Sweeper) { s.onSweep = onSweep }
}

// WithSweepErrorLogger reports a failed sweep. A failure is not fatal: the
// next tick tries again.
func WithSweepErrorLogger(onError func(error)) SweeperOption {
	return func(s *Sweeper) { s.onError = onError }
}

// WithSweepGrace overrides how long an orphan is spared, for tests.
func WithSweepGrace(grace time.Duration) SweeperOption {
	return func(s *Sweeper) { s.grace = grace }
}

func NewSweeper(
	repo *Repo,
	files storage.Storage,
	referenced ReferenceLookup,
	interval time.Duration,
	opts ...SweeperOption,
) *Sweeper {
	s := &Sweeper{
		repo:       repo,
		files:      files,
		referenced: referenced,
		interval:   interval,
		grace:      OrphanGrace,
		onSweep:    func(int) {},
		onError:    func(error) {},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run sweeps on every tick until ctx is cancelled. It deliberately does not
// sweep immediately on start: a restart is exactly when an operator is least
// likely to want an unattended delete, and the first tick comes soon enough.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepOnce()
		}
	}
}

// SweepOnce runs a single sweep. It is exported so a test -- or a future
// manual "clean up now" action -- can trigger one without waiting for a tick.
func (s *Sweeper) SweepOnce() {
	deleted, err := SweepOrphans(s.repo, s.files, s.referenced, s.grace, time.Now())
	if err != nil {
		s.onError(err)
	}
	if deleted > 0 {
		s.onSweep(deleted)
	}
}
