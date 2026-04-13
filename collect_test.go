package grove

import (
	"context"
	"errors"
	"fmt"
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

func TestCollect_FnReturnsError(t *testing.T) {
	type T any
	ctx := context.Background()
	e := errors.New("setup function error")

	res, err := Collect(ctx, func(tg *TypedGrove[T]) error {
		return e
	})

	if len(res) != 0 {
		t.Errorf("expect 0 res, got: %v", res)
	}
	if !errors.Is(err, e) {
		t.Errorf("expected setup error, got: %v", err)
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

func TestFirst_MultiErrorOneSuccess(t *testing.T) {
	type T any
	var me MultiError
	ctx := context.Background()
	e := errors.New("expected error.")

	// random user defined function
	f := func(a, b int, expectError bool) (int, error) {
		if expectError {
			return 0, e
		}
		return a + b, nil
	}

	numProcs := 10000
	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		for j := 0; j < numProcs; j++ {
			// prevent race
			i := j

			// success path
			if i == 420 {
				tg.SubmitFirst("func", func(ctx context.Context) (T, error) {
					r, err := f(i, 10, false)
					return r, err
				})
			} else {
				// error path
				tg.SubmitFirst("func-fail", func(ctx context.Context) (T, error) {
					r, err := f(i, 10, true)
					return r, err
				})
			}
		}

		return nil
	})

	if err == nil {
		t.Errorf("expected err, got nil")
	}
	if !errors.Is(err, e) {
		t.Errorf("expected 'expected error', got: %v", err)
	}
	if !errors.As(err, &me) {
		t.Errorf("expected multi-error, got: %v", err)
	}
	if res != 430 {
		t.Errorf("expected result 430, got: %d", res)
	}
}

func TestFirst_AllFail(t *testing.T) {
	type T any
	ctx := context.Background()
	expectedErrors := []error{}

	numProcs := 10
	for i := 0; i < numProcs; i++ {
		expectedErrors = append(
			expectedErrors, fmt.Errorf("func_%d failed", i),
		)
	}

	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		// all the errors fail
		for i := 0; i < numProcs; i++ {
			ci := i
			name := fmt.Sprintf("func_%d", ci)
			tg.SubmitFirst(name, func(ctx context.Context) (T, error) {
				return *new(T), expectedErrors[ci]
			})
		}

		return nil
	})

	if res != nil {
		t.Errorf("expected all errors, got: %v", res)
	}
	if err == nil {
		t.Error("expected err, got nil")
	}

	for i := 0; i < numProcs; i++ {
		if !errors.Is(err, expectedErrors[i]) {
			t.Errorf("expected err_%d, not found in error chain", i)
		}
	}
}

func TestFirst_ContextCancelled(t *testing.T) {
	type T any
	ctx := context.Background()
	e := errors.New("context cancelled")

	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		// this function will cancel the context
		tg.SubmitFirst("cancel-context", func(ctx context.Context) (T, error) {
			return *new(T), e
		})

		// this function should not run, will not have context in err chain
		tg.SubmitFirst("context-cancelled", func(ctx context.Context) (T, error) {
			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(1 * time.Second):
				return "something", nil
			}

		})

		return nil
	})

	if res != nil {
		t.Errorf("expected no res, got: %v", res)
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("expected err chain to not have context.Canceled")
	}
}

func TestFirst_SetupError(t *testing.T) {
	type T any
	ctx := context.Background()

	res, err := First(ctx, func(tg *TypedGrove[T]) error {
		return errors.New("expected error")
	})

	if res != nil {
		t.Errorf("expected nil res, got: %v", res)
	}
	if err == nil {
		t.Error("expected err, got nil")
	}
}

func TestRace_HappyPath(t *testing.T) {
	type T any
	ctx := context.Background()

	// define func where f1 is fastest
	f1 := func() string {
		return "fastest"
	}

	res, err := Race(ctx, func(tg *TypedGrove[T]) error {

		// run the slowest functions first
		tg.SubmitRace("slow1", func(ctx context.Context) (T, error) {
			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "slow", nil
			}
		})
		tg.SubmitRace("slow2", func(ctx context.Context) (T, error) {
			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "slow", nil
			}
		})

		// run the fastest function second
		tg.SubmitRace("fast", func(ctx context.Context) (T, error) {
			return f1(), nil
		})

		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if res == nil {
		t.Log(res)
		t.Errorf("expected fastest task first")
	}
}

func TestRace_ErrorFirst(t *testing.T) {
	type T any
	ctx := context.Background()
	e := errors.New("fastest error")

	// define 3 functions where f1 is fastest
	// and returns an error
	f1 := func() (string, error) {
		return "", e
	}

	res, err := Race(ctx, func(tg *TypedGrove[T]) error {

		// run the slowest functions first
		tg.SubmitRace("slow1", func(ctx context.Context) (T, error) {
			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "slow", nil
			}
		})
		tg.SubmitRace("slow2", func(ctx context.Context) (T, error) {
			select {
			case <-ctx.Done():
				return *new(T), ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "slow", nil
			}
		})

		// run the fastest function second
		tg.SubmitRace("fast", func(ctx context.Context) (T, error) {
			res, err := f1()

			if err != nil {
				return *new(T), err
			}
			return res, nil
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected err, got nil")
	}
	if res != nil {
		t.Errorf("expected no res, got: %v", res)
	}
	if !errors.Is(err, e) {
		t.Errorf("expected 'fastest error', got: %v", err)
	}
}

func TestRace_ContextCanceledFound(t *testing.T) {
	type T any
	ctx := context.Background()
	ctxError := errors.New("context cancelled on purpose")

	numProcs := 1000

	res, err := Race(ctx, func(tg *TypedGrove[T]) error {
		// cancel the parent context first
		// so that all goroutines flood the
		// context.Cancelled block.
		tg.grove.cancel(ctxError)

		// now try to race multiple goroutines
		for i := 0; i < numProcs; i++ {
			ci := i
			name := fmt.Sprintf("func_%d", ci)
			tg.SubmitRace(name, func(ctx context.Context) (T, error) {
				select {
				case <-ctx.Done():
					return *new(T), ctx.Err()
				default:
					return fmt.Sprintf("completed %d", ci), nil
				}
			})
		}

		return nil
	})

	if res != nil {
		t.Errorf("expected nil res, got: %v", res)
	}
	t.Log(err)
}

func TestRace_SetupError(t *testing.T) {
	type T any
	ctx := context.Background()

	res, err := Race(ctx, func(tg *TypedGrove[T]) error {
		return errors.New("expected error")
	})

	if res != nil {
		t.Errorf("expected nil res, got: %v", res)
	}
	if err == nil {
		t.Error("expected err, got nil")
	}
}
