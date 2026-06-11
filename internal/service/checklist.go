package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var maxChecklistSearchKeywordRunes = 50

const checklistTargetDateLayout = "2006-01-02"

// ChecklistService defines checklist behavior.
type ChecklistService interface {
	CreateChecklist(ctx context.Context, input request.CreateChecklistInput) (*domain.Checklist, error)
	ListChecklists(ctx context.Context, input request.ListChecklistsInput) ([]domain.Checklist, error)
	GetChecklist(ctx context.Context, checklistID string) (*domain.Checklist, error)
	UpdateChecklist(ctx context.Context, checklistID string, input request.UpdateChecklistInput) (*domain.Checklist, error)
	AddChecklistLineItems(ctx context.Context, checklistID string, input request.AddChecklistLineItemsInput) (*domain.Checklist, error)
	RemoveChecklistLineItems(ctx context.Context, checklistID string, input request.RemoveChecklistLineItemsInput) (*domain.Checklist, error)
	UpdateChecklistLineItemStatus(ctx context.Context, checklistID string, lineItemID string, input request.UpdateChecklistLineItemStatusInput) (*domain.Checklist, error)
	DeleteChecklist(ctx context.Context, checklistID string) error
}

type checklistService struct {
	repo     repository.ChecklistRepository
	itemRepo repository.ItemRepository
}

// NewChecklistService creates a checklist service.
func NewChecklistService(repo repository.ChecklistRepository, itemRepo repository.ItemRepository) ChecklistService {
	return &checklistService{repo: repo, itemRepo: itemRepo}
}

// CreateChecklist creates a new checklist.
func (s *checklistService) CreateChecklist(ctx context.Context, input request.CreateChecklistInput) (*domain.Checklist, error) {
	// TODO: Read and verify checklist owner from auth context after user accounts are implemented.
	userID, err := parseOptionalObjectID(input.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("checklist name is required")
	}

	targetDate, err := parseChecklistTargetDate(input.TargetDate)
	if err != nil {
		return nil, err
	}

	items, err := s.newChecklistLineItems(ctx, input.Items)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	checklist := &domain.Checklist{
		ID:          bson.NewObjectID(),
		UserID:      userID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		TargetDate:  targetDate,
		Items:       items,
		Status:      domain.ChecklistStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, checklist); err != nil {
		return nil, fmt.Errorf("create checklist failed: %w", err)
	}

	return checklist, nil
}

// ListChecklists lists all checklists or searches checklists by keyword.
func (s *checklistService) ListChecklists(ctx context.Context, input request.ListChecklistsInput) ([]domain.Checklist, error) {
	var (
		checklists []domain.Checklist
		err        error
	)
	if input.HasQ {
		checklists, err = s.searchChecklistsByKeyword(ctx, input.UserID, input.Q)
	} else {
		checklists, err = s.listChecklists(ctx, input.UserID)
	}
	if err != nil {
		return nil, fmt.Errorf("list checklists failed: %w", err)
	}

	return checklists, nil
}

func (s *checklistService) listChecklists(ctx context.Context, userID string) ([]domain.Checklist, error) {
	// TODO: Filter by authenticated user after user accounts/auth are implemented.
	if strings.TrimSpace(userID) == "" {
		return s.repo.ListAll(ctx)
	}

	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.ListByUserID(ctx, objectID)
}

func (s *checklistService) searchChecklistsByKeyword(ctx context.Context, userID string, keyword string) ([]domain.Checklist, error) {
	// TODO: Filter by authenticated user after user accounts/auth are implemented.
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("checklist search keyword is required")
	}
	if utf8.RuneCountInString(keyword) > maxChecklistSearchKeywordRunes {
		return nil, fmt.Errorf("checklist search keyword is too long")
	}

	if strings.TrimSpace(userID) == "" {
		return s.repo.SearchByKeyword(ctx, keyword)
	}

	objectID, err := parseObjectID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	return s.repo.SearchByKeywordAndUserID(ctx, objectID, keyword)
}

// GetChecklist gets a single checklist by ID.
func (s *checklistService) GetChecklist(ctx context.Context, checklistID string) (*domain.Checklist, error) {
	objectID, err := parseObjectID(checklistID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	checklist, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("get checklist failed: %w", err)
	}
	if checklist.Status == domain.ChecklistStatusDeleted {
		return nil, fmt.Errorf("checklist not found")
	}

	return checklist, nil
}

// UpdateChecklist updates checklist metadata.
func (s *checklistService) UpdateChecklist(ctx context.Context, checklistID string, input request.UpdateChecklistInput) (*domain.Checklist, error) {
	objectID, err := parseObjectID(checklistID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	checklist, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("get checklist failed: %w", err)
	}
	if checklist.Status == domain.ChecklistStatusDeleted {
		return nil, fmt.Errorf("checklist not found")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("checklist name is required")
	}

	targetDate, err := parseChecklistTargetDate(input.TargetDate)
	if err != nil {
		return nil, err
	}

	checklist.Name = name
	checklist.Description = strings.TrimSpace(input.Description)
	checklist.TargetDate = targetDate
	checklist.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, checklist); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("update checklist failed: %w", err)
	}

	return checklist, nil
}

// AddChecklistLineItems adds line items to an existing checklist.
func (s *checklistService) AddChecklistLineItems(ctx context.Context, checklistID string, input request.AddChecklistLineItemsInput) (*domain.Checklist, error) {
	checklist, err := s.getEditableChecklist(ctx, checklistID)
	if err != nil {
		return nil, err
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("checklist line items are required")
	}

	items, err := s.newChecklistLineItems(ctx, input.Items)
	if err != nil {
		return nil, err
	}

	checklist.Items = append(checklist.Items, items...)
	checklist.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, checklist); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("update checklist failed: %w", err)
	}

	return checklist, nil
}

// RemoveChecklistLineItems removes line items from an existing checklist.
func (s *checklistService) RemoveChecklistLineItems(ctx context.Context, checklistID string, input request.RemoveChecklistLineItemsInput) (*domain.Checklist, error) {
	checklist, err := s.getEditableChecklist(ctx, checklistID)
	if err != nil {
		return nil, err
	}
	if len(input.LineItemIDs) == 0 {
		return nil, fmt.Errorf("checklist line item ids are required")
	}

	lineItemIDs, err := parseChecklistLineItemIDs(input.LineItemIDs)
	if err != nil {
		return nil, err
	}

	remainingItems, err := removeChecklistLineItems(checklist.Items, lineItemIDs)
	if err != nil {
		return nil, err
	}

	checklist.Items = remainingItems
	checklist.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, checklist); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("update checklist failed: %w", err)
	}

	return checklist, nil
}

// UpdateChecklistLineItemStatus updates a single checklist line item status.
func (s *checklistService) UpdateChecklistLineItemStatus(ctx context.Context, checklistID string, lineItemID string, input request.UpdateChecklistLineItemStatusInput) (*domain.Checklist, error) {
	checklistObjectID, err := parseObjectID(checklistID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}
	lineItemObjectID, err := parseObjectID(lineItemID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	status, err := parseChecklistLineItemStatus(input.Status)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateLineItemStatus(ctx, checklistObjectID, lineItemObjectID, status, time.Now().UTC()); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist line item not found")
		}
		return nil, fmt.Errorf("update checklist line item status failed: %w", err)
	}

	checklist, err := s.repo.GetByID(ctx, checklistObjectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("get checklist failed: %w", err)
	}

	return checklist, nil
}

// DeleteChecklist logically deletes a checklist by ID.
func (s *checklistService) DeleteChecklist(ctx context.Context, checklistID string) error {
	objectID, err := parseObjectID(checklistID)
	if err != nil {
		return fmt.Errorf("invalid input")
	}

	if err := s.repo.DeleteByID(ctx, objectID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("checklist not found")
		}
		return fmt.Errorf("delete checklist failed: %w", err)
	}

	return nil
}

func (s *checklistService) getEditableChecklist(ctx context.Context, checklistID string) (*domain.Checklist, error) {
	objectID, err := parseObjectID(checklistID)
	if err != nil {
		return nil, fmt.Errorf("invalid input")
	}

	checklist, err := s.repo.GetByID(ctx, objectID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("checklist not found")
		}
		return nil, fmt.Errorf("get checklist failed: %w", err)
	}
	if checklist.Status == domain.ChecklistStatusDeleted {
		return nil, fmt.Errorf("checklist not found")
	}

	return checklist, nil
}

func parseChecklistTargetDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("checklist target_date is required")
	}

	targetDate, err := time.Parse(checklistTargetDateLayout, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid input")
	}

	return targetDate, nil
}

func parseChecklistLineItemStatus(value string) (domain.LineItemStatus, error) {
	status := domain.LineItemStatus(strings.TrimSpace(value))
	if status == "" {
		return "", fmt.Errorf("checklist line item status is required")
	}
	switch status {
	case domain.LineItemStatusChecked, domain.LineItemStatusUnchecked:
		return status, nil
	default:
		return "", fmt.Errorf("checklist line item status is invalid")
	}
}

func (s *checklistService) newChecklistLineItems(ctx context.Context, inputs []request.ChecklistLineItemInput) ([]domain.LineItem, error) {
	if len(inputs) == 0 {
		return []domain.LineItem{}, nil
	}

	items := make([]domain.LineItem, 0, len(inputs))
	for _, input := range inputs {
		lineItem, err := s.newChecklistLineItem(ctx, input)
		if err != nil {
			return nil, err
		}
		items = append(items, lineItem)
	}

	return items, nil
}

func (s *checklistService) newChecklistLineItem(ctx context.Context, input request.ChecklistLineItemInput) (domain.LineItem, error) {
	referenceType := domain.LineItemType(strings.TrimSpace(input.ReferenceType))
	referenceIDValue := strings.TrimSpace(input.ReferenceID)

	lineItem := domain.LineItem{
		ID:            bson.NewObjectID(),
		ReferenceType: referenceType,
		Status:        domain.LineItemStatusUnchecked,
	}

	switch referenceType {
	case domain.LineItemTypeItem:
		if input.Snapshot != nil {
			return domain.LineItem{}, fmt.Errorf("invalid input")
		}
		referenceID, err := parseObjectID(referenceIDValue)
		if err != nil {
			return domain.LineItem{}, fmt.Errorf("invalid input")
		}
		if err := s.validateLineItemReferenceItem(ctx, referenceID); err != nil {
			return domain.LineItem{}, err
		}
		lineItem.ReferenceID = referenceID
	case domain.LineItemTypeSnapshot:
		if referenceIDValue != "" {
			return domain.LineItem{}, fmt.Errorf("invalid input")
		}
		if input.Snapshot == nil {
			return domain.LineItem{}, fmt.Errorf("invalid input")
		}
		snapshotName := strings.TrimSpace(input.Snapshot.Name)
		if snapshotName == "" {
			return domain.LineItem{}, fmt.Errorf("invalid input")
		}
		lineItem.Snapshot = &domain.ItemSnapshot{Name: snapshotName}
	default:
		return domain.LineItem{}, fmt.Errorf("invalid input")
	}

	return lineItem, nil
}

func (s *checklistService) validateLineItemReferenceItem(ctx context.Context, itemID bson.ObjectID) error {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("checklist line item reference item not found")
		}
		return fmt.Errorf("get checklist line item reference item failed: %w", err)
	}
	if item.Status == domain.ItemStatusDeleted {
		return fmt.Errorf("checklist line item reference item not found")
	}

	return nil
}

func parseChecklistLineItemIDs(values []string) (map[bson.ObjectID]struct{}, error) {
	lineItemIDs := make(map[bson.ObjectID]struct{}, len(values))
	for _, value := range values {
		lineItemID, err := parseObjectID(value)
		if err != nil {
			return nil, fmt.Errorf("invalid input")
		}
		lineItemIDs[lineItemID] = struct{}{}
	}

	return lineItemIDs, nil
}

func removeChecklistLineItems(items []domain.LineItem, lineItemIDs map[bson.ObjectID]struct{}) ([]domain.LineItem, error) {
	remainingItems := make([]domain.LineItem, 0, len(items))
	removedIDs := make(map[bson.ObjectID]struct{}, len(lineItemIDs))

	for _, item := range items {
		if _, ok := lineItemIDs[item.ID]; ok {
			removedIDs[item.ID] = struct{}{}
			continue
		}
		remainingItems = append(remainingItems, item)
	}
	if len(removedIDs) != len(lineItemIDs) {
		return nil, fmt.Errorf("checklist line item not found")
	}

	return remainingItems, nil
}
