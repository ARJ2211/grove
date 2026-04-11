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

func (tg *TypedGrove[T]) Submit(
	name string,
	fn func(ctx context.Context) (T, error),
) {
}
