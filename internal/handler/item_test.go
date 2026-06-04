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

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeItemService struct {
	item  *domain.Item
	items []domain.Item
	err   error
}

func (s *fakeItemService) CreateItem(_ context.Context, _ request.CreateItemInput) (*domain.Item, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) ListItems(_ context.Context, _ request.ListItemsInput) ([]domain.Item, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.items != nil {
		return s.items, nil
	}
	return []domain.Item{*s.defaultItem()}, nil
}

func (s *fakeItemService) GetItem(_ context.Context, _ string) (*domain.Item, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) UpdateItem(_ context.Context, _ string, _ request.UpdateItemInput) (*domain.Item, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultItem(), nil
}

func (s *fakeItemService) DeleteItem(_ context.Context, _ string) error {
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

func TestCreateItemHandlerRequiresName(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.POST("/api/v1/item", itemHandler.CreateItem)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreateItemHandlerReturnsItem(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.POST("/api/v1/item", itemHandler.CreateItem)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("user_id", bson.NewObjectID().Hex())
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestCreateItemHandlerRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.POST("/api/v1/item", itemHandler.CreateItem)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("user_id", "bad-user-id")
	_ = writer.WriteField("name", "黑色双肩包")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/item", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "user_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerAllowsMissingUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestListItemsHandlerRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?user_id=bad-user-id", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "user_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerSearchesByQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{
		items: []domain.Item{{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "手机充电器"}},
	})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=充电", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"name":"手机充电器"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerSearchesDescriptionByQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{
		items: []domain.Item{{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "转换插头", Description: "支持手机充电器"}},
	})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=充电", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"description":"支持手机充电器"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerRejectsEmptyQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=+%20", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "q is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListItemsHandlerMapsTooLongQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{err: errors.New("item search keyword is too long")})
	router.GET("/api/v1/item", itemHandler.ListItems)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item?q=充电", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetItemHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{err: errors.New("item not found")})
	router.GET("/api/v1/item/:item_id", itemHandler.GetItem)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/item/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateItemHandlerRequiresName(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.PUT("/api/v1/item/:item_id", itemHandler.UpdateItem)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/item/"+bson.NewObjectID().Hex(), body)
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

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	itemHandler := NewItemHandler(&fakeItemService{})
	router.DELETE("/api/v1/item/:item_id", itemHandler.DeleteItem)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/item/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
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
