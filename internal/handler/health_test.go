package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeHealthHandlerService struct {
	healthErr error
	readyErr  error
}

// Health returns the configured liveness result for handler tests.
func (s *fakeHealthHandlerService) Health(context.Context) error { return s.healthErr }

// Ready returns the configured readiness result for handler tests.
func (s *fakeHealthHandlerService) Ready(context.Context) error { return s.readyErr }

var _ service.HealthService = (*fakeHealthHandlerService)(nil)

// TestHealthHandlerReturnsExpectedStatuses verifies health endpoint status codes.
func TestHealthHandlerReturnsExpectedStatuses(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		path       string
		serviceErr error
		wantCode   int
	}{
		{name: "health ok", path: "/health", wantCode: http.StatusOK},
		{name: "health unavailable", path: "/health", serviceErr: errors.New("failed"), wantCode: http.StatusServiceUnavailable},
		{name: "ready ok", path: "/ready", wantCode: http.StatusOK},
		{name: "ready unavailable", path: "/ready", serviceErr: errors.New("failed"), wantCode: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &fakeHealthHandlerService{}
			if test.path == "/health" {
				svc.healthErr = test.serviceErr
			} else {
				svc.readyErr = test.serviceErr
			}
			handler := NewHealthHandler(svc)
			router := gin.New()
			router.GET("/health", handler.Health)
			router.GET("/ready", handler.Ready)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != test.wantCode {
				t.Fatalf("expected status %d, got %d", test.wantCode, recorder.Code)
			}
		})
	}
}
