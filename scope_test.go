package grove

import (
	"context"
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
