package grove

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ARJ2211/grove/internal"
)

// PanicError.Error() tests

func TestPanicError_Error_WithString(t *testing.T) {
	e := internal.NewPanicError(
		"index out of range",
		[]byte("goroutine 1 [running]"),
	)
	msg := e.Error()

	if !strings.Contains(msg, "index out of range") {
		t.Errorf("expected message to contain panic value, got: %s", msg)
	}
	if !strings.Contains(msg, "goroutine 1 [running]") {
		t.Errorf("expected message to contain stack trace, got: %s", msg)
	}
}

func TestPanicError_Error_WithInteger(t *testing.T) {
	e := internal.NewPanicError(42, []byte("goroutine 1 [running]"))
	msg := e.Error()

	if !strings.Contains(msg, "42") {
		t.Errorf("expected message to contain panic value 42, got: %s", msg)
	}
}

func TestPanicError_Error_WithStruct(t *testing.T) {
	type myStruct struct{ Code int }
	e := internal.NewPanicError(
		myStruct{Code: 500},
		[]byte("goroutine 1 [running]"),
	)
	msg := e.Error()

	if !strings.Contains(msg, "500") {
		t.Errorf("expected message to contain struct field value, got: %s", msg)
	}
}

func TestPanicError_Error_WithError(t *testing.T) {
	inner := errors.New("database connection lost")
	e := internal.NewPanicError(inner, []byte("goroutine 1 [running]"))
	msg := e.Error()

	if !strings.Contains(msg, "database connection lost") {
		t.Errorf("expected message to contain inner error message, got: %s", msg)
	}
}

// PanicError.Unwrap() tests

func TestPanicError_Unwrap_WhenValueIsError(t *testing.T) {
	inner := errors.New("something specific")
	e := internal.NewPanicError(inner, []byte("trace"))

	unwrapped := e.Unwrap()
	if unwrapped == nil {
		t.Fatal("expected non-nil error from Unwrap, got nil")
	}
	if !errors.Is(unwrapped, inner) {
		t.Errorf("expected unwrapped error to be inner, got: %v", unwrapped)
	}
}

func TestPanicError_Unwrap_WhenValueIsNotError(t *testing.T) {
	e := internal.NewPanicError("just a string", []byte("trace"))

	unwrapped := e.Unwrap()
	if unwrapped != nil {
		t.Errorf("expected nil from Unwrap for non-error value, got: %v", unwrapped)
	}
}

func TestPanicError_Unwrap_ErrorsAs(t *testing.T) {
	inner := fmt.Errorf("wrapped: %w", errors.New("root cause"))
	e := internal.NewPanicError(inner, []byte("trace"))

	var target internal.PanicError
	if !errors.As(e, &target) {
		t.Error("expected errors.As to find PanicError")
	}
}

// Join tests

func TestJoin_AllNil(t *testing.T) {
	err := Join(nil, nil, nil)
	if err != nil {
		t.Errorf("expected nil when all errors are nil, got: %v", err)
	}
}

func TestJoin_OneNonNil(t *testing.T) {
	sentinel := errors.New("only error")
	err := Join(nil, sentinel, nil)

	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	if err != sentinel {
		t.Errorf("expected the exact sentinel error to be returned, got: %v", err)
	}
}

func TestJoin_MultipleNonNil(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")
	err := Join(err1, err2)

	if err == nil {
		t.Fatal("expected non-nil error, got nil")
	}
	var me MultiError
	if !errors.As(err, &me) {
		t.Errorf("expected a MultiError, got: %T", err)
	}
}

func TestJoin_ErrorsIs_TraversesThroughMultiError(t *testing.T) {
	sentinel := errors.New("something specific")
	err := Join(sentinel, errors.New("other error"))

	if !errors.Is(err, sentinel) {
		t.Errorf("expected errors.Is to find sentinel inside MultiError")
	}
}

// MultiError.Error() tests

func TestMultiError_Error_ContainsAllErrors(t *testing.T) {
	err1 := errors.New("first error")
	err2 := errors.New("second error")
	me := MultiError{errors: []error{err1, err2}}
	msg := me.Error()

	if !strings.Contains(msg, "first error") {
		t.Errorf("expected output to contain first error, got: %s", msg)
	}
	if !strings.Contains(msg, "second error") {
		t.Errorf("expected output to contain second error, got: %s", msg)
	}
}

func TestMultiError_Error_ShowsCorrectCount(t *testing.T) {
	me := MultiError{errors: []error{errors.New("a"), errors.New("b"), errors.New("c")}}
	msg := me.Error()

	if !strings.Contains(msg, "3") {
		t.Errorf("expected output to contain error count, got: %s", msg)
	}
}

func TestMultiError_Error_OnlyNonNilErrors(t *testing.T) {
	err := Join(errors.New("real error"), nil)

	if _, ok := err.(MultiError); ok {
		t.Error("expected single error, not MultiError, when only one is non-nil")
	}
}
