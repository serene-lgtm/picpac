package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// AIHandler handles AI suggestion HTTP requests.
type AIHandler struct {
	svc service.AISuggestionService
}

// NewAIHandler creates an AI handler.
func NewAIHandler(svc service.AISuggestionService) *AIHandler {
	return &AIHandler{svc: svc}
}

// RecommendPackItems handles pack item recommendation requests.
func (h *AIHandler) RecommendPackItems(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	var input request.RecommendPackItemsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	input.PackName = strings.TrimSpace(input.PackName)
	input.Description = strings.TrimSpace(input.Description)
	if input.PackName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_name is required"})
		return
	}
	input.UserID = userID

	recommendedItems, err := h.svc.RecommendPackItems(c.Request.Context(), input)
	if err != nil {
		respondAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.RecommendPackItemsResponse{RecommendedItems: buildRecommendedItemResponses(recommendedItems)})
}

func respondAIError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "required"),
		strings.Contains(message, "invalid input"),
		strings.Contains(message, "too long"):
		status = http.StatusBadRequest
	}

	c.JSON(status, gin.H{"error": message})
}

func buildRecommendedItemResponses(items []service.RecommendedItem) []response.RecommendedItemResponse {
	responses := make([]response.RecommendedItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, response.RecommendedItemResponse{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	return responses
}
