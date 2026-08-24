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
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeItemRepository struct {
	created       *domain.Item
	listed        []domain.Item
	searched      []domain.Item
	listUserID    bson.ObjectID
	filter        repository.ItemFilter
	searchUserID  bson.ObjectID
	searchKeyword string
	gotItemID     bson.ObjectID
	got           *domain.Item
	updated       *domain.Item
	deleted       bson.ObjectID
	err           error
	getErr        error
	deleteErr     error
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

func (r *fakeItemRepository) ListByUserID(_ context.Context, userID bson.ObjectID) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.listUserID = userID
	return r.listed, nil
}

func (r *fakeItemRepository) ListByFilter(_ context.Context, filter repository.ItemFilter) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.filter = filter
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

func (r *fakeItemRepository) SearchByKeywordAndUserID(_ context.Context, userID bson.ObjectID, keyword string) ([]domain.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchUserID = userID
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeItemRepository) GetByID(_ context.Context, itemID bson.ObjectID) (*domain.Item, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	r.gotItemID = itemID
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
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = itemID
	return nil
}

type fakeUploadService struct {
	objectKey string
	err       error
}

type fakeItemCategoryService struct {
	categoryID bson.ObjectID
	err        error
}

func (s *fakeItemCategoryService) ListCategories(_ context.Context) ([]domain.Category, error) {
	return nil, nil
}

func (s *fakeItemCategoryService) GetCategory(_ context.Context, _ string) (*domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.categoryID == bson.NilObjectID {
		s.categoryID = bson.NewObjectID()
	}
	return &domain.Category{ID: s.categoryID, Key: "other", Name: "其他"}, nil
}

func (s *fakeItemCategoryService) GetCategoryByID(_ context.Context, _ bson.ObjectID) (*domain.Category, error) {
	return nil, nil
}

func (s *fakeItemCategoryService) ResolveCategoryID(_ context.Context, _ string) (bson.ObjectID, error) {
	if s.err != nil {
		return bson.NilObjectID, s.err
	}
	if s.categoryID == bson.NilObjectID {
		s.categoryID = bson.NewObjectID()
	}
	return s.categoryID, nil
}

func (s *fakeUploadService) Upload(_ context.Context, objectKey string, _ string, _ io.Reader) error {
	if s.err != nil {
		return s.err
	}
	s.objectKey = objectKey
	return nil
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
	categoryID := bson.NewObjectID()
	repo := &fakeItemRepository{}
	svc := NewItemService(repo, &fakeUploadService{}, &fakeItemCategoryService{categoryID: categoryID})

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
	if item.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID.Hex(), item.UserID.Hex())
	}
	if item.CategoryID != categoryID {
		t.Fatalf("expected category id %s, got %s", categoryID.Hex(), item.CategoryID.Hex())
	}
	if item.Name != "黑色双肩包" || item.SourceImageObjectKey != "" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Status != domain.ItemStatusCreated {
		t.Fatalf("expected status=created, got %s", item.Status)
	}
}

func TestCreateItemRejectsInvalidCategory(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, &fakeItemCategoryService{err: errors.New("invalid category")})

	_, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID:     bson.NewObjectID().Hex(),
		CategoryID: bson.NewObjectID().Hex(),
		Name:       "黑色双肩包",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid category") {
		t.Fatalf("expected invalid category error, got %v", err)
	}
}

func TestCreateItemUploadsImageWhenProvided(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakeItemRepository{}
	uploader := &fakeUploadService{}
	svc := NewItemService(repo, uploader, nil)

	item, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID:   userID.Hex(),
		Name:     "黑色双肩包",
		File:     bytes.NewReader(testPNGBytes()),
		FileName: "bag.jpg",
	})
	if err != nil {
		t.Fatalf("CreateItem returned error: %v", err)
	}
	if item.SourceImageObjectKey == "" {
		t.Fatalf("expected source image object key to be populated")
	}
	if item.SourceImageObjectKey != uploader.objectKey {
		t.Fatalf("expected source image object key %s, got %s", uploader.objectKey, item.SourceImageObjectKey)
	}
}

func TestCreateItemRejectsMissingName(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		UserID: bson.NewObjectID().Hex(),
	})
	if err == nil || !strings.Contains(err.Error(), "item name is required") {
		t.Fatalf("expected item name error, got %v", err)
	}
}

func TestCreateItemRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.CreateItem(context.Background(), request.CreateItemInput{
		Name: "黑色双肩包",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListItemsReturnsCurrentUserResults(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Item{{UserID: userID, Name: "黑色双肩包"}}
	repo := &fakeItemRepository{listed: expected}
	svc := NewItemService(repo, &fakeUploadService{}, nil)

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{UserID: userID.Hex()})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "黑色双肩包" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if repo.filter.UserID != userID {
		t.Fatalf("expected list by user id %s, got %s", userID.Hex(), repo.filter.UserID.Hex())
	}
}

func TestListItemsFiltersByCategory(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	categoryID := bson.NewObjectID()
	repo := &fakeItemRepository{listed: []domain.Item{{Name: "护照", CategoryID: categoryID}}}
	svc := NewItemService(repo, &fakeUploadService{}, &fakeItemCategoryService{categoryID: categoryID})

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID:        userID.Hex(),
		CategoryID:    categoryID.Hex(),
		HasCategoryID: true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].CategoryID != categoryID {
		t.Fatalf("unexpected items: %+v", items)
	}
	if !repo.filter.HasCategoryID || repo.filter.CategoryID != categoryID {
		t.Fatalf("expected category filter %s, got %+v", categoryID.Hex(), repo.filter)
	}
}

func TestListItemsRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListItemsSearchesByChineseKeyword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Item{{Name: "手机充电器"}}
	repo := &fakeItemRepository{listed: expected}
	svc := NewItemService(repo, &fakeUploadService{}, nil)

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: userID.Hex(),
		Q:      "  充电  ",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "手机充电器" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if repo.filter.Keyword != "充电" {
		t.Fatalf("expected trimmed keyword, got %q", repo.filter.Keyword)
	}
	if repo.filter.UserID != userID {
		t.Fatalf("expected search by user id %s, got %s", userID.Hex(), repo.filter.UserID.Hex())
	}
}

func TestListItemsFiltersByKeywordAndCategory(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	categoryID := bson.NewObjectID()
	repo := &fakeItemRepository{listed: []domain.Item{{Name: "手机充电器", CategoryID: categoryID}}}
	svc := NewItemService(repo, &fakeUploadService{}, &fakeItemCategoryService{categoryID: categoryID})

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID:        userID.Hex(),
		Q:             "充电",
		HasQ:          true,
		CategoryID:    categoryID.Hex(),
		HasCategoryID: true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if !repo.filter.HasKeyword || repo.filter.Keyword != "充电" {
		t.Fatalf("expected keyword filter, got %+v", repo.filter)
	}
	if !repo.filter.HasCategoryID || repo.filter.CategoryID != categoryID {
		t.Fatalf("expected category filter %s, got %+v", categoryID.Hex(), repo.filter)
	}
}

func TestListItemsSearchesByDescriptionKeyword(t *testing.T) {
	t.Parallel()

	repo := &fakeItemRepository{listed: []domain.Item{{Name: "转换插头", Description: "支持手机充电器"}}}
	svc := NewItemService(repo, &fakeUploadService{}, nil)

	items, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: bson.NewObjectID().Hex(),
		Q:      "充电",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].Description != "支持手机充电器" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestListItemsRejectsEmptySearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: bson.NewObjectID().Hex(),
		HasQ:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "item search keyword is required") {
		t.Fatalf("expected keyword required error, got %v", err)
	}
}

func TestListItemsRejectsTooLongSearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)
	keyword := strings.Repeat("行", maxItemSearchKeywordRunes+1)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: bson.NewObjectID().Hex(),
		Q:      keyword,
		HasQ:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "item search keyword is too long") {
		t.Fatalf("expected keyword too long error, got %v", err)
	}
}

func TestListItemsRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{UserID: "bad-user-id"})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListItemsWrapsSearchRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{err: errors.New("db down")}, &fakeUploadService{}, nil)

	_, err := svc.ListItems(context.Background(), request.ListItemsInput{
		UserID: bson.NewObjectID().Hex(),
		Q:      "充电",
		HasQ:   true,
	})
	if err == nil || !strings.Contains(err.Error(), "list items failed") {
		t.Fatalf("expected list items failure, got %v", err)
	}
}

func TestGetItemMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewItemService(&fakeItemRepository{}, &fakeUploadService{}, nil)

	_, err := svc.GetItem(context.Background(), bson.NewObjectID().Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestGetItemRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: bson.NewObjectID(),
			Status: domain.ItemStatusCreated,
		},
	}, &fakeUploadService{}, nil)

	_, err := svc.GetItem(context.Background(), itemID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestGetItemHidesDeletedItem(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: userID,
			Status: domain.ItemStatusDeleted,
		},
	}, &fakeUploadService{}, nil)

	_, err := svc.GetItem(context.Background(), itemID.Hex(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestUpdateItemReplacesFieldsAndOptionalImage(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	categoryID := bson.NewObjectID()
	repo := &fakeItemRepository{
		got: &domain.Item{
			ID:                   itemID,
			UserID:               userID,
			Name:                 "旧名称",
			Description:          "old",
			SourceImageObjectKey: "items/item_old/source.jpg",
			Status:               domain.ItemStatusCreated,
			UpdatedAt:            time.Now().Add(-time.Hour),
		},
	}
	uploader := &fakeUploadService{}
	svc := NewItemService(repo, uploader, &fakeItemCategoryService{categoryID: categoryID})

	item, err := svc.UpdateItem(context.Background(), itemID.Hex(), userID.Hex(), request.UpdateItemInput{
		CategoryID:    categoryID.Hex(),
		HasCategoryID: true,
		Name:          "新名称",
		File:          bytes.NewReader(testPNGBytes()),
		FileName:      "bag.png",
	})
	if err != nil {
		t.Fatalf("UpdateItem returned error: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "新名称" {
		t.Fatalf("expected repository update to be called")
	}
	if repo.updated.CategoryID != categoryID {
		t.Fatalf("expected category id %s, got %s", categoryID.Hex(), repo.updated.CategoryID.Hex())
	}
	if item.SourceImageObjectKey != uploader.objectKey {
		t.Fatalf("expected image object key %s, got %s", uploader.objectKey, item.SourceImageObjectKey)
	}
}

func TestUpdateItemKeepsCategoryWhenNotProvided(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	categoryID := bson.NewObjectID()
	repo := &fakeItemRepository{
		got: &domain.Item{
			ID:         itemID,
			UserID:     userID,
			CategoryID: categoryID,
			Name:       "旧名称",
			Status:     domain.ItemStatusCreated,
		},
	}
	svc := NewItemService(repo, &fakeUploadService{}, &fakeItemCategoryService{categoryID: bson.NewObjectID()})

	_, err := svc.UpdateItem(context.Background(), itemID.Hex(), userID.Hex(), request.UpdateItemInput{
		Name: "新名称",
	})
	if err != nil {
		t.Fatalf("UpdateItem returned error: %v", err)
	}
	if repo.updated.CategoryID != categoryID {
		t.Fatalf("expected category id to remain %s, got %s", categoryID.Hex(), repo.updated.CategoryID.Hex())
	}
}

func TestUpdateItemRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: bson.NewObjectID(),
			Status: domain.ItemStatusCreated,
		},
	}, &fakeUploadService{}, nil)

	_, err := svc.UpdateItem(context.Background(), itemID.Hex(), bson.NewObjectID().Hex(), request.UpdateItemInput{
		Name: "新名称",
	})
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}

func TestDeleteItemWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: userID,
			Status: domain.ItemStatusCreated,
		},
		deleteErr: errors.New("db down"),
	}, &fakeUploadService{}, nil)

	err := svc.DeleteItem(context.Background(), itemID.Hex(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "delete item failed") {
		t.Fatalf("expected delete item failure, got %v", err)
	}
}

func TestDeleteItemMarksStatusDeleted(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	repo := &fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: userID,
			Status: domain.ItemStatusCreated,
		},
	}
	svc := NewItemService(repo, &fakeUploadService{}, nil)

	err := svc.DeleteItem(context.Background(), itemID.Hex(), userID.Hex())
	if err != nil {
		t.Fatalf("DeleteItem returned error: %v", err)
	}
	if repo.deleted != itemID {
		t.Fatalf("expected repository delete to be called")
	}
}

func TestDeleteItemRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	svc := NewItemService(&fakeItemRepository{
		got: &domain.Item{
			ID:     itemID,
			UserID: bson.NewObjectID(),
			Status: domain.ItemStatusCreated,
		},
	}, &fakeUploadService{}, nil)

	err := svc.DeleteItem(context.Background(), itemID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "item not found") {
		t.Fatalf("expected item not found, got %v", err)
	}
}
