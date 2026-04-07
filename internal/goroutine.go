package internal

import (
	"fmt"
	"runtime/debug"
)

// Catches all panics and wraps them as an error.
type PanicError struct {
	value any
	stack []byte
}

// Creates a new panic error
func NewPanicError(v any, s []byte) PanicError {
	return PanicError{
		value: v,
		stack: s,
	}
}

// Allows PanicError to adhere to the error contract.
func (e PanicError) Error() string {
	strVal := fmt.Sprintf("%v", e.value)
	strStack := string(e.stack)

	msg := fmt.Sprintf("panic: %s \nStack trace: \n%s", strVal, strStack)
	return msg
}

// Adds chain traversal support for errors.Is and errors.As
func (e PanicError) Unwrap() error {
	err, ok := e.value.(error)

	if !ok {
		return nil
	}
	return err
}

// Run will Run the required fn under a grove.
func Run(name string, fn func() error, errChan chan<- error) {
	go func() {
		err := CapturePanic(fn)

		if err != nil {
			errChan <- fmt.Errorf("task [%s] - %w", name, err)
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
			err = NewPanicError(r, s)
		}
	}()

	return fn()
}
