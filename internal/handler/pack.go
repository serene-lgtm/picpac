package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// PackHandler handles pack HTTP requests.
type PackHandler struct {
	svc service.PackService
}

// NewPackHandler creates a pack handler.
func NewPackHandler(svc service.PackService) *PackHandler {
	return &PackHandler{svc: svc}
}

// CreatePack handles pack creation requests.
func (h *PackHandler) CreatePack(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	var input request.CreatePackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if !validateObjectIDs(input.Items) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items contains invalid item_id"})
		return
	}
	input.UserID = userID

	pack, err := h.svc.CreatePack(c.Request.Context(), input)
	if err != nil {
		respondPackError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildPackResponse(pack))
}

// ListPacks handles pack list requests.
func (h *PackHandler) ListPacks(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	q, hasQ := c.GetQuery("q")
	q = strings.TrimSpace(q)
	if hasQ && q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	packs, err := h.svc.ListPacks(c.Request.Context(), request.ListPacksInput{
		UserID: userID,
		Q:      q,
		HasQ:   hasQ,
	})
	if err != nil {
		respondPackError(c, err)
		return
	}

	responses := make([]response.PackResponse, 0, len(packs))
	for _, pack := range packs {
		responses = append(responses, buildPackResponse(&pack))
	}

	c.JSON(http.StatusOK, response.ListPacksResponse{Packs: responses})
}

// GetPack handles pack detail requests.
func (h *PackHandler) GetPack(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	packID := strings.TrimSpace(c.Param("pack_id"))
	if packID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is required"})
		return
	}
	if !validateRequiredObjectID(packID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is invalid"})
		return
	}

	pack, err := h.svc.GetPack(c.Request.Context(), packID, userID)
	if err != nil {
		respondPackError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildPackResponse(pack))
}

// UpdatePack handles pack update requests.
func (h *PackHandler) UpdatePack(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	packID := strings.TrimSpace(c.Param("pack_id"))
	if packID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is required"})
		return
	}
	if !validateRequiredObjectID(packID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is invalid"})
		return
	}

	var input request.UpdatePackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if !validateObjectIDs(input.Items) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items contains invalid item_id"})
		return
	}

	pack, err := h.svc.UpdatePack(c.Request.Context(), packID, userID, input)
	if err != nil {
		respondPackError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildPackResponse(pack))
}

// DeletePack handles pack deletion requests.
func (h *PackHandler) DeletePack(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
		return
	}

	packID := strings.TrimSpace(c.Param("pack_id"))
	if packID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is required"})
		return
	}
	if !validateRequiredObjectID(packID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pack_id is invalid"})
		return
	}

	if err := h.svc.DeletePack(c.Request.Context(), packID, userID); err != nil {
		respondPackError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.DeletePackResponse{Deleted: true})
}

func respondPackError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid input"),
		strings.Contains(message, "pack name is required"),
		strings.Contains(message, "pack search keyword is required"),
		strings.Contains(message, "pack search keyword is too long"):
		status = http.StatusBadRequest
	case strings.Contains(message, "pack not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "pack item not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "create pack failed"),
		strings.Contains(message, "list packs failed"),
		strings.Contains(message, "get pack failed"),
		strings.Contains(message, "update pack failed"),
		strings.Contains(message, "delete pack failed"):
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}

func buildPackResponse(pack *domain.Pack) response.PackResponse {
	items := make([]string, 0, len(pack.Items))
	for _, itemID := range pack.Items {
		items = append(items, itemID.Hex())
	}

	return response.PackResponse{
		ID:          pack.ID.Hex(),
		UserID:      pack.UserID.Hex(),
		Name:        pack.Name,
		Description: pack.Description,
		Items:       items,
		Status:      string(pack.Status),
	}
}

func validateObjectIDs(values []string) bool {
	for _, value := range values {
		if !validateRequiredObjectID(value) {
			return false
		}
	}
	return true
}

func validateRequiredObjectID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	return validateOptionalObjectID(trimmed)
}
