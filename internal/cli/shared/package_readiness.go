package shared

import (
	"context"
	"errors"
	"strings"

	"github.com/leszko11/google-play-console-cli/internal/gpc"
)

type PackageReadiness string

const (
	PackageReadinessUnknown                PackageReadiness = ""
	PackageReadinessUninitialized          PackageReadiness = "uninitialized"
	PackageReadinessDraftBootstrapRequired PackageReadiness = "draft_bootstrap_required"
	PackageReadinessReady                  PackageReadiness = "ready"
)

type PackageReadinessClient interface {
	VerifyPackageAccess(ctx context.Context, packageName string) error
	CreateEdit(ctx context.Context, packageName string) (gpc.EditInfo, error)
	ValidateEdit(ctx context.Context, packageName, editID string) error
	DeleteEdit(ctx context.Context, packageName, editID string) error
}

type PackageReadinessResult struct {
	Status    PackageReadiness `json:"status"`
	Code      string           `json:"code,omitempty"`
	Detail    string           `json:"detail,omitempty"`
	NextStep  string           `json:"nextStep,omitempty"`
	Warning   string           `json:"warning,omitempty"`
	AccessOK  bool             `json:"accessOk,omitempty"`
	CleanupOK bool             `json:"cleanupOk,omitempty"`
}

func DetectPackageReadiness(ctx context.Context, client PackageReadinessClient, packageName string) (PackageReadinessResult, error) {
	if err := client.VerifyPackageAccess(ctx, packageName); err != nil {
		if errors.Is(err, gpc.ErrPackageNotFound) {
			return PackageReadinessResult{
				Status:   PackageReadinessUninitialized,
				Code:     "package_uninitialized",
				Detail:   "package is not initialized in Google Play yet",
				NextStep: "Upload the first APK or AAB once in Play Console, then rerun `gpc release init --package-name <package>`.",
			}, nil
		}
		return PackageReadinessResult{}, err
	}

	result := PackageReadinessResult{AccessOK: true}
	edit, err := client.CreateEdit(ctx, packageName)
	if err != nil {
		return PackageReadinessResult{}, err
	}
	defer func() {
		if cleanupErr := client.DeleteEdit(ctx, packageName, edit.ID); cleanupErr == nil {
			result.CleanupOK = true
		}
	}()

	if err := client.ValidateEdit(ctx, packageName, edit.ID); err != nil {
		if IsDraftAppError(err) {
			return PackageReadinessResult{
				Status:    PackageReadinessDraftBootstrapRequired,
				Code:      "draft_app_requires_bootstrap_release",
				Detail:    "package is still in Play's draft bootstrap state",
				NextStep:  "Run `gpc release init --package-name <package> --dir ./play` to generate the bootstrap workspace and next release command.",
				Warning:   err.Error(),
				AccessOK:  true,
				CleanupOK: result.CleanupOK,
			}, nil
		}
		return PackageReadinessResult{}, err
	}

	return PackageReadinessResult{
		Status:    PackageReadinessReady,
		Code:      "package_ready",
		Detail:    "package access and metadata edits are ready",
		AccessOK:  true,
		CleanupOK: result.CleanupOK,
	}, nil
}

func IsDraftAppError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "only releases with status draft may be created on draft app")
}
