package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pack_mate/internal/domain"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeHandlerCategoryService struct {
	err error
}

func (s *fakeHandlerCategoryService) ListCategories(_ context.Context) ([]domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []domain.Category{{ID: bson.NewObjectID(), Key: "document", Name: "证件"}}, nil
}

func (s *fakeHandlerCategoryService) GetCategory(_ context.Context, _ string) (*domain.Category, error) {
	return nil, nil
}

func (s *fakeHandlerCategoryService) GetCategoryByID(_ context.Context, _ bson.ObjectID) (*domain.Category, error) {
	return nil, nil
}

func (s *fakeHandlerCategoryService) ResolveCategoryID(_ context.Context, _ string) (bson.ObjectID, error) {
	return bson.NilObjectID, nil
}

func TestListCategoriesHandlerReturnsCategories(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	categoryHandler := NewCategoryHandler(&fakeHandlerCategoryService{})
	router.GET("/api/v1/categories", categoryHandler.ListCategories)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"key":"document"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
