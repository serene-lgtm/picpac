package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// CategoryHandler handles category HTTP requests.
type CategoryHandler struct {
	svc service.CategoryService
}

// NewCategoryHandler creates a category handler.
func NewCategoryHandler(svc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// ListCategories handles category list requests.
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		respondCategoryError(c, err)
		return
	}

	responses := make([]response.CategoryResponse, 0, len(categories))
	for _, category := range categories {
		responses = append(responses, buildCategoryResponse(&category))
	}

	c.JSON(http.StatusOK, response.ListCategoriesResponse{Categories: responses})
}

func respondCategoryError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid category"):
		status = http.StatusBadRequest
	case strings.Contains(message, "category not found"):
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": message})
}

func buildCategoryResponse(category *domain.Category) response.CategoryResponse {
	return response.CategoryResponse{
		ID:   category.ID.Hex(),
		Key:  category.Key,
		Name: category.Name,
	}
}
