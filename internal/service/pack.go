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
	GetPack(ctx context.Context, packID string, userID string) (*domain.Pack, error)
	UpdatePack(ctx context.Context, packID string, userID string, input request.UpdatePackInput) (*domain.Pack, error)
	DeletePack(ctx context.Context, packID string, userID string) error
}

type packService struct {
	repo  repository.PackRepository
	items repository.ItemRepository
}

// NewPackService creates a pack service.
func NewPackService(repo repository.PackRepository, items repository.ItemRepository) PackService {
	return &packService{repo: repo, items: items}
}

// CreatePack creates a new pack.
func (s *packService) CreatePack(ctx context.Context, input request.CreatePackInput) (*domain.Pack, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	items, err := parseOptionalObjectIDs(input.Items)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	if err := s.validateOwnedItems(ctx, userID, items); err != nil {
		return nil, err
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
	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.ListByUserID(ctx, objectID)
}

func (s *packService) searchPacksByKeyword(ctx context.Context, userID string, keyword string) ([]domain.Pack, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("pack search keyword is required")
	}
	if utf8.RuneCountInString(keyword) > maxPackSearchKeywordRunes {
		return nil, fmt.Errorf("pack search keyword is too long")
	}

	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.SearchByKeywordAndUserID(ctx, objectID, keyword)
}

// GetPack gets a single pack by ID.
func (s *packService) GetPack(ctx context.Context, packID string, userID string) (*domain.Pack, error) {
	pack, err := s.getOwnedPack(ctx, packID, userID)
	if err != nil {
		return nil, err
	}

	return pack, nil
}

// UpdatePack updates an existing pack.
func (s *packService) UpdatePack(ctx context.Context, packID string, userID string, input request.UpdatePackInput) (*domain.Pack, error) {
	pack, err := s.getOwnedPack(ctx, packID, userID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	items, err := parseOptionalObjectIDs(input.Items)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	if err := s.validateOwnedItems(ctx, pack.UserID, items); err != nil {
		return nil, err
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
func (s *packService) DeletePack(ctx context.Context, packID string, userID string) error {
	objectID, err := parseObjectID(packID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}
	if _, err := s.getOwnedPack(ctx, packID, userID); err != nil {
		return err
	}

	if err := s.repo.DeleteByID(ctx, objectID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("pack not found")
		}
		return fmt.Errorf("delete pack failed: %w", err)
	}

	return nil
}

func (s *packService) getOwnedPack(ctx context.Context, packID string, userID string) (*domain.Pack, error) {
	objectID, err := parseObjectID(packID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	currentUserID, err := parseObjectID(userID)
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
	if pack.Status == domain.PackStatusDeleted || pack.UserID != currentUserID {
		return nil, fmt.Errorf("pack not found")
	}

	return pack, nil
}

func (s *packService) validateOwnedItems(ctx context.Context, userID bson.ObjectID, itemIDs []bson.ObjectID) error {
	for _, itemID := range itemIDs {
		item, err := s.items.GetByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("pack item not found")
			}
			return fmt.Errorf("get item failed: %w", err)
		}
		if item.Status == domain.ItemStatusDeleted || item.UserID != userID {
			return fmt.Errorf("pack item not found")
		}
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
