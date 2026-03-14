package shared

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type UsageError struct {
	Message string
	Cause   error
}

func (e UsageError) Error() string {
	return e.Message
}

func (e UsageError) Unwrap() error {
	return e.Cause
}

func NewUsageError(message string) error {
	return UsageError{Message: message}
}

func UsageErrorf(format string, args ...any) error {
	return UsageError{Message: fmt.Sprintf(format, args...)}
}

func WrapUsageError(err error) error {
	if err == nil {
		return nil
	}
	return UsageError{Message: err.Error(), Cause: err}
}

func IsUsageError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, flag.ErrHelp) {
		return true
	}
	var usageErr UsageError
	if errors.As(err, &usageErr) {
		return true
	}
	return false
}

func IsLikelyUsageError(err error) bool {
	if IsUsageError(err) {
		return true
	}
	if err == nil {
		return false
	}

	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "flag provided but not defined") {
		return true
	}
	if strings.HasPrefix(msg, "invalid value") {
		return true
	}
	if strings.HasPrefix(msg, "unsupported output format") {
		return true
	}
	if strings.HasPrefix(msg, "--") && strings.Contains(msg, "required") {
		return true
	}
	return false
}

func DefaultUsageFunc(c *ffcli.Command) string {
	return ffcli.DefaultUsageFunc(c)
}
