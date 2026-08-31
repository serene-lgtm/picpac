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

// AISuggestionService defines AI suggestion behavior.
type AISuggestionService interface {
	RecommendPackItems(ctx context.Context, input request.RecommendPackItemsInput) ([]RecommendedItem, error)
}

// RecommendedItem defines one recommended item summary.
type RecommendedItem struct {
	ID   string
	Name string
}

type aiSuggestionService struct {
	items       ItemService
	categories  CategoryService
	recommender agent.RecommendationPlanner
}

// NewAISuggestionService creates an AI suggestion service.
func NewAISuggestionService(items ItemService, categories CategoryService, recommender agent.RecommendationPlanner) AISuggestionService {
	return &aiSuggestionService{
		items:       items,
		categories:  categories,
		recommender: recommender,
	}
}

// RecommendPackItems recommends existing item IDs for a pack name.
func (s *aiSuggestionService) RecommendPackItems(ctx context.Context, input request.RecommendPackItemsInput) ([]RecommendedItem, error) {
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

func mapCategoryByID(categories []domain.Category) map[bson.ObjectID]domain.Category {
	categoryByID := make(map[bson.ObjectID]domain.Category, len(categories))
	for _, category := range categories {
		categoryByID[category.ID] = category
	}
	return categoryByID
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
