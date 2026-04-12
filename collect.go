package grove

import (
	"context"
	"sync"
)

// struct used to collect values
// from the user defined function
type TypedGrove[T any] struct {
	grove   *Grove     // which scope or grove are the results in
	mu      sync.Mutex // locking mech
	results []T        // the results of the function
	found   bool       // detect if there is a first result
}

// this function will call grove.Go which will
// wrap the actual function whose value we need
// and append it to the results of TypedGrove.
func (tg *TypedGrove[T]) Submit(
	name string,
	fn func(ctx context.Context) (T, error),
) {
	tg.grove.Go(name, func(ctx context.Context) error {
		var result T
		var err error

		// call the function
		result, err = fn(ctx)

		if err != nil {
			return err
		}

		tg.mu.Lock()
		defer tg.mu.Unlock()
		tg.results = append(tg.results, result)

		return nil
	})
}

// collect all the T values from that grove
// and return them along with the errors
func Collect[T any](ctx context.Context, fn func(*TypedGrove[T]) error) ([]T, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// create Grove and TypedGrove
	g := Grove{
		ctx:    ctx,
		cancel: cancel,
		closed: false,
		errs:   []error{},
	}

	tg := TypedGrove[T]{
		grove:   &g,
		results: []T{},
		found:   false,
	}

	// run the gorutines that the user has defined
	if err := fn(&tg); err != nil {
		g.errs = append(g.errs, err)
	}

	// wait for the grove to finish collecting all errors and results
	g.wg.Wait()

	// close the grove
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	return tg.results, Join(g.errs...)
}

// run the goroutines but also cancel the
// context on successful execution
func (tg *TypedGrove[T]) SubmitFirst(
	name string,
	fn func(ctx context.Context) (T, error),
) {
	tg.grove.Go(name, func(ctx context.Context) error {
		var result T
		var err error

		// call the function
		result, err = fn(ctx)

		if err != nil {
			return err
		}

		tg.mu.Lock()
		defer tg.mu.Unlock()

		// if there already is a result, then skip it
		if !tg.found {
			tg.found = true
			tg.results = append(tg.results, result)
			tg.grove.cancel(nil) // cancel the context
		}

		return nil
	})
}
