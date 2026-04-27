package grove

import (
	"context"
	"time"
)

// task structure for the supervisor
// to know current state of a task
type task struct {
	fn         func(ctx context.Context) error // the goroutine to be run
	name       string                          // the name of the user defined task
	retries    int                             // the number of current retries
	maxRetries int                             // the max number of retries
	delay      time.Duration                   // the duration of task
}

// supervisor strategy for task restarts
type Strategy int

const (
	RestartOnFailure Strategy = iota
	OneForOne                 // when one task fails, only restart that task
	OneForAll                 // when one task fails, restart ALL the tasks
)
