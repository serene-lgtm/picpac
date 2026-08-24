package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const defaultCategoryKey = "other"

// CategoryService defines category behavior.
type CategoryService interface {
	ListCategories(ctx context.Context) ([]domain.Category, error)
	GetCategory(ctx context.Context, categoryID string) (*domain.Category, error)
	GetCategoryByID(ctx context.Context, categoryID bson.ObjectID) (*domain.Category, error)
	ResolveCategoryID(ctx context.Context, categoryID string) (bson.ObjectID, error)
}

type categoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService creates a category service.
func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

// ListCategories lists all configured system categories.
func (s *categoryService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	categories, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories failed: %w", err)
	}
	return categories, nil
}

// GetCategory returns a category by string ID.
func (s *categoryService) GetCategory(ctx context.Context, categoryID string) (*domain.Category, error) {
	objectID, err := parseObjectID(categoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid category")
	}
	return s.GetCategoryByID(ctx, objectID)
}

// GetCategoryByID returns a category by ObjectID.
func (s *categoryService) GetCategoryByID(ctx context.Context, categoryID bson.ObjectID) (*domain.Category, error) {
	if categoryID == bson.NilObjectID {
		return s.getDefaultCategory(ctx)
	}

	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("invalid category")
		}
		return nil, fmt.Errorf("get category failed: %w", err)
	}
	return category, nil
}

// ResolveCategoryID returns a valid category ID, defaulting empty values to other.
func (s *categoryService) ResolveCategoryID(ctx context.Context, categoryID string) (bson.ObjectID, error) {
	categoryID = strings.TrimSpace(categoryID)
	if categoryID == "" {
		category, err := s.getDefaultCategory(ctx)
		if err != nil {
			return bson.NilObjectID, err
		}
		return category.ID, nil
	}

	category, err := s.GetCategory(ctx, categoryID)
	if err != nil {
		return bson.NilObjectID, err
	}
	return category.ID, nil
}

func (s *categoryService) getDefaultCategory(ctx context.Context) (*domain.Category, error) {
	category, err := s.repo.GetByKey(ctx, defaultCategoryKey)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("default category not found")
		}
		return nil, fmt.Errorf("get default category failed: %w", err)
	}
	return category, nil
}
