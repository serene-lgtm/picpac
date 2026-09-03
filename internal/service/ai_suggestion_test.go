package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service/agent"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeAIItemService struct {
	items []domain.Item
	err   error
	input request.ListItemsInput
}

func (s *fakeAIItemService) CreateItem(_ context.Context, _ request.CreateItemInput) (*domain.Item, error) {
	return nil, nil
}

func (s *fakeAIItemService) CreateItemsBatch(_ context.Context, _ request.BatchCreateItemsInput) ([]domain.Item, error) {
	return nil, nil
}

func (s *fakeAIItemService) ListItems(_ context.Context, input request.ListItemsInput) ([]domain.Item, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.input = input
	return s.items, nil
}

func (s *fakeAIItemService) GetItem(_ context.Context, _ string, _ string) (*domain.Item, error) {
	return nil, nil
}

func (s *fakeAIItemService) UpdateItem(_ context.Context, _ string, _ string, _ request.UpdateItemInput) (*domain.Item, error) {
	return nil, nil
}

func (s *fakeAIItemService) DeleteItem(_ context.Context, _ string, _ string) error {
	return nil
}

type fakeAICategoryService struct {
	categories []domain.Category
	err        error
}

func (s *fakeAICategoryService) ListCategories(_ context.Context) ([]domain.Category, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.categories, nil
}

func (s *fakeAICategoryService) GetCategory(_ context.Context, _ string) (*domain.Category, error) {
	return nil, nil
}

func (s *fakeAICategoryService) GetCategoryByID(_ context.Context, _ bson.ObjectID) (*domain.Category, error) {
	return nil, nil
}

func (s *fakeAICategoryService) ResolveCategoryID(_ context.Context, _ string) (bson.ObjectID, error) {
	return bson.NilObjectID, nil
}

type fakeRecommendationAgent struct {
	refs   []string
	err    error
	called bool
	input  agent.RecommendationInput
}

type fakeItemDraftExtractionAgent struct {
	drafts []agent.ItemDraft
	err    error
	called bool
	input  agent.ItemDraftExtractionInput
}

func (a *fakeItemDraftExtractionAgent) ExtractItemDrafts(_ context.Context, input agent.ItemDraftExtractionInput) ([]agent.ItemDraft, error) {
	a.called = true
	a.input = input
	if a.err != nil {
		return nil, a.err
	}
	return a.drafts, nil
}

func (a *fakeRecommendationAgent) Recommend(_ context.Context, input agent.RecommendationInput) (*agent.RecommendationResult, error) {
	a.called = true
	a.input = input
	if a.err != nil {
		return nil, a.err
	}
	return &agent.RecommendationResult{Refs: a.refs}, nil
}

func TestAISuggestionRecommendsExistingItems(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID().Hex()
	categoryID := bson.NewObjectID()
	items := make([]domain.Item, 0, 17)
	for index := range 17 {
		items = append(items, domain.Item{
			ID:          bson.NewObjectID(),
			UserID:      bson.NewObjectID(),
			CategoryID:  categoryID,
			Name:        fmt.Sprintf("item %d", index+1),
			Description: "description",
		})
	}
	refs := []string{"i1", "i2", "i2", "i99"}
	for index := 3; index <= 17; index++ {
		refs = append(refs, fmt.Sprintf("i%d", index))
	}
	recommender := &fakeRecommendationAgent{refs: refs}
	svc := NewAISuggestionService(
		&fakeAIItemService{items: items},
		&fakeAICategoryService{categories: []domain.Category{{ID: categoryID, Key: "electronics", Name: "电子设备"}}},
		recommender,
		&fakeItemDraftExtractionAgent{},
	)

	recommendedItems, err := svc.RecommendPackItems(context.Background(), request.RecommendPackItemsInput{
		UserID:      userID,
		PackName:    "日本出差",
		Description: "东京 5 天",
	})
	if err != nil {
		t.Fatalf("RecommendPackItems returned error: %v", err)
	}
	if len(recommendedItems) != defaultPackItemRecommendationLimit {
		t.Fatalf("expected %d recommendations, got %d", defaultPackItemRecommendationLimit, len(recommendedItems))
	}
	if recommendedItems[0].ID != items[0].ID.Hex() || recommendedItems[0].Name != "item 1" {
		t.Fatalf("unexpected first recommendation: %+v", recommendedItems[0])
	}
	if recommendedItems[1].ID != items[1].ID.Hex() || recommendedItems[2].ID != items[2].ID.Hex() {
		t.Fatalf("unexpected recommendation order: %+v", recommendedItems[:3])
	}
	if recommender.input.Limit != defaultPackItemRecommendationLimit {
		t.Fatalf("expected recommender limit %d, got %d", defaultPackItemRecommendationLimit, recommender.input.Limit)
	}
	if recommender.input.Scenario != "pack_item_recommendation" || recommender.input.Subject.Type != "pack" {
		t.Fatalf("unexpected recommender context: %+v", recommender.input)
	}
	if recommender.input.Candidates[0].Ref != "i1" || recommender.input.Candidates[0].Attributes["category_key"] != "electronics" {
		t.Fatalf("unexpected recommender candidate: %+v", recommender.input.Candidates[0])
	}
}

func TestAISuggestionReturnsEmptyWhenUserHasNoItems(t *testing.T) {
	t.Parallel()

	recommender := &fakeRecommendationAgent{}
	svc := NewAISuggestionService(
		&fakeAIItemService{},
		&fakeAICategoryService{},
		recommender,
		&fakeItemDraftExtractionAgent{},
	)

	recommendedItems, err := svc.RecommendPackItems(context.Background(), request.RecommendPackItemsInput{
		UserID:   bson.NewObjectID().Hex(),
		PackName: "周末露营",
	})
	if err != nil {
		t.Fatalf("RecommendPackItems returned error: %v", err)
	}
	if len(recommendedItems) != 0 {
		t.Fatalf("expected empty recommendations, got %+v", recommendedItems)
	}
	if recommender.called {
		t.Fatalf("expected recommender not to be called")
	}
}

func TestAISuggestionRequiresPackName(t *testing.T) {
	t.Parallel()

	svc := NewAISuggestionService(&fakeAIItemService{}, &fakeAICategoryService{}, &fakeRecommendationAgent{}, &fakeItemDraftExtractionAgent{})

	_, err := svc.RecommendPackItems(context.Background(), request.RecommendPackItemsInput{
		UserID: bson.NewObjectID().Hex(),
	})
	if err == nil || !strings.Contains(err.Error(), "pack_name is required") {
		t.Fatalf("expected pack_name error, got %v", err)
	}
}

func TestAISuggestionWrapsItemListError(t *testing.T) {
	t.Parallel()

	svc := NewAISuggestionService(
		&fakeAIItemService{err: errors.New("db down")},
		&fakeAICategoryService{},
		&fakeRecommendationAgent{},
		&fakeItemDraftExtractionAgent{},
	)

	_, err := svc.RecommendPackItems(context.Background(), request.RecommendPackItemsInput{
		UserID:   bson.NewObjectID().Hex(),
		PackName: "日本出差",
	})
	if err == nil || !strings.Contains(err.Error(), "list items failed") {
		t.Fatalf("expected list items failure, got %v", err)
	}
}

func TestAISuggestionWrapsPlannerError(t *testing.T) {
	t.Parallel()

	categoryID := bson.NewObjectID()
	svc := NewAISuggestionService(
		&fakeAIItemService{items: []domain.Item{{ID: bson.NewObjectID(), CategoryID: categoryID, Name: "充电器"}}},
		&fakeAICategoryService{categories: []domain.Category{{ID: categoryID, Key: "electronics", Name: "电子设备"}}},
		&fakeRecommendationAgent{err: errors.New("llm down")},
		&fakeItemDraftExtractionAgent{},
	)

	_, err := svc.RecommendPackItems(context.Background(), request.RecommendPackItemsInput{
		UserID:   bson.NewObjectID().Hex(),
		PackName: "日本出差",
	})
	if err == nil || !strings.Contains(err.Error(), "recommend pack items failed") {
		t.Fatalf("expected planner failure, got %v", err)
	}
}

func TestAISuggestionGeneratesItemDrafts(t *testing.T) {
	t.Parallel()

	electronicsID := bson.NewObjectID()
	otherID := bson.NewObjectID()
	extractor := &fakeItemDraftExtractionAgent{
		drafts: []agent.ItemDraft{
			{Name: " 手机 ", CategoryKey: "electronics"},
			{Name: "手机", CategoryKey: "electronics"},
			{Name: "相机", CategoryKey: "unknown"},
			{Name: "", CategoryKey: "electronics"},
		},
	}
	svc := NewAISuggestionService(
		&fakeAIItemService{},
		&fakeAICategoryService{categories: []domain.Category{
			{ID: electronicsID, Key: "electronics", Name: "电子设备"},
			{ID: otherID, Key: "other", Name: "其他"},
		}},
		&fakeRecommendationAgent{},
		extractor,
	)

	drafts, err := svc.GenerateItemDrafts(context.Background(), request.GenerateItemDraftsInput{
		UserID: bson.NewObjectID().Hex(),
		Text:   "请帮我添加手机，手机，相机",
	})
	if err != nil {
		t.Fatalf("GenerateItemDrafts returned error: %v", err)
	}
	if len(drafts) != 2 {
		t.Fatalf("expected 2 drafts, got %+v", drafts)
	}
	if drafts[0].Name != "手机" || drafts[0].CategoryID != electronicsID.Hex() {
		t.Fatalf("unexpected first draft: %+v", drafts[0])
	}
	if drafts[1].Name != "相机" || drafts[1].CategoryID != otherID.Hex() || drafts[1].CategoryKey != "other" {
		t.Fatalf("expected unknown category to fallback to other, got %+v", drafts[1])
	}
	if !extractor.called || len(extractor.input.Categories) != 2 {
		t.Fatalf("expected extractor to receive categories, got %+v", extractor.input)
	}
}

func TestAISuggestionGenerateItemDraftsRequiresText(t *testing.T) {
	t.Parallel()

	svc := NewAISuggestionService(&fakeAIItemService{}, &fakeAICategoryService{}, &fakeRecommendationAgent{}, &fakeItemDraftExtractionAgent{})

	_, err := svc.GenerateItemDrafts(context.Background(), request.GenerateItemDraftsInput{
		UserID: bson.NewObjectID().Hex(),
	})
	if err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("expected text required error, got %v", err)
	}
}

func TestAISuggestionGenerateItemDraftsWrapsExtractorError(t *testing.T) {
	t.Parallel()

	svc := NewAISuggestionService(
		&fakeAIItemService{},
		&fakeAICategoryService{categories: []domain.Category{{ID: bson.NewObjectID(), Key: "other", Name: "其他"}}},
		&fakeRecommendationAgent{},
		&fakeItemDraftExtractionAgent{err: errors.New("llm down")},
	)

	_, err := svc.GenerateItemDrafts(context.Background(), request.GenerateItemDraftsInput{
		UserID: bson.NewObjectID().Hex(),
		Text:   "请帮我添加手机",
	})
	if err == nil || !strings.Contains(err.Error(), "extract item drafts failed") {
		t.Fatalf("expected extractor failure, got %v", err)
	}
}
