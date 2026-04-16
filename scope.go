package grove

import (
	"context"
	"time"

	"github.com/ARJ2211/grove/internal"
)

// this structure is used to create a scope
// where the user can set deadlines or timeouts
// the main difference between a scope and a grove
// is that we do not cancel the context of the grove
// when there is an error.
type Scope struct {
	grove    *Grove
	timeout  time.Duration
	deadline time.Time
}

// constructor to set timeout on the scope
func (g *Grove) WithTimeout(d time.Duration) *Scope {
	return &Scope{
		grove:    g,
		timeout:  d,
		deadline: time.Time{},
	}
}

// costructor to set deadline on the scope
func (g *Grove) WithDeadline(t time.Time) *Scope {
	return &Scope{
		grove:    g,
		timeout:  0,
		deadline: t,
	}
}

// run the functions under the scope
func (s *Scope) Go(name string, fn func(ctx context.Context) error) {
	g := s.grove

	// check if the grove is closed
	g.mu.Lock()
	if g.closed {
		panic(ErrClosedGrove)
	}

	// add the goroutine to the waitgroup
	g.wg.Add(1)
	g.mu.Unlock()

	var derivedCtx context.Context
	var cancel context.CancelFunc

	// set the appropriate context (each goroutine will have
	// their own derieved context

	if !s.deadline.IsZero() {
		derivedCtx, cancel = context.WithDeadline(g.ctx, s.deadline)
		defer cancel()
	} else {
		derivedCtx, cancel = context.WithTimeout(g.ctx, s.timeout)
		defer cancel()
	}

	// run the go function under the derived contexts
	// we do not cancel the scope context when there is
	// an error within a deadline or timeout scope
	go func() {
		defer g.wg.Done()
		err := internal.CapturePanic(func() error { return fn(derivedCtx) })

		if err != nil {
			g.mu.Lock()
			g.errs = append(g.errs, err)
			g.mu.Unlock()
		}
	}()
}
