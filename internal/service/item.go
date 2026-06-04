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
	ListItems(ctx context.Context, input request.ListItemsInput) ([]domain.Item, error)
	GetItem(ctx context.Context, itemID string) (*domain.Item, error)
	UpdateItem(ctx context.Context, itemID string, input request.UpdateItemInput) (*domain.Item, error)
	DeleteItem(ctx context.Context, itemID string) error
}

type itemService struct {
	repo     repository.ItemRepository
	uploader UploadService
}

// NewItemService creates an item service.
func NewItemService(repo repository.ItemRepository, uploader UploadService) ItemService {
	return &itemService{
		repo:     repo,
		uploader: uploader,
	}
}

// CreateItem creates a new item.
func (s *itemService) CreateItem(ctx context.Context, input request.CreateItemInput) (*domain.Item, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("item name is required")
	}

	// TODO: Verify item ownership after user accounts/auth are implemented.
	userID, err := parseOptionalObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	now := time.Now().UTC()
	item := &domain.Item{
		ID:                 bson.NewObjectID(),
		UserID:             userID,
		Name:               name,
		Description:        strings.TrimSpace(input.Description),
		SourceImageURL:     "",
		ImageThumbnailURL:  "",
		AIRenderedImageURL: "",
		Status:             domain.ItemStatusCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if input.File != nil {
		imageURL, err := s.uploadItemImage(ctx, item.ID, input.FileName, input.File)
		if err != nil {
			return nil, err
		}
		item.SourceImageURL = imageURL
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create item failed: %w", err)
	}

	return item, nil
}

// ListItems lists all items or searches items by keyword.
func (s *itemService) ListItems(ctx context.Context, input request.ListItemsInput) ([]domain.Item, error) {
	var (
		items []domain.Item
		err   error
	)
	if input.HasQ {
		items, err = s.searchItemsByKeyword(ctx, input.UserID, input.Q)
	} else {
		items, err = s.listItems(ctx, input.UserID)
	}
	if err != nil {
		return nil, fmt.Errorf("list items failed: %w", err)
	}

	return items, nil
}

func (s *itemService) listItems(ctx context.Context, userID string) ([]domain.Item, error) {
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

func (s *itemService) searchItemsByKeyword(ctx context.Context, userID string, keyword string) ([]domain.Item, error) {
	// TODO: Filter by authenticated user after user accounts/auth are implemented.
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("item search keyword is required")
	}
	if utf8.RuneCountInString(keyword) > maxItemSearchKeywordRunes {
		return nil, fmt.Errorf("item search keyword is too long")
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

// GetItem gets a single item by ID.
func (s *itemService) GetItem(ctx context.Context, itemID string) (*domain.Item, error) {
	objectID, err := parseObjectID(itemID)
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
	if item.Status == domain.ItemStatusDeleted {
		return nil, fmt.Errorf("item not found")
	}

	return item, nil
}

// UpdateItem updates an existing item.
func (s *itemService) UpdateItem(ctx context.Context, itemID string, input request.UpdateItemInput) (*domain.Item, error) {
	objectID, err := parseObjectID(itemID)
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
	if item.Status == domain.ItemStatusDeleted {
		return nil, fmt.Errorf("item not found")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("item name is required")
	}

	item.Name = name
	item.Description = strings.TrimSpace(input.Description)
	item.UpdatedAt = time.Now().UTC()

	if input.File != nil {
		imageURL, err := s.uploadItemImage(ctx, item.ID, input.FileName, input.File)
		if err != nil {
			return nil, err
		}
		item.SourceImageURL = imageURL
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
func (s *itemService) DeleteItem(ctx context.Context, itemID string) error {
	objectID, err := parseObjectID(itemID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}

	if err := s.repo.DeleteByID(ctx, objectID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("item not found")
		}
		return fmt.Errorf("delete item failed: %w", err)
	}

	return nil
}

func (s *itemService) uploadItemImage(ctx context.Context, itemID bson.ObjectID, fileName string, file io.ReadSeeker) (string, error) {
	body, contentType, err := readUpload(file)
	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	objectKey := buildItemObjectKey(itemID, fileName, contentType)
	imageURL, err := s.uploader.Upload(ctx, objectKey, contentType, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("upload item image failed: %w", err)
	}

	return imageURL, nil
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
