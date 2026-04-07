package grove

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ARJ2211/grove/internal"
)

// Alias for the public API so users can do errors.Is
type PanicError = internal.PanicError

// Catches a slice of errors when multiple goroutines
// are ran under a grove.
type MultiError struct {
	errors []error
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
