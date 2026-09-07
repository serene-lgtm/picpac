package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeItemService struct {
	item         *domain.Item
	items        []domain.Item
	err          error
	createInput  request.CreateItemInput
	batchInput   request.BatchCreateItemsInput
	listInput    request.ListItemsInput
	gotItemID    string
	gotUserID    string
	updateItemID string
	updateUserID string
	updateInput  request.UpdateItemInput
	deleteItemID string
	deleteUserID string
}

type fakeCategoryService struct {
	category *domain.Category
	err      error
}

func (s *fakeCategoryService) ListCategories(_ context.Context) ([]domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []domain.Category{*s.defaultCategory()}, nil
}

func (s *fakeCategoryService) GetCategory(_ context.Context, _ string) (*domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultCategory(), nil
}

func (s *fakeCategoryService) GetCategoryByID(_ context.Context, _ bson.ObjectID) (*domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultCategory(), nil
}

func (s *fakeCategoryService) ResolveCategoryID(_ context.Context, _ string) (bson.ObjectID, error) {
	if s.err != nil {
		return bson.NilObjectID, s.err
	}
	return s.defaultCategory().ID, nil
}

func (s *fakeCategoryService) defaultCategory() *domain.Category {
	if s.category != nil {
		return s.category
	}
	return &domain.Category{ID: bson.NewObjectID(), Key: "other", Name: "其他"}
}

func (s *fakeItemService) CreateItem(_ context.Context, input request.CreateItemInput) (*domain.Item, error) {
	s.createInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) CreateItemsBatch(_ context.Context, input request.BatchCreateItemsInput) ([]domain.Item, error) {
	s.batchInput = input
	if s.err != nil {
		return nil, s.err
	}
	if s.items != nil {
		return s.items, nil
	}
	return []domain.Item{*s.defaultItem()}, nil
}

func (s *fakeItemService) ListItems(_ context.Context, input request.ListItemsInput) ([]domain.Item, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	if s.items != nil {
		return s.items, nil
	}
	return []domain.Item{*s.defaultItem()}, nil
}

func (s *fakeItemService) GetItem(_ context.Context, itemID string, userID string) (*domain.Item, error) {
	s.gotItemID = itemID
	s.gotUserID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) UpdateItem(_ context.Context, itemID string, userID string, input request.UpdateItemInput) (*domain.Item, error) {
	s.updateItemID = itemID
	s.updateUserID = userID
	s.updateInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) DeleteItem(_ context.Context, itemID string, userID string) error {
	s.deleteItemID = itemID
	s.deleteUserID = userID
	return s.err
}

func (s *fakeItemService) defaultItem() *domain.Item {
	if s.item != nil {
		return s.item
	}
	return &domain.Item{
		ID:          bson.NewObjectID(),
		UserID:      bson.NewObjectID(),
		Name:        "黑色双肩包",
		Description: "日常出差用",
	}
}

func testItemPNGBytes() []byte {
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

func newAuthenticatedItemRouter(t *testing.T, itemService *fakeItemService) (*gin.Engine, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	itemHandler := NewItemHandler(itemService, &fakeCategoryService{}, fakeObjectURLSigner{})
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authMiddleware := NewAuthMiddleware(tokenService, &fakeAuthService{user: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: "users/default/avatar.png"}, Status: domain.UserStatusCreated}})
	itemRoutes := router.Group("/api/v1/item")
	itemRoutes.Use(authMiddleware.RequireAuth())
	itemRoutes.POST("", itemHandler.CreateItem)
	itemRoutes.POST("/batch", itemHandler.CreateItemsBatch)
	itemRoutes.GET("", itemHandler.ListItems)
	itemRoutes.GET("/:item_id", itemHandler.GetItem)
	itemRoutes.PUT("/:item_id", itemHandler.UpdateItem)
	itemRoutes.DELETE("/:item_id", itemHandler.DeleteItem)

	return router, token, userID.Hex()
}

func TestCreateItemHandlerRequiresName(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreateItemHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedItemRouter(t, itemService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.createInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, itemService.createInput.UserID)
	}
}

func TestCreateItemHandlerPassesCategoryID(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)
	categoryID := bson.NewObjectID().Hex()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.WriteField("category_id", categoryID)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.createInput.CategoryID != categoryID {
		t.Fatalf("expected category id %s, got %s", categoryID, itemService.createInput.CategoryID)
	}
}

func TestCreateItemHandlerPassesMultiplePhotos(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "相机")
	for _, fileName := range []string{"front.png", "back.png"} {
		part, err := writer.CreateFormFile("photos", fileName)
		if err != nil {
			t.Fatalf("CreateFormFile returned error: %v", err)
		}
		if _, err := part.Write(testItemPNGBytes()); err != nil {
			t.Fatalf("write photo returned error: %v", err)
		}
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if len(itemService.createInput.Photos) != 2 {
		t.Fatalf("expected two photos, got %+v", itemService.createInput.Photos)
	}
}

func TestCreateItemHandlerSignsImageURL(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{item: &domain.Item{
		ID:     bson.NewObjectID(),
		UserID: bson.NewObjectID(),
		Name:   "黑色双肩包",
		Photos: []domain.ItemPhoto{{
			ID:               bson.NewObjectID(),
			SourceObjectKey:  "items/user_1/item_1/photos/photo_1/source.jpg",
			DisplayObjectKey: "items/user_1/item_1/photos/photo_1/display.jpg",
		}},
		Status: domain.ItemStatusCreated,
	}}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp struct {
		CoverImageURL string `json:"cover_image_url"`
		Photos        []struct {
			SourceImageURL string `json:"source_image_url"`
			ImageURL       string `json:"image_url"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	expectedSourceURL := "https://signed.example/items/user_1/item_1/photos/photo_1/source.jpg?Expires=3600&Signature=test"
	expectedDisplayURL := "https://signed.example/items/user_1/item_1/photos/photo_1/display.jpg?Expires=3600&Signature=test"
	if resp.CoverImageURL != expectedDisplayURL || len(resp.Photos) != 1 ||
		resp.Photos[0].SourceImageURL != expectedSourceURL || resp.Photos[0].ImageURL != expectedDisplayURL {
		t.Fatalf("unexpected photo response: %+v", resp)
	}
}

func TestCreateItemHandlerUsesDefaultCoverWhenPhotosEmpty(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{item: &domain.Item{
		ID:     bson.NewObjectID(),
		UserID: bson.NewObjectID(),
		Name:   "护照",
		Status: domain.ItemStatusCreated,
	}}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "护照")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp struct {
		CoverImageURL string `json:"cover_image_url"`
		Photos        []struct {
			ImageURL string `json:"image_url"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	expectedCoverURL := "https://signed.example/items/default/cover.jpg?Expires=3600&Signature=test"
	if resp.CoverImageURL != expectedCoverURL || len(resp.Photos) != 0 {
		t.Fatalf("unexpected default cover response: %+v", resp)
	}
}

func TestCreateItemsBatchHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	categoryID := bson.NewObjectID().Hex()
	itemService := &fakeItemService{items: []domain.Item{{
		ID:         bson.NewObjectID(),
		UserID:     bson.NewObjectID(),
		CategoryID: bson.NewObjectID(),
		Name:       "手机",
		Status:     domain.ItemStatusCreated,
	}}}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedItemRouter(t, itemService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item/batch", bytes.NewBufferString(`{"items":[{"name":"手机","category_id":"`+categoryID+`"}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if itemService.batchInput.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, itemService.batchInput.UserID)
	}
	if len(itemService.batchInput.Items) != 1 || itemService.batchInput.Items[0].Name != "手机" || itemService.batchInput.Items[0].CategoryID != categoryID {
		t.Fatalf("unexpected batch input: %+v", itemService.batchInput)
	}
	if !strings.Contains(recorder.Body.String(), `"name":"手机"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreateItemsBatchHandlerRequiresItems(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item/batch", bytes.NewBufferString(`{"items":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "items are required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestItemRoutesRequireAuthorization(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, _, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListItemsHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedItemRouter(t, itemService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.listInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, itemService.listInput.UserID)
	}
}

func TestListItemsHandlerSearchesByQ(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{
		items: []domain.Item{{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "手机充电器"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=充电", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"name":"手机充电器"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerFiltersByCategoryID(t *testing.T) {
	t.Parallel()

	categoryID := bson.NewObjectID().Hex()
	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?category_id="+categoryID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !itemService.listInput.HasCategoryID || itemService.listInput.CategoryID != categoryID {
		t.Fatalf("expected category filter %s, got %+v", categoryID, itemService.listInput)
	}
}

func TestListItemsHandlerRejectsEmptyCategoryID(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?category_id=", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestListItemsHandlerRejectsEmptyQ(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=+%20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "q is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestGetItemHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{err: errors.New("item not found")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item/"+bson.NewObjectID().Hex(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateItemHandlerRequiresName(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, &fakeItemService{})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/item/"+bson.NewObjectID().Hex(), body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdateItemHandlerPassesCategoryID(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedItemRouter(t, itemService)
	itemID := bson.NewObjectID().Hex()
	categoryID := bson.NewObjectID().Hex()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.WriteField("category_id", categoryID)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/item/"+itemID, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.updateUserID != userID || itemService.updateItemID != itemID {
		t.Fatalf("unexpected update call: user=%s item=%s", itemService.updateUserID, itemService.updateItemID)
	}
	if !itemService.updateInput.HasCategoryID || itemService.updateInput.CategoryID != categoryID {
		t.Fatalf("expected category id %s, got %+v", categoryID, itemService.updateInput)
	}
}

func TestUpdateItemHandlerKeepsCategoryWhenMissing(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedItemRouter(t, itemService)
	itemID := bson.NewObjectID().Hex()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/item/"+itemID, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.updateInput.HasCategoryID || itemService.updateInput.CategoryID != "" {
		t.Fatalf("expected category to be omitted, got %+v", itemService.updateInput)
	}
}

func TestDeleteItemHandlerReturnsDeletedTrue(t *testing.T) {
	t.Parallel()

	itemService := &fakeItemService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedItemRouter(t, itemService)
	itemID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/item/"+itemID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if itemService.deleteUserID != userID || itemService.deleteItemID != itemID {
		t.Fatalf("unexpected delete call: user=%s item=%s", itemService.deleteUserID, itemService.deleteItemID)
	}
	var resp struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Deleted {
		t.Fatalf("expected deleted=true")
	}
}
