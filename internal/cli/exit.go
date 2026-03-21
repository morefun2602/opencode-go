package cli

import (
	"errors"
	"fmt"
	"strings"

)

type codeErr struct {
	code int
	err  error
}

func (e codeErr) Error() string { return e.err.Error() }

func (e codeErr) Unwrap() error { return e.err }

func withCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return codeErr{code: code, err: err}
}

func exitFromErr(err error) int {
	if err == nil {
		return 0
	}
	var ce codeErr
	if errors.As(err, &ce) {
		return ce.code
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand") {
		return 2
	}
	if strings.Contains(msg, "required flag") || strings.Contains(msg, "invalid argument") {
		return 2
	}
	if strings.Contains(msg, "accepts ") && strings.Contains(msg, "arg") {
		return 2
	}
	if strings.Contains(msg, "unknown command") {
		return 2
	}
	return 1
}

func usageErr(msg string) error {
	return codeErr{code: 2, err: fmt.Errorf("%s", msg)}
}
