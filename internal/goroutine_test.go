package internal

import (
	"errors"
	"strings"
	"testing"

	"github.com/ARJ2211/grove"
)

// Tests for Run()

func TestRun_NilReturn(t *testing.T) {
	errChan := make(chan error, 1)

	Run("t1", func() error { return nil }, errChan)

	chanVal := <-errChan
	if chanVal != nil {
		t.Errorf("expected nil, got: %v", chanVal)
	}
}

func TestRun_ErrorReturn(t *testing.T) {
	errChan := make(chan error, 1)

	fnError := errors.New("injected error")
	Run("t2", func() error { return fnError }, errChan)

	chanVal := <-errChan

	if chanVal == nil {
		t.Errorf("expected %v, got: %v", fnError, chanVal)
	}
	if !strings.Contains(chanVal.Error(), "injected error") {
		t.Errorf("expected %v, got: %v", fnError, chanVal)
	}
	if !strings.Contains(chanVal.Error(), "t2") {
		t.Errorf("expected error to have 't2', got %v", chanVal)
	}
	if !errors.Is(chanVal, fnError) {
		t.Errorf("expected %v, got: %v", fnError, chanVal)
	}
}

func TestRun_PanicWithError(t *testing.T) {
	errChan := make(chan error, 1)

	// dummy function that will panic when
	// b is 0
	panicFn := func(a, b int) int {
		if b == 0 {
			err := errors.New("division by 0")
			panic(err)
		}

		return a / b
	}
	Run("t3", func() error { panicFn(4, 0); return nil }, errChan)

	chanVal := <-errChan
	var pe grove.PanicError

	if !errors.As(chanVal, &pe) {
		t.Errorf("expected panic error, got: %v", chanVal)
	}
	if !strings.Contains(pe.Error(), "division by 0") {
		t.Errorf("expected panic error, got: %v", chanVal)
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace")
	}
}

func TestRun_PanicWithNonError(t *testing.T) {
	errChan := make(chan error, 1)

	// dummy function that will panic when
	// b is 0 (panic is a string)
	panicFn := func(a, b int) int {
		if b == 0 {
			panic("division by 0")
		}

		return a / b
	}

	Run("t4", func() error { panicFn(4, 0); return nil }, errChan)

	chanVal := <-errChan
	var pe grove.PanicError

	if !errors.As(chanVal, &pe) {
		t.Errorf("expected panic error, got: %v", chanVal)
	}
	if !strings.Contains(pe.Error(), "division by 0") {
		t.Errorf("expected panic error, got: %v", chanVal)
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace")
	}
}

// Test for CapturePanic()

func TestCapturePanic_NilReturn(t *testing.T) {
	err := CapturePanic(func() error { return nil })

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCapturePanic_ErrorReturn(t *testing.T) {
	err := CapturePanic(func() error { return errors.New("string error") })

	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "string error") {
		t.Errorf("expected string panic, got: %v", err.Error())
	}
}

func TestCapturePanic_PanicWithString(t *testing.T) {
	err := CapturePanic(func() error { panic("string panic") })

	var pe grove.PanicError
	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !(errors.As(err, &pe)) {
		t.Errorf("expected a panic error, got: %v", err)
	}
	if !strings.Contains(pe.Error(), "string panic") {
		t.Errorf("expected string panic, got: %v", pe.Error())
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace")
	}
}

func TestCapturePanic_PanicWithError(t *testing.T) {
	fnError := errors.New("string error panic")
	err := CapturePanic(func() error { panic(fnError) })

	var pe grove.PanicError
	if err == nil {
		t.Errorf("expected error, got: %v", err)
	}
	if !(errors.Is(err, fnError)) {
		t.Errorf("expected to find fnError in the error chain")
	}
	if !(errors.As(err, &pe)) {
		t.Errorf("expected a panic error, got: %v", err)
	}
	if !strings.Contains(pe.Error(), "string error panic") {
		t.Errorf("expected string panic, got: %v", pe.Error())
	}
	if !strings.Contains(pe.Error(), "goroutine") {
		t.Errorf("expected stack trace")
	}
}
