package shared

import (
	"context"
	"time"
)

const (
	DefaultTimeout       = 90 * time.Second
	DefaultUploadTimeout = 5 * time.Minute
)

func ContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return context.WithTimeout(parent, timeout)
}

func ContextWithUploadTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultUploadTimeout
	}
	return context.WithTimeout(parent, timeout)
}
