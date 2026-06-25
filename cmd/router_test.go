package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeItemService struct{}

func (s *fakeItemService) CreateItem(_ context.Context, _ request.CreateItemInput) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) ListItems(_ context.Context, _ request.ListItemsInput) ([]domain.Item, error) {
	return []domain.Item{}, nil
}

func (s *fakeItemService) GetItem(_ context.Context, _ string, _ string) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) UpdateItem(_ context.Context, _ string, _ string, _ request.UpdateItemInput) (*domain.Item, error) {
	return &domain.Item{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "item"}, nil
}

func (s *fakeItemService) DeleteItem(_ context.Context, _ string, _ string) error {
	return nil
}

type fakePackService struct{}

func (s *fakePackService) CreatePack(_ context.Context, _ request.CreatePackInput) (*domain.Pack, error) {
	return &domain.Pack{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "pack"}, nil
}

func (s *fakePackService) ListPacks(_ context.Context, _ request.ListPacksInput) ([]domain.Pack, error) {
	return []domain.Pack{}, nil
}

func (s *fakePackService) GetPack(_ context.Context, _ string, _ string) (*domain.Pack, error) {
	return &domain.Pack{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "pack"}, nil
}

func (s *fakePackService) UpdatePack(_ context.Context, _ string, _ string, _ request.UpdatePackInput) (*domain.Pack, error) {
	return &domain.Pack{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "pack"}, nil
}

func (s *fakePackService) DeletePack(_ context.Context, _ string, _ string) error {
	return nil
}

type fakeChecklistService struct{}

func (s *fakeChecklistService) CreateChecklist(_ context.Context, _ request.CreateChecklistInput) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) ListChecklists(_ context.Context, _ request.ListChecklistsInput) ([]domain.Checklist, error) {
	return []domain.Checklist{}, nil
}

func (s *fakeChecklistService) GetChecklist(_ context.Context, _ string, _ string) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) UpdateChecklist(_ context.Context, _ string, _ string, _ request.UpdateChecklistInput) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) AddChecklistLineItems(_ context.Context, _ string, _ string, _ request.AddChecklistLineItemsInput) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) RemoveChecklistLineItems(_ context.Context, _ string, _ string, _ request.RemoveChecklistLineItemsInput) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) UpdateChecklistLineItemStatus(_ context.Context, _ string, _ string, _ string, _ request.UpdateChecklistLineItemStatusInput) (*domain.Checklist, error) {
	return &domain.Checklist{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "checklist"}, nil
}

func (s *fakeChecklistService) DeleteChecklist(_ context.Context, _ string, _ string) error {
	return nil
}

type fakeAuthService struct{}

func (s *fakeAuthService) SendPhoneCode(_ context.Context, _ request.SendPhoneCodeInput) error {
	return nil
}

func (s *fakeAuthService) LoginWithPhone(_ context.Context, _ request.PhoneLoginInput) (*service.AuthResult, error) {
	return &service.AuthResult{AccessToken: "access", RefreshToken: "refresh", User: &domain.User{ID: bson.NewObjectID(), DisplayName: "user", Status: domain.UserStatusCreated}}, nil
}

func (s *fakeAuthService) Refresh(_ context.Context, _ request.RefreshTokenInput) (*service.RefreshResult, error) {
	return &service.RefreshResult{AccessToken: "access"}, nil
}

func (s *fakeAuthService) Logout(_ context.Context, _ request.LogoutInput) error {
	return nil
}

func (s *fakeAuthService) Me(_ context.Context, _ string) (*domain.User, error) {
	return &domain.User{ID: bson.NewObjectID(), DisplayName: "user", Status: domain.UserStatusCreated}, nil
}

type fakeTokenService struct{}

func (s *fakeTokenService) CreateAccessToken(_ bson.ObjectID) (string, error) {
	return "access", nil
}

func (s *fakeTokenService) ParseAccessToken(_ string) (bson.ObjectID, error) {
	return bson.NewObjectID(), nil
}

func (s *fakeTokenService) CreateRefreshToken() (string, string, error) {
	return "refresh", "hash", nil
}

func (s *fakeTokenService) HashToken(_ string) string {
	return "hash"
}

func TestRegisterAPIRoutesExposesEndpoints(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerAPIRoutes(router, &fakeItemService{}, &fakePackService{}, &fakeChecklistService{}, &fakeAuthService{}, &fakeTokenService{})

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/item"},
		{method: http.MethodGet, path: "/api/v1/item"},
		{method: http.MethodGet, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
		{method: http.MethodPut, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
		{method: http.MethodDelete, path: "/api/v1/item/" + bson.NewObjectID().Hex()},
		{method: http.MethodPost, path: "/api/v1/pack"},
		{method: http.MethodGet, path: "/api/v1/pack"},
		{method: http.MethodGet, path: "/api/v1/pack/" + bson.NewObjectID().Hex()},
		{method: http.MethodPut, path: "/api/v1/pack/" + bson.NewObjectID().Hex()},
		{method: http.MethodDelete, path: "/api/v1/pack/" + bson.NewObjectID().Hex()},
		{method: http.MethodPost, path: "/api/v1/checklist"},
		{method: http.MethodGet, path: "/api/v1/checklist"},
		{method: http.MethodGet, path: "/api/v1/checklist/" + bson.NewObjectID().Hex()},
		{method: http.MethodPut, path: "/api/v1/checklist/" + bson.NewObjectID().Hex()},
		{method: http.MethodPost, path: "/api/v1/checklist/" + bson.NewObjectID().Hex() + "/items"},
		{method: http.MethodDelete, path: "/api/v1/checklist/" + bson.NewObjectID().Hex() + "/items"},
		{method: http.MethodPatch, path: "/api/v1/checklist/" + bson.NewObjectID().Hex() + "/items/" + bson.NewObjectID().Hex() + "/status"},
		{method: http.MethodDelete, path: "/api/v1/checklist/" + bson.NewObjectID().Hex()},
		{method: http.MethodPost, path: "/api/v1/auth/phone/code"},
		{method: http.MethodPost, path: "/api/v1/auth/phone/login"},
		{method: http.MethodPost, path: "/api/v1/auth/refresh"},
		{method: http.MethodPost, path: "/api/v1/auth/logout"},
		{method: http.MethodGet, path: "/api/v1/me"},
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
