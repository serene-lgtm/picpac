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
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeChecklistService struct {
	checklist  *domain.Checklist
	checklists []domain.Checklist
	err        error
}

func (s *fakeChecklistService) CreateChecklist(_ context.Context, _ request.CreateChecklistInput) (*domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) ListChecklists(_ context.Context, _ request.ListChecklistsInput) ([]domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.checklists != nil {
		return s.checklists, nil
	}
	return []domain.Checklist{*s.defaultChecklist()}, nil
}

func (s *fakeChecklistService) GetChecklist(_ context.Context, _ string) (*domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) UpdateChecklist(_ context.Context, _ string, _ request.UpdateChecklistInput) (*domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) AddChecklistLineItems(_ context.Context, _ string, _ request.AddChecklistLineItemsInput) (*domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) RemoveChecklistLineItems(_ context.Context, _ string, _ request.RemoveChecklistLineItemsInput) (*domain.Checklist, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) DeleteChecklist(_ context.Context, _ string) error {
	return s.err
}

func (s *fakeChecklistService) defaultChecklist() *domain.Checklist {
	if s.checklist != nil {
		return s.checklist
	}
	return &domain.Checklist{
		ID:          bson.NewObjectID(),
		UserID:      bson.NewObjectID(),
		Name:        "日本出差 checklist",
		Description: "东京 5 天",
		TargetDate:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Items: []domain.LineItem{{
			ID:            bson.NewObjectID(),
			ReferenceID:   bson.NewObjectID(),
			ReferenceType: domain.LineItemTypeItem,
			Status:        domain.LineItemStatusUnchecked,
		}, {
			ID:            bson.NewObjectID(),
			ReferenceType: domain.LineItemTypeSnapshot,
			Snapshot:      &domain.ItemSnapshot{Name: "临时雨伞"},
			Status:        domain.LineItemStatusUnchecked,
		}},
		Status: domain.ChecklistStatusCreated,
	}
}

func TestCreateChecklistHandlerReturnsChecklist(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.POST("/api/v1/checklist", checklistHandler.CreateChecklist)

	body := bytes.NewBufferString(`{"name":"日本出差 checklist","target_date":"2026-07-01","items":[{"reference_type":"item","reference_id":"` + bson.NewObjectID().Hex() + `"},{"reference_type":"snapshot","snapshot":{"name":"临时雨伞"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"target_date":"2026-07-01"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"snapshot":{"name":"临时雨伞"}`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestCreateChecklistHandlerRequiresFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.POST("/api/v1/checklist", checklistHandler.CreateChecklist)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist", bytes.NewBufferString(`{"name":"清单"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateChecklistHandlerRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.POST("/api/v1/checklist", checklistHandler.CreateChecklist)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist", bytes.NewBufferString(`{"user_id":"bad-user-id","name":"清单","target_date":"2026-07-01"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "user_id is invalid") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestListChecklistsHandlerSearchesByQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.GET("/api/v1/checklist", checklistHandler.ListChecklists)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist?q=东京", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp struct {
		Checklists []struct {
			Name string `json:"name"`
		} `json:"checklists"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Checklists) != 1 || resp.Checklists[0].Name != "日本出差 checklist" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestListChecklistsHandlerRejectsEmptyQ(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.GET("/api/v1/checklist", checklistHandler.ListChecklists)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist?q=+%20", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetChecklistHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{err: errors.New("checklist not found")})
	router.GET("/api/v1/checklist/:checklist_id", checklistHandler.GetChecklist)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist/"+bson.NewObjectID().Hex(), nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateChecklistHandlerRejectsItemsField(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.PUT("/api/v1/checklist/:checklist_id", checklistHandler.UpdateChecklist)

	body := bytes.NewBufferString(`{"name":"清单","target_date":"2026-07-01","items":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/checklist/"+bson.NewObjectID().Hex(), body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "items cannot be updated here") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestAddChecklistLineItemsHandlerReturnsChecklist(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.POST("/api/v1/checklist/:checklist_id/items", checklistHandler.AddChecklistLineItems)

	body := bytes.NewBufferString(`{"items":[{"reference_type":"snapshot","snapshot":{"name":"临时雨伞"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist/"+bson.NewObjectID().Hex()+"/items", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"snapshot":{"name":"临时雨伞"}`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestRemoveChecklistLineItemsHandlerReturnsChecklist(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.DELETE("/api/v1/checklist/:checklist_id/items", checklistHandler.RemoveChecklistLineItems)

	body := bytes.NewBufferString(`{"line_item_ids":["` + bson.NewObjectID().Hex() + `"]}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/checklist/"+bson.NewObjectID().Hex()+"/items", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestDeleteChecklistHandlerReturnsDeletedTrue(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	checklistHandler := NewChecklistHandler(&fakeChecklistService{})
	router.DELETE("/api/v1/checklist/:checklist_id", checklistHandler.DeleteChecklist)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/checklist/"+bson.NewObjectID().Hex(), nil)
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
