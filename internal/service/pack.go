package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var maxPackSearchKeywordRunes = 50

// PackService defines pack behavior.
type PackService interface {
	CreatePack(ctx context.Context, input request.CreatePackInput) (*domain.Pack, error)
	ListPacks(ctx context.Context, input request.ListPacksInput) ([]domain.Pack, error)
	GetPack(ctx context.Context, packID string) (*domain.Pack, error)
	UpdatePack(ctx context.Context, packID string, input request.UpdatePackInput) (*domain.Pack, error)
	DeletePack(ctx context.Context, packID string) error
}

type packService struct {
	repo repository.PackRepository
}

// NewPackService creates a pack service.
func NewPackService(repo repository.PackRepository) PackService {
	return &packService{repo: repo}
}

// CreatePack creates a new pack.
func (s *packService) CreatePack(ctx context.Context, input request.CreatePackInput) (*domain.Pack, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	// TODO: Read and verify pack owner from auth context after user accounts are implemented.
	userID, err := parseOptionalObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	items, err := parseOptionalObjectIDs(input.Items)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	now := time.Now().UTC()
	pack := &domain.Pack{
		ID:          bson.NewObjectID(),
		UserID:      userID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Items:       items,
		Status:      domain.PackStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, pack); err != nil {
		return nil, fmt.Errorf("create pack failed: %w", err)
	}

	return pack, nil
}

// ListPacks lists all packs or searches packs by keyword.
func (s *packService) ListPacks(ctx context.Context, input request.ListPacksInput) ([]domain.Pack, error) {
	var (
		packs []domain.Pack
		err   error
	)
	if input.HasQ {
		packs, err = s.searchPacksByKeyword(ctx, input.UserID, input.Q)
	} else {
		packs, err = s.listPacks(ctx, input.UserID)
	}
	if err != nil {
		return nil, fmt.Errorf("list packs failed: %w", err)
	}

	return packs, nil
}

func (s *packService) listPacks(ctx context.Context, userID string) ([]domain.Pack, error) {
	// TODO: Filter by authenticated user after user accounts/auth are implemented.
	if strings.TrimSpace(userID) == "" {
		return s.repo.ListAll(ctx)
	}

	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.ListByUserID(ctx, objectID)
}

func (s *packService) searchPacksByKeyword(ctx context.Context, userID string, keyword string) ([]domain.Pack, error) {
	// TODO: Filter by authenticated user after user accounts/auth are implemented.
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("pack search keyword is required")
	}
	if utf8.RuneCountInString(keyword) > maxPackSearchKeywordRunes {
		return nil, fmt.Errorf("pack search keyword is too long")
	}

	if strings.TrimSpace(userID) == "" {
		return s.repo.SearchByKeyword(ctx, keyword)
	}

	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.SearchByKeywordAndUserID(ctx, objectID, keyword)
}

// GetPack gets a single pack by ID.
func (s *packService) GetPack(ctx context.Context, packID string) (*domain.Pack, error) {
	objectID, err := parseObjectID(packID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	pack, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("pack not found")
		}
		return nil, fmt.Errorf("get pack failed: %w", err)
	}
	if pack.Status == domain.PackStatusDeleted {
		return nil, fmt.Errorf("pack not found")
	}

	return pack, nil
}

// UpdatePack updates an existing pack.
func (s *packService) UpdatePack(ctx context.Context, packID string, input request.UpdatePackInput) (*domain.Pack, error) {
	objectID, err := parseObjectID(packID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	pack, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("pack not found")
		}
		return nil, fmt.Errorf("get pack failed: %w", err)
	}
	if pack.Status == domain.PackStatusDeleted {
		return nil, fmt.Errorf("pack not found")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	items, err := parseOptionalObjectIDs(input.Items)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	pack.Name = name
	pack.Description = strings.TrimSpace(input.Description)
	pack.Items = items
	pack.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, pack); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("pack not found")
		}
		return nil, fmt.Errorf("update pack failed: %w", err)
	}

	return pack, nil
}

// DeletePack logically deletes a pack by ID.
func (s *packService) DeletePack(ctx context.Context, packID string) error {
	objectID, err := parseObjectID(packID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}

	if err := s.repo.DeleteByID(ctx, objectID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("pack not found")
		}
		return fmt.Errorf("delete pack failed: %w", err)
	}

	return nil
}

func parseOptionalObjectIDs(values []string) ([]bson.ObjectID, error) {
	if len(values) == 0 {
		return []bson.ObjectID{}, nil
	}

	objectIDs := make([]bson.ObjectID, 0, len(values))
	for _, value := range values {
		objectID, err := parseObjectID(value)
		if err != nil {
			return nil, err
		}
		objectIDs = append(objectIDs, objectID)
	}

	return objectIDs, nil
}
