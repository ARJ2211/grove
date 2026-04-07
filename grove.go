package grove

import (
	"context"
	"sync"
)

type Grove struct {
	ctx    context.Context         // hold the context of the goroutine
	cancel context.CancelCauseFunc // capture any error or cause of cancel
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
		errs:   []error{},
	}

	// run all the goroutines that the user registered
	if err := fn(&g); err != nil {
		g.errs = append(g.errs, err)
	}

	// wait for the grove to finish collecting all errors
	g.wg.Wait()
	return Join(g.errs...)
}

// func (g *Grove) Go(name string, fn func(ctx context.Context) error) {

// 	g.wg.Add(1)
// 	defer g.wg.Done()

// }
// func (g *Grove) Context() context.Context
