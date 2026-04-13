package grove

import (
	"context"
	"errors"
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

// collect only the FIRST successful value
// and any errors associated with all the
// goroutines.
func First[T any](ctx context.Context, fn func(*TypedGrove[T]) error) (T, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

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

	if err := fn(&tg); err != nil {
		g.errs = append(g.errs, err)
	}

	// wait for the grove to finish collecting all errors and results
	g.wg.Wait()

	// close the grove
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	if len(tg.results) > 0 {
		return tg.results[0], Join(g.errs...)
	}

	return *new(T), Join(g.errs...)
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

		// if context is cancelled, which means
		// there was either a panic or a result
		// was found, return early
		if errors.Is(err, context.Canceled) {
			return nil
		}

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

// this will returns the FIRST successfull OR
// unsuccessful task. This will return the first
// task that either has a result or an error.
func Race[T any](ctx context.Context, fn func(tg *TypedGrove[T]) error) (T, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// create grove
	g := Grove{
		ctx:    ctx,
		cancel: cancel,
		closed: false,
		errs:   []error{},
	}

	// create typed grove
	tg := TypedGrove[T]{
		grove:   &g,
		results: []T{},
		found:   false,
	}

	if err := fn(&tg); err != nil {
		g.errs = append(g.errs, err)
	}

	// wait for the grove to finish collecting all errors and results
	// to prevent any goroutine leaks
	g.wg.Wait()

	// close the grove
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	if len(tg.results) > 0 {
		return tg.results[0], Join(g.errs...)
	}

	// if no result, means it was an error
	// there can be two cases:
	if len(g.errs) == 0 {
		// case 1: no errors means that the
		// context was cancelled before
		// any functions could run
		return *new(T), ctx.Err()
	}
	// case 2: there is only 1
	// error which was the first
	// error it encountered.
	return *new(T), g.errs[0]
}

// run a task with a race, if an error or a result
// is found, close the context for that grove
func (tg *TypedGrove[T]) SubmitRace(
	name string,
	fn func(ctx context.Context) (T, error),
) {
	tg.grove.Go(name, func(ctx context.Context) error {
		var result T
		var err error

		// call the function
		result, err = fn(ctx)

		// if context is cancelled, which means
		// there was either a panic or a result
		// was found, return early. We also
		// set found to true
		if errors.Is(err, context.Canceled) {
			tg.mu.Lock()
			defer tg.mu.Unlock()

			if !tg.found {
				tg.found = true
				tg.grove.cancel(nil) // cancel the context
			}
			return nil
		}

		// even if there is an error, close
		// the context. Return the first
		// result irrespective of succ or fail.
		if err != nil {
			tg.mu.Lock()
			defer tg.mu.Unlock()

			if !tg.found {
				tg.found = true
				tg.grove.cancel(nil) // cancel the context
			}
			return err
		}

		// in happy path, append the result
		tg.mu.Lock()
		defer tg.mu.Unlock()

		if !tg.found {
			tg.found = true
			tg.results = append(tg.results, result)
			tg.grove.cancel(nil) // cancel the context
		}
		return nil
	})
}
