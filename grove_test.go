package grove

import (
	"context"
	"testing"
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
