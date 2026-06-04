package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ItemHandler handles item HTTP requests.
type ItemHandler struct {
	svc service.ItemService
}

// NewItemHandler creates an item handler.
func NewItemHandler(svc service.ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
}

// CreateItem handles item creation requests.
func (h *ItemHandler) CreateItem(c *gin.Context) {
	userID := strings.TrimSpace(c.PostForm("user_id"))
	name := strings.TrimSpace(c.PostForm("name"))
	description := c.PostForm("description")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	// TODO: Read current user from auth context after user accounts are implemented.
	if !validateOptionalObjectID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is invalid"})
		return
	}

	input := request.CreateItemInput{
		UserID:      userID,
		Name:        name,
		Description: description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.File = file
		input.FileName = header.Filename
	}

	item, err := h.svc.CreateItem(c.Request.Context(), input)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildItemResponse(item))
}

// ListItems handles item list requests.
func (h *ItemHandler) ListItems(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	// TODO: Read current user from auth context after user accounts are implemented.
	if !validateOptionalObjectID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is invalid"})
		return
	}

	q, hasQ := c.GetQuery("q")
	q = strings.TrimSpace(q)
	if hasQ && q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	items, err := h.svc.ListItems(c.Request.Context(), request.ListItemsInput{
		UserID: userID,
		Q:      q,
		HasQ:   hasQ,
	})
	if err != nil {
		respondItemError(c, err)
		return
	}

	responses := make([]response.ItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, buildItemResponse(&item))
	}

	c.JSON(http.StatusOK, response.ListItemsResponse{Items: responses})
}

// GetItem handles item detail requests.
func (h *ItemHandler) GetItem(c *gin.Context) {
	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	item, err := h.svc.GetItem(c.Request.Context(), itemID)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildItemResponse(item))
}

// UpdateItem handles item update requests.
func (h *ItemHandler) UpdateItem(c *gin.Context) {
	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	description := c.PostForm("description")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	input := request.UpdateItemInput{
		Name:        name,
		Description: description,
	}
	if file, header, err := c.Request.FormFile("image"); err == nil {
		defer file.Close()
		input.File = file
		input.FileName = header.Filename
	}

	item, err := h.svc.UpdateItem(c.Request.Context(), itemID, input)
	if err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildItemResponse(item))
}

// DeleteItem handles item deletion requests.
func (h *ItemHandler) DeleteItem(c *gin.Context) {
	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_id is required"})
		return
	}

	if err := h.svc.DeleteItem(c.Request.Context(), itemID); err != nil {
		respondItemError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.DeleteItemResponse{Deleted: true})
}

func respondItemError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid input"),
		strings.Contains(message, "item name is required"),
		strings.Contains(message, "item search keyword is required"),
		strings.Contains(message, "item search keyword is too long"):
		status = http.StatusBadRequest
	case strings.Contains(message, "item not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "upload item image failed"):
		status = http.StatusBadGateway
	case strings.Contains(message, "create item failed"),
		strings.Contains(message, "list items failed"),
		strings.Contains(message, "get item failed"),
		strings.Contains(message, "update item failed"),
		strings.Contains(message, "delete item failed"):
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}

func buildItemResponse(item *domain.Item) response.ItemResponse {
	return response.ItemResponse{
		ID:                 item.ID.Hex(),
		UserID:             item.UserID.Hex(),
		Name:               item.Name,
		Description:        item.Description,
		SourceImageURL:     item.SourceImageURL,
		ImageThumbnailURL:  item.ImageThumbnailURL,
		AIRenderedImageURL: item.AIRenderedImageURL,
		Status:             string(item.Status),
	}
}

func validateOptionalObjectID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	_, err := bson.ObjectIDFromHex(trimmed)
	return err == nil
}
