package grove

import (
	"fmt"
	"strconv"
	"strings"
)

// Catches all panics and wraps them as an error.
type PanicError struct {
	value any
	stack []byte
}

// Catches a slice of errors when multiple goroutines
// are ran under a grove.
type MultiError struct {
	errors []error
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

// Allows MultiError to adhere to the error contract.
func (me MultiError) Error() string {
	var sb strings.Builder
	errLen := len(me.errors)

	fmt.Fprintf(&sb, "%d errors occurred:\n", errLen)

	for i, e := range me.errors {
		sb.WriteString("   [")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString("]: ")
		sb.WriteString(e.Error())
		sb.WriteString("\n")
	}

	return sb.String()
}

// Adds chain traversal support for errors.Is and errors.As
func (me MultiError) Unwrap() []error {
	return me.errors
}

// Joins the different errors that the different goroutines in
// grove encounter and return them based on the following 3:
// 1. All are nil: Returns nil.
// 2. One is not nil: Returns that specific error.
// 3. 2 or more are not nil: Return MultiError
func Join(errs ...error) error {
	nonNilErrs := []error{}

	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	switch len(nonNilErrs) {
	case 0:
		return nil
	case 1:
		return nonNilErrs[0]
	default:
		me := MultiError{
			errors: nonNilErrs,
		}
		return me
	}
}
