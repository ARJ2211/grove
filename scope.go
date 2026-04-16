package grove

import (
	"context"
	"time"
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
	return nil
}

// costructor to set deadline on the scope
func (g *Grove) WithDeadline(t time.Time) *Scope {
	return nil
}

// run the functions under the scope
func (s *Scope) Go(name string, fn func(ctx context.Context) error) {}
