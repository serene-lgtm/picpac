package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeItemRepository struct {
	created       *domain.Item
	listed        []domain.Item
	searched      []domain.Item
	searchKeyword string
	got           *domain.Item
	updated       *domain.Item
	deleted       bson.ObjectID
	err           error
}

func (r *fakeItemRepository) Create(_ context.Context, item *domain.Item) error {
	if r.err != nil {
		return r.err
	}
	r.created = item
	return nil
}

func (r *fakeItemRepository) ListAll(_ context.Context) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeItemRepository) ListByUserID(_ context.Context, _ bson.ObjectID) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeItemRepository) SearchByKeyword(_ context.Context, keyword string) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeItemRepository) SearchByKeywordAndUserID(_ context.Context, _ bson.ObjectID, keyword string) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeItemRepository) GetByID(_ context.Context, _ bson.ObjectID) (*domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeItemRepository) Update(_ context.Context, item *domain.Item) error {
	if r.err != nil {
		return r.err
	}
	r.updated = item
	return nil
}

func (r *fakeItemRepository) DeleteByID(_ context.Context, itemID bson.ObjectID) error {
	if r.err != nil {
		return r.err
	}
	if r.got != nil && r.got.Status == domain.ItemStatusDeleted {
		return mongo.ErrNoDocuments
	}
	r.deleted = itemID
	return nil
}

type fakeUploadService struct {
	url string
	err error
}

func (s *fakeUploadService) Upload(_ context.Context, _ string, _ string, _ io.Reader) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.url, nil
}

func testPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0x0f, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d, 0xb1, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}

func TestCreateItemStoresItemWithoutImage(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakeItemRepository{}
	svc := NewItemService(repo, &fakeUploadService{})

	item, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID:      userID.Hex(),
		Name:        "黑色双肩包",
		Description: "日常出差用",
	})
	if err != nil {
		t.Fatalf("CreateItem returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatalf("expected repository create to be called")
	}
	if item.Name != "黑色双肩包" || item.SourceImageURL != "" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Status != domain.ItemStatusCreated {
		t.Fatalf("expected status=created, got %s", item.Status)
	}
}

func TestCreateItemUploadsImageWhenProvided(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakeItemRepository{}
	uploader := &fakeUploadService{url: "https://cos.example/items/item_1/source.jpg"}
	svc := NewItemService(repo, uploader)

	item, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID:   userID.Hex(),
		Name:     "黑色双肩包",
		File:     bytes.NewReader(testPNGBytes()),
		FileName: "bag.jpg",
	})
	if err != nil {
		t.Fatalf("CreateItem returned error: %v", err)
	}
	if item.SourceImageURL == "" {
		t.Fatalf("expected source image url to be populated")
	}
}

func TestCreateItemRejectsMissingName(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{})

	_, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID: bson.NewObjectID().Hex(),
	})
	if err == nil || !strings.Contains(err.Error(), "item name is required") {
		t.Fatalf("expected item name error, got %v", err)
	}
}

func TestCreateItemAllowsMissingUserID(t *testing.T) {
	t.Parallel()

	repo := &fakeItemRepository{}
	svc := NewItemService(repo, &fakeUploadService{})

	item, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		Name: "黑色双肩包",
	})
	if err != nil {
		t.Fatalf("CreateItem returned error: %v", err)
	}
	if !item.UserID.IsZero() {
		t.Fatalf("expected zero user id, got %s", item.UserID.Hex())
	}
}

func TestListItemsReturnsRepositoryResults(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Item{{UserID: userID, Name: "黑色双肩包"}}
	svc := NewItemService(&fakeItemRepository{listed: expected}, &fakeUploadService{})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{UserID: userID.Hex()})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "黑色双肩包" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListItemsAllowsMissingUserID(t *testing.T) {
	t.Parallel()

	expected := []domain.Item{{Name: "黑色双肩包"}}
	svc := NewItemService(&fakeItemRepository{listed: expected}, &fakeUploadService{})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "黑色双肩包" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListItemsSearchesByChineseKeyword(t *testing.T) {
	t.Parallel()

	expected := []domain.Item{{Name: "手机充电器"}}
	repo := &fakeItemRepository{searched: expected}
	svc := NewItemService(repo, &fakeUploadService{})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		Q:    "  充电  ",
		HasQ: true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "手机充电器" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if repo.searchKeyword != "充电" {
		t.Fatalf("expected trimmed keyword, got %q", repo.searchKeyword)
	}
}

func TestListItemsSearchesByDescriptionKeyword(t *testing.T) {
	t.Parallel()

	expected := []domain.Item{{Name: "转换插头", Description: "支持手机充电器"}}
	repo := &fakeItemRepository{searched: expected}
	svc := NewItemService(repo, &fakeUploadService{})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		Q:    "充电",
		HasQ: true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Description != "支持手机充电器" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListItemsSearchesByUserID(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Item{{UserID: userID, Name: "手机充电器"}}
	svc := NewItemService(&fakeItemRepository{searched: expected}, &fakeUploadService{})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: userID.Hex(),
		Q:      "充电",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "手机充电器" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListItemsRejectsEmptySearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{})

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "item search keyword is required") {
		t.Fatalf("expected keyword required error, got %v", err)
	}
}

func TestListItemsRejectsTooLongSearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{})
	keyword := strings.Repeat("行", maxItemSearchKeywordRunes+1)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{Q: keyword, HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "item search keyword is too long") {
		t.Fatalf("expected keyword too long error, got %v", err)
	}
}

func TestListItemsRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{})

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{UserID: "bad-user-id"})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListItemsWrapsSearchRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{err: errors.New("db down")}, &fakeUploadService{})

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{Q: "充电", HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "list items failed") {
		t.Fatalf("expected list items failure, got %v", err)
	}
}

func TestGetItemMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{})

	_, err := svc.GetItem(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestGetItemHidesDeletedItem(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			Status: domain.ItemStatusDeleted,
		},
	}, &fakeUploadService{})

	_, err := svc.GetItem(context.Background(), itemID.Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestUpdateItemReplacesFieldsAndOptionalImage(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	repo := &fakeItemRepository{
		got: &domain.Item{
			ID:                itemID,
			UserID:            bson.NewObjectID(),
			Name:              "旧名称",
			Description:       "old",
			SourceImageURL:    "old-url",
			ImageThumbnailURL: "",
			Status:            domain.ItemStatusCreated,
			UpdatedAt:         time.Now().Add(-time.Hour),
		},
	}
	uploader := &fakeUploadService{url: "https://cos.example/items/item_1/source.png"}
	svc := NewItemService(repo, uploader)

	item, err := svc.UpdateItem(context.Background(), itemID.Hex(), request.UpdateItemInput{
		Name:     "新名称",
		File:     bytes.NewReader(testPNGBytes()),
		FileName: "bag.png",
	})
	if err != nil {
		t.Fatalf("UpdateItem returned error: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "新名称" {
		t.Fatalf("expected repository update to be called")
	}
	if item.SourceImageURL != uploader.url {
		t.Fatalf("expected image url to be replaced, got %s", item.SourceImageURL)
	}
}

func TestUpdateItemRejectsDeletedItem(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			Status: domain.ItemStatusDeleted,
		},
	}, &fakeUploadService{})

	_, err := svc.UpdateItem(context.Background(), itemID.Hex(), request.UpdateItemInput{
		Name: "新名称",
	})
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestDeleteItemWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{err: errors.New("db down")}, &fakeUploadService{})

	err := svc.DeleteItem(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "delete item failed") {
		t.Fatalf("expected wrapped delete error, got %v", err)
	}
}

func TestDeleteItemMarksStatusDeleted(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	repo := &fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			Status: domain.ItemStatusCreated,
		},
	}
	svc := NewItemService(repo, &fakeUploadService{})

	err := svc.DeleteItem(context.Background(), itemID.Hex())
	if err != nil {
		t.Fatalf("DeleteItem returned error: %v", err)
	}
	if repo.deleted != itemID {
		t.Fatalf("expected repository delete to be called")
	}
}

func TestDeleteItemRejectsDeletedItem(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			Status: domain.ItemStatusDeleted,
		},
	}, &fakeUploadService{})

	err := svc.DeleteItem(context.Background(), itemID.Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}
