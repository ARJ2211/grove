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
