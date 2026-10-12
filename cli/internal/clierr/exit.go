package clierr

import "fmt"

// ExitError is a soft failure: message already printed (or empty); process should
// exit with Code without inventing extra stderr chrome.
type ExitError struct {
	Code    int
	Message string // optional; if non-empty, caller may print before returning
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("exit %d", e.Code)
}

// Exit returns an ExitError with the given code (message optional).
func Exit(code int, message string) error {
	return &ExitError{Code: code, Message: message}
}

// AsExit extracts ExitError from err.
func AsExit(err error) (*ExitError, bool) {
	if err == nil {
		return nil, false
	}
	e, ok := err.(*ExitError)
	return e, ok
}
