package pipeline

import (
	"errors"
	"fmt"
)

var (
	// ErrSkip is used as a return value when container execution should be
	// skipped at runtime. It is not returned as an error by any function.
	ErrSkip = errors.New("Skipped")

	// ErrCancel is used as a return value when the container execution receives
	// a cancellation signal from the context.
	ErrCancel = errors.New("Cancelled")
)

// An ExitError reports an unsuccessful exit.
type ExitError struct {
	Name string
	Code int
}

// Error returns the error message in string format.
func (e *ExitError) Error() string {
	return fmt.Sprintf("%s : exit code %d", e.Name, e.Code)
}

// An OomError reports the process received an OOMKill from the kernel.
type OomError struct {
	Name string
	Code int
}

// Error reteurns the error message in string format.
func (e *OomError) Error() string {
	return fmt.Sprintf("%s : received oom kill", e.Name)
}
