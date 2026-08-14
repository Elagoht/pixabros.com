package media

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pixabros/internal/storage"
)

// countingLookup records how many times the sweep asked for references.
type countingLookup struct {
	mu    sync.Mutex
	calls int
	ids   map[string]bool
	err   error
}

func (c *countingLookup) lookup() (map[string]bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.ids, c.err
}

func (c *countingLookup) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func TestSweeper_RunSweepsOnEveryTick(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")
	lookup := &countingLookup{ids: map[string]bool{}}

	sweeper := NewSweeper(repo, files, lookup.lookup, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sweeper.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for lookup.count() < 2 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("sweeper ran %d times in 2s, want at least 2", lookup.count())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after its context was cancelled")
	}
}

// The sweeper sweeps as soon as it starts.
//
// It used to wait a full interval, on the reasoning that a restart is when an
// operator least wants an unattended delete. That reasoning was wrong: what
// protects a recent upload is the grace period below, not the delay, and a
// startup sweep can only remove what the next tick would have removed anyway.
// What the delay did add was six hours after every restart in which nothing
// proved the sweeper was running at all.
func TestSweeper_SweepsOnStartup(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	lookup := &countingLookup{ids: map[string]bool{}}

	sweeper := NewSweeper(repo, storage.NewLocalDisk(t.TempDir(), "/media"), lookup.lookup, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	go sweeper.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	if got := lookup.count(); got == 0 {
		t.Error("the sweeper did nothing on startup, so nothing proves it is running")
	}
}

// And the startup sweep is safe because of the grace period: an image uploaded
// a moment ago and not yet attached to anything is left alone.
func TestSweeper_StartupSweepSparesARecentUpload(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	fresh, err := repo.Create("media/og_image/2026-fresh.webp", 1200, 630)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sweeper := NewSweeper(
		repo,
		storage.NewLocalDisk(t.TempDir(), "/media"),
		func() (map[string]bool, error) { return map[string]bool{}, nil },
		time.Hour,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go sweeper.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	if _, err := repo.FindByID(fresh.ID); err != nil {
		t.Errorf("a just-uploaded image was swept away: %v", err)
	}
}

// A sweep that found nothing still says so. Silence is what made a stopped
// sweeper look exactly like a working one.
func TestSweeper_ReportsEvenAnEmptySweep(t *testing.T) {
	repo := NewRepo(setupTestDB(t))

	reported := make(chan int, 4)
	sweeper := NewSweeper(
		repo,
		storage.NewLocalDisk(t.TempDir(), "/media"),
		func() (map[string]bool, error) { return map[string]bool{}, nil },
		time.Hour,
		WithSweepLogger(func(deleted int) { reported <- deleted }),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sweeper.Run(ctx)

	select {
	case deleted := <-reported:
		if deleted != 0 {
			t.Errorf("deleted = %d, want 0 with nothing to sweep", deleted)
		}
	case <-time.After(2 * time.Second):
		t.Error("the sweeper swept nothing and said nothing")
	}
}

// A failing lookup must not stop the schedule: the next tick tries again.
func TestSweeper_ReportsErrorsAndKeepsGoing(t *testing.T) {
	repo := NewRepo(setupTestDB(t))
	lookup := &countingLookup{err: errors.New("database is busy")}

	var mu sync.Mutex
	var reported []error
	sweeper := NewSweeper(repo, storage.NewLocalDisk(t.TempDir(), "/media"), lookup.lookup, 10*time.Millisecond,
		WithSweepErrorLogger(func(err error) {
			mu.Lock()
			reported = append(reported, err)
			mu.Unlock()
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go sweeper.Run(ctx)

	deadline := time.After(2 * time.Second)
	for lookup.count() < 2 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("sweeper stopped after an error: %d runs", lookup.count())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if len(reported) == 0 {
		t.Error("the error was never reported")
	}
}

// The grace period is the whole reason an upload can be stored before the form
// referencing it is saved.
func TestSweeper_SweepOnceRespectsTheGracePeriod(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)
	files := storage.NewLocalDisk(t.TempDir(), "/media")

	fresh, err := repo.Create("fresh.webp", 100, 100)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var swept int
	sweeper := NewSweeper(repo, files, func() (map[string]bool, error) {
		return map[string]bool{}, nil
	}, time.Hour, WithSweepLogger(func(deleted int) { swept = deleted }))

	sweeper.SweepOnce()

	if swept != 0 {
		t.Errorf("swept %d images, want 0 while inside the grace period", swept)
	}
	if _, err := repo.FindByID(fresh.ID); err != nil {
		t.Errorf("a just-uploaded image was deleted: %v", err)
	}

	// With no grace at all the same orphan goes.
	zeroGrace := NewSweeper(repo, files, func() (map[string]bool, error) {
		return map[string]bool{}, nil
	}, time.Hour,
		WithSweepGrace(0),
		WithSweepLogger(func(deleted int) { swept = deleted }),
	)
	zeroGrace.SweepOnce()

	if swept != 1 {
		t.Errorf("swept %d images, want 1 once the grace period is zero", swept)
	}
	if _, err := repo.FindByID(fresh.ID); !errors.Is(err, ErrMediaNotFound) {
		t.Errorf("FindByID() after the sweep error = %v, want ErrMediaNotFound", err)
	}
}

// A referenced image is never swept, whatever its age.
func TestSweeper_LeavesReferencedImagesAlone(t *testing.T) {
	conn := setupTestDB(t)
	repo := NewRepo(conn)

	kept, err := repo.Create("kept.webp", 100, 100)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sweeper := NewSweeper(repo, storage.NewLocalDisk(t.TempDir(), "/media"), func() (map[string]bool, error) {
		return map[string]bool{kept.ID: true}, nil
	}, time.Hour, WithSweepGrace(0))

	sweeper.SweepOnce()

	if _, err := repo.FindByID(kept.ID); err != nil {
		t.Errorf("a referenced image was swept: %v", err)
	}
}
