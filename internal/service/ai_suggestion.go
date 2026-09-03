package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service/agent"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const defaultPackItemRecommendationLimit = 15

const maxRecommendPackNameRunes = 64
const maxRecommendPackDescriptionRunes = 200
const maxItemDraftTextRunes = 500
const maxItemDraftCount = 50

// AISuggestionService defines AI suggestion behavior.
type AISuggestionService interface {
	RecommendPackItems(ctx context.Context, input request.RecommendPackItemsInput) ([]RecommendedItem, error)
	GenerateItemDrafts(ctx context.Context, input request.GenerateItemDraftsInput) ([]ItemDraft, error)
}

// RecommendedItem defines one recommended item summary.
type RecommendedItem struct {
	ID   string
	Name string
}

// ItemDraft defines one AI-extracted item draft.
type ItemDraft struct {
	Name         string
	CategoryID   string
	CategoryKey  string
	CategoryName string
}

type aiSuggestionService struct {
	items       ItemService
	categories  CategoryService
	recommender agent.RecommendationPlanner
	extractor   agent.ItemDraftExtractionAgent
}

// NewAISuggestionService creates an AI suggestion service.
func NewAISuggestionService(items ItemService, categories CategoryService, recommender agent.RecommendationPlanner, extractor agent.ItemDraftExtractionAgent) AISuggestionService {
	return &aiSuggestionService{
		items:       items,
		categories:  categories,
		recommender: recommender,
		extractor:   extractor,
	}
}

// RecommendPackItems recommends existing item summaries for a pack name.
func (s *aiSuggestionService) RecommendPackItems(ctx context.Context, input request.RecommendPackItemsInput) ([]RecommendedItem, error) {
	if s.recommender == nil {
		return nil, fmt.Errorf("recommendation planner is not configured")
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return nil, fmt.Errorf("invalid input")
	}

	packName := strings.TrimSpace(input.PackName)
	if packName == "" {
		return nil, fmt.Errorf("pack_name is required")
	}
	if utf8.RuneCountInString(packName) > maxRecommendPackNameRunes {
		return nil, fmt.Errorf("pack_name is too long")
	}
	description := strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(description) > maxRecommendPackDescriptionRunes {
		return nil, fmt.Errorf("description is too long")
	}

	items, err := s.items.ListItems(ctx, request.ListItemsInput{UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("list items failed: %w", err)
	}
	if len(items) == 0 {
		return []RecommendedItem{}, nil
	}

	categories, err := s.categories.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories failed: %w", err)
	}
	categoryByID := mapCategoryByID(categories)
	candidates, refToItem := buildItemRecommendationCandidates(items, categoryByID)
	if len(candidates) == 0 {
		return []RecommendedItem{}, nil
	}

	result, err := s.recommender.Recommend(ctx, agent.RecommendationInput{
		Scenario: "pack_item_recommendation",
		Subject: agent.RecommendationSubject{
			Type:        "pack",
			Name:        packName,
			Description: description,
		},
		Limit:      defaultPackItemRecommendationLimit,
		Candidates: candidates,
	})
	if err != nil {
		return nil, fmt.Errorf("recommend pack items failed: %w", err)
	}
	if result == nil {
		return []RecommendedItem{}, nil
	}

	return normalizeRecommendedItems(result.Refs, refToItem, defaultPackItemRecommendationLimit), nil
}

// GenerateItemDrafts extracts item drafts from natural language.
func (s *aiSuggestionService) GenerateItemDrafts(ctx context.Context, input request.GenerateItemDraftsInput) ([]ItemDraft, error) {
	if s.extractor == nil {
		return nil, fmt.Errorf("item draft extractor is not configured")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, fmt.Errorf("invalid input")
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if utf8.RuneCountInString(text) > maxItemDraftTextRunes {
		return nil, fmt.Errorf("text is too long")
	}

	categories, err := s.categories.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories failed: %w", err)
	}
	categoryByKey := mapCategoryByKey(categories)
	if _, ok := categoryByKey[defaultCategoryKey]; !ok {
		return nil, fmt.Errorf("default category not found")
	}
	agentCategories := buildItemDraftCategories(categories)

	drafts, err := s.extractor.ExtractItemDrafts(ctx, agent.ItemDraftExtractionInput{
		Text:       text,
		Categories: agentCategories,
	})
	if err != nil {
		return nil, fmt.Errorf("extract item drafts failed: %w", err)
	}

	return normalizeItemDrafts(drafts, categoryByKey, maxItemDraftCount), nil
}

func mapCategoryByID(categories []domain.Category) map[bson.ObjectID]domain.Category {
	categoryByID := make(map[bson.ObjectID]domain.Category, len(categories))
	for _, category := range categories {
		categoryByID[category.ID] = category
	}
	return categoryByID
}

func mapCategoryByKey(categories []domain.Category) map[string]domain.Category {
	categoryByKey := make(map[string]domain.Category, len(categories))
	for _, category := range categories {
		categoryByKey[category.Key] = category
	}
	return categoryByKey
}

func buildItemDraftCategories(categories []domain.Category) []agent.ItemDraftCategory {
	agentCategories := make([]agent.ItemDraftCategory, 0, len(categories))
	for _, category := range categories {
		agentCategories = append(agentCategories, agent.ItemDraftCategory{
			Key:  category.Key,
			Name: category.Name,
		})
	}
	return agentCategories
}

func normalizeItemDrafts(drafts []agent.ItemDraft, categoryByKey map[string]domain.Category, limit int) []ItemDraft {
	normalized := make([]ItemDraft, 0, len(drafts))
	seen := make(map[string]struct{}, len(drafts))
	defaultCategory := categoryByKey[defaultCategoryKey]
	for _, draft := range drafts {
		name := strings.TrimSpace(draft.Name)
		if name == "" {
			continue
		}
		key := normalizeItemDraftNameKey(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		category, ok := categoryByKey[strings.TrimSpace(draft.CategoryKey)]
		if !ok {
			category = defaultCategory
		}
		normalized = append(normalized, ItemDraft{
			Name:         name,
			CategoryID:   category.ID.Hex(),
			CategoryKey:  category.Key,
			CategoryName: category.Name,
		})
		if len(normalized) == limit {
			break
		}
	}
	return normalized
}

func normalizeItemDraftNameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), ""))
}

func buildItemRecommendationCandidates(items []domain.Item, categoryByID map[bson.ObjectID]domain.Category) ([]agent.RecommendationCandidate, map[string]RecommendedItem) {
	candidates := make([]agent.RecommendationCandidate, 0, len(items))
	refToItem := make(map[string]RecommendedItem, len(items))
	for index, item := range items {
		ref := fmt.Sprintf("i%d", index+1)
		category := categoryByID[item.CategoryID]
		candidates = append(candidates, agent.RecommendationCandidate{
			Ref:         ref,
			Name:        item.Name,
			Description: item.Description,
			Attributes: map[string]string{
				"category_key":  category.Key,
				"category_name": category.Name,
			},
		})
		refToItem[ref] = RecommendedItem{
			ID:   item.ID.Hex(),
			Name: item.Name,
		}
	}
	return candidates, refToItem
}

func normalizeRecommendedItems(itemRefs []string, refToItem map[string]RecommendedItem, limit int) []RecommendedItem {
	recommendedItems := make([]RecommendedItem, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, ref := range itemRefs {
		ref = strings.TrimSpace(ref)
		item, ok := refToItem[ref]
		if !ok {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		recommendedItems = append(recommendedItems, item)
		if len(recommendedItems) == limit {
			break
		}
	}
	return recommendedItems
}
