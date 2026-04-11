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
