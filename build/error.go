package build

import (
	"errors"
	"fmt"
)

var (
	// ErrSkip is used as a return value when container execution should be
	// skipped at runtime. It is not returned as an error by any function.
	ErrSkip = errors.New("Skip")

	// ErrTerm is used as a return value when the runner should terminate
	// execution and exit. It is not returned as an error by any function.
	ErrTerm = errors.New("Terminate")
)

// An ExitError reports an unsuccessful exit.
type ExitError struct {
	Name string
	Code int
}

// Error reteurns the error message in string format.
func (e *ExitError) Error() string {
	return fmt.Sprintf("%s : exit code %d", e.Name, e.Code)
}

// An OomError reports the process received an OOMKill from the kernel.
type OomError struct {
	Name string
}

// Error reteurns the error message in string format.
func (e *OomError) Error() string {
	return fmt.Sprintf("%s : received oom kill", e.Name)
}
