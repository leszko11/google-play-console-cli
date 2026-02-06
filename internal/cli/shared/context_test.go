package shared

import (
	"context"
	"testing"
	"time"
)

func TestContextWithTimeout_Default(t *testing.T) {
	ctx, cancel := ContextWithTimeout(context.Background(), 0)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Fatalf("expected positive timeout, got %v", remaining)
	}
}

func TestContextWithUploadTimeout_Default(t *testing.T) {
	ctx, cancel := ContextWithUploadTimeout(context.Background(), 0)
	defer cancel()

	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("expected deadline to be set")
	}
}
