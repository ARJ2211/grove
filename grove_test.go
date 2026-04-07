package grove

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRun_HappyPath(t *testing.T) {
	// test function 1
	f1 := func() error {
		return nil
	}

	// test function 2
	f2 := func() error {
		return nil
	}

	// create a grove and launch the test functions
	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		g.Go("f1", func(ctx context.Context) error {
			err := f1()
			return err
		})

		g.Go("f2", func(ctx context.Context) error {
			err := f2()
			return err
		})

		return nil
	})

	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestRun_OneFailure(t *testing.T) {
	var me MultiError
	customError := errors.New("expected error.")

	// test function 1 (Happy Path)
	f1 := func() error {
		return nil
	}

	// test function 2 (Failiure)
	f2 := func() error {
		return customError
	}

	// run f1 and f2 under the grove and expect one error
	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		// launch test function 1
		g.Go("f1", func(ctx context.Context) error {
			err := f1()
			return err
		})

		// launch test function 2
		g.Go("f2", func(ctx context.Context) error {
			err := f2()
			return err
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected 1 error, got nil")
	}
	if !errors.Is(err, customError) {
		t.Errorf("expected customError, got %v", err)
	}
	if errors.As(err, &me) {
		t.Errorf("expected customError, got MultiError")
	}
}

func TestRun_MultipleFailures(t *testing.T) {
	e1 := errors.New("error 1")
	e2 := errors.New("error 2")
	e3 := errors.New("error 3")

	f1 := func() error {
		return e1
	}
	f2 := func() error {
		return e2
	}
	f3 := func() error {
		return e3
	}

	// launch all goroutines
	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		// launch function 1
		g.Go("f1", func(ctx context.Context) error {
			return f1()
		})

		// launch function 2
		g.Go("f2", func(ctx context.Context) error {
			return f2()
		})

		// launch function 3
		g.Go("f3", func(ctx context.Context) error {
			return f3()
		})

		return nil
	})

	var me MultiError
	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !errors.As(err, &me) {
		t.Errorf("expected MultiError, got: %v", err)
	}
	if !errors.Is(err, e1) {
		t.Errorf("expected e1 in chain, got: %v", err)
	}
	if !errors.Is(err, e2) {
		t.Errorf("expected e2 in chain, got: %v", err)
	}
	if !errors.Is(err, e3) {
		t.Errorf("expected e3 in chain, got: %v", err)
	}
}

func TestRun_PanicRecovery(t *testing.T) {
	e := errors.New("custom panic error")

	//panic function
	f := func(x, y int) (int, error) {
		if y == 0 {
			panic(e)
		}
		return x / y, nil
	}

	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		g.Go("panic_task", func(ctx context.Context) error {
			_, err := f(4, 0)
			return err
		})

		return nil
	})

	var pe PanicError
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(err, e) {
		t.Errorf("expected custom panic error in chain, got %v", err)
	}
	if !errors.As(err, &pe) {
		t.Errorf("expected panic error, got %v", err)
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace in error, got: %v", pe)
	}
}

func TestRun_PanicRecovery_StringValue(t *testing.T) {
	//panic function
	f := func(x, y int) (int, error) {
		if y == 0 {
			panic("something went wrong")
		}
		return x / y, nil
	}

	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		g.Go("panic_task", func(ctx context.Context) error {
			_, err := f(4, 0)
			return err
		})

		return nil
	})

	var pe PanicError
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("expected 'something went wrong', got: %v", err)
	}
	if !errors.As(err, &pe) {
		t.Errorf("expected panic error, got %v", err)
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace in error, got: %v", pe)
	}
}

func TestRun_ZeroGoroutines(t *testing.T) {
	ctx := context.Background()
	err := Run(ctx, func(g *Grove) error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error")
	}
}

func TestRun_ParentContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	err := Run(ctx, func(g *Grove) error {
		// cancel parent context after 75ms
		g.Go("cancel_context", func(ctx context.Context) error {
			time.Sleep(75 * time.Millisecond)
			cancel()
			return nil
		})

		g.Go("long_task", func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected cancelled context, got: %v", err)
	}
}

func TestRun_NestedGrove(t *testing.T) {
	parentCTX := context.Background()
	e := errors.New("nested error")

	err := Run(parentCTX, func(g *Grove) error {
		g.Go("some_func", func(ctx context.Context) error {
			return nil
		})

		// nested grove that should percolate its errors up.
		nestedCTX := context.Background()
		err := Run(nestedCTX, func(g *Grove) error {
			g.Go("nested_func1", func(ctx context.Context) error {
				panic(e)
			})

			return nil
		})

		return err
	})

	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !errors.Is(err, e) {
		t.Errorf("expected nested error in chain, got: %v", err)
	}
}

func TestRun_ClosedGrove(t *testing.T) {
	var savedGrove *Grove
	ctx := context.Background()

	defer func() {
		r := recover()

		if r == nil {
			t.Errorf("expected panic recovery")
		}

		err, ok := r.(error)
		if !ok {
			t.Errorf("expected an error object, got: %T", r)
		}
		if !strings.Contains(err.Error(), "cannot access closed grove") {
			t.Errorf("incorrect panic received, got: %v", err)
		}
	}()

	err := Run(ctx, func(g *Grove) error {
		savedGrove = g

		g.Go("function", func(ctx context.Context) error {
			return nil
		})

		return nil
	})

	savedGrove.Go("error_func", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error in, got: %v", err)
	}
}
