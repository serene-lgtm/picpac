package service

import (
	"context"
	"fmt"
	"time"

	"pack_mate/internal/repository"
)

const healthReadinessTimeout = 2 * time.Second

// HealthService defines process and dependency health checks.
type HealthService interface {
	Health(ctx context.Context) error
	Ready(ctx context.Context) error
}

type healthService struct {
	repo repository.HealthRepository
}

// NewHealthService creates a health service.
func NewHealthService(repo repository.HealthRepository) HealthService {
	return &healthService{repo: repo}
}

// Health confirms that the application process can serve requests.
func (s *healthService) Health(_ context.Context) error {
	return nil
}

// Ready confirms that required application dependencies are available.
func (s *healthService) Ready(ctx context.Context) error {
	if s.repo == nil {
		return fmt.Errorf("health repository is not configured")
	}
	checkCtx, cancel := context.WithTimeout(ctx, healthReadinessTimeout)
	defer cancel()
	if err := s.repo.Ping(checkCtx); err != nil {
		return fmt.Errorf("mongodb is unavailable: %w", err)
	}
	return nil
}
