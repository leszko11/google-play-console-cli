package shared

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type fakePackageReadinessClient struct {
	verifyErr   error
	validateErr error
}

func (f *fakePackageReadinessClient) VerifyPackageAccess(_ context.Context, _ string) error {
	return f.verifyErr
}

func (f *fakePackageReadinessClient) CreateEdit(_ context.Context, _ string) (gpc.EditInfo, error) {
	return gpc.EditInfo{ID: "edit-1"}, nil
}

func (f *fakePackageReadinessClient) ValidateEdit(_ context.Context, _, _ string) error {
	return f.validateErr
}

func (f *fakePackageReadinessClient) DeleteEdit(_ context.Context, _, _ string) error {
	return nil
}

func TestDetectPackageReadiness_Uninitialized(t *testing.T) {
	res, err := DetectPackageReadiness(context.Background(), &fakePackageReadinessClient{
		verifyErr: fmt.Errorf("%w: missing package", gpc.ErrPackageNotFound),
	}, "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != PackageReadinessUninitialized {
		t.Fatalf("unexpected readiness: %+v", res)
	}
}

func TestDetectPackageReadiness_DraftBootstrapRequired(t *testing.T) {
	res, err := DetectPackageReadiness(context.Background(), &fakePackageReadinessClient{
		validateErr: errors.New("androidpublisher api error (400): Only releases with status draft may be created on draft app."),
	}, "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != PackageReadinessDraftBootstrapRequired {
		t.Fatalf("unexpected readiness: %+v", res)
	}
}

func TestDetectPackageReadiness_DeletedEditMapsToDraftBootstrapRequired(t *testing.T) {
	res, err := DetectPackageReadiness(context.Background(), &fakePackageReadinessClient{
		validateErr: errors.New("androidpublisher api error (400): This Edit has been deleted."),
	}, "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != PackageReadinessDraftBootstrapRequired {
		t.Fatalf("unexpected readiness: %+v", res)
	}
}

func TestDetectPackageReadiness_Ready(t *testing.T) {
	res, err := DetectPackageReadiness(context.Background(), &fakePackageReadinessClient{}, "com.example.app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != PackageReadinessReady {
		t.Fatalf("unexpected readiness: %+v", res)
	}
}
