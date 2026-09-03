package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var maxItemSearchKeywordRunes = 50

// ItemService defines item CRUD behavior.
type ItemService interface {
	CreateItem(ctx context.Context, input request.CreateItemInput) (*domain.Item, error)
	CreateItemsBatch(ctx context.Context, input request.BatchCreateItemsInput) ([]domain.Item, error)
	ListItems(ctx context.Context, input request.ListItemsInput) ([]domain.Item, error)
	GetItem(ctx context.Context, itemID string, userID string) (*domain.Item, error)
	UpdateItem(ctx context.Context, itemID string, userID string, input request.UpdateItemInput) (*domain.Item, error)
	DeleteItem(ctx context.Context, itemID string, userID string) error
}

type itemService struct {
	repo       repository.ItemRepository
	uploader   UploadService
	categories CategoryService
}

// NewItemService creates an item service.
func NewItemService(repo repository.ItemRepository, uploader UploadService, categories CategoryService) ItemService {
	return &itemService{
		repo:       repo,
		uploader:   uploader,
		categories: categories,
	}
}

const maxBatchCreateItemsCount = 50

// CreateItem creates a new item.
func (s *itemService) CreateItem(ctx context.Context, input request.CreateItemInput) (*domain.Item, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("item name is required")
	}

	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	categoryID, err := s.resolveCategoryID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	item := &domain.Item{
		ID:                       bson.NewObjectID(),
		UserID:                   userID,
		CategoryID:               categoryID,
		Name:                     name,
		Description:              strings.TrimSpace(input.Description),
		SourceImageObjectKey:     "",
		ImageThumbnailObjectKey:  "",
		AIRenderedImageObjectKey: "",
		Status:                   domain.ItemStatusCreated,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if input.File != nil {
		objectKey, err := s.uploadItemImage(ctx, item.ID, input.FileName, input.File)
		if err != nil {
			return nil, err
		}
		item.SourceImageObjectKey = objectKey
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create item failed: %w", err)
	}

	return item, nil
}

// CreateItemsBatch creates multiple items atomically.
func (s *itemService) CreateItemsBatch(ctx context.Context, input request.BatchCreateItemsInput) ([]domain.Item, error) {
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("items are required")
	}
	if len(input.Items) > maxBatchCreateItemsCount {
		return nil, fmt.Errorf("items are too many")
	}

	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	items := make([]domain.Item, 0, len(input.Items))
	now := time.Now().UTC()
	for _, itemInput := range input.Items {
		name := strings.TrimSpace(itemInput.Name)
		if name == "" {
			return nil, fmt.Errorf("item name is required")
		}
		categoryID, err := s.resolveCategoryID(ctx, itemInput.CategoryID)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.Item{
			ID:                       bson.NewObjectID(),
			UserID:                   userID,
			CategoryID:               categoryID,
			Name:                     name,
			Description:              strings.TrimSpace(itemInput.Description),
			SourceImageObjectKey:     "",
			ImageThumbnailObjectKey:  "",
			AIRenderedImageObjectKey: "",
			Status:                   domain.ItemStatusCreated,
			CreatedAt:                now,
			UpdatedAt:                now,
		})
	}

	if err := s.repo.CreateMany(ctx, items); err != nil {
		return nil, fmt.Errorf("create items failed: %w", err)
	}

	return items, nil
}

// ListItems lists all items or searches items by keyword.
func (s *itemService) ListItems(ctx context.Context, input request.ListItemsInput) ([]domain.Item, error) {
	userID, err := parseObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	filter := repository.ItemFilter{UserID: userID}
	if input.HasQ {
		keyword := strings.TrimSpace(input.Q)
		if keyword == "" {
			return nil, fmt.Errorf("item search keyword is required")
		}
		if utf8.RuneCountInString(keyword) > maxItemSearchKeywordRunes {
			return nil, fmt.Errorf("item search keyword is too long")
		}
		filter.Keyword = keyword
		filter.HasKeyword = true
	}
	if input.HasCategoryID {
		category, err := s.getCategory(ctx, input.CategoryID)
		if err != nil {
			return nil, err
		}
		filter.CategoryID = category.ID
		filter.HasCategoryID = true
	}

	items, err := s.repo.ListByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list items failed: %w", err)
	}

	return items, nil
}

// GetItem gets a single item by ID.
func (s *itemService) GetItem(ctx context.Context, itemID string, userID string) (*domain.Item, error) {
	item, err := s.getOwnedItem(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// UpdateItem updates an existing item.
func (s *itemService) UpdateItem(ctx context.Context, itemID string, userID string, input request.UpdateItemInput) (*domain.Item, error) {
	item, err := s.getOwnedItem(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("item name is required")
	}

	item.Name = name
	item.Description = strings.TrimSpace(input.Description)
	if input.HasCategoryID {
		categoryID, err := s.resolveCategoryID(ctx, input.CategoryID)
		if err != nil {
			return nil, err
		}
		item.CategoryID = categoryID
	}
	item.UpdatedAt = time.Now().UTC()

	if input.File != nil {
		objectKey, err := s.uploadItemImage(ctx, item.ID, input.FileName, input.File)
		if err != nil {
			return nil, err
		}
		item.SourceImageObjectKey = objectKey
	}

	if err := s.repo.Update(ctx, item); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("update item failed: %w", err)
	}

	return item, nil
}

// DeleteItem logically deletes a single item by ID.
func (s *itemService) DeleteItem(ctx context.Context, itemID string, userID string) error {
	objectID, err := parseObjectID(itemID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}
	if _, err := s.getOwnedItem(ctx, itemID, userID); err != nil {
		return err
	}

	if err := s.repo.DeleteByID(ctx, objectID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("item not found")
		}
		return fmt.Errorf("delete item failed: %w", err)
	}

	return nil
}

func (s *itemService) resolveCategoryID(ctx context.Context, categoryID string) (bson.ObjectID, error) {
	if s.categories == nil {
		return bson.NilObjectID, nil
	}
	return s.categories.ResolveCategoryID(ctx, categoryID)
}

func (s *itemService) getCategory(ctx context.Context, categoryID string) (*domain.Category, error) {
	if s.categories == nil {
		objectID, err := parseObjectID(categoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category %s: ", categoryID)
		}
		return &domain.Category{ID: objectID}, nil
	}
	return s.categories.GetCategory(ctx, categoryID)
}

func (s *itemService) getOwnedItem(ctx context.Context, itemID string, userID string) (*domain.Item, error) {
	objectID, err := parseObjectID(itemID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	currentUserID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	item, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("get item failed: %w", err)
	}
	if item.Status == domain.ItemStatusDeleted || item.UserID != currentUserID {
		return nil, fmt.Errorf("item not found")
	}

	return item, nil
}

func (s *itemService) uploadItemImage(ctx context.Context, itemID bson.ObjectID, fileName string, file io.ReadSeeker) (string, error) {
	body, contentType, err := readUpload(file)
	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	objectKey := buildItemObjectKey(itemID, fileName, contentType)
	if err := s.uploader.Upload(ctx, objectKey, contentType, bytes.NewReader(body)); err != nil {
		return "", fmt.Errorf("upload item image failed: %w", err)
	}

	return objectKey, nil
}

func parseObjectID(value string) (bson.ObjectID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return bson.ObjectID{}, fmt.Errorf("invalid input")
	}

	objectID, err := bson.ObjectIDFromHex(trimmed)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("invalid input")
	}

	return objectID, nil
}

func parseOptionalObjectID(value string) (bson.ObjectID, error) {
	if strings.TrimSpace(value) == "" {
		return bson.ObjectID{}, nil
	}
	return parseObjectID(value)
}

func readUpload(file io.ReadSeeker) ([]byte, string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("empty file")
	}

	contentType := http.DetectContentType(body)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("invalid content type")
	}

	return body, contentType, nil
}

func buildItemObjectKey(itemID bson.ObjectID, fileName string, contentType string) string {
	ext := extensionForUpload(fileName, contentType)
	return fmt.Sprintf("items/item_%s/source%s", itemID.Hex(), ext)
}

func extensionForUpload(fileName string, contentType string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if ext != "" {
		return ext
	}

	extensions, err := mime.ExtensionsByType(contentType)
	if err == nil && len(extensions) > 0 {
		return strings.ToLower(extensions[0])
	}

	switch contentType {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
