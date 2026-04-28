package grove

import (
	"context"
	"fmt"
	"time"

	"github.com/ARJ2211/grove/internal"
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

// the result of a task that will be propogated
// back to the supervisor over a channel
type taskResult struct {
	task task  // the task that failed / succeeded
	err  error // the error propogated
}

// supervisor strategy for task restarts
type Strategy int

const (
	RestartOnFailure Strategy = iota
	OneForOne                 // when one task fails, only restart that task
	OneForAll                 // when one task fails, restart ALL the tasks
)

// supervisor registry to track the tasks in the
// supervisor grove
type supervisorRegistry struct {
	tasks []task // slice of tasks that it needs
}

// used to append the tasks or register them in the registry
func (reg *supervisorRegistry) Go(name string, fn func(ctx context.Context) error) {
	task := task{
		name:       name,
		fn:         fn,
		retries:    0,
		maxRetries: -1, // unlimited
		delay:      30 * time.Millisecond,
	}

	// append the task into the registry
	reg.tasks = append(reg.tasks, task)
}

// start a supervisor to track and maintain
// the goroutines under it.
func Supervise(
	ctx context.Context,
	strategy Strategy,
	fn func(*supervisorRegistry) error,
) error {
	// create the empty registry
	reg := supervisorRegistry{}

	// run the function and collect errors
	// and register the tasks that need to be run
	var errs []error
	if err := fn(&reg); err != nil {
		errs = append(errs, err)
	}

	// create the resChan
	resChan := make(chan taskResult, len(reg.tasks))

	// run the tasks
	for _, t := range reg.tasks {
		launchTask(ctx, t, resChan)
	}

	// count of all running tasks
	running := len(reg.tasks)

supervisorLoop:
	for {
		select {
		case res := <-resChan:
			// handle res
			running -= 1
			if res.err != nil {
				errs = append(errs, res.err)
			}

			if running == 0 {
				break supervisorLoop
			}
		case <-ctx.Done():
			for {
				<-resChan
				running -= 1
				if running == 0 {
					break
				}
			}

			// append a context cancelled error
			errs = append(errs, ctx.Err())
		}
	}

	return Join(errs...)
}

// launch a task into the supervisor
func launchTask(
	ctx context.Context,
	task task,
	resChan chan<- taskResult,
) {
	go func() {
		// run the function under a Capture Panic wrapper
		err := internal.CapturePanic(func() error { return task.fn(ctx) })

		var tr taskResult
		if err != nil {
			tr = taskResult{
				task: task,
				err:  fmt.Errorf("task [%s] - %w", task.name, err),
			}
		} else {
			tr = taskResult{
				task: task,
				err:  nil,
			}
		}

		// send the task result over the channel
		resChan <- tr
	}()
}
