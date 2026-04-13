package grove

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

func TestCollect_HappyPath(t *testing.T) {
	ctx := context.Background()
	type T any

	// can be any random user defined function that returns some value
	add := func(a, b int) int {
		return a + b
	}

	res, err := Collect(ctx, func(tg *TypedGrove[T]) error {
		for i := 0; i < 100; i++ {
			io := i
			tg.Submit("add", func(ctx context.Context) (T, error) {
				r := add(io, 2)
				return r, nil
			})
		}

		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if len(res) != 100 {
		t.Errorf("expected 100 return values, got: %d", len(res))
	}
}

func TestCollect_OneFails(t *testing.T) {
	type T any
	e := errors.New("expected fail.")
	ctx := context.Background()

	// random user defined function that returns a value
	f := func(expectError bool) (string, error) {
		if expectError {
			return "fail", e
		}
		return "success", nil
	}

	res, err := Collect(ctx, func(tg *TypedGrove[T]) error {
		numRuns := 5
		failAt := 3

		for i := 0; i < numRuns; i++ {
			if i == failAt {
				tg.Submit("task-fail", func(ctx context.Context) (T, error) {
					r, err := f(true)
					return r, err
				})
			} else {
				tg.Submit("task-pass", func(ctx context.Context) (T, error) {
					r, err := f(false)
					return r, err
				})
			}
		}

		return nil
	})

	if err == nil {
		t.Error("expected err, got: nil")
	}
	if slices.Contains(res, "fail") {
		t.Errorf("expected only success, got: %v", res)
	}
	if !errors.Is(err, e) {
		t.Errorf("expected 'expected fail.', got: %v", err)
	}
}

func TestFirst_HappyPath(t *testing.T) {
	type T any
	ctx := context.Background()

	// random user defined function
	f := func(a, b int) int {
		return a + b
	}

	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		// only one goroutine
		tg.SubmitFirst("func", func(ctx context.Context) (T, error) {
			return f(2, 3), nil
		})

		return nil
	})

	if err != nil {
		t.Errorf("expected nil err, got: %v", err)
	}
	if res != 5 {
		t.Errorf("expected 5, got: %d", res)
	}
}

func TestCollect_MultiRunHappyPath(t *testing.T) {
	type T any
	ctx := context.Background()

	// random user defined function
	f := func(a, b int) int {
		time.Sleep(1 * time.Second)
		return a + b
	}

	numProc := 10000

	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		// launch multiple but expect only one value stored
		for i := 0; i < numProc; i++ {
			io := i
			tg.SubmitFirst("task", func(ctx context.Context) (T, error) {
				return f(io, 10), nil
			})
		}

		return nil
	})

	if res == nil {
		t.Error("expected some res value")
	}
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}
