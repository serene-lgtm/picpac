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
	listInput    request.ListItemsInput
	gotItemID    string
	gotUserID    string
	updateItemID string
	updateUserID string
	updateInput  request.UpdateItemInput
	deleteItemID string
	deleteUserID string
}

func (s *fakeItemService) CreateItem(_ context.Context, input request.CreateItemInput) (*domain.Item, error) {
	s.createInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
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
		ID:                 bson.NewObjectID(),
		UserID:             bson.NewObjectID(),
		Name:               "黑色双肩包",
		Description:        "日常出差用",
		SourceImageURL:     "",
		ImageThumbnailURL:  "",
		AIRenderedImageURL: "",
	}
}

func newAuthenticatedItemRouter(t *testing.T, itemService *fakeItemService) (*gin.Engine, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	itemHandler := NewItemHandler(itemService)
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authMiddleware := NewAuthMiddleware(tokenService, &fakeAuthService{user: &domain.User{ID: userID, DisplayName: "用户8000", Status: domain.UserStatusCreated}})
	itemRoutes := router.Group("/api/v1/item")
	itemRoutes.Use(authMiddleware.RequireAuth())
	itemRoutes.POST("", itemHandler.CreateItem)
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
