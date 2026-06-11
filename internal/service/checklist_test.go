package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeChecklistRepository struct {
	created       *domain.Checklist
	listed        []domain.Checklist
	searched      []domain.Checklist
	searchKeyword string
	got           *domain.Checklist
	updated       *domain.Checklist
	statusUpdated bool
	status        domain.LineItemStatus
	statusLineID  bson.ObjectID
	deleted       bson.ObjectID
	err           error
	updateErr     error
}

func (r *fakeChecklistRepository) Create(_ context.Context, checklist *domain.Checklist) error {
	if r.err != nil {
		return r.err
	}
	r.created = checklist
	return nil
}

func (r *fakeChecklistRepository) ListAll(_ context.Context) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) ListByUserID(_ context.Context, _ bson.ObjectID) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) SearchByKeyword(_ context.Context, keyword string) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) SearchByKeywordAndUserID(_ context.Context, _ bson.ObjectID, keyword string) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) GetByID(_ context.Context, _ bson.ObjectID) (*domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeChecklistRepository) Update(_ context.Context, checklist *domain.Checklist) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.updated = checklist
	return nil
}

func (r *fakeChecklistRepository) UpdateLineItemStatus(_ context.Context, _ bson.ObjectID, lineItemID bson.ObjectID, status domain.LineItemStatus, updatedAt time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.statusUpdated = true
	r.statusLineID = lineItemID
	r.status = status
	if r.got != nil {
		r.got.UpdatedAt = updatedAt
		for i := range r.got.Items {
			if r.got.Items[i].ID == lineItemID {
				r.got.Items[i].Status = status
				return nil
			}
		}
	}
	return mongo.ErrNoDocuments
}

func (r *fakeChecklistRepository) DeleteByID(_ context.Context, checklistID bson.ObjectID) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = checklistID
	return nil
}

func validChecklistItemRepository() *fakeItemRepository {
	return &fakeItemRepository{
		got: &domain.Item{
			ID:     bson.NewObjectID(),
			Status: domain.ItemStatusCreated,
		},
	}
}

func TestCreateChecklistStoresChecklist(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		UserID:      userID.Hex(),
		Name:        "日本出差 checklist",
		Description: "东京 5 天",
		TargetDate:  "2026-07-01",
		Items: []request.ChecklistLineItemInput{
			{ReferenceType: string(domain.LineItemTypeItem), ReferenceID: itemID.Hex()},
			{ReferenceType: string(domain.LineItemTypeSnapshot), Snapshot: &request.ItemSnapshotInput{Name: "临时雨伞"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateChecklist returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatalf("expected repository create to be called")
	}
	if checklist.UserID != userID || checklist.Status != domain.ChecklistStatusCreated {
		t.Fatalf("unexpected checklist: %+v", checklist)
	}
	if checklist.TargetDate.Format(checklistTargetDateLayout) != "2026-07-01" {
		t.Fatalf("unexpected target date: %s", checklist.TargetDate)
	}
	if len(checklist.Items) != 2 {
		t.Fatalf("expected 2 line items, got %+v", checklist.Items)
	}
	if checklist.Items[0].ReferenceID != itemID || checklist.Items[0].Status != domain.LineItemStatusUnchecked {
		t.Fatalf("unexpected item line: %+v", checklist.Items[0])
	}
	if !checklist.Items[1].ReferenceID.IsZero() || checklist.Items[1].ReferenceType != domain.LineItemTypeSnapshot {
		t.Fatalf("unexpected snapshot line: %+v", checklist.Items[1])
	}
	if checklist.Items[1].Snapshot == nil || checklist.Items[1].Snapshot.Name != "临时雨伞" {
		t.Fatalf("unexpected snapshot: %+v", checklist.Items[1].Snapshot)
	}
}

func TestCreateChecklistAllowsMissingUserID(t *testing.T) {
	t.Parallel()

	repo := &fakeChecklistRepository{}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		Name:       "日本出差 checklist",
		TargetDate: "2026-07-01",
	})
	if err != nil {
		t.Fatalf("CreateChecklist returned error: %v", err)
	}
	if !checklist.UserID.IsZero() {
		t.Fatalf("expected zero user id, got %s", checklist.UserID.Hex())
	}
}

func TestCreateChecklistRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, validChecklistItemRepository())

	tests := []request.CreateChecklistInput{
		{UserID: "bad-user-id", Name: "清单", TargetDate: "2026-07-01"},
		{UserID: bson.NewObjectID().Hex(), TargetDate: "2026-07-01"},
		{UserID: bson.NewObjectID().Hex(), Name: "清单"},
		{UserID: bson.NewObjectID().Hex(), Name: "清单", TargetDate: "bad-date"},
		{
			UserID:     bson.NewObjectID().Hex(),
			Name:       "清单",
			TargetDate: "2026-07-01",
			Items:      []request.ChecklistLineItemInput{{ReferenceType: string(domain.LineItemTypeItem)}},
		},
		{
			UserID:     bson.NewObjectID().Hex(),
			Name:       "清单",
			TargetDate: "2026-07-01",
			Items:      []request.ChecklistLineItemInput{{ReferenceType: string(domain.LineItemTypeSnapshot), ReferenceID: bson.NewObjectID().Hex()}},
		},
		{
			UserID:     bson.NewObjectID().Hex(),
			Name:       "清单",
			TargetDate: "2026-07-01",
			Items:      []request.ChecklistLineItemInput{{ReferenceType: string(domain.LineItemTypeSnapshot)}},
		},
		{
			UserID:     bson.NewObjectID().Hex(),
			Name:       "清单",
			TargetDate: "2026-07-01",
			Items:      []request.ChecklistLineItemInput{{ReferenceType: string(domain.LineItemTypeSnapshot), Snapshot: &request.ItemSnapshotInput{Name: " "}}},
		},
		{
			UserID:     bson.NewObjectID().Hex(),
			Name:       "清单",
			TargetDate: "2026-07-01",
			Items:      []request.ChecklistLineItemInput{{ReferenceType: string(domain.LineItemTypeItem), ReferenceID: bson.NewObjectID().Hex(), Snapshot: &request.ItemSnapshotInput{Name: "不应出现"}}},
		},
	}

	for _, input := range tests {
		_, err := svc.CreateChecklist(context.Background(), input)
		if err == nil {
			t.Fatalf("expected error for input %+v", input)
		}
	}
}

func TestCreateChecklistRejectsMissingReferenceItem(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, &fakeItemRepository{})

	_, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		Name:       "清单",
		TargetDate: "2026-07-01",
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeItem),
			ReferenceID:   bson.NewObjectID().Hex(),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item reference item not found") {
		t.Fatalf("expected reference item not found, got %v", err)
	}
}

func TestCreateChecklistRejectsDeletedReferenceItem(t *testing.T) {
	t.Parallel()

	itemRepo := &fakeItemRepository{
		got: &domain.Item{ID: bson.NewObjectID(), Status: domain.ItemStatusDeleted},
	}
	svc := NewChecklistService(&fakeChecklistRepository{}, itemRepo)

	_, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		Name:       "清单",
		TargetDate: "2026-07-01",
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeItem),
			ReferenceID:   bson.NewObjectID().Hex(),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item reference item not found") {
		t.Fatalf("expected reference item not found, got %v", err)
	}
}

func TestListChecklistsSearchesByKeyword(t *testing.T) {
	t.Parallel()

	expected := []domain.Checklist{{Name: "日本出差 checklist", Description: "东京 5 天", Status: domain.ChecklistStatusCreated}}
	repo := &fakeChecklistRepository{searched: expected}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklists, err := svc.ListChecklists(context.Background(), request.ListChecklistsInput{
		Q:    "  东京  ",
		HasQ: true,
	})
	if err != nil {
		t.Fatalf("ListChecklists returned error: %v", err)
	}
	if len(checklists) != 1 || checklists[0].Description != "东京 5 天" {
		t.Fatalf("unexpected checklists: %+v", checklists)
	}
	if repo.searchKeyword != "东京" {
		t.Fatalf("expected trimmed keyword, got %q", repo.searchKeyword)
	}
}

func TestListChecklistsRejectsTooLongSearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, validChecklistItemRepository())
	keyword := strings.Repeat("行", maxChecklistSearchKeywordRunes+1)

	_, err := svc.ListChecklists(context.Background(), request.ListChecklistsInput{Q: keyword, HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "checklist search keyword is too long") {
		t.Fatalf("expected keyword too long error, got %v", err)
	}
}

func TestGetChecklistMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, validChecklistItemRepository())

	_, err := svc.GetChecklist(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "checklist not found") {
		t.Fatalf("expected checklist not found, got %v", err)
	}
}

func TestUpdateChecklistReplacesEditableFields(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	originalUpdatedAt := time.Now().Add(-time.Hour)
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			UserID: bson.NewObjectID(),
			Name:   "旧清单",
			Items: []domain.LineItem{{
				ID:            lineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "旧快照"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status:    domain.ChecklistStatusCreated,
			UpdatedAt: originalUpdatedAt,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.UpdateChecklist(context.Background(), checklistID.Hex(), request.UpdateChecklistInput{
		Name:       "新清单",
		TargetDate: "2026-07-02",
	})
	if err != nil {
		t.Fatalf("UpdateChecklist returned error: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "新清单" {
		t.Fatalf("expected repository update to be called")
	}
	if checklist.TargetDate.Format(checklistTargetDateLayout) != "2026-07-02" {
		t.Fatalf("unexpected target date: %s", checklist.TargetDate)
	}
	if !checklist.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance")
	}
	if len(checklist.Items) != 1 || checklist.Items[0].ID != lineItemID {
		t.Fatalf("expected metadata update to preserve line items, got %+v", checklist.Items)
	}
}

func TestAddChecklistLineItemsAppendsItems(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	existingLineItemID := bson.NewObjectID()
	originalUpdatedAt := time.Now().Add(-time.Hour)
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID: checklistID,
			Items: []domain.LineItem{{
				ID:            existingLineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "旧快照"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status:    domain.ChecklistStatusCreated,
			UpdatedAt: originalUpdatedAt,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.AddChecklistLineItems(context.Background(), checklistID.Hex(), request.AddChecklistLineItemsInput{
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeSnapshot),
			Snapshot:      &request.ItemSnapshotInput{Name: "新快照"},
		}},
	})
	if err != nil {
		t.Fatalf("AddChecklistLineItems returned error: %v", err)
	}
	if repo.updated == nil {
		t.Fatalf("expected repository update to be called")
	}
	if len(checklist.Items) != 2 {
		t.Fatalf("expected 2 line items, got %+v", checklist.Items)
	}
	if checklist.Items[0].ID != existingLineItemID {
		t.Fatalf("expected existing line item to be preserved, got %+v", checklist.Items[0])
	}
	if checklist.Items[1].Snapshot == nil || checklist.Items[1].Snapshot.Name != "新快照" {
		t.Fatalf("unexpected appended line item: %+v", checklist.Items[1])
	}
	if !checklist.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance")
	}
}

func TestAddChecklistLineItemsRejectsMissingReferenceItem(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, &fakeItemRepository{})

	_, err := svc.AddChecklistLineItems(context.Background(), checklistID.Hex(), request.AddChecklistLineItemsInput{
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeItem),
			ReferenceID:   bson.NewObjectID().Hex(),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item reference item not found") {
		t.Fatalf("expected reference item not found, got %v", err)
	}
}

func TestRemoveChecklistLineItemsRemovesItems(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	removeLineItemID := bson.NewObjectID()
	keepLineItemID := bson.NewObjectID()
	originalUpdatedAt := time.Now().Add(-time.Hour)
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID: checklistID,
			Items: []domain.LineItem{{
				ID:            removeLineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "移除"},
				Status:        domain.LineItemStatusUnchecked,
			}, {
				ID:            keepLineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "保留"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status:    domain.ChecklistStatusCreated,
			UpdatedAt: originalUpdatedAt,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.RemoveChecklistLineItems(context.Background(), checklistID.Hex(), request.RemoveChecklistLineItemsInput{
		LineItemIDs: []string{removeLineItemID.Hex()},
	})
	if err != nil {
		t.Fatalf("RemoveChecklistLineItems returned error: %v", err)
	}
	if repo.updated == nil {
		t.Fatalf("expected repository update to be called")
	}
	if len(checklist.Items) != 1 || checklist.Items[0].ID != keepLineItemID {
		t.Fatalf("unexpected remaining items: %+v", checklist.Items)
	}
	if !checklist.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance")
	}
}

func TestRemoveChecklistLineItemsRejectsMissingLineItem(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	_, err := svc.RemoveChecklistLineItems(context.Background(), checklistID.Hex(), request.RemoveChecklistLineItemsInput{
		LineItemIDs: []string{bson.NewObjectID().Hex()},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item not found") {
		t.Fatalf("expected line item not found, got %v", err)
	}
}

func TestUpdateChecklistLineItemStatusChecksItem(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID: checklistID,
			Items: []domain.LineItem{{
				ID:            lineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "护照"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.UpdateChecklistLineItemStatus(context.Background(), checklistID.Hex(), lineItemID.Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusChecked),
	})
	if err != nil {
		t.Fatalf("UpdateChecklistLineItemStatus returned error: %v", err)
	}
	if !repo.statusUpdated || repo.statusLineID != lineItemID || repo.status != domain.LineItemStatusChecked {
		t.Fatalf("expected repository status update, got statusUpdated=%v statusLineID=%s status=%s", repo.statusUpdated, repo.statusLineID.Hex(), repo.status)
	}
	if checklist.Items[0].Status != domain.LineItemStatusChecked {
		t.Fatalf("expected checked status, got %+v", checklist.Items[0])
	}
}

func TestUpdateChecklistLineItemStatusUnchecksItem(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID: checklistID,
			Items: []domain.LineItem{{
				ID:            lineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "护照"},
				Status:        domain.LineItemStatusChecked,
			}},
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	checklist, err := svc.UpdateChecklistLineItemStatus(context.Background(), checklistID.Hex(), lineItemID.Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusUnchecked),
	})
	if err != nil {
		t.Fatalf("UpdateChecklistLineItemStatus returned error: %v", err)
	}
	if checklist.Items[0].Status != domain.LineItemStatusUnchecked {
		t.Fatalf("expected unchecked status, got %+v", checklist.Items[0])
	}
}

func TestUpdateChecklistLineItemStatusRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID().Hex()
	lineItemID := bson.NewObjectID().Hex()
	svc := NewChecklistService(&fakeChecklistRepository{}, validChecklistItemRepository())

	tests := []struct {
		checklistID string
		lineItemID  string
		status      string
	}{
		{checklistID: "bad-checklist-id", lineItemID: lineItemID, status: string(domain.LineItemStatusChecked)},
		{checklistID: checklistID, lineItemID: "bad-line-item-id", status: string(domain.LineItemStatusChecked)},
		{checklistID: checklistID, lineItemID: lineItemID, status: ""},
		{checklistID: checklistID, lineItemID: lineItemID, status: "done"},
	}

	for _, test := range tests {
		_, err := svc.UpdateChecklistLineItemStatus(context.Background(), test.checklistID, test.lineItemID, request.UpdateChecklistLineItemStatusInput{Status: test.status})
		if err == nil {
			t.Fatalf("expected error for input %+v", test)
		}
	}
}

func TestUpdateChecklistLineItemStatusMapsMissingLineItem(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{got: &domain.Checklist{}}, validChecklistItemRepository())

	_, err := svc.UpdateChecklistLineItemStatus(context.Background(), bson.NewObjectID().Hex(), bson.NewObjectID().Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusChecked),
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item not found") {
		t.Fatalf("expected line item not found, got %v", err)
	}
}

func TestUpdateChecklistLineItemStatusWrapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{
		got:       &domain.Checklist{},
		updateErr: errors.New("database down"),
	}, validChecklistItemRepository())

	_, err := svc.UpdateChecklistLineItemStatus(context.Background(), bson.NewObjectID().Hex(), bson.NewObjectID().Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusChecked),
	})
	if err == nil || !strings.Contains(err.Error(), "update checklist line item status failed") {
		t.Fatalf("expected update status failure, got %v", err)
	}
}

func TestDeleteChecklistDeletesByID(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	repo := &fakeChecklistRepository{}
	svc := NewChecklistService(repo, validChecklistItemRepository())

	if err := svc.DeleteChecklist(context.Background(), checklistID.Hex()); err != nil {
		t.Fatalf("DeleteChecklist returned error: %v", err)
	}
	if repo.deleted != checklistID {
		t.Fatalf("expected deleted checklist id %s, got %s", checklistID.Hex(), repo.deleted.Hex())
	}
}

func TestDeleteChecklistMapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{err: errors.New("database down")}, validChecklistItemRepository())

	err := svc.DeleteChecklist(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "delete checklist failed") {
		t.Fatalf("expected delete checklist failure, got %v", err)
	}
}
