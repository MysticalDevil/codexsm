package cli

import "fmt"

// ExitError carries a user-facing error plus an intended process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *ExitError) ExitCode() int {
	if e == nil || e.Code <= 0 {
		return 1
	}
	return e.Code
}

// WithExitCode wraps an error with a process exit code for main() handling.
func WithExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: code, Err: fmt.Errorf("%w", err)}
}
