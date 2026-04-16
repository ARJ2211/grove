package grove

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestScope_TimeoutTaskCompletesInTime(t *testing.T) {
	ctx := context.Background()

	err := Run(ctx, func(g *Grove) error {
		// create a scope of 100 ms timeout
		scope := g.WithTimeout(100 * time.Millisecond)
		scope.Go("task-50ms", func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		return nil
	})

	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func TestScope_TimeoutTaskExceedsTimeout(t *testing.T) {
	ctx := context.Background()

	// a timer for a long task (timer is context aware)
	timer := time.NewTimer(100 * time.Second)
	defer timer.Stop()

	err := Run(ctx, func(g *Grove) error {
		// create a scope of 50 ms timeout
		scope := g.WithTimeout(50 * time.Millisecond)
		scope.Go("task-100ms", func(ctx context.Context) error {
			// making the function context aware using timer
			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected err, got: %v", err)
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context deadline exceeded err, got: %v", err)
	}
}

func TestScope_TimeoutDoesNotCancelSiblings(t *testing.T) {
	ctx := context.Background()

	// these errors should not be in the error chain
	// will be called when the context deadline has passed
	// but we do not cancel the context in a scope
	e1 := errors.New("normal task 1 failed")
	e2 := errors.New("normal task 2 failed")
	e3 := errors.New("normal task 3 failed")

	err := Run(ctx, func(g *Grove) error {
		scope := g.WithTimeout(100 * time.Millisecond)
		scope.Go("long-task", func(ctx context.Context) error {
			{
				// wait till done channel is triggered
				<-ctx.Done()
				return ctx.Err()
			}
		})

		// these tasks clearly run AFTER the scoped context
		// has exceeded its deadline but we do not cancel
		// the parent context in that case
		g.Go("normal-task1", func(ctx context.Context) error {
			select {
			case <-time.After(110 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return e1
			}
		})
		g.Go("normal-task2", func(ctx context.Context) error {
			select {
			case <-time.After(110 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return e2
			}
		})
		g.Go("normal-task3", func(ctx context.Context) error {
			select {
			case <-time.After(110 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return e3
			}
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected err, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded err, got: %v", err)
	}
	if errors.Is(err, e1) || errors.Is(err, e2) || errors.Is(err, e3) {
		t.Errorf("expected e1, e2, e3 not in chain, got: %v", err)
	}
}

func TestScope_TwoTasksWithDifferentTimeouts(t *testing.T) {
	var me MultiError
	ctx := context.Background()

	ctxE := errors.New("parent context cancelled")

	err := Run(ctx, func(g *Grove) error {
		// scope 1 (50 ms)
		scope1 := g.WithTimeout(50 * time.Millisecond)
		scope1.Go("long-task-scope1", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

		// scope 2 (30 ms)
		scope2 := g.WithTimeout(30 * time.Millisecond)
		scope2.Go("long-task-scope2", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

		g.Go("normal-task", func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctxE
			}
		})

		return nil
	})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.As(err, &me) {
		t.Errorf("expected multierror, got: %v", err)
	}
	if errors.Is(err, ctxE) {
		t.Errorf("expected ctxE to not be in error chain, got: %v", err)
	}
	for i, e := range me.Unwrap() {
		if e != context.DeadlineExceeded {
			t.Errorf("expected deadline exceeded got: %v @ %d", e, i)
		}
	}
}
