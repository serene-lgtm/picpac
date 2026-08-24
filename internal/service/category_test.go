package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeCategoryRepository struct {
	categories []domain.Category
	got        *domain.Category
	other      *domain.Category
	err        error
}

func (r *fakeCategoryRepository) ListAll(_ context.Context) ([]domain.Category, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.categories, nil
}

func (r *fakeCategoryRepository) GetByID(_ context.Context, categoryID bson.ObjectID) (*domain.Category, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil || r.got.ID != categoryID {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeCategoryRepository) GetByKey(_ context.Context, key string) (*domain.Category, error) {
	if r.err != nil {
		return nil, r.err
	}
	if key == defaultCategoryKey && r.other != nil {
		return r.other, nil
	}
	return nil, mongo.ErrNoDocuments
}

func (r *fakeCategoryRepository) UpsertByKey(_ context.Context, _ *domain.Category) error {
	return r.err
}

func TestListCategoriesReturnsCategories(t *testing.T) {
	t.Parallel()

	category := domain.Category{ID: bson.NewObjectID(), Key: "document", Name: "证件"}
	svc := NewCategoryService(&fakeCategoryRepository{categories: []domain.Category{category}})

	categories, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	if len(categories) != 1 || categories[0].Key != "document" {
		t.Fatalf("unexpected categories: %+v", categories)
	}
}

func TestResolveCategoryIDDefaultsEmptyToOther(t *testing.T) {
	t.Parallel()

	other := &domain.Category{ID: bson.NewObjectID(), Key: defaultCategoryKey, Name: "其他"}
	svc := NewCategoryService(&fakeCategoryRepository{other: other})

	categoryID, err := svc.ResolveCategoryID(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveCategoryID returned error: %v", err)
	}
	if categoryID != other.ID {
		t.Fatalf("expected default category %s, got %s", other.ID.Hex(), categoryID.Hex())
	}
}

func TestResolveCategoryIDRejectsUnknownCategory(t *testing.T) {
	t.Parallel()

	svc := NewCategoryService(&fakeCategoryRepository{})

	_, err := svc.ResolveCategoryID(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "invalid category") {
		t.Fatalf("expected invalid category error, got %v", err)
	}
}

func TestListCategoriesWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewCategoryService(&fakeCategoryRepository{err: errors.New("db down")})

	_, err := svc.ListCategories(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list categories failed") {
		t.Fatalf("expected list categories failure, got %v", err)
	}
}
