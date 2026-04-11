package grove

import (
	"context"
	"testing"
)

func TestCollect_HappyPath(t *testing.T) {
	ctx := context.Background()
	type T any

	add := func(a, b int) int {
		return a + b
	}

	res, err := Collect(ctx, func(tg *TypedGrove[T]) error {
		for i := 0; i < 10; i++ {
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
	if len(res) != 10 {
		t.Errorf("expected 10 return values, got: %d", len(res))
	}
	t.Log(res)
}
