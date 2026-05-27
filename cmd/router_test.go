package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeItemService struct{}

func (s *fakeItemService) CreateItem(_ context.Context, _ request.CreateItemInput) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) ListItems(_ context.Context, _ string) ([]domain.Item, error) {
	return []domain.Item{}, nil
}

func (s *fakeItemService) GetItem(_ context.Context, _ string) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) UpdateItem(_ context.Context, _ string, _ request.UpdateItemInput) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) DeleteItem(_ context.Context, _ string) error {
	return nil
}

func TestRegisterAPIRoutesExposesItemEndpoints(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerAPIRoutes(router, &fakeItemService{})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/item"},
		{method: http.MethodGet, path: "/api/v1/item"},
		{method: http.MethodGet, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
		{method: http.MethodPut, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
		{method: http.MethodDelete, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, bytes.NewBuffer(nil))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("expected route %s %s to be registered", test.method, test.path)
		}
	}
}
