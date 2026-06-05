package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakePackService struct {
	pack  *domain.Pack
	packs []domain.Pack
	err   error
}

func (s *fakePackService) CreatePack(_ context.Context, _ request.CreatePackInput) (*domain.Pack, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) ListPacks(_ context.Context, _ request.ListPacksInput) ([]domain.Pack, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.packs != nil {
		return s.packs, nil
	}
	return []domain.Pack{*s.defaultPack()}, nil
}

func (s *fakePackService) GetPack(_ context.Context, _ string) (*domain.Pack, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) UpdatePack(_ context.Context, _ string, _ request.UpdatePackInput) (*domain.Pack, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) DeletePack(_ context.Context, _ string) error {
	return s.err
}

func (s *fakePackService) defaultPack() *domain.Pack {
	if s.pack != nil {
		return s.pack
	}
	return &domain.Pack{
		ID:          bson.NewObjectID(),
		UserID:      bson.NewObjectID(),
		Name:        "日本出差",
		Description: "东京 5 天商务行程",
		Items:       []bson.ObjectID{bson.NewObjectID()},
		Status:      domain.PackStatusCreated,
	}
}

func TestCreatePackHandlerReturnsPack(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.POST("/api/v1/pack", packHandler.CreatePack)

	body := bytes.NewBufferString(`{"name":"日本出差","user_id":"` + bson.NewObjectID().Hex() + `","items":["` + bson.NewObjectID().Hex() + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var resp struct {
		Name   string   `json:"name"`
		Items  []string `json:"items"`
		Status string   `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "日本出差" || len(resp.Items) != 1 || resp.Status != string(domain.PackStatusCreated) {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreatePackHandlerRequiresName(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.POST("/api/v1/pack", packHandler.CreatePack)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", bytes.NewBufferString(`{"description":"东京"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreatePackHandlerRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.POST("/api/v1/pack", packHandler.CreatePack)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", bytes.NewBufferString(`{"name":"日本出差","user_id":"bad-user-id"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "user_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreatePackHandlerRejectsInvalidItemID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.POST("/api/v1/pack", packHandler.CreatePack)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", bytes.NewBufferString(`{"name":"日本出差","items":["bad-item-id"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "items contains invalid item_id") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListPacksHandlerReturnsPacks(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?status=deleted", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp struct {
		Packs []struct {
			Name string `json:"name"`
		} `json:"packs"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Packs) != 1 || resp.Packs[0].Name != "日本出差" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestListPacksHandlerReturnsEmptyList(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{packs: []domain.Pack{}})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"packs":[]`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListPacksHandlerRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?user_id=bad-user-id", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "user_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListPacksHandlerSearchesByQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{
		packs: []domain.Pack{{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "日本出差", Description: "东京 5 天"}},
	})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?q=东京", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"description":"东京 5 天"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListPacksHandlerRejectsEmptyQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?q=+%20", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "q is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListPacksHandlerMapsTooLongQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{err: errors.New("pack search keyword is too long")})
	router.GET("/api/v1/pack", packHandler.ListPacks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?q=东京", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetPackHandlerReturnsPack(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.GET("/api/v1/pack/:pack_id", packHandler.GetPack)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"name":"日本出差"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestGetPackHandlerRejectsInvalidPackID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.GET("/api/v1/pack/:pack_id", packHandler.GetPack)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack/bad-pack-id", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "pack_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestGetPackHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{err: errors.New("pack not found")})
	router.GET("/api/v1/pack/:pack_id", packHandler.GetPack)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdatePackHandlerReturnsPack(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.PUT("/api/v1/pack/:pack_id", packHandler.UpdatePack)

	body := bytes.NewBufferString(`{"name":"日本出差","description":"东京 6 天商务行程","items":["` + bson.NewObjectID().Hex() + `"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pack/"+bson.NewObjectID().Hex(), body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"name":"日本出差"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdatePackHandlerRequiresName(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.PUT("/api/v1/pack/:pack_id", packHandler.UpdatePack)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pack/"+bson.NewObjectID().Hex(), bytes.NewBufferString(`{"description":"东京"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdatePackHandlerRejectsInvalidPackID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.PUT("/api/v1/pack/:pack_id", packHandler.UpdatePack)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pack/bad-pack-id", bytes.NewBufferString(`{"name":"日本出差"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "pack_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdatePackHandlerRejectsInvalidItemID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.PUT("/api/v1/pack/:pack_id", packHandler.UpdatePack)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pack/"+bson.NewObjectID().Hex(), bytes.NewBufferString(`{"name":"日本出差","items":["bad-item-id"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "items contains invalid item_id") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdatePackHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{err: errors.New("pack not found")})
	router.PUT("/api/v1/pack/:pack_id", packHandler.UpdatePack)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pack/"+bson.NewObjectID().Hex(), bytes.NewBufferString(`{"name":"日本出差"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestDeletePackHandlerReturnsDeletedTrue(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.DELETE("/api/v1/pack/:pack_id", packHandler.DeletePack)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/"+bson.NewObjectID().Hex(), nil)
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

func TestDeletePackHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{err: errors.New("pack not found")})
	router.DELETE("/api/v1/pack/:pack_id", packHandler.DeletePack)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestDeletePackHandlerRejectsInvalidPackID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	packHandler := NewPackHandler(&fakePackService{})
	router.DELETE("/api/v1/pack/:pack_id", packHandler.DeletePack)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/bad-pack-id", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "pack_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
