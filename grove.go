package grove

import (
	"context"
	"errors"
	"sync"

	"github.com/ARJ2211/grove/internal"
)

type Grove struct {
	ctx    context.Context         // hold the context of the goroutine
	cancel context.CancelCauseFunc // capture any error or cause of cancel
	closed bool                    //check if the grove is closed
	wg     sync.WaitGroup          // track the number of goroutines under a grove
	mu     sync.Mutex              // lock the common resources [errs]
	errs   []error                 // catch the errors within the grove
}

// Created a grove and runs the functions under it.
func Run(ctx context.Context, fn func(*Grove) error) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	g := Grove{
		ctx:    ctx,
		cancel: cancel,
		closed: false,
		errs:   []error{},
	}

	// run all the goroutines that the user registered
	if err := fn(&g); err != nil {
		g.errs = append(g.errs, err)
	}

	// wait for the grove to finish collecting all errors
	g.wg.Wait()

	// close the grove
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	return Join(g.errs...)
}

// Launches all the goroutines within the grove.
func (g *Grove) Go(name string, fn func(ctx context.Context) error) {
	g.mu.Lock()
	// check if the grove is closed
	if g.closed {
		panic(ErrClosedGrove)
	}

	// add the goroutine to the waitgroup
	g.wg.Add(1)
	g.mu.Unlock()

	go func() {
		defer g.wg.Done()
		err := internal.CapturePanic(func() error { return fn(g.ctx) })

		if err != nil {
			g.mu.Lock()
			g.errs = append(g.errs, err)
			g.mu.Unlock()

			g.cancel(err)
		}
	}()
}

// Expose the grove context to the user.
func (g *Grove) Context() context.Context {
	return g.ctx
}

var (
	ErrClosedGrove error = errors.New("cannot access closed grove")
)
