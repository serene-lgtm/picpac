package service

import (
	"context"
	"errors"
	"testing"
)

type fakeHealthRepository struct {
	err error
}

// Ping returns the configured dependency result for service tests.
func (r *fakeHealthRepository) Ping(context.Context) error {
	return r.err
}

// TestHealthServiceChecksMongoDB verifies readiness dependency handling.
func TestHealthServiceChecksMongoDB(t *testing.T) {
	t.Parallel()

	svc := NewHealthService(&fakeHealthRepository{})
	if err := svc.Health(context.Background()); err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if err := svc.Ready(context.Background()); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}

	wantErr := errors.New("mongo unavailable")
	svc = NewHealthService(&fakeHealthRepository{err: wantErr})
	if err := svc.Ready(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected readiness error to wrap MongoDB error, got %v", err)
	}
}

// TestHealthServiceRejectsMissingRepository verifies misconfiguration handling.
func TestHealthServiceRejectsMissingRepository(t *testing.T) {
	t.Parallel()

	svc := NewHealthService(nil)
	if err := svc.Ready(context.Background()); err == nil {
		t.Fatal("expected readiness error when repository is missing")
	}
}
