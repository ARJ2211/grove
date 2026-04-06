package internal

import (
	"fmt"
	"runtime/debug"

	"github.com/ARJ2211/grove"
)

// Run will Run the required fn under a grove.
func Run(name string, fn func() error, errChan chan<- error) {
	go func() {
		err := CapturePanic(fn)

		if err != nil {
			errChan <- fmt.Errorf("task [%s]: %w", name, err)
		} else {
			errChan <- nil
		}
	}()
}

// This will capture the panic and return a panic error
func CapturePanic(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// grove encountered a panic.
			s := debug.Stack()
			err = grove.NewPanicError(r, s)
		}
	}()

	return fn()
}
